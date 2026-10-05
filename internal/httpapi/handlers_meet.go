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

// createMeet нь багшийн нэрийн өмнөөс Meet үүсгэж хадгална.
func (s *Server) createMeet(ctx context.Context, teacherID, courseID, title string, start time.Time, durMin int) (*store.Meeting, error) {
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
		MeetURL: mt.MeetURL, EventID: mt.EventID}
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

// handleCreateMeeting: {title, starts_at (RFC3339), duration_min, course_id?}
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
	}
	if !decode(w, r, &in) {
		return
	}
	in.Title = strings.TrimSpace(in.Title)
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
	m, err := s.createMeet(r.Context(), c.UID, in.CourseID, in.Title, in.StartsAt, in.DurationMin)
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
	writeJSON(w, http.StatusOK, ms)
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
	access := course.Price == 0 && course.Published
	if c, ok := s.principal(r); ok && !c.IsGuest() && !access {
		access = c.UID == course.TeacherID
		if !access {
			access, _ = s.store.IsEnrolled(r.Context(), c.UID, course.ID)
		}
	}
	if !course.Published && !access {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	if !access {
		for i := range ms {
			ms[i].MeetURL = ""
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"meetings": ms, "access": access})
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
	m, err := s.createMeet(r.Context(), conv.TeacherID, "", "Уулзалт: "+conv.VisitorName, time.Now(), 60)
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
