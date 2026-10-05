package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// Хяналтын үйл явдлууд: монгол нэр, багшид шууд мэдэгдэх эсэх, зөрчилд тооцох эсэх.
type eventMeta struct {
	Label     string
	Notify    bool
	Violation bool
}

var eventInfo = map[string]eventMeta{
	"tab_switch":      {"Өөр цонх эсвэл таб руу шилжсэн", false, true},
	"copy":            {"Хуулах оролдлого", true, true},
	"context_menu":    {"Баруун товч дарсан", false, true},
	"print":           {"Хэвлэх оролдлого", true, true},
	"save":            {"Хуудсыг хадгалах оролдлого", true, true},
	"screenshot":      {"Дэлгэцийн зураг авах оролдлого", true, true},
	"devtools":        {"Хөгжүүлэгчийн хэрэгсэл нээсэн", true, true},
	"download":        {"Файл татах оролдлого", true, true},
	"idle":            {"2 минутаас дээш хөдөлгөөнгүй", false, false},
	"ping_missed":     {"Идэвхийн шалгалтад хариу өгөөгүй", true, false},
	"auto_block":      {"Хичээл автоматаар зогссон", true, false},
	"face_missing":    {"Камерт царай харагдаагүй", false, true},
	"eyes_closed":     {"Нүдээ аньсан (нойрмоглосон)", false, false},
	"look_away":       {"Дэлгэцээс өөр тийш харсан", false, true},
	"head_down":       {"Толгой доош харсан (утас ашиглаж магадгүй)", false, true},
	"phone":           {"Гар утас илэрсэн", true, true},
	"camera_off":      {"Камераа унтраасан", true, true},
	"exam_start":      {"Шалгалт эхэлсэн", false, false},
	"exam_submit":     {"Шалгалт өгсөн", false, false},
	"exam_terminated": {"Шалгалт зөрчлөөр хаагдсан", false, true},
	"teacher_remind":  {"Багш сануулга илгээсэн", false, false},
}

func eventLabel(t string) string {
	if m, ok := eventInfo[t]; ok {
		return m.Label
	}
	return t
}

// logEvents нь логт бичиж, ноцтой зөрчлийг багшид мэдэгдэнэ (нэг суралцагч, нэг төрлөөр 2 минутад нэг).
func (s *Server) logEvents(r *http.Request, evs []store.ActivityEvent) {
	if len(evs) == 0 {
		return
	}
	if err := s.store.AddActivityEvents(r.Context(), evs); err != nil {
		s.log.Warn("activity events", "err", err)
	}
	var ns []*store.Notification
	n := s.notif
	n.mu.Lock()
	for _, e := range evs {
		m := eventInfo[e.Type]
		if !m.Notify || e.UserID == e.TeacherID {
			continue
		}
		k := "act|" + e.UserID + "|" + e.Type
		if t, ok := n.msgLast[k]; ok && time.Since(t) < msgThrottle {
			continue
		}
		n.msgLast[k] = time.Now()
		ns = append(ns, &store.Notification{UserID: e.TeacherID, Type: "violation", Title: "⚠️ " + e.UserName + ": " + m.Label, Body: short(e.Detail), Link: "/me#students"})
	}
	n.mu.Unlock()
	s.notify(r.Context(), ns...)
}

// handleActivityStart: POST /api/activity/start {"course_id","lesson_id","kind"}
func (s *Server) handleActivityStart(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		CourseID string `json:"course_id"`
		LessonID string `json:"lesson_id"`
		Kind     string `json:"kind"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Kind != "exam" {
		in.Kind = "lesson"
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, in.CourseID, in.LessonID)
	if !ok {
		return
	}
	name := s.displayName(r, c.UID, c.Name)
	sess := &store.StudySession{UserID: c.UID, UserName: name, CourseID: course.ID, LessonID: l.ID, TeacherID: course.TeacherID, Kind: in.Kind, Title: l.Title,
		StartedAt: time.Now(), LastAt: time.Now(), IP: s.clientIP(r)}
	if err := s.store.CreateSession(r.Context(), sess); s.storeErr(w, r, err) {
		return
	}
	done, err := s.lessonActiveSec(r, c.UID, l.ID)
	if s.storeErr(w, r, err) {
		return
	}
	camera := course.Camera
	if camera == "" {
		camera = "optional"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"session_id": sess.ID,
		"watermark":  map[string]string{"name": name, "id": tail(c.UID, 6), "ip": sess.IP},
		"policy": map[string]any{"camera": camera, "ping_sec": 300, "ping_answer_sec": 30, "idle_sec": 120, "beat_sec": 15,
			"active_min": l.ActiveMin, "active_done_sec": done, "owner": course.TeacherID == c.UID},
	})
}

// lessonActiveSec — суралцагчийн тухайн хичээлд нийт идэвхтэй суралцсан секунд.
func (s *Server) lessonActiveSec(r *http.Request, uid, lessonID string) (int, error) {
	ss, err := s.store.Sessions(r.Context(), store.ActivityFilter{UserID: uid, LessonID: lessonID}, 1000)
	total := 0
	for _, x := range ss {
		total += x.ActiveSec
	}
	return total, err
}

const maxBeatEvents = 40

// handleActivityBeat: POST /api/activity/beat — 15 секунд тутам: идэвхтэй/идэвхгүй/өөр цонхонд
// байсан секунд, анхаарлын оноо, үйл явдлууд. Хугацааг сүүлийн цохилтоос хойш өнгөрсөн хугацаагаар хязгаарлана.
func (s *Server) handleActivityBeat(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		SessionID string  `json:"session_id"`
		Active    int     `json:"active"`
		Idle      int     `json:"idle"`
		Away      int     `json:"away"`
		FocusSum  float64 `json:"focus_sum"`
		FocusN    int     `json:"focus_n"`
		Camera    bool    `json:"camera"`
		AttemptID string  `json:"attempt_id"`
		End       string  `json:"end"`
		Events    []struct {
			Type   string `json:"type"`
			Detail string `json:"detail"`
		} `json:"events"`
	}
	if !decode(w, r, &in) {
		return
	}
	sess, err := s.store.SessionByID(r.Context(), in.SessionID)
	if err != nil || sess.UserID != c.UID {
		writeErr(w, http.StatusNotFound, "сесс олдсонгүй")
		return
	}
	if sess.Ended {
		writeErr(w, http.StatusConflict, "сесс дууссан")
		return
	}
	// Өнгөрсөн бодит хугацаанаас их секунд тоолохгүй (хуурамч "идэвхтэй" цагаас хамгаална).
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
		if _, ok := eventInfo[e.Type]; !ok || strings.HasPrefix(e.Type, "exam_") || e.Type == "teacher_remind" {
			continue
		}
		b.Counts[e.Type]++
		d := e.Detail
		if utf8.RuneCountInString(d) > 200 {
			d = string([]rune(d)[:200])
		}
		evs = append(evs, store.ActivityEvent{UserID: c.UID, UserName: sess.UserName, CourseID: sess.CourseID, LessonID: sess.LessonID, TeacherID: sess.TeacherID,
			SessionID: sess.ID, Type: e.Type, Detail: strings.TrimSpace(sess.Title + " · " + d)})
	}
	if err := s.store.AddSessionBeat(r.Context(), sess.ID, b); s.storeErr(w, r, err) {
		return
	}
	s.logEvents(r, evs)
	if in.AttemptID != "" {
		for _, e := range evs {
			if eventInfo[e.Type].Violation {
				_ = s.store.AddAttemptViolation(r.Context(), in.AttemptID)
			}
		}
	}
	done, _ := s.lessonActiveSec(r, c.UID, sess.LessonID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active_done_sec": done})
}

// handleRemindStudent: POST /api/me/students/{uid}/remind {"course_id","message"} — багш суралцагчид сануулга илгээнэ.
func (s *Server) handleRemindStudent(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	var in struct {
		CourseID string `json:"course_id"`
		Message  string `json:"message"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Message = strings.TrimSpace(in.Message)
	if n := utf8.RuneCountInString(in.Message); n < 1 || n > 500 {
		writeErr(w, http.StatusBadRequest, "сануулга 1-500 тэмдэгт")
		return
	}
	uid := r.PathValue("uid")
	course, err := s.store.CourseByID(r.Context(), in.CourseID)
	if err != nil || course.TeacherID != c.UID {
		writeErr(w, http.StatusNotFound, "сургалт олдсонгүй")
		return
	}
	member, err := s.courseMember(r.Context(), uid, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if !member {
		if ss, _ := s.store.Sessions(r.Context(), store.ActivityFilter{UserID: uid, CourseID: course.ID}, 1); len(ss) == 0 {
			writeErr(w, http.StatusNotFound, "энэ суралцагч таны сургалтад алга")
			return
		}
	}
	teacher := s.displayName(r, c.UID, c.Name)
	s.notify(r.Context(), &store.Notification{UserID: uid, Type: "remind", Title: "📣 " + teacher + " багшаас сануулга", Body: short(in.Message), Link: "/c/" + course.ID})
	s.logEvents(r, []store.ActivityEvent{{UserID: uid, UserName: s.displayName(r, uid, ""), CourseID: course.ID, TeacherID: c.UID, Type: "teacher_remind", Detail: in.Message}})
	writeJSON(w, http.StatusOK, map[string]any{"sent": true})
}

func fmtDur(sec int) string {
	if sec < 60 {
		return fmt.Sprintf("%d сек", sec)
	}
	if sec < 3600 {
		return fmt.Sprintf("%d мин", sec/60)
	}
	return fmt.Sprintf("%d ц %02d мин", sec/3600, sec%3600/60)
}
