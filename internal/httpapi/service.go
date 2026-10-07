package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// APIError — HTTP ба gRPC хоёуланд нь нэг ижил утгатай алдаа (код + монгол мессеж + нэмэлт талбар).
type APIError struct {
	Code  int // HTTP статус
	Msg   string
	Extra map[string]any // хариуд нэмэгдэх талбарууд (ж: lesson_price, state)
}

func (e *APIError) Error() string { return e.Msg }

func apiError(code int, msg string, extra map[string]any) *APIError {
	return &APIError{Code: code, Msg: msg, Extra: extra}
}

// apiErr нь APIError-ийг HTTP хариу болгоно; бусад алдааг storeErr шийднэ. true бол хариу бичигдсэн.
func (s *Server) apiErr(w http.ResponseWriter, r *http.Request, err error) bool {
	if err == nil {
		return false
	}
	var ae *APIError
	if errors.As(err, &ae) {
		body := map[string]any{"error": ae.Msg}
		for k, v := range ae.Extra {
			body[k] = v
		}
		writeJSON(w, ae.Code, body)
		return true
	}
	return s.storeErr(w, r, err)
}

// LessonFor нь хэрэглэгч (эсвэл багш) хичээлийг үзэх эрхтэй эсэхийг бүрэн шалгана:
// нийтлэгдсэн, төлбөр, дараалсан нээлт. Эрхгүй бол *APIError.
func (s *Server) LessonFor(ctx context.Context, uid, cid, lid string) (*store.Course, *store.Lesson, error) {
	course, err := s.store.CourseByID(ctx, cid)
	if err != nil {
		return nil, nil, err
	}
	if !course.Published && course.TeacherID != uid {
		return nil, nil, apiError(http.StatusNotFound, "олдсонгүй", nil)
	}
	l, err := s.store.LessonByID(ctx, cid, lid)
	if err != nil {
		return nil, nil, err
	}
	if hiddenFrom(l, course, uid) {
		return nil, nil, errLessonHidden
	}
	has, err := s.lessonAccess(ctx, uid, course, l)
	if err != nil {
		return nil, nil, err
	}
	if !has {
		return nil, nil, apiError(http.StatusPaymentRequired, "энэ хичээл төлбөртэй", map[string]any{"lesson_price": l.Price, "course_price": course.Price})
	}
	if b := s.lessonBlocked(ctx, uid, course, l.ID); b != nil {
		b.Title = l.Title
		return nil, nil, blockedErr(b)
	}
	if course.Drip && !l.AlwaysOpen && course.TeacherID != uid {
		lessons, err := s.store.LessonsByCourse(ctx, course.ID)
		if err != nil {
			return nil, nil, err
		}
		lessons = visibleLessons(lessons, false)
		progress, err := s.store.LessonProgress(ctx, uid, course.ID)
		if err != nil {
			return nil, nil, err
		}
		enrolled, _ := s.store.IsEnrolled(ctx, uid, course.ID)
		if st := dripState(course, lessons, l, progress, fullAccess(course, uid, enrolled), time.Now(), s.dripFacts(ctx, uid, course.ID)); !st.Open {
			return nil, nil, apiError(http.StatusLocked, "энэ хичээл хараахан нээгдээгүй", map[string]any{"state": st})
		}
	}
	return course, l, nil
}

// ActivityStart — идэвхийн сесс эхлүүлсний хариу (усан тэмдэг, хяналтын бодлого).
type ActivityStart struct {
	SessionID string            `json:"session_id"`
	Watermark map[string]string `json:"watermark"`
	Policy    ActivityPolicy    `json:"policy"`
}

type ActivityPolicy struct {
	Camera        string `json:"camera"`
	PingSec       int    `json:"ping_sec"`
	PingAnswerSec int    `json:"ping_answer_sec"`
	IdleSec       int    `json:"idle_sec"`
	BeatSec       int    `json:"beat_sec"`
	ActiveMin     int    `json:"active_min"`
	ActiveDoneSec int    `json:"active_done_sec"`
	Owner         bool   `json:"owner"`
	MaxWarn       int    `json:"max_warn"` // таб солих сануулгын тоо (хэтэрвэл хичээл зогсож хаагдана)
}

// StartActivity нь хичээл үзэх идэвхийн сесс нээнэ (HTTP ба gRPC).
func (s *Server) StartActivity(ctx context.Context, uid, fallbackName, ip, courseID, lessonID, kind string) (*ActivityStart, error) {
	if kind != "exam" {
		kind = "lesson"
	}
	course, l, err := s.LessonFor(ctx, uid, courseID, lessonID)
	if err != nil {
		return nil, err
	}
	name := s.displayName(ctx, uid, fallbackName)
	sess := &store.StudySession{UserID: uid, UserName: name, CourseID: course.ID, LessonID: l.ID, TeacherID: course.TeacherID, Kind: kind, Title: l.Title,
		StartedAt: time.Now(), LastAt: time.Now(), IP: ip}
	if err := s.store.CreateSession(ctx, sess); err != nil {
		return nil, err
	}
	done, err := s.lessonActiveSec(ctx, uid, l.ID)
	if err != nil {
		return nil, err
	}
	camera := course.Camera
	if camera == "" {
		camera = "optional"
	}
	return &ActivityStart{
		SessionID: sess.ID,
		Watermark: map[string]string{"name": name, "id": tail(uid, 6), "ip": sess.IP},
		Policy: ActivityPolicy{Camera: camera, PingSec: 300, PingAnswerSec: 30, IdleSec: 120, BeatSec: 15,
			ActiveMin: l.ActiveMin, ActiveDoneSec: done, Owner: course.TeacherID == uid, MaxWarn: maxWarnings(course)},
	}, nil
}

// BeatInput — 15 секунд тутмын идэвхийн тайлан.
type BeatInput struct {
	SessionID string      `json:"session_id"`
	Active    int         `json:"active"`
	Idle      int         `json:"idle"`
	Away      int         `json:"away"`
	FocusSum  float64     `json:"focus_sum"`
	FocusN    int         `json:"focus_n"`
	Camera    bool        `json:"camera"`
	AttemptID string      `json:"attempt_id"`
	End       string      `json:"end"`
	Events    []BeatEvent `json:"events"`
}

type BeatEvent struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

// RecordBeat нь сессэд идэвхийн секунд, зөрчлийг нэмж, нийт идэвхтэй секундийг буцаана.
// Хугацааг сүүлийн цохилтоос хойш өнгөрсөн бодит хугацаагаар хязгаарлана (хуурамч цагаас хамгаална).
func (s *Server) RecordBeat(ctx context.Context, uid string, in BeatInput) (int, error) {
	sess, err := s.store.SessionByID(ctx, in.SessionID)
	if err != nil || sess.UserID != uid {
		return 0, apiError(http.StatusNotFound, "сесс олдсонгүй", nil)
	}
	if sess.Ended {
		return 0, apiError(http.StatusConflict, "сесс дууссан", nil)
	}
	budget := int(time.Since(sess.LastAt).Seconds()) + 5
	budget = min(max(budget, 0), 180)
	clamp := func(v int) int {
		v = min(max(v, 0), budget)
		budget -= v
		return v
	}
	b := store.SessionBeat{ActiveSec: clamp(in.Active), IdleSec: clamp(in.Idle), AwaySec: clamp(in.Away), Camera: in.Camera, Counts: map[string]int{}}
	if in.FocusN > 0 && in.FocusN <= 400 {
		b.FocusN, b.FocusSum = in.FocusN, min(max(in.FocusSum, 0), float64(in.FocusN))
	}
	if in.End != "" {
		if _, ok := eventInfo[in.End]; ok || in.End == "closed" {
			b.End = in.End
		} else {
			b.End = "closed"
		}
	}
	var evs []store.ActivityEvent
	for i, e := range in.Events {
		if i >= maxBeatEvents {
			break
		}
		if _, ok := eventInfo[e.Type]; !ok || strings.HasPrefix(e.Type, "exam_") || e.Type == "teacher_remind" || e.Type == "teacher_unblock" {
			continue
		}
		b.Counts[e.Type]++
		d := e.Detail
		if utf8.RuneCountInString(d) > 200 {
			d = string([]rune(d)[:200])
		}
		evs = append(evs, store.ActivityEvent{UserID: uid, UserName: sess.UserName, CourseID: sess.CourseID, LessonID: sess.LessonID, TeacherID: sess.TeacherID,
			SessionID: sess.ID, Type: e.Type, Detail: strings.TrimSpace(sess.Title + " · " + d)})
	}
	if err := s.store.AddSessionBeat(ctx, sess.ID, b); err != nil {
		return 0, err
	}
	s.logEvents(ctx, evs)
	if in.AttemptID != "" {
		for _, e := range evs {
			if eventInfo[e.Type].Violation {
				_ = s.store.AddAttemptViolation(ctx, in.AttemptID)
			}
		}
	}
	done, _ := s.lessonActiveSec(ctx, uid, sess.LessonID)
	return done, nil
}

// Analytics — багшийн самбарын нэгтгэл (gRPC-д ч ижил).
func (s *Server) Analytics(ctx context.Context, teacherID, courseID string, days int) (*analyticsData, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	data, _, _, err := s.buildAnalytics(ctx, teacherID, courseID, "", days)
	return data, err
}

// PublicTeacherByUsername — нээлттэй профайлын өгөгдөл (кэштэй).
func (s *Server) PublicTeacherByUsername(ctx context.Context, username string) (*PublicProfile, error) {
	p, err := s.publicProfile(ctx, username)
	if err != nil {
		return nil, err
	}
	return &p.Data, nil
}

// PublicCourseByID — нээлттэй сургалт + хичээлийн тойм (кэштэй).
func (s *Server) PublicCourseByID(ctx context.Context, id string) (*PublicCourse, error) {
	c, err := s.publicCourse(ctx, id)
	if err != nil {
		return nil, err
	}
	return &c.Data, nil
}

// VerifyToken — gRPC metadata-гийн Bearer токеныг шалгана.
func (s *Server) VerifyToken(tok string) (uid, role, name string, err error) {
	c, err := s.tokens.Verify(tok)
	if err != nil {
		return "", "", "", err
	}
	if c.IsGuest() {
		return "", c.Role, c.Name, nil
	}
	return c.UID, c.Role, c.Name, nil
}

// Store нь gRPC үйлчилгээнд (ижил сан).
func (s *Server) Store() store.Store { return s.store }
