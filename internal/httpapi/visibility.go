package httpapi

import (
	"net/http"

	"surgalt/internal/store"
)

// Хичээлийн харагдах байдал: багш бэлтгэж дуусаагүй хичээлээ "Хаалттай" болговол суралцагчдад огт
// харагдахгүй — сургалтын хуудасны жагсаалт, хайлт, хичээлийн тоо, дараалсан нээлт, цол, шууд холбоос,
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
