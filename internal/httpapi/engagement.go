package httpapi

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"surgalt/internal/store"
)

const (
	reflectionMinWords = 3
	reflectionMaxChars = 2000
)

// handleSaveReflection: POST /api/courses/{id}/lessons/{lid}/reflect {"text":"..."}
// Хичээл үзсэний дараа "юу сурсан бэ?" гэсэн товч дүгнэлт. Идэвхтэй суралцсаны нэг нотолгоо.
func (s *Server) handleSaveReflection(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	if course.TeacherID == c.UID {
		writeErr(w, http.StatusBadRequest, "өөрийн хичээлд дүгнэлт бичихгүй")
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Text = strings.TrimSpace(in.Text)
	words := len(strings.Fields(in.Text))
	if utf8.RuneCountInString(in.Text) > reflectionMaxChars {
		writeErr(w, http.StatusBadRequest, "дүгнэлт 2000 тэмдэгтээс хэтрэхгүй")
		return
	}
	if words < reflectionMinWords {
		writeErr(w, http.StatusBadRequest, "дор хаяж 3 үгээр бичнэ үү")
		return
	}
	ref := &store.Reflection{UserID: c.UID, UserName: s.displayName(r.Context(), c.UID, c.Name), CourseID: course.ID, LessonID: l.ID, Lesson: l.Title,
		TeacherID: course.TeacherID, Text: in.Text, Words: words}
	if err := s.store.SaveReflection(r.Context(), ref); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, ref)
}

const videoWatchMaxBuckets = 600 // 600×10 сек = 100 минут хүртэлх видео

// handleVideoProgress: POST /api/courses/{id}/lessons/{lid}/progress/{bid} {"duration":320,"buckets":{"0":1,"1":2,...}}
// Видеоны аль 10 секундийн хэсгийг хэдэн удаа үзсэнийг бүртгэнэ (давтаж үзсэн ба алгассан хэсгийг олоход).
func (s *Server) handleVideoProgress(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	bid := r.PathValue("bid")
	if !blockIDRe.MatchString(bid) && bid != "main" {
		writeErr(w, http.StatusBadRequest, "буруу хэсэг")
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	if course.TeacherID == c.UID {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	var in struct {
		Duration int         `json:"duration"`
		Buckets  map[int]int `json:"buckets"`
	}
	if !decode(w, r, &in) {
		return
	}
	clean := make(map[int]int, len(in.Buckets))
	for k, n := range in.Buckets {
		if k < 0 || k >= videoWatchMaxBuckets || n <= 0 {
			continue
		}
		clean[k] = min(n, 50)
	}
	v := store.VideoWatch{UserID: c.UID, UserName: s.displayName(r.Context(), c.UID, c.Name), CourseID: course.ID, LessonID: l.ID, TeacherID: course.TeacherID, BlockID: bid,
		Duration: min(max(in.Duration, 0), videoWatchMaxBuckets*store.VideoBucketSec), Buckets: clean}
	if err := s.store.AddVideoWatch(r.Context(), v); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
