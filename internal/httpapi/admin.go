package httpapi

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"surgalt/internal/auth"
	"surgalt/internal/files"
	"surgalt/internal/store"
)

// ---- Хянагчийн самбар ----
// Тусдаа (холбоосгүй) замаар (ADMIN_PATH) нээгддэг; нууц үгээр (ADMIN_LOGIN + ADMIN_PASSWORD_HASH) эсвэл
// ADMIN_EMAILS-ийн Google/Microsoft бүртгэлээр нэвтэрнэ. Сайтын хандалт (өдөр, 7 хоног, сар), багш нарын дэлгэрэнгүй
// (агуулга, орлого, татсан дүн), суралцагчид, гүйлгээний жагсаалт, орлого-зарлагын нэгтгэл.

const (
	adminCookie = "sg_adm"
	adminTTL    = 12 * time.Hour
)

// adminPathRe — ADMIN_PATH: нэг түвшний зам (жишээ нь /sg-control-7f3a).
var adminPathRe = regexp.MustCompile(`^/[A-Za-z0-9_-]{3,64}$`)

// reservedPaths — сайтын бусад хуудастай давхцах ёсгүй.
var reservedPaths = map[string]bool{"/login": true, "/privacy": true, "/terms": true, "/intro2": true, "/me": true,
	"/studio": true, "/reset": true, "/healthz": true}

// ValidAdminPath — main.go-д ADMIN_PATH-ыг шалгахад.
func ValidAdminPath(p string) bool {
	return adminPathRe.MatchString(p) && !reservedPaths[strings.ToLower(p)]
}

func (s *Server) adminPath() string {
	if s.cfg.AdminPath != "" {
		return s.cfg.AdminPath
	}
	return "/admin"
}

func (s *Server) adminPassOn() bool { return s.cfg.AdminLogin != "" && s.cfg.AdminPassHash != "" }

type adminSess struct {
	L string `json:"l"`
	H string `json:"h"` // нууц үгийн hash-ийн MAC: нууц үг солигдвол хуучин нэвтрэлт хүчингүй болно
}

func (s *Server) adminHashTag() string {
	return s.tokens.MAC("admin-pass", s.cfg.AdminLogin+"|"+s.cfg.AdminPassHash)
}

// adminCookieOK — нууц үгээр нэвтэрсэн хянагчийн cookie хүчинтэй эсэх.
func (s *Server) adminCookieOK(r *http.Request) bool {
	if !s.adminPassOn() {
		return false
	}
	c, err := r.Cookie(adminCookie)
	if err != nil || c.Value == "" {
		return false
	}
	var v adminSess
	if s.tokens.Open(c.Value, &v) != nil {
		return false
	}
	return constantEq(v.L, s.cfg.AdminLogin) && constantEq(v.H, s.adminHashTag())
}

// sameOrigin — cookie-гоор баталгаажих өөрчлөх хүсэлт өөр сайтаас ирээгүйг шалгана (SameSite=Strict-ийн нэмэлт хамгаалалт).
func (s *Server) sameOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	if o == "" {
		sfs := r.Header.Get("Sec-Fetch-Site")
		return sfs == "" || sfs == "same-origin"
	}
	u, err := url.Parse(o)
	if err != nil || u.Host == "" {
		return false
	}
	hosts := []string{r.Host}
	if pu, err := url.Parse(s.cfg.PublicURL); err == nil && pu.Host != "" {
		hosts = append(hosts, pu.Host)
	}
	if s.cfg.TrustProxy {
		if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
			hosts = append(hosts, fh)
		}
	}
	for _, h := range hosts {
		if strings.EqualFold(u.Host, h) {
			return true
		}
	}
	return false
}

func (s *Server) isHTTPS(r *http.Request) bool {
	return r.TLS != nil || (s.cfg.TrustProxy && r.Header.Get("X-Forwarded-Proto") == "https") ||
		(strings.HasPrefix(s.cfg.PublicURL, "https://") && !strings.HasPrefix(r.Host, "127.0.0.1") && !strings.HasPrefix(r.Host, "localhost"))
}

// pageAdmin: GET <ADMIN_PATH> — самбарын хуудас (хайлтад гарахгүй, кэшлэгдэхгүй, замаа referrer-ээр алдахгүй).
func (s *Server) pageAdmin(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Robots-Tag", "noindex, nofollow")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Frame-Options", "DENY")
	if err := s.tmpl.ExecuteTemplate(w, "admin.html", map[string]any{"PublicURL": s.cfg.PublicURL}); err != nil {
		s.log.Error("template", "err", err, "page", "admin")
	}
}

// handleAdminLogin: POST /api/admin/login {login, password}
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	if !s.adminPassOn() {
		writeErr(w, http.StatusNotFound, "нууц үгээр нэвтрэх тохируулаагүй (ADMIN_LOGIN, ADMIN_PASSWORD_HASH)")
		return
	}
	if !s.sameOrigin(r) {
		writeErr(w, http.StatusForbidden, "хүсэлтийн эх сурвалж буруу")
		return
	}
	ip := s.clientIP(r)
	if !s.admLim.Allow(ip) {
		w.Header().Set("Retry-After", "60")
		writeErr(w, http.StatusTooManyRequests, "олон удаа оролдлоо — 1 минутын дараа дахин оролдоно уу")
		return
	}
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	ok, err := auth.CheckPassword(r.Context(), s.cfg.AdminPassHash, in.Password)
	if err != nil {
		writeErr(w, http.StatusServiceUnavailable, "түр завгүй байна")
		return
	}
	if !ok || !constantEq(strings.TrimSpace(in.Login), s.cfg.AdminLogin) {
		s.log.Warn("хянагчийн нэвтрэлт амжилтгүй", "ip", ip)
		writeErr(w, http.StatusUnauthorized, "нэвтрэх нэр эсвэл нууц үг буруу")
		return
	}
	tok, err := s.tokens.Seal(adminSess{L: s.cfg.AdminLogin, H: s.adminHashTag()}, adminTTL)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "дотоод алдаа")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: adminCookie, Value: tok, Path: "/api/admin", MaxAge: int(adminTTL.Seconds()),
		HttpOnly: true, Secure: s.isHTTPS(r), SameSite: http.SameSiteStrictMode})
	s.log.Info("хянагч нэвтэрлээ", "ip", ip)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "login": s.cfg.AdminLogin, "via": "password"})
}

// handleAdminLogout: POST /api/admin/logout
func (s *Server) handleAdminLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: adminCookie, Value: "", Path: "/api/admin", MaxAge: -1, HttpOnly: true,
		Secure: s.isHTTPS(r), SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}

// handleAdminMe: GET /api/admin/me — хянагчаар нэвтэрсэн эсэх (самбар нэвтрэх хэсгээ харуулах эсэхээ шийднэ).
func (s *Server) handleAdminMe(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	if s.adminCookieOK(r) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "login": s.cfg.AdminLogin, "via": "password"})
		return
	}
	if c, ok := s.principal(r); ok && !c.IsGuest() {
		if u, err := s.store.UserByID(r.Context(), c.UID); err == nil && s.isAdmin(u) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "login": u.Email, "via": "oauth"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": false, "password": s.adminPassOn()})
}

// ---- сайтын хандалт ----

type hitBuf struct {
	mu sync.Mutex
	hs []store.SiteHit
}

var hitPathRe = regexp.MustCompile(`^/[A-Za-z0-9/_.~%-]*$`)

// handleHit: POST /api/hit {p, vid, ref} — хөтөч сайтын аль нэг хуудсыг нээхэд (JS ажилласан жинхэнэ зочин л тоологдоно).
func (s *Server) handleHit(w http.ResponseWriter, r *http.Request) {
	var in struct {
		P   string `json:"p"`
		VID string `json:"vid"`
		Ref string `json:"ref"`
	}
	if !decode(w, r, &in) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
	ua := r.UserAgent()
	p, _, _ := strings.Cut(in.P, "?")
	if ua == "" || botRe.MatchString(ua) || !hitPathRe.MatchString(p) || strings.EqualFold(p, s.adminPath()) {
		return
	}
	if utf8.RuneCountInString(p) > 120 {
		p = string([]rune(p)[:120])
	}
	h := store.SiteHit{At: time.Now(), Path: p, Referrer: refName(in.Ref), Device: deviceOf(ua)}
	if c, ok := s.principal(r); ok && !c.IsGuest() {
		h.UserID = c.UID
	}
	// Давтагдаагүй зочныг хөтчөөр нь тоолно (нэвтрэхээс өмнө ба дараа нь нэг хүн хоёр тоологдохгүй).
	switch vid := cleanVID(in.VID); {
	case vid != "":
		h.VisitorID = "a:" + vid
	case h.UserID != "":
		h.VisitorID = "u:" + h.UserID
	default:
		h.VisitorID = "ip:" + s.tokens.MAC("ip", s.clientIP(r))
	}
	s.hits.mu.Lock()
	s.hits.hs = append(s.hits.hs, h)
	n := len(s.hits.hs)
	s.hits.mu.Unlock()
	if n >= 500 {
		go s.flushHits(context.Background())
	}
}

// flushHits — хуримтлагдсан хандалтыг нэг багцаар хадгална (ClickHouse-д мөр бүрээр бичихгүй).
func (s *Server) flushHits(ctx context.Context) {
	s.hits.mu.Lock()
	hs := s.hits.hs
	s.hits.hs = nil
	s.hits.mu.Unlock()
	if len(hs) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := s.store.AddSiteHits(ctx, hs); err != nil {
		s.log.Warn("сайтын хандалт хадгалах", "err", err, "n", len(hs))
		s.hits.mu.Lock()
		if len(s.hits.hs) < 20000 { // сан түр унасан бол дараагийн удаа дахин оролдоно
			s.hits.hs = append(hs, s.hits.hs...)
		}
		s.hits.mu.Unlock()
	}
}

func (s *Server) hitLoop(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.flushHits(context.Background())
			return
		case <-t.C:
			s.flushHits(ctx)
		}
	}
}

// ---- платформын агшин зураг (20 секунд кэшлэнэ) ----

type fileStat struct {
	Count int   `json:"count"`
	Bytes int64 `json:"bytes"`
}

type adminSnap struct {
	at      time.Time
	users   []store.User
	user    map[string]*store.User
	courses []store.Course
	course  map[string]*store.Course
	books   []store.Book
	orders  []store.Order
	enrolls []store.Enrollment
	payouts []store.Payout
	files   map[string]fileStat
}

type adminCache struct {
	mu   sync.Mutex
	snap *adminSnap
}

func (s *Server) adminInvalidate() {
	s.adm.mu.Lock()
	s.adm.snap = nil
	s.adm.mu.Unlock()
}

func (s *Server) adminSnapshot(ctx context.Context) (*adminSnap, error) {
	s.adm.mu.Lock()
	defer s.adm.mu.Unlock()
	if s.adm.snap != nil && time.Since(s.adm.snap.at) < 20*time.Second {
		return s.adm.snap, nil
	}
	sn := &adminSnap{at: time.Now()}
	var wg sync.WaitGroup
	errs := make([]error, 6)
	run := func(i int, f func() error) {
		wg.Add(1)
		go func() { defer wg.Done(); errs[i] = f() }()
	}
	run(0, func() (err error) { sn.users, err = s.store.AdminUsers(ctx); return })
	run(1, func() (err error) { sn.courses, err = s.store.AdminCourses(ctx); return })
	run(2, func() (err error) { sn.books, err = s.store.AdminBooks(ctx); return })
	run(3, func() (err error) { sn.orders, err = s.store.AdminOrders(ctx, time.Time{}); return })
	run(4, func() (err error) { sn.enrolls, err = s.store.AdminEnrollments(ctx); return })
	run(5, func() (err error) { sn.payouts, err = s.store.Payouts(ctx, "", "", 1_000_000); return })
	wg.Wait()
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	sn.user = make(map[string]*store.User, len(sn.users))
	for i := range sn.users {
		sn.user[sn.users[i].ID] = &sn.users[i]
	}
	sn.course = make(map[string]*store.Course, len(sn.courses))
	for i := range sn.courses {
		sn.course[sn.courses[i].ID] = &sn.courses[i]
	}
	// Багш бүрийн файлын сан (дискнээс; зэрэг 8-аар).
	sn.files = map[string]fileStat{}
	if s.files != nil {
		var mu sync.Mutex
		sem := make(chan struct{}, 8)
		for _, u := range sn.users {
			if u.Role != store.RoleTeacher {
				continue
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(id string) {
				defer func() { <-sem; wg.Done() }()
				var st fileStat
				if l, err := s.files.List(id); err == nil {
					st.Count = len(l)
				}
				st.Bytes, _ = s.files.Usage(id)
				mu.Lock()
				sn.files[id] = st
				mu.Unlock()
			}(u.ID)
		}
		wg.Wait()
	}
	s.adm.snap = sn
	return sn, nil
}

// ---- туслах ----

var orderKindLabel = map[string]string{
	store.OrderKindCourse: "Сургалт", store.OrderKindLesson: "Хичээл", store.OrderKindStorage: "Файлын багтаамж",
	store.OrderKindLate: "Хоцролтын төлбөр", store.OrderKindFee: "Оролцооны төлбөр", store.OrderKindMeeting: "Шууд хичээл",
	store.OrderKindBook: "Ном", store.OrderKindSlot: "Уулзалтын цаг",
}

func kindLabel(k string) string {
	if l, ok := orderKindLabel[k]; ok {
		return l
	}
	return k
}

func dayStart(t time.Time) time.Time {
	t = t.In(time.Local)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

func localDay(t time.Time) string   { return t.In(time.Local).Format("2006-01-02") }
func localMonth(t time.Time) string { return t.In(time.Local).Format("2006-01") }

type personRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

func (sn *adminSnap) person(id string) personRef {
	if u := sn.user[id]; u != nil {
		n := u.DisplayName
		if n == "" {
			n = u.Username
		}
		return personRef{ID: id, Name: n, Username: u.Username}
	}
	if id == "" {
		return personRef{}
	}
	return personRef{ID: id, Name: "Устгагдсан бүртгэл"}
}

func paidAt(o *store.Order) time.Time {
	if o.PaidAt != nil {
		return *o.PaidAt
	}
	return o.CreatedAt
}

func intParam(r *http.Request, key string, def, lo, hi int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || n < lo || n > hi {
		return def
	}
	return n
}

// ---- Тойм ----

// handleAdminOverview: GET /api/admin/overview?days=30
func (s *Server) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	ctx := r.Context()
	days := intParam(r, "days", 30, 7, 366)
	now := time.Now()
	today := dayStart(now)
	since := today.AddDate(0, 0, -(days - 1))
	_, off := now.Zone()
	s.flushHits(ctx)
	tr, err := s.store.SiteTraffic(ctx, store.TrafficQuery{Since: since, Today: today, Week: today.AddDate(0, 0, -6),
		Month: today.AddDate(0, 0, -29), Online: now.Add(-5 * time.Minute), TZOffset: off})
	if s.storeErr(w, r, err) {
		return
	}
	sn, err := s.adminSnapshot(ctx)
	if s.storeErr(w, r, err) {
		return
	}
	type dayRow struct {
		Day     string `json:"day"`
		Views   int64  `json:"views"`
		Unique  int64  `json:"unique"`
		Members int64  `json:"members"`
		Signups int    `json:"signups"`
		Revenue int64  `json:"revenue"`
		Orders  int    `json:"orders"`
	}
	idx := map[string]int{}
	daily := make([]dayRow, days)
	for i := range daily {
		k := since.AddDate(0, 0, i).Format("2006-01-02")
		daily[i].Day, idx[k] = k, i
	}
	for _, d := range tr.Daily {
		if i, ok := idx[d.Day]; ok {
			daily[i].Views, daily[i].Unique, daily[i].Members = d.Views, d.Unique, d.Members
		}
	}
	week, month := today.AddDate(0, 0, -6), today.AddDate(0, 0, -29)
	users := map[string]int{"total": len(sn.users)}
	for _, u := range sn.users {
		users[string(u.Role)]++
		if !u.CreatedAt.Before(today) {
			users["today"]++
		}
		if !u.CreatedAt.Before(week) {
			users["week"]++
		}
		if !u.CreatedAt.Before(month) {
			users["month"]++
		}
		if i, ok := idx[localDay(u.CreatedAt)]; ok {
			daily[i].Signups++
		}
	}
	type money struct {
		Total int64 `json:"total"`
		Count int   `json:"count"`
	}
	inc := map[string]*money{"today": {}, "week": {}, "month": {}, "period": {}, "all": {}, "sales": {}, "storage": {}, "fees": {}, "pending": {}}
	add := func(k string, v int64) { inc[k].Total += v; inc[k].Count++ }
	for i := range sn.orders {
		o := &sn.orders[i]
		if o.Status != store.OrderPaid {
			add("pending", o.Amount)
			continue
		}
		t, got := paidAt(o), o.Amount+o.Fee // хүлээн авсан мөнгө: үнэ + үйлчилгээний хураамж
		add("all", got)
		if o.Kind == store.OrderKindStorage {
			add("storage", o.Amount)
		} else {
			add("sales", o.Amount)
		}
		if o.Fee > 0 {
			add("fees", o.Fee)
		}
		if !t.Before(today) {
			add("today", got)
		}
		if !t.Before(week) {
			add("week", got)
		}
		if !t.Before(month) {
			add("month", got)
		}
		if !t.Before(since) {
			add("period", got)
		}
		if i, ok := idx[localDay(t)]; ok {
			daily[i].Revenue += got
			daily[i].Orders++
		}
	}
	pay := map[string]*money{"pending": {}, "paid": {}}
	for _, p := range sn.payouts {
		if m := pay[p.Status]; m != nil {
			m.Total += p.Amount
			m.Count++
		}
	}
	content := map[string]int64{"courses": int64(len(sn.courses)), "books": int64(len(sn.books))}
	for _, c := range sn.courses {
		content["lessons"] += int64(c.LessonCount)
		if c.Published {
			content["courses_published"]++
		}
		content["course_views"] += c.Views
	}
	for _, b := range sn.books {
		if b.Published {
			content["books_published"]++
		}
	}
	for _, f := range sn.files {
		content["files"] += int64(f.Count)
		content["file_bytes"] += f.Bytes
	}
	for _, e := range sn.enrolls {
		if e.Lesson {
			content["lesson_purchases"]++
		} else {
			content["enrollments"]++
		}
	}
	openReports := 0
	if rs, err := s.store.Reports(ctx, store.ReportOpen, 500); err == nil {
		openReports = len(rs)
	}
	// Шилдэг багш (нийт борлуулалтаар) ба сүүлийн гүйлгээ, бүртгэл.
	rev := map[string]int64{}
	for i := range sn.orders {
		if o := &sn.orders[i]; o.Status == store.OrderPaid && o.Kind != store.OrderKindStorage {
			rev[o.TeacherID] += o.Amount
		}
	}
	type topT struct {
		personRef
		Revenue int64 `json:"revenue"`
	}
	top := []topT{}
	for id, v := range rev {
		top = append(top, topT{sn.person(id), v})
	}
	sort.Slice(top, func(i, j int) bool { return top[i].Revenue > top[j].Revenue })
	if len(top) > 5 {
		top = top[:5]
	}
	recent := []map[string]any{}
	for i := range sn.orders {
		if len(recent) >= 8 {
			break
		}
		if o := &sn.orders[i]; o.Status == store.OrderPaid {
			recent = append(recent, s.txRow(sn, o))
		}
	}
	signups := []map[string]any{}
	for _, u := range sn.users {
		if len(signups) >= 8 {
			break
		}
		signups = append(signups, map[string]any{"id": u.ID, "name": u.DisplayName, "username": u.Username, "role": u.Role, "created_at": u.CreatedAt})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"days": days, "traffic": tr, "daily": daily, "users": users, "income": inc, "payouts": pay, "content": content,
		"open_reports": openReports, "top_teachers": top, "recent_orders": recent, "recent_users": signups,
	})
}

// ---- Багш нар ----

type teacherRow struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	AvatarURL    string     `json:"avatar_url"`
	CreatedAt    time.Time  `json:"created_at"`
	Courses      int        `json:"courses"`
	Published    int        `json:"courses_published"`
	Lessons      int        `json:"lessons"`
	Books        int        `json:"books"`
	Files        int        `json:"files"`
	FileBytes    int64      `json:"file_bytes"`
	Students     int        `json:"students"`
	Sales        int        `json:"sales"`
	Revenue      int64      `json:"revenue"`
	PaidOut      int64      `json:"paid_out"`
	Pending      int64      `json:"pending"`
	Available    int64      `json:"available"`
	ProfileViews int64      `json:"profile_views"`
	CourseViews  int64      `json:"course_views"`
	LastSaleAt   *time.Time `json:"last_sale_at,omitempty"`
}

func (s *Server) teacherRows(sn *adminSnap) map[string]*teacherRow {
	rows := map[string]*teacherRow{}
	for _, u := range sn.users {
		if u.Role == store.RoleTeacher {
			f := sn.files[u.ID]
			rows[u.ID] = &teacherRow{ID: u.ID, Name: u.DisplayName, Username: u.Username, Email: u.Email, AvatarURL: u.AvatarURL,
				CreatedAt: u.CreatedAt, ProfileViews: u.ProfileViews, Files: f.Count, FileBytes: f.Bytes}
		}
	}
	for _, c := range sn.courses {
		if t := rows[c.TeacherID]; t != nil {
			t.Courses++
			t.Lessons += c.LessonCount
			t.CourseViews += c.Views
			if c.Published {
				t.Published++
			}
		}
	}
	for _, b := range sn.books {
		if t := rows[b.TeacherID]; t != nil {
			t.Books++
		}
	}
	studs := map[string]map[string]bool{}
	for _, e := range sn.enrolls {
		if studs[e.TeacherID] == nil {
			studs[e.TeacherID] = map[string]bool{}
		}
		studs[e.TeacherID][e.UserID] = true
	}
	for id, m := range studs {
		if t := rows[id]; t != nil {
			t.Students = len(m)
		}
	}
	for i := range sn.orders {
		o := &sn.orders[i]
		t := rows[o.TeacherID]
		if t == nil || o.Status != store.OrderPaid || o.Kind == store.OrderKindStorage {
			continue
		}
		t.Sales++
		t.Revenue += o.Amount
		if pa := paidAt(o); t.LastSaleAt == nil || pa.After(*t.LastSaleAt) {
			t.LastSaleAt = &pa
		}
	}
	for _, p := range sn.payouts {
		if t := rows[p.TeacherID]; t != nil {
			switch p.Status {
			case store.PayoutPaid:
				t.PaidOut += p.Amount
			case store.PayoutPending:
				t.Pending += p.Amount
			}
		}
	}
	for _, t := range rows {
		t.Available = max(0, t.Revenue-t.PaidOut-t.Pending)
	}
	return rows
}

// handleAdminTeachers: GET /api/admin/teachers
func (s *Server) handleAdminTeachers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	sn, err := s.adminSnapshot(r.Context())
	if s.storeErr(w, r, err) {
		return
	}
	out := []*teacherRow{}
	for _, t := range s.teacherRows(sn) {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Revenue != out[j].Revenue {
			return out[i].Revenue > out[j].Revenue
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	writeJSON(w, http.StatusOK, map[string]any{"teachers": out})
}

// handleAdminTeacher: GET /api/admin/teachers/{id} — нэг багшийн дэлгэрэнгүй.
func (s *Server) handleAdminTeacher(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	ctx := r.Context()
	sn, err := s.adminSnapshot(ctx)
	if s.storeErr(w, r, err) {
		return
	}
	id := r.PathValue("id")
	u := sn.user[id]
	row := s.teacherRows(sn)[id]
	if u == nil || row == nil {
		writeErr(w, http.StatusNotFound, "багш олдсонгүй")
		return
	}
	type courseRow struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Published bool      `json:"published"`
		Price     int64     `json:"price"`
		Lessons   int       `json:"lessons"`
		Students  int       `json:"students"`
		Views     int64     `json:"views"`
		Sales     int       `json:"sales"`
		Revenue   int64     `json:"revenue"`
		CreatedAt time.Time `json:"created_at"`
	}
	studs := map[string]map[string]bool{}
	firstAt := map[string]time.Time{}
	perStudent := map[string]int{}
	for _, e := range sn.enrolls {
		if e.TeacherID != id {
			continue
		}
		if studs[e.CourseID] == nil {
			studs[e.CourseID] = map[string]bool{}
		}
		if !studs[e.CourseID][e.UserID] {
			perStudent[e.UserID]++
		}
		studs[e.CourseID][e.UserID] = true
		if t, ok := firstAt[e.UserID]; !ok || (!e.At.IsZero() && e.At.Before(t)) {
			firstAt[e.UserID] = e.At
		}
	}
	csales, crev := map[string]int{}, map[string]int64{}
	sales := []map[string]any{}
	for i := range sn.orders {
		o := &sn.orders[i]
		if o.TeacherID != id || o.Status != store.OrderPaid || o.Kind == store.OrderKindStorage {
			continue
		}
		csales[o.CourseID]++
		crev[o.CourseID] += o.Amount
		if len(sales) < 300 {
			sales = append(sales, s.txRow(sn, o))
		}
	}
	courses := []courseRow{}
	for _, c := range sn.courses {
		if c.TeacherID == id {
			courses = append(courses, courseRow{c.ID, c.Title, c.Published, c.Price, c.LessonCount, len(studs[c.ID]), c.Views,
				csales[c.ID], crev[c.ID], c.CreatedAt})
		}
	}
	books := []store.Book{}
	for _, b := range sn.books {
		if b.TeacherID == id {
			books = append(books, b)
		}
	}
	students := []map[string]any{}
	for uid, n := range perStudent {
		p := sn.person(uid)
		var at any
		if t := firstAt[uid]; !t.IsZero() {
			at = t
		}
		students = append(students, map[string]any{"id": uid, "name": p.Name, "username": p.Username, "courses": n, "since": at})
	}
	sort.Slice(students, func(i, j int) bool { return students[i]["name"].(string) < students[j]["name"].(string) })
	if len(students) > 500 {
		students = students[:500]
	}
	files := []map[string]any{}
	if s.files != nil {
		if l, err := s.files.List(id); err == nil {
			for i, f := range l {
				if i >= 200 {
					break
				}
				name := f.OriginalName
				if name == "" {
					name = f.Name
				}
				files = append(files, map[string]any{"name": name, "size": f.Size, "type": f.ContentType, "visibility": f.Visibility,
					"status": f.Status, "at": f.ModTime, "url": f.URL})
			}
		}
	}
	payouts := []store.Payout{}
	for _, p := range sn.payouts {
		if p.TeacherID == id {
			p.Bank = bankName(p.Bank)
			payouts = append(payouts, p)
		}
	}
	var acct map[string]any
	if a, err := s.store.PayoutAccount(ctx, id); err == nil {
		acct = map[string]any{"bank": bankName(a.Bank), "account_no": a.AccountNo, "account_name": a.AccountName, "schedule": a.Schedule}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"teacher": row,
		"profile": map[string]any{"headline": u.Headline, "location": u.Location, "phone": u.Phone, "subjects": u.Subjects,
			"storage_extra_bytes": u.StorageExtraBytes, "storage_expires_at": u.StorageExpiresAt},
		"account": acct, "courses": courses, "books": books, "students": students, "files": files, "sales": sales, "payouts": payouts,
	})
}

// ---- Суралцагчид ----

type studentRow struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Username   string     `json:"username"`
	Email      string     `json:"email"`
	CreatedAt  time.Time  `json:"created_at"`
	Courses    int        `json:"courses"`
	Lessons    int        `json:"lessons"`
	Teachers   int        `json:"teachers"`
	Orders     int        `json:"orders"`
	Spent      int64      `json:"spent"`
	LastPaidAt *time.Time `json:"last_paid_at,omitempty"`
}

func studentRows(sn *adminSnap) map[string]*studentRow {
	rows := map[string]*studentRow{}
	for _, u := range sn.users {
		if u.Role != store.RoleTeacher {
			rows[u.ID] = &studentRow{ID: u.ID, Name: u.DisplayName, Username: u.Username, Email: u.Email, CreatedAt: u.CreatedAt}
		}
	}
	teachers := map[string]map[string]bool{}
	for _, e := range sn.enrolls {
		st := rows[e.UserID]
		if st == nil {
			continue
		}
		if e.Lesson {
			st.Lessons++
		} else {
			st.Courses++
		}
		if teachers[e.UserID] == nil {
			teachers[e.UserID] = map[string]bool{}
		}
		teachers[e.UserID][e.TeacherID] = true
	}
	for id, m := range teachers {
		rows[id].Teachers = len(m)
	}
	for i := range sn.orders {
		o := &sn.orders[i]
		st := rows[o.UserID]
		if st == nil || o.Status != store.OrderPaid {
			continue
		}
		st.Orders++
		st.Spent += o.Amount
		if pa := paidAt(o); st.LastPaidAt == nil || pa.After(*st.LastPaidAt) {
			st.LastPaidAt = &pa
		}
	}
	return rows
}

// handleAdminStudents: GET /api/admin/students
func (s *Server) handleAdminStudents(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	sn, err := s.adminSnapshot(r.Context())
	if s.storeErr(w, r, err) {
		return
	}
	out := []*studentRow{}
	for _, st := range studentRows(sn) {
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	writeJSON(w, http.StatusOK, map[string]any{"students": out})
}

// handleAdminStudent: GET /api/admin/students/{id}
func (s *Server) handleAdminStudent(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	sn, err := s.adminSnapshot(r.Context())
	if s.storeErr(w, r, err) {
		return
	}
	id := r.PathValue("id")
	row := studentRows(sn)[id]
	u := sn.user[id]
	if row == nil || u == nil {
		writeErr(w, http.StatusNotFound, "суралцагч олдсонгүй")
		return
	}
	enrolls := []map[string]any{}
	for _, e := range sn.enrolls {
		if e.UserID != id {
			continue
		}
		title := ""
		if c := sn.course[e.CourseID]; c != nil {
			title = c.Title
		}
		var at any
		if !e.At.IsZero() {
			at = e.At
		}
		enrolls = append(enrolls, map[string]any{"course_id": e.CourseID, "course": title, "teacher": sn.person(e.TeacherID), "lesson": e.Lesson, "at": at})
	}
	orders := []map[string]any{}
	for i := range sn.orders {
		if o := &sn.orders[i]; o.UserID == id {
			orders = append(orders, s.txRow(sn, o))
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"student": row, "profile": map[string]any{"phone": u.Phone, "location": u.Location},
		"enrollments": enrolls, "orders": orders})
}

// ---- Гүйлгээ ----

func (s *Server) txRow(sn *adminSnap, o *store.Order) map[string]any {
	return map[string]any{"id": o.ID, "kind": o.Kind, "kind_label": kindLabel(o.Kind), "title": o.Title, "amount": o.Amount, "fee": o.Fee,
		"status": o.Status, "created_at": o.CreatedAt, "paid_at": o.PaidAt, "buyer": sn.person(o.UserID), "teacher": sn.person(o.TeacherID)}
}

// handleAdminTransactions: GET /api/admin/transactions?status=paid|pending|all&kind=&from=YYYY-MM-DD&to=YYYY-MM-DD&q=&format=csv
func (s *Server) handleAdminTransactions(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	sn, err := s.adminSnapshot(r.Context())
	if s.storeErr(w, r, err) {
		return
	}
	q := r.URL.Query()
	status, kind := q.Get("status"), q.Get("kind")
	if status == "" {
		status = "paid"
	}
	var from, to time.Time
	if t, err := time.ParseInLocation("2006-01-02", q.Get("from"), time.Local); err == nil {
		from = t
	}
	if t, err := time.ParseInLocation("2006-01-02", q.Get("to"), time.Local); err == nil {
		to = t.AddDate(0, 0, 1)
	}
	needle := strings.ToLower(strings.TrimSpace(q.Get("q")))
	match := func(o *store.Order) bool {
		if status != "all" && string(o.Status) != status {
			return false
		}
		if kind != "" && o.Kind != kind {
			return false
		}
		t := o.CreatedAt
		if o.Status == store.OrderPaid {
			t = paidAt(o)
		}
		if (!from.IsZero() && t.Before(from)) || (!to.IsZero() && !t.Before(to)) {
			return false
		}
		if needle == "" {
			return true
		}
		b, te := sn.person(o.UserID), sn.person(o.TeacherID)
		hay := strings.ToLower(o.ID + " " + o.Title + " " + b.Name + " " + b.Username + " " + te.Name + " " + te.Username)
		return strings.Contains(hay, needle)
	}
	type sum struct {
		Count int   `json:"count"`
		Total int64 `json:"total"`
	}
	tot := map[string]*sum{"paid": {}, "pending": {}, "fees": {}}
	byKind := map[string]*sum{}
	list := []*store.Order{}
	for i := range sn.orders {
		o := &sn.orders[i]
		if !match(o) {
			continue
		}
		list = append(list, o)
		if t := tot[string(o.Status)]; t != nil {
			t.Count++
			t.Total += o.Amount + o.Fee
		}
		if o.Status == store.OrderPaid && o.Fee > 0 {
			tot["fees"].Count++
			tot["fees"].Total += o.Fee
		}
		if o.Status == store.OrderPaid {
			if byKind[o.Kind] == nil {
				byKind[o.Kind] = &sum{}
			}
			byKind[o.Kind].Count++
			byKind[o.Kind].Total += o.Amount
		}
	}
	if q.Get("format") == "csv" {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="guilgee-%s.csv"`, time.Now().Format("2006-01-02")))
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM: Excel монгол үсгийг зөв уншина
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"Огноо", "Төлөв", "Төрөл", "Нэр", "Үнэ (₮)", "Хураамж (₮)", "Нийт (₮)", "Худалдан авагч", "Багш", "Захиалгын дугаар"})
		for _, o := range list {
			t, st := o.CreatedAt, "Хүлээгдэж буй"
			if o.Status == store.OrderPaid {
				t, st = paidAt(o), "Төлөгдсөн"
			}
			b, te := sn.person(o.UserID), sn.person(o.TeacherID)
			_ = cw.Write([]string{t.In(time.Local).Format("2006-01-02 15:04"), st, kindLabel(o.Kind), o.Title, strconv.FormatInt(o.Amount, 10), strconv.FormatInt(o.Fee, 10), strconv.FormatInt(o.Amount+o.Fee, 10),
				strings.TrimSpace(b.Name + " (@" + b.Username + ")"), te.Name, o.ID})
		}
		cw.Flush()
		return
	}
	limit := intParam(r, "limit", 500, 1, 5000)
	rows := []map[string]any{}
	for i, o := range list {
		if i >= limit {
			break
		}
		rows = append(rows, s.txRow(sn, o))
	}
	kinds := []map[string]any{}
	for k, v := range byKind {
		kinds = append(kinds, map[string]any{"kind": k, "label": kindLabel(k), "count": v.Count, "total": v.Total})
	}
	sort.Slice(kinds, func(i, j int) bool { return kinds[i]["total"].(int64) > kinds[j]["total"].(int64) })
	labels := []map[string]string{}
	for _, k := range []string{store.OrderKindCourse, store.OrderKindLesson, store.OrderKindBook, store.OrderKindMeeting, store.OrderKindSlot,
		store.OrderKindFee, store.OrderKindLate, store.OrderKindStorage} {
		labels = append(labels, map[string]string{"key": k, "label": kindLabel(k)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows": rows, "matched": len(list), "totals": tot, "by_kind": kinds, "kinds": labels})
}

// ---- Орлого, зарлага ----

var expenseCategories = []string{"Сервер, дэд бүтэц", "Сурталчилгаа", "Цалин", "Түрээс", "Банкны шимтгэл", "Татвар", "Хууль, нягтлан", "Бусад"}

// handleAdminFinance: GET /api/admin/finance?months=12 — сар бүрийн орлого (борлуулалт, багтаамж), зарлага (багшид
// шилжүүлсэн, платформын зардал), цэвэр дүн; багшид өгөх үлдэгдэл.
func (s *Server) handleAdminFinance(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	ctx := r.Context()
	months := intParam(r, "months", 12, 1, 36)
	sn, err := s.adminSnapshot(ctx)
	if s.storeErr(w, r, err) {
		return
	}
	allExp, err := s.store.Expenses(ctx, time.Time{})
	if s.storeErr(w, r, err) {
		return
	}
	now := time.Now().In(time.Local)
	first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local).AddDate(0, -(months - 1), 0)
	type monthRow struct {
		Month    string `json:"month"`
		Sales    int64  `json:"sales"`
		Storage  int64  `json:"storage"`
		Fees     int64  `json:"fees"`
		Income   int64  `json:"income"`
		Payouts  int64  `json:"payouts"`
		Expenses int64  `json:"expenses"`
		Outgo    int64  `json:"outgo"`
		Net      int64  `json:"net"`
		Orders   int    `json:"orders"`
	}
	idx := map[string]int{}
	rows := make([]monthRow, months)
	for i := range rows {
		k := first.AddDate(0, i, 0).Format("2006-01")
		rows[i].Month, idx[k] = k, i
	}
	type totals struct {
		Sales, Storage, Fees, Income, Payouts, Pending, Expenses, Outgo, Net int64
	}
	var all, rng totals
	for i := range sn.orders {
		o := &sn.orders[i]
		if o.Status != store.OrderPaid {
			continue
		}
		in := func(t *totals) {
			t.Income += o.Amount + o.Fee
			t.Fees += o.Fee
			if o.Kind == store.OrderKindStorage {
				t.Storage += o.Amount
			} else {
				t.Sales += o.Amount
			}
		}
		in(&all)
		if j, ok := idx[localMonth(paidAt(o))]; ok {
			in(&rng)
			m := &rows[j]
			m.Income += o.Amount + o.Fee
			m.Fees += o.Fee
			m.Orders++
			if o.Kind == store.OrderKindStorage {
				m.Storage += o.Amount
			} else {
				m.Sales += o.Amount
			}
		}
	}
	for _, p := range sn.payouts {
		switch p.Status {
		case store.PayoutPending:
			all.Pending += p.Amount
		case store.PayoutPaid:
			all.Payouts += p.Amount
			at := p.CreatedAt
			if p.ProcessedAt != nil {
				at = *p.ProcessedAt
			}
			if j, ok := idx[localMonth(at)]; ok {
				rng.Payouts += p.Amount
				rows[j].Payouts += p.Amount
			}
		}
	}
	exps := []store.Expense{}
	byCat := map[string]int64{}
	for _, e := range allExp {
		all.Expenses += e.Amount
		if j, ok := idx[localMonth(e.Date)]; ok {
			rng.Expenses += e.Amount
			rows[j].Expenses += e.Amount
			byCat[e.Category] += e.Amount
			exps = append(exps, e)
		}
	}
	for i := range rows {
		m := &rows[i]
		m.Outgo = m.Payouts + m.Expenses
		m.Net = m.Income - m.Outgo
	}
	for _, t := range []*totals{&all, &rng} {
		t.Outgo = t.Payouts + t.Expenses
		t.Net = t.Income - t.Outgo
	}
	j := func(t totals) map[string]int64 {
		return map[string]int64{"sales": t.Sales, "storage": t.Storage, "fees": t.Fees, "income": t.Income, "payouts": t.Payouts, "pending": t.Pending,
			"expenses": t.Expenses, "outgo": t.Outgo, "net": t.Net}
	}
	allJ := j(all)
	allJ["owed"] = max(0, all.Sales-all.Payouts)           // багшид шилжүүлээгүй үлдэгдэл (хүлээгдэж буй хүсэлт орно)
	allJ["profit"] = all.Storage + all.Fees - all.Expenses // платформын өөрийн ашиг (багтаамж + үйлчилгээний хураамж − зардал)
	allJ["cash"] = all.Income - all.Payouts - all.Expenses // данс дахь мөнгө (онолын)
	cats := []map[string]any{}
	for k, v := range byCat {
		cats = append(cats, map[string]any{"category": k, "total": v})
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i]["total"].(int64) > cats[j]["total"].(int64) })
	writeJSON(w, http.StatusOK, map[string]any{"months": rows, "range": j(rng), "all": allJ, "expenses": exps, "by_category": cats,
		"categories": expenseCategories})
}

// handleAdminAddExpense: POST /api/admin/expenses {date, category, amount, note}
func (s *Server) handleAdminAddExpense(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var in struct {
		Date     string `json:"date"`
		Category string `json:"category"`
		Amount   int64  `json:"amount"`
		Note     string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	d, err := time.ParseInLocation("2006-01-02", in.Date, time.Local)
	switch {
	case err != nil:
		writeErr(w, http.StatusBadRequest, "огноо буруу")
		return
	case d.After(time.Now().AddDate(0, 0, 1)):
		writeErr(w, http.StatusBadRequest, "ирээдүйн огноогоор зардал бүртгэхгүй")
		return
	case in.Amount <= 0 || in.Amount > 1_000_000_000_000:
		writeErr(w, http.StatusBadRequest, "дүн буруу")
		return
	}
	in.Category = strings.Join(strings.Fields(in.Category), " ")
	in.Note = strings.TrimSpace(in.Note)
	if in.Category == "" {
		in.Category = "Бусад"
	}
	if utf8.RuneCountInString(in.Category) > 40 || utf8.RuneCountInString(in.Note) > 300 {
		writeErr(w, http.StatusBadRequest, "ангилал 40, тайлбар 300 тэмдэгтээс хэтрэхгүй")
		return
	}
	e := &store.Expense{Date: d, Category: in.Category, Amount: in.Amount, Note: in.Note, CreatedBy: admin.DisplayName}
	if err := s.store.SaveExpense(r.Context(), e); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

// handleAdminDeleteExpense: DELETE /api/admin/expenses/{id}
func (s *Server) handleAdminDeleteExpense(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if err := s.store.DeleteExpense(r.Context(), r.PathValue("id")); s.storeErr(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Систем: файлын сан, өгөгдлийн сан, сервер ----

type sysCache struct {
	mu    sync.Mutex
	at    time.Time
	files *files.Space
	ferr  string
}

// handleAdminSystem: GET /api/admin/system[?fresh=1] — файлын сангийн хэмжээ ба дискний сул зай, өгөгдлийн сангийн
// хүснэгт бүрийн хэмжээ, серверийн төлөв. Файлын санг гүйх нь удаан байж болох тул 60 секунд кэшлэнэ.
func (s *Server) handleAdminSystem(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	out := map[string]any{}
	s.sys.mu.Lock()
	if s.files != nil && (s.sys.files == nil || time.Since(s.sys.at) > time.Minute || r.URL.Query().Get("fresh") == "1") {
		sp, err := s.files.Space()
		s.sys.files, s.sys.ferr, s.sys.at = sp, "", time.Now()
		if err != nil {
			s.sys.ferr = err.Error()
		}
	}
	out["files"], out["files_err"], out["files_at"] = s.sys.files, s.sys.ferr, s.sys.at
	s.sys.mu.Unlock()
	if ds, ok := s.store.(store.DBStatser); ok {
		if st, err := ds.DBStats(r.Context()); err == nil {
			out["db"] = st
		} else {
			out["db_err"] = err.Error()
		}
	} else {
		out["db"] = &store.DBStats{Engine: "Санах ой", Version: "туршилтын горим — өгөгдөл хадгалагдахгүй", Tables: []store.DBTable{}, Disks: []store.DBDisk{}}
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	out["server"] = map[string]any{"started_at": s.started, "uptime_sec": int64(time.Since(s.started).Seconds()), "mem_alloc": ms.Alloc,
		"mem_sys": ms.Sys, "goroutines": runtime.NumGoroutine(), "ws": s.hub.Connections(), "go": runtime.Version(), "version": assetVersion}
	writeJSON(w, http.StatusOK, out)
}
