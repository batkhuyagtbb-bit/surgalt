package httpapi

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"time"

	"surgalt/internal/store"
)

// lessonForUser — суралцагч (эсвэл багш) энэ хичээлийг үзэх эрхтэй эсэхийг бүрэн шалгана:
// нийтлэгдсэн, төлбөр, дараалсан нээлт. Эрхгүй бол хариуг бичээд false.
func (s *Server) lessonForUser(w http.ResponseWriter, r *http.Request, uid, cid, lid string) (*store.Course, *store.Lesson, bool) {
	course, l, err := s.LessonFor(r.Context(), uid, cid, lid)
	if s.apiErr(w, r, err) {
		return nil, nil, false
	}
	return course, l, true
}

func validateExam(e *store.Exam) string {
	if e == nil {
		return ""
	}
	switch {
	case e.TimeMin < 0 || e.TimeMin > 600:
		return "шалгалтын хугацаа 0-600 минут"
	case e.Attempts < 0 || e.Attempts > 100:
		return "оролдлогын тоо 0-100"
	case e.PassPct < 0 || e.PassPct > 100:
		return "тэнцэх хувь 0-100"
	}
	return validateDue(&e.Due)
}

// examQuestions — шалгалтын асуултууд (оролдлогын дарааллаар).
func examQuestions(l *store.Lesson, order []string) []store.Block {
	byID := map[string]store.Block{}
	var ids []string
	for _, b := range l.Blocks {
		if b.Type == "quiz" && b.Quiz != nil {
			byID[b.ID] = b
			ids = append(ids, b.ID)
		}
	}
	if len(order) == 0 {
		order = ids
	}
	out := make([]store.Block, 0, len(order))
	for _, id := range order {
		if b, ok := byID[id]; ok {
			out = append(out, b)
		}
	}
	return out
}

func (s *Server) attemptView(a *store.ExamAttempt, l *store.Lesson) map[string]any {
	qs := examQuestions(l, a.Order)
	view := make([]store.Block, len(qs))
	for i, b := range qs {
		b.Quiz = s.viewerQuiz(b.ID, b.Quiz)
		view[i] = b
	}
	out := map[string]any{"attempt": a, "questions": view, "now": time.Now()}
	return out
}

// handleExamInfo: GET .../exam — тохиргоо ба өөрийн оролдлогууд (асуултыг харуулахгүй).
func (s *Server) handleExamInfo(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	_, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	if l.Exam == nil {
		writeErr(w, http.StatusNotFound, "энэ хичээл шалгалт биш")
		return
	}
	atts, err := s.store.ExamAttempts(r.Context(), store.ActivityFilter{UserID: c.UID, LessonID: l.ID})
	if s.storeErr(w, r, err) {
		return
	}
	left := -1
	if l.Exam.Attempts > 0 {
		left = max(0, l.Exam.Attempts-len(atts))
	}
	course, _ := s.store.CourseByID(r.Context(), r.PathValue("id"))
	ds := DueState{Open: true}
	if course != nil {
		ds = s.dueFor(r, c.UID, course, l)
	}
	writeJSON(w, http.StatusOK, map[string]any{"exam": l.Exam, "questions": len(examQuestions(l, nil)), "attempts": atts, "left": left, "due": ds})
}

// handleExamStart: POST .../exam/start — шинэ оролдлого (эсвэл дуусаагүйг үргэлжлүүлнэ).
func (s *Server) handleExamStart(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	if l.Exam == nil {
		writeErr(w, http.StatusNotFound, "энэ хичээл шалгалт биш")
		return
	}
	qs := examQuestions(l, nil)
	if len(qs) == 0 {
		writeErr(w, http.StatusConflict, "шалгалтад асуулт алга")
		return
	}
	if dueBlocked(w, s.dueFor(r, c.UID, course, l)) { // хугацаа: хаалттай 423, хоцорсон төлбөр 402
		return
	}
	atts, err := s.store.ExamAttempts(r.Context(), store.ActivityFilter{UserID: c.UID, LessonID: l.ID})
	if s.storeErr(w, r, err) {
		return
	}
	for i := range atts {
		a := &atts[i]
		if a.Status != store.AttemptActive {
			continue
		}
		if a.DeadlineAt == nil || time.Now().Before(*a.DeadlineAt) {
			writeJSON(w, http.StatusOK, s.attemptView(a, l)) // дахин ачаалсан: үргэлжлүүлнэ
			return
		}
		s.finishAttempt(r, a, l, course, nil, store.AttemptExpired, "хугацаа дууссан")
	}
	if l.Exam.Attempts > 0 && len(atts) >= l.Exam.Attempts && course.TeacherID != c.UID {
		writeErr(w, http.StatusConflict, fmt.Sprintf("оролдлогын тоо дууссан (%d)", l.Exam.Attempts))
		return
	}
	a := &store.ExamAttempt{UserID: c.UID, UserName: s.displayName(r.Context(), c.UID, c.Name), CourseID: course.ID, LessonID: l.ID, TeacherID: course.TeacherID,
		StartedAt: time.Now(), Status: store.AttemptActive}
	if l.Exam.TimeMin > 0 {
		d := a.StartedAt.Add(time.Duration(l.Exam.TimeMin) * time.Minute)
		a.DeadlineAt = &d
	}
	for _, b := range qs {
		a.Order = append(a.Order, b.ID)
	}
	if l.Exam.Shuffle {
		rand.Shuffle(len(a.Order), func(i, j int) { a.Order[i], a.Order[j] = a.Order[j], a.Order[i] })
	}
	if err := s.store.CreateExamAttempt(r.Context(), a); s.storeErr(w, r, err) {
		return
	}
	s.logEvents(r.Context(), []store.ActivityEvent{{UserID: c.UID, UserName: a.UserName, CourseID: course.ID, LessonID: l.ID, TeacherID: course.TeacherID, Type: "exam_start", Detail: l.Title}})
	writeJSON(w, http.StatusCreated, s.attemptView(a, l))
}

// handleExamSubmit: POST .../exam/submit {"attempt_id", "answers": {bid: QuizAnswer}, "terminate": "copy"}
// terminate хоосон биш бол зөрчлөөр хаагдсан — одоогийн хариултаар дүгнэнэ.
func (s *Server) handleExamSubmit(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		AttemptID string                `json:"attempt_id"`
		Answers   map[string]QuizAnswer `json:"answers"`
		Terminate string                `json:"terminate"`
	}
	if !decode(w, r, &in) {
		return
	}
	a, err := s.store.ExamAttemptByID(r.Context(), in.AttemptID)
	if err != nil || a.UserID != c.UID || a.LessonID != r.PathValue("lid") {
		writeErr(w, http.StatusNotFound, "оролдлого олдсонгүй")
		return
	}
	course, err := s.store.CourseByID(r.Context(), a.CourseID)
	if s.storeErr(w, r, err) {
		return
	}
	l, err := s.store.LessonByID(r.Context(), a.CourseID, a.LessonID)
	if s.storeErr(w, r, err) {
		return
	}
	if a.Status != store.AttemptActive {
		writeErr(w, http.StatusConflict, "энэ оролдлого аль хэдийн дууссан")
		return
	}
	status, reason := store.AttemptSubmitted, ""
	if in.Terminate != "" {
		status, reason = store.AttemptTerminated, eventLabel(in.Terminate)
	} else if a.DeadlineAt != nil && time.Now().After(a.DeadlineAt.Add(30*time.Second)) { // сүлжээний 30 сек хүлцэл
		status, reason = store.AttemptExpired, "хугацаа хэтэрсэн"
	}
	if !s.finishAttempt(r, a, l, course, in.Answers, status, reason) {
		writeErr(w, http.StatusConflict, "энэ оролдлого аль хэдийн дууссан")
		return
	}
	out := map[string]any{"attempt": a}
	if l.Exam != nil && l.Exam.ShowAnswers {
		rev := map[string]any{}
		for _, b := range examQuestions(l, nil) {
			rev[b.ID] = revealQuiz(b.ID, b.Quiz)
		}
		out["reveal"] = rev
	}
	writeJSON(w, http.StatusOK, out)
}

// finishAttempt нь дүгнэж хадгалаад багшид мэдэгдэнэ. Аль хэдийн дууссан бол false.
func (s *Server) finishAttempt(r *http.Request, a *store.ExamAttempt, l *store.Lesson, course *store.Course, answers map[string]QuizAnswer, status, reason string) bool {
	a.Results = map[string]bool{}
	a.Score, a.Max = 0, 0
	for _, b := range examQuestions(l, a.Order) {
		pts := quizPoints(b.Quiz)
		a.Max += pts
		ok := gradeQuiz(b.ID, b.Quiz, answers[b.ID])
		a.Results[b.ID] = ok
		if ok {
			a.Score += pts
		}
	}
	if a.Max > 0 {
		a.Pct = int(math.Round(a.Score / a.Max * 100))
	}
	pass := 0
	if l.Exam != nil {
		pass = l.Exam.PassPct
	}
	a.Passed = status == store.AttemptSubmitted && a.Pct >= pass
	now := time.Now()
	a.Status, a.Reason, a.FinishedAt = status, reason, &now
	if err := s.store.FinishExamAttempt(r.Context(), a); err != nil {
		if !errors.Is(err, store.ErrConflict) {
			s.log.Warn("exam finish", "err", err)
		}
		return false
	}
	typ, title := "exam_submit", fmt.Sprintf("📝 %s шалгалт өглөө: %d%%", a.UserName, a.Pct)
	if status == store.AttemptTerminated {
		typ, title = "exam_terminated", fmt.Sprintf("⛔ %s-ийн шалгалт зөрчлөөр хаагдлаа", a.UserName)
	}
	s.logEvents(r.Context(), []store.ActivityEvent{{UserID: a.UserID, UserName: a.UserName, CourseID: a.CourseID, LessonID: a.LessonID, TeacherID: a.TeacherID,
		Type: typ, Detail: fmt.Sprintf("%s · %d%% · %s", l.Title, a.Pct, reason)}})
	if a.UserID != course.TeacherID {
		s.notify(r.Context(), &store.Notification{UserID: course.TeacherID, Type: "exam", Title: title, Body: short(l.Title + " · " + reason), Link: "/me#students"})
	}
	if a.Passed {
		_ = s.store.MarkLessonCompleted(r.Context(), a.UserID, a.CourseID, a.LessonID)
	}
	return true
}

func (s *Server) displayName(ctx context.Context, uid, fallback string) string {
	if u, err := s.store.UserByID(ctx, uid); err == nil && u.DisplayName != "" {
		return u.DisplayName
	}
	return fallback
}
