package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/oauth2"

	"surgalt/internal/meet"
	"surgalt/internal/store"
)

type meetState struct {
	State    string `json:"s"`
	Verifier string `json:"v"`
	UID      string `json:"u"`
}

const meetCookie = "surgalt_meet"

func (s *Server) meetEnabled(w http.ResponseWriter) bool {
	if s.Meet == nil {
		writeErr(w, http.StatusServiceUnavailable, "Google Meet тохируулаагүй (GOOGLE_CLIENT_ID)")
		return false
	}
	return true
}

// handleMeetConnect: студиос POST хийж, Google-ийн зөвшөөрлийн URL авна.
// (Токен header-ээр ирдэг тул шууд redirect биш — URL буцааж, браузер шилжинэ.)
func (s *Server) handleMeetConnect(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok || !s.meetEnabled(w) {
		return
	}
	st := meetState{State: randHex(16), Verifier: oauth2.GenerateVerifier(), UID: c.UID}
	sealed, err := s.tokens.Seal(st, 10*time.Minute)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "дотоод алдаа")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: meetCookie, Value: sealed, Path: "/auth/", MaxAge: 600, HttpOnly: true,
		Secure: strings.HasPrefix(s.baseURL(r), "https://"), SameSite: http.SameSiteLaxMode})
	writeJSON(w, http.StatusOK, map[string]string{"url": s.Meet.AuthURL(st.State, st.Verifier)})
}

func (s *Server) handleMeetCallback(w http.ResponseWriter, r *http.Request) {
	back := func(status string) { http.Redirect(w, r, "/me#meet="+url.QueryEscape(status), http.StatusFound) }
	if s.Meet == nil {
		back("disabled")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: meetCookie, Value: "", Path: "/auth/", MaxAge: -1, HttpOnly: true})
	var st meetState
	ck, err := r.Cookie(meetCookie)
	if err != nil || s.tokens.Open(ck.Value, &st) != nil || !constantEq(st.State, r.URL.Query().Get("state")) || r.URL.Query().Get("error") != "" {
		back("error")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	refresh, err := s.Meet.Exchange(ctx, r.URL.Query().Get("code"), st.Verifier)
	if err != nil {
		s.log.Warn("meet exchange", "err", err)
		back("error")
		return
	}
	enc, err := s.tokens.Encrypt(refresh)
	if err == nil {
		err = s.store.SetGoogleToken(ctx, st.UID, enc)
	}
	if err != nil {
		s.log.Error("meet save", "err", err)
		back("error")
		return
	}
	back("connected")
}

func (s *Server) handleMeetDisconnect(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	if err := s.store.SetGoogleToken(r.Context(), c.UID, ""); s.storeErr(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// createMeet нь багшийн нэрийн өмнөөс Meet үүсгэж хадгална. price > 0 бол төлбөртэй шууд хичээл.
func (s *Server) createMeet(ctx context.Context, teacherID, courseID, title string, start time.Time, durMin int, price int64, membersFree bool) (*store.Meeting, error) {
	u, err := s.store.UserByID(ctx, teacherID)
	if err != nil {
		return nil, err
	}
	if u.GoogleToken == "" {
		return nil, meet.ErrNotConnected
	}
	refresh, err := s.tokens.Decrypt(u.GoogleToken)
	if err != nil {
		return nil, meet.ErrNotConnected
	}
	mt, err := s.Meet.Create(ctx, refresh, title, "Surgalt — "+u.DisplayName, start, time.Duration(durMin)*time.Minute)
	if err != nil {
		return nil, err
	}
	m := &store.Meeting{TeacherID: teacherID, CourseID: courseID, Title: title, StartsAt: start, DurationMin: durMin,
		MeetURL: mt.MeetURL, EventID: mt.EventID, Price: price, MembersFree: membersFree}
	return m, s.store.CreateMeeting(ctx, m)
}

func (s *Server) meetErr(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, meet.ErrNotConnected) {
		writeErr(w, http.StatusPreconditionRequired, "Google Meet-ээ Студи → Профайл хэсгээс холбоно уу")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
}

// meetingPriceErr — шууд хичээлийн үнийн шалгалт. Төлбөртэй бол сургалттай холбоотой байх ёстой
// (суралцагчид сургалтын хуудаснаас худалдаж авна).
func meetingPriceErr(price int64, courseID string) string {
	switch {
	case price < 0 || price > maxPrice:
		return "үнэ 0-100,000,000₮"
	case price > 0 && courseID == "":
		return "төлбөртэй шууд хичээлийг сургалттай холбоно уу — суралцагчид сургалтын хуудаснаас худалдаж авна"
	}
	return ""
}

// meetingOpen — Meet холбоосыг харах (нэгдэх) эрх. Багш үргэлж; төлбөртэй бол худалдаж авсан
// (эсвэл "элссэн хүнд үнэгүй" үед элссэн) хүн; үнэгүй бол сургалтын дүрэм: үнэгүй нийтлэгдсэн
// сургалт эсвэл элссэн хүн.
func meetingOpen(m *store.Meeting, course *store.Course, uid string, enrolled bool, bought map[string]bool) bool {
	switch {
	case uid != "" && uid == m.TeacherID:
		return true
	case m.Price > 0:
		return uid != "" && (bought[m.ID] || (m.MembersFree && enrolled))
	case course == nil:
		return false
	}
	return (course.Price == 0 && course.Published) || enrolled
}

// handleCreateMeeting: {title, starts_at (RFC3339), duration_min, course_id?, price?, members_free?}
func (s *Server) handleCreateMeeting(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok || !s.meetEnabled(w) {
		return
	}
	var in struct {
		Title       string    `json:"title"`
		StartsAt    time.Time `json:"starts_at"`
		DurationMin int       `json:"duration_min"`
		CourseID    string    `json:"course_id"`
		Price       int64     `json:"price"`
		MembersFree bool      `json:"members_free"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
	if msg := meetingPriceErr(in.Price, in.CourseID); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	switch {
	case in.Title == "" || utf8.RuneCountInString(in.Title) > 200:
		writeErr(w, http.StatusBadRequest, "гарчиг 1-200 тэмдэгт")
		return
	case in.DurationMin < 10 || in.DurationMin > 480:
		writeErr(w, http.StatusBadRequest, "үргэлжлэх хугацаа 10-480 минут")
		return
	case in.StartsAt.Before(time.Now().Add(-5 * time.Minute)):
		writeErr(w, http.StatusBadRequest, "эхлэх цаг өнгөрсөн байна")
		return
	}
	if in.CourseID != "" {
		course, err := s.store.CourseByID(r.Context(), in.CourseID)
		if s.storeErr(w, r, err) {
			return
		}
		if course.TeacherID != c.UID {
			writeErr(w, http.StatusNotFound, "олдсонгүй")
			return
		}
	}
	m, err := s.createMeet(r.Context(), c.UID, in.CourseID, in.Title, in.StartsAt, in.DurationMin, in.Price, in.MembersFree)
	if err != nil {
		s.meetErr(w, r, err)
		return
	}
	if in.CourseID != "" {
		s.profiles.Delete(c.Name) // профайл дээрх "Удахгүй болох" жагсаалт шууд шинэчлэгдэнэ
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) handleMyMeetings(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	ms, err := s.store.Meetings(r.Context(), c.UID, "", time.Now(), 100)
	if s.storeErr(w, r, err) {
		return
	}
	if buyers, err := s.store.MeetingBuyers(r.Context(), c.UID); err == nil {
		for i := range ms {
			ms[i].Buyers = buyers[ms[i].ID]
		}
	}
	writeJSON(w, http.StatusOK, ms)
}

// handleMeetingPrice: PUT /api/me/meetings/{id} {price, members_free} — багш үнийг өөрчилнө.
// Аль хэдийн худалдаж авсан хүмүүсийн эрх хэвээр; хүлээгдэж буй захиалга дараагийн удаа шинэ үнээр.
func (s *Server) handleMeetingPrice(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	m, err := s.store.MeetingByID(r.Context(), r.PathValue("id"))
	if err != nil || m.TeacherID != c.UID {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	var in struct {
		Price       int64 `json:"price"`
		MembersFree bool  `json:"members_free"`
	}
	if !decode(w, r, &in) {
		return
	}
	if msg := meetingPriceErr(in.Price, m.CourseID); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.SetMeetingPrice(r.Context(), m.ID, in.Price, in.MembersFree); s.storeErr(w, r, err) {
		return
	}
	m.Price, m.MembersFree = in.Price, in.MembersFree
	s.profiles.Delete(c.Name) // профайл дээрх үнэ шууд шинэчлэгдэнэ
	writeJSON(w, http.StatusOK, m)
}

// handleBuyMeeting: POST /api/meetings/{id}/buy — төлбөртэй шууд хичээлд бүртгүүлэх захиалга.
func (s *Server) handleBuyMeeting(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	m, err := s.store.MeetingByID(r.Context(), r.PathValue("id"))
	if err != nil || m.CourseID == "" {
		writeErr(w, http.StatusNotFound, "шууд хичээл олдсонгүй")
		return
	}
	course, err := s.store.CourseByID(r.Context(), m.CourseID)
	if err != nil || (!course.Published && course.TeacherID != c.UID) {
		writeErr(w, http.StatusNotFound, "шууд хичээл олдсонгүй")
		return
	}
	enrolled, _ := s.store.IsEnrolled(r.Context(), c.UID, course.ID)
	bought, _ := s.store.MeetingAccess(r.Context(), c.UID)
	if meetingOpen(m, course, c.UID, enrolled, bought) {
		writeJSON(w, http.StatusOK, map[string]any{"unlocked": true, "meet_url": m.MeetURL})
		return
	}
	switch {
	case m.Price <= 0:
		writeErr(w, http.StatusBadRequest, "энэ шууд хичээлд сургалтад элссэн суралцагчид нэгдэнэ")
		return
	case m.StartsAt.Add(time.Duration(m.DurationMin) * time.Minute).Before(time.Now()):
		writeErr(w, http.StatusGone, "шууд хичээл дууссан байна")
		return
	}
	o, err := s.store.CreateOrGetPendingMeetingOrder(r.Context(), c.UID, m, "Шууд хичээл: "+m.Title+" ("+course.Title+")")
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"unlocked": false, "order": o,
		"payment": map[string]any{"amount": o.Amount, "currency": "MNT", "dev_pay": s.cfg.DevPayments}})
}

// handleCourseMeetings: товлосон шууд хичээлүүд. Meet холбоос зөвхөн элссэн/багш/үнэгүй сургалтад.
func (s *Server) handleCourseMeetings(w http.ResponseWriter, r *http.Request) {
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	ms, err := s.store.Meetings(r.Context(), course.TeacherID, course.ID, time.Now(), 20)
	if s.storeErr(w, r, err) {
		return
	}
	uid, enrolled, bought := "", false, map[string]bool{}
	if c, ok := s.principal(r); ok && !c.IsGuest() {
		uid = c.UID
		enrolled, _ = s.store.IsEnrolled(r.Context(), uid, course.ID)
		bought, _ = s.store.MeetingAccess(r.Context(), uid)
	}
	access := uid == course.TeacherID || (course.Price == 0 && course.Published) || enrolled // сургалтын (үнэгүй) шууд хичээлд
	if !course.Published && !access {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	type courseMeeting struct {
		store.Meeting
		Access bool `json:"access"` // энэ хүн нэгдэх эрхтэй (холбоос харагдана)
		Bought bool `json:"bought,omitempty"`
	}
	out := make([]courseMeeting, len(ms))
	for i := range ms {
		open := meetingOpen(&ms[i], course, uid, enrolled, bought)
		if !open {
			ms[i].MeetURL = ""
		}
		out[i] = courseMeeting{Meeting: ms[i], Access: open, Bought: bought[ms[i].ID]}
	}
	writeJSON(w, http.StatusOK, map[string]any{"meetings": out, "access": access})
}

// handleChatMeet: багш чатаас нэг товчоор Meet үүсгэж, холбоосыг ярианд автоматаар илгээнэ.
func (s *Server) handleChatMeet(w http.ResponseWriter, r *http.Request) {
	_, conv, role, ok := s.loadConv(w, r)
	if !ok || !s.meetEnabled(w) {
		return
	}
	if role != store.SenderTeacher {
		writeErr(w, http.StatusForbidden, "зөвхөн багш Meet үүсгэнэ")
		return
	}
	m, err := s.createMeet(r.Context(), conv.TeacherID, "", "Уулзалт: "+conv.VisitorName, time.Now(), 60, 0, false)
	if err != nil {
		s.meetErr(w, r, err)
		return
	}
	msg := &store.Message{ConversationID: conv.ID, Sender: store.SenderTeacher,
		Body: "📹 Google Meet уулзалт бэлэн боллоо, нэгдээрэй: " + m.MeetURL}
	if err := s.store.AddMessage(r.Context(), msg); s.storeErr(w, r, err) {
		return
	}
	if s.Publish != nil {
		s.Publish(*msg)
	}
	writeJSON(w, http.StatusCreated, map[string]any{"meeting": m, "message": msg})
}
