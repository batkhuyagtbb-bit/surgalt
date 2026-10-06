package httpapi

import (
	"context"
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
	"teacher_unblock": {"Багш хоригийг цуцалсан", false, false},
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
func (s *Server) logEvents(ctx context.Context, evs []store.ActivityEvent) {
	if len(evs) == 0 {
		return
	}
	if err := s.store.AddActivityEvents(ctx, evs); err != nil {
		s.log.Warn("activity events", "err", err)
	}
	var ns []*store.Notification
	n := s.notif
	n.mu.Lock()
	var blocked []store.ActivityEvent
	for _, e := range evs {
		m := eventInfo[e.Type]
		if !m.Notify || e.UserID == e.TeacherID {
			continue
		}
		if e.Type == "auto_block" && e.LessonID != "" { // багшид улаан "Нээх" товчтой мэдэгдэл (тусад нь)
			blocked = append(blocked, e)
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
	for _, e := range blocked {
		title, what := e.LessonID, "хичээл"
		if l, err := s.store.LessonByID(ctx, e.CourseID, e.LessonID); err == nil {
			title = l.Title
			if l.Exam != nil {
				what = "шалгалт"
			}
		}
		body := short(e.Detail)
		if prev, err := s.store.ActivityEvents(ctx, store.ActivityFilter{UserID: e.UserID, CourseID: e.CourseID}, 500); err == nil {
			n := 0
			for _, x := range prev {
				if x.Type == "auto_block" {
					n++
				}
			}
			if n > 1 {
				body = fmt.Sprintf("Энэ сургалтад %d дахь удаагаа хаагдлаа · %s", n, body)
			}
		}
		ns = append(ns, &store.Notification{UserID: e.TeacherID, Type: "blocked", Title: "⛔ " + e.UserName + ": «" + title + "» " + what + " хаагдсан",
			Body: body, Link: "unblock:" + e.UserID + ":" + e.LessonID})
	}
	s.notify(ctx, ns...)
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
	out, err := s.StartActivity(r.Context(), c.UID, c.Name, s.clientIP(r), in.CourseID, in.LessonID, in.Kind)
	if s.apiErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) lessonActiveSec(ctx context.Context, uid, lessonID string) (int, error) {
	ss, err := s.store.Sessions(ctx, store.ActivityFilter{UserID: uid, LessonID: lessonID}, 1000)
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
	var in BeatInput
	if !decode(w, r, &in) {
		return
	}
	done, err := s.RecordBeat(r.Context(), c.UID, in)
	if s.apiErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "active_done_sec": done})
}

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
	teacher := s.displayName(r.Context(), c.UID, c.Name)
	s.notify(r.Context(), &store.Notification{UserID: uid, Type: "remind", Title: "📣 " + teacher + " багшаас сануулга", Body: short(in.Message), Link: "/c/" + course.ID})
	s.logEvents(r.Context(), []store.ActivityEvent{{UserID: uid, UserName: s.displayName(r.Context(), uid, ""), CourseID: course.ID, TeacherID: c.UID, Type: "teacher_remind", Detail: in.Message}})
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
