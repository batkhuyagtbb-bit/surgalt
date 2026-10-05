package httpapi

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Ном, өгүүллийн дэлгүүр.
// Хамгаалалт: эх файл уншигчид очдоггүй; хуудас бүрийг сервер эрх шалгаж, уншигчийн нэр/ID/огноог
// пикселд шингээн нэг нэгээр нь өгнө; төлөөгүй хүнд зөвхөн эхний хэдэн хуудас; хэт хурдан олон хуудас
// татвал хязгаарлаж, багшид мэдэгдэнэ. Бүх үйлдэл (үзсэн, уншсан, сонирхсон, төлбөрийн хана, худалдан авалт) логт.

const (
	defaultPreviewPages = 3
	bookPagesPerMinute  = 40
)

type bookLimiter struct {
	mu   sync.Mutex
	win  map[string]*[2]int64 // key -> {window start unix, count}
	seen map[string]time.Time // лог давхардуулахгүй
}

var bookLim = &bookLimiter{win: map[string]*[2]int64{}, seen: map[string]time.Time{}}

// allow — нэг минутад bookPagesPerMinute хуудас.
func (l *bookLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().Unix()
	if len(l.win) > 100_000 {
		l.win = map[string]*[2]int64{}
	}
	w := l.win[key]
	if w == nil || now-w[0] >= 60 {
		l.win[key] = &[2]int64{now, 1}
		return true
	}
	w[1]++
	return w[1] <= bookPagesPerMinute
}

// once — key-г ttl хугацаанд нэг л удаа true.
func (l *bookLimiter) once(key string, ttl time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.seen) > 200_000 {
		l.seen = map[string]time.Time{}
	}
	if t, ok := l.seen[key]; ok && time.Since(t) < ttl {
		return false
	}
	l.seen[key] = time.Now()
	return true
}

type bookInput struct {
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	CoverURL    string `json:"cover_url"`
	Price       int64  `json:"price"`
	Published   bool   `json:"published"`
	PreviewN    int    `json:"preview_pages"`
}

func (in *bookInput) validate(teacherID string) string {
	in.Title, in.Author, in.CoverURL = strings.TrimSpace(in.Title), strings.TrimSpace(in.Author), strings.TrimSpace(in.CoverURL)
	if in.Kind != "article" {
		in.Kind = "book"
	}
	if in.PreviewN == 0 {
		in.PreviewN = defaultPreviewPages
	}
	switch {
	case in.Title == "" || utf8.RuneCountInString(in.Title) > 200:
		return "нэр 1-200 тэмдэгт"
	case utf8.RuneCountInString(in.Author) > 120:
		return "зохиогчийн нэр хэт урт"
	case utf8.RuneCountInString(in.Description) > 5000:
		return "тайлбар хэт урт"
	case in.Price < 0 || in.Price > maxPrice:
		return "үнэ 0-100,000,000₮"
	case in.PreviewN < 1 || in.PreviewN > 20:
		return "үнэгүй үзүүлэх хуудас 1-20"
	case in.CoverURL != "" && !validURL(in.CoverURL) && !(files.OwnedBy(in.CoverURL, teacherID) && strings.Contains(in.CoverURL, "/"+files.Public+"/")):
		return "нүүр зураг: өөрийн нээлттэй зураг эсвэл http(s) холбоос"
	}
	return ""
}

// PublicBook — нээлттэй мэдээлэл (орлого, лог байхгүй).
type PublicBook struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description,omitempty"`
	Cover       string `json:"cover"`
	Price       int64  `json:"price"`
	Pages       int    `json:"pages"`
	PreviewN    int    `json:"preview_pages"`
}

func publicBook(b *store.Book, withDesc bool) PublicBook {
	p := PublicBook{ID: b.ID, Kind: b.Kind, Title: b.Title, Author: b.Author, Cover: b.CoverURL, Price: b.Price, Pages: b.Pages, PreviewN: b.PreviewN}
	if p.Cover == "" {
		p.Cover = "/api/books/" + b.ID + "/cover"
	}
	if withDesc {
		p.Description = b.Description
	}
	return p
}

func (s *Server) ownBook(w http.ResponseWriter, r *http.Request, teacherID string) (*store.Book, bool) {
	b, err := s.store.BookByID(r.Context(), r.PathValue("id"))
	if err != nil || b.TeacherID != teacherID {
		writeErr(w, http.StatusNotFound, "ном олдсонгүй")
		return nil, false
	}
	return b, true
}

func (s *Server) bookChanged(r *http.Request, teacherID, username string) {
	s.profiles.Delete(username)
	_ = teacherID
}

// ---- багш ----

func (s *Server) handleMyBooks(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	bs, err := s.store.BooksByTeacher(r.Context(), c.UID, false)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, bs)
}

func (s *Server) handleCreateBook(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	var in bookInput
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(c.UID); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	b := &store.Book{TeacherID: c.UID, Kind: in.Kind, Title: in.Title, Author: in.Author, Description: in.Description, CoverURL: in.CoverURL, Price: in.Price, PreviewN: in.PreviewN}
	if err := s.store.CreateBook(r.Context(), b); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (s *Server) handleUpdateBook(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	cur, ok := s.ownBook(w, r, c.UID)
	if !ok {
		return
	}
	var in bookInput
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(c.UID); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	if in.Published && cur.Pages == 0 {
		writeErr(w, http.StatusConflict, "эхлээд номын файлаа оруулна уу")
		return
	}
	b := &store.Book{ID: cur.ID, TeacherID: c.UID, Kind: in.Kind, Title: in.Title, Author: in.Author, Description: in.Description, CoverURL: in.CoverURL, Price: in.Price, Published: in.Published, PreviewN: in.PreviewN}
	if err := s.store.UpdateBook(r.Context(), b); s.storeErr(w, r, err) {
		return
	}
	s.bookChanged(r, c.UID, c.Name)
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) handleDeleteBook(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	b, ok := s.ownBook(w, r, c.UID)
	if !ok {
		return
	}
	if b.Sales > 0 {
		writeErr(w, http.StatusConflict, "худалдаж авсан хүмүүс байгаа тул устгах боломжгүй — нийтлэлээс хасна уу")
		return
	}
	if err := s.store.DeleteBook(r.Context(), b.ID, c.UID); s.storeErr(w, r, err) {
		return
	}
	_ = s.files.DeleteBookPages(c.UID, b.ID)
	s.bookChanged(r, c.UID, c.Name)
	w.WriteHeader(http.StatusNoContent)
}

// handleUploadBookPage: POST /api/me/books/{id}/pages/{n} — түүхий зураг (багшийн хөтөч PDF-ээс зурж илгээнэ).
func (s *Server) handleUploadBookPage(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	b, ok := s.ownBook(w, r, c.UID)
	if !ok {
		return
	}
	n, _ := strconv.Atoi(r.PathValue("n"))
	r.Body = http.MaxBytesReader(w, r.Body, files.MaxBookPageBytes+1024)
	if err := s.files.SaveBookPage(c.UID, b.ID, n, r.Body); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleBookPagesDone: POST /api/me/books/{id}/pages-done {"total": N}
func (s *Server) handleBookPagesDone(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	b, ok := s.ownBook(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Total int `json:"total"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Total < 1 || in.Total > files.MaxBookPages {
		writeErr(w, http.StatusBadRequest, "хуудасны тоо буруу")
		return
	}
	if err := s.files.BookPagesReady(c.UID, b.ID, in.Total); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.SetBookPages(r.Context(), b.ID, c.UID, in.Total); s.storeErr(w, r, err) {
		return
	}
	s.bookChanged(r, c.UID, c.Name)
	writeJSON(w, http.StatusOK, map[string]any{"pages": in.Total})
}

// handleBookEvents: GET /api/me/book-events?book=&limit= — лог.
func (s *Server) handleBookEvents(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	evs, err := s.store.BookEvents(r.Context(), c.UID, r.URL.Query().Get("book"), limit)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, evs)
}

// ---- уншигч ----

func (s *Server) publishedBook(w http.ResponseWriter, r *http.Request) (*store.Book, string, string, bool) {
	b, err := s.store.BookByID(r.Context(), r.PathValue("id"))
	uid, name := "", "Зочин"
	if p, ok := s.principal(r); ok && !p.IsGuest() {
		uid, name = p.UID, p.Name
	}
	if err != nil || (!b.Published && b.TeacherID != uid) {
		writeErr(w, http.StatusNotFound, "ном олдсонгүй")
		return nil, "", "", false
	}
	return b, uid, name, true
}

// fullAccess — бүх хуудсыг уншиж болох эсэх.
func (s *Server) bookFull(r *http.Request, b *store.Book, uid string) bool {
	if uid == "" {
		return false
	}
	if uid == b.TeacherID || b.Price == 0 {
		return true
	}
	ok, err := s.store.HasBookAccess(r.Context(), uid, b.ID)
	return err == nil && ok
}

func (s *Server) logBook(r *http.Request, b *store.Book, uid, name, typ, detail string, dedupe time.Duration) {
	if uid == b.TeacherID && uid != "" {
		return // багш өөрөө
	}
	who := uid
	if who == "" {
		who = s.clientIP(r)
	}
	if dedupe > 0 && !bookLim.once(typ+"|"+b.ID+"|"+who, dedupe) {
		return
	}
	if uid != "" {
		name = s.displayName(r, uid, name)
	}
	if err := s.store.AddBookEvent(r.Context(), store.BookEvent{BookID: b.ID, TeacherID: b.TeacherID, UserID: uid, UserName: name, Type: typ, Detail: detail, IP: s.clientIP(r)}); err != nil {
		s.log.Warn("book event", "err", err)
	}
}

// handleBook: GET /api/books/{id}
func (s *Server) handleBook(w http.ResponseWriter, r *http.Request) {
	b, uid, _, ok := s.publishedBook(w, r)
	if !ok {
		return
	}
	t, err := s.store.UserByID(r.Context(), b.TeacherID)
	if s.storeErr(w, r, err) {
		return
	}
	full := s.bookFull(r, b, uid)
	writeJSON(w, http.StatusOK, map[string]any{"book": publicBook(b, true), "teacher": publicTeacher(t),
		"access": map[string]any{"full": full, "owner": uid == b.TeacherID, "logged_in": uid != "", "preview_pages": min(b.PreviewN, b.Pages)}})
}

// handleBookPage: GET /api/books/{id}/pages/{n} — уншигчийн тэмдэгтэй JPEG.
func (s *Server) handleBookPage(w http.ResponseWriter, r *http.Request) {
	b, uid, name, ok := s.publishedBook(w, r)
	if !ok {
		return
	}
	n, _ := strconv.Atoi(r.PathValue("n"))
	if n < 1 || n > b.Pages {
		writeErr(w, http.StatusNotFound, "хуудас олдсонгүй")
		return
	}
	full := s.bookFull(r, b, uid)
	if !full && n > b.PreviewN {
		s.logBook(r, b, uid, name, "paywall", fmt.Sprintf("%d-р хуудаснаас цааш унших гэсэн", b.PreviewN), time.Hour)
		msg := "Цааш унших бол номыг худалдаж авна уу"
		if b.Price == 0 {
			msg = "Цааш унших бол нэвтэрнэ үү"
		}
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"error": msg, "price": b.Price, "need_login": uid == ""})
		return
	}
	who := uid
	if who == "" {
		who = s.clientIP(r)
	}
	if uid != b.TeacherID && !bookLim.allow(who+"|"+b.ID) {
		s.logBook(r, b, uid, name, "limit", "Минутад 40-өөс олон хуудас татах гэсэн", 10*time.Minute)
		if uid != "" && bookLim.once("notify-limit|"+uid+"|"+b.ID, 30*time.Minute) {
			s.notify(r.Context(), &store.Notification{UserID: b.TeacherID, Type: "violation", Title: "⚠️ " + s.displayName(r, uid, name) + ": номыг хэт хурдан татах гэсэн", Body: short(b.Title), Link: "/me#books"})
		}
		writeErr(w, http.StatusTooManyRequests, "Хэт хурдан эргүүлж байна — түр хүлээгээд үргэлжлүүлнэ үү")
		return
	}
	raw, err := s.files.BookPage(b.TeacherID, b.ID, n)
	if err != nil {
		writeErr(w, http.StatusNotFound, "хуудас олдсонгүй")
		return
	}
	if full {
		s.logBook(r, b, uid, name, "read", "", 6*time.Hour)
	} else {
		s.logBook(r, b, uid, name, "preview", "", time.Hour)
	}
	mark := "surgalt.mn  PREVIEW"
	if full {
		mark = fmt.Sprintf("surgalt.mn  @%s  #%s  %s", asciiOnly(name), tail(uid, 6), time.Now().Format("2006-01-02"))
	}
	out, err := watermark(raw, mark)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "хуудсыг бэлтгэж чадсангүй")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "image/jpeg")
	h.Set("Cache-Control", "private, no-store")
	h.Set("Content-Disposition", "inline")
	h.Set("X-Robots-Tag", "noindex")
	_, _ = w.Write(out)
}

// handleBookCover: GET /api/books/{id}/cover — нүүр зураггүй бол эхний хуудсыг жижгээр.
func (s *Server) handleBookCover(w http.ResponseWriter, r *http.Request) {
	b, _, _, ok := s.publishedBook(w, r)
	if !ok {
		return
	}
	if b.Pages == 0 {
		writeErr(w, http.StatusNotFound, "нүүр алга")
		return
	}
	raw, err := s.files.BookPage(b.TeacherID, b.ID, 1)
	if err != nil {
		writeErr(w, http.StatusNotFound, "нүүр алга")
		return
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "нүүр уншигдсангүй")
		return
	}
	sb := src.Bounds()
	tw := 360
	th := sb.Dy() * tw / max(1, sb.Dx())
	dst := image.NewRGBA(image.Rect(0, 0, tw, th))
	xdraw.ApproxBiLinear.Scale(dst, dst.Bounds(), src, sb, draw.Over, nil)
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 78})
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(buf.Bytes())
}

// handleBuyBook: POST /api/books/{id}/buy
func (s *Server) handleBuyBook(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	b, err := s.store.BookByID(r.Context(), r.PathValue("id"))
	if err != nil || !b.Published {
		writeErr(w, http.StatusNotFound, "ном олдсонгүй")
		return
	}
	if s.bookFull(r, b, c.UID) {
		writeJSON(w, http.StatusOK, map[string]any{"unlocked": true})
		return
	}
	o, err := s.store.CreateOrGetPendingBookOrder(r.Context(), c.UID, b)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"unlocked": false, "order": o,
		"payment": map[string]any{"amount": o.Amount, "currency": "MNT", "dev_pay": s.cfg.DevPayments}})
}

// handleBookInterest: POST /api/books/{id}/interest — "Сонирхож байна".
func (s *Server) handleBookInterest(w http.ResponseWriter, r *http.Request) {
	b, uid, name, ok := s.publishedBook(w, r)
	if !ok {
		return
	}
	s.logBook(r, b, uid, name, "interest", "Сонирхож байна гэж дарсан", 24*time.Hour)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- усан тэмдэг ----

func asciiOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 128 && (r == '.' || r == '_' || r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "user"
	}
	return b.String()
}

// watermark нь текстийг хуудсан дээр давхар мөрөөр (бүдэг) ба доод хэсэгт (тод) шингээнэ.
func watermark(raw []byte, text string) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	page := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(page, page.Bounds(), src, b.Min, draw.Src)

	face := basicfont.Face7x13
	tw := font.MeasureString(face, text).Ceil() + 4
	small := image.NewRGBA(image.Rect(0, 0, tw, 16))
	d := &font.Drawer{Dst: small, Src: image.NewUniform(color.RGBA{30, 45, 110, 255}), Face: face, Dot: fixed.P(2, 12)}
	d.DrawString(text)
	scale := max(2, b.Dx()/(tw*2))
	big := image.NewRGBA(image.Rect(0, 0, tw*scale, 16*scale))
	xdraw.NearestNeighbor.Scale(big, big.Bounds(), small, small.Bounds(), draw.Src, nil)

	faint := image.NewUniform(color.Alpha{34})
	stepY := max(big.Bounds().Dy()*4, b.Dy()/6)
	for row, y := 0, stepY/2; y < b.Dy(); row, y = row+1, y+stepY {
		off := (row % 2) * big.Bounds().Dx() / 2
		for x := -off; x < b.Dx(); x += big.Bounds().Dx() + big.Bounds().Dx()/3 {
			r := image.Rect(x, y, x+big.Bounds().Dx(), y+big.Bounds().Dy())
			draw.DrawMask(page, r, big, image.Point{}, faint, image.Point{}, draw.Over)
		}
	}
	// Доод мөр: уншихад саад болохгүй, гэхдээ тайрахад хэцүү.
	foot := image.Rect(8, b.Dy()-small.Bounds().Dy()-6, 8+tw, b.Dy()-6)
	draw.DrawMask(page, foot, small, image.Point{}, image.NewUniform(color.Alpha{150}), image.Point{}, draw.Over)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, page, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// pageBook: GET /b/{id} — номын нээлттэй хуудас.
func (s *Server) pageBook(w http.ResponseWriter, r *http.Request) {
	b, err := s.store.BookByID(r.Context(), r.PathValue("id"))
	uid, name := "", "Зочин"
	if p, ok := s.principal(r); ok && !p.IsGuest() {
		uid, name = p.UID, p.Name
	}
	if err != nil || !b.Published {
		s.pageError(w, r, store.ErrNotFound)
		return
	}
	t, err := s.store.UserByID(r.Context(), b.TeacherID)
	if err != nil {
		s.pageError(w, r, err)
		return
	}
	s.logBook(r, b, uid, name, "view", "", 30*time.Minute)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=30")
	if err := s.tmpl.ExecuteTemplate(w, "book.html", map[string]any{"D": map[string]any{"Book": publicBook(b, true), "Teacher": publicTeacher(t)}, "PublicURL": s.cfg.PublicURL}); err != nil {
		s.log.Error("template", "err", err, "page", "book")
	}
}
