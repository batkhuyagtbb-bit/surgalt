package httpapi

import (
	"net/http"
	"strings"

	"surgalt/internal/store"
)

// Хичээлийн харагдах байдал: багш бэлтгэж дуусаагүй хичээлээ "Хаалттай" болговол суралцагчдад огт
// харагдахгүй — сургалтын хуудасны жагсаалт, хайлт, хичээлийн тоо, дараалсан нээлт, эрэмбэ, шууд холбоос,
// худалдан авалт бүгдээс хасагдана. Багш өөрөө бүгдийг харж, засна.

// visibleLessons — суралцагчид харагдах хичээлүүд (owner=true бол бүгд).
func visibleLessons(ls []store.Lesson, owner bool) []store.Lesson {
	if owner {
		return ls
	}
	out := make([]store.Lesson, 0, len(ls))
	for _, l := range ls {
		if !l.Hidden {
			out = append(out, l)
		}
	}
	return out
}

// hiddenFrom — хичээл энэ хэрэглэгчээс нуугдсан эсэх (багш өөрийн хичээлийг үргэлж харна).
func hiddenFrom(l *store.Lesson, course *store.Course, uid string) bool {
	return l.Hidden && !teaches(course, uid)
}

var errLessonHidden = apiError(http.StatusNotFound, "хичээл олдсонгүй", nil)

// sectionClosed — бүлэг (модуль) хаалттай эсэх: exceptID-ээс бусад дор хаяж нэг хичээлтэй, бүгд нь хаалттай.
// Хаалттай модульд нэмсэн/зөөсөн хичээл ч хаалттай болж, модуль бүхэлдээ хамт хаалттай хэвээр үлдэнэ.
func sectionClosed(ls []store.Lesson, section string, skip func(id string) bool) bool {
	n, key := 0, secKey(section)
	for _, l := range ls {
		if (skip != nil && skip(l.ID)) || secKey(l.Section) != key {
			continue
		}
		if !l.Hidden {
			return false
		}
		n++
	}
	return n > 0
}

// secKey — бүлгийн нэрийг харьцуулах хэлбэрт (илүүдэл зай хасна).
func secKey(s string) string { return strings.Join(strings.Fields(s), " ") }

// hideMovedIntoClosed — хичээлүүд бүлэг сольсны дараа: нээлттэй хичээл хаалттай модуль руу орсон бол хамт хаагдана.
// before — өмнөх төлөв, after — шинэ төлөв (бүлэг нь шинэчлэгдсэн).
func (s *Server) hideMovedIntoClosed(r *http.Request, courseID string, before, after []store.Lesson) error {
	old := make(map[string]string, len(before))
	for _, l := range before {
		old[l.ID] = secKey(l.Section)
	}
	moved := map[string]bool{}
	for _, l := range after {
		if prev, ok := old[l.ID]; ok && prev != secKey(l.Section) && !l.Hidden {
			moved[l.ID] = true
		}
	}
	for _, l := range after {
		if moved[l.ID] && sectionClosed(after, l.Section, func(id string) bool { return moved[id] }) {
			if err := s.store.SetLessonHidden(r.Context(), courseID, l.ID, true); err != nil {
				return err
			}
		}
	}
	return nil
}

// handleSectionVisibility: PUT /api/courses/{id}/section-visibility {section, hidden} — модулийн бүх хичээлийг
// нэг дор, нэг түгжээн дор нээх/хаах (хичээл бүрт тусдаа хүсэлт илгээвэл зарим нь амжилтгүй болж хагас нээлттэй үлдэж болно).
func (s *Server) handleSectionVisibility(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Section string `json:"section"`
		Hidden  bool   `json:"hidden"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Section = strings.Join(strings.Fields(in.Section), " ")
	n, err := s.store.SetSectionHidden(r.Context(), course.ID, in.Section, in.Hidden)
	if s.storeErr(w, r, err) {
		return
	}
	s.courseEdited(r.Context(), course)
	writeJSON(w, http.StatusOK, map[string]any{"section": in.Section, "hidden": in.Hidden, "changed": n})
}

// handleLessonVisibility: PUT /api/courses/{id}/lessons/{lid}/visibility {hidden} — багш хичээлийг
// суралцагчдад нээх / хаах (бэлтгэж дуусаагүй).
func (s *Server) handleLessonVisibility(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Hidden bool `json:"hidden"`
	}
	if !decode(w, r, &in) {
		return
	}
	if err := s.store.SetLessonHidden(r.Context(), course.ID, r.PathValue("lid"), in.Hidden); s.storeErr(w, r, err) {
		return
	}
	s.courseEdited(r.Context(), course) // сургалтын хуудас, хайлт, профайл шууд шинэчлэгдэнэ
	writeJSON(w, http.StatusOK, map[string]any{"hidden": in.Hidden})
}
