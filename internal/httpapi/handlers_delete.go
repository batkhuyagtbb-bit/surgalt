package httpapi

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Хичээл, ном устгахад цаана нь байгаа файлуудыг хамт цэвэрлэнэ. Файл багшийн санд нэг л удаа
// хадгалагддаг тул өөр хичээл/номд ашиглаж байгаа файлыг устгахгүй (зөвхөн өнчирсөн файл).

// lessonFileRefs — хичээлийн холбож буй багшийн сангийн замууд (/files/<teacher>/<vis>/<name>).
func lessonFileRefs(l *store.Lesson) []string {
	var out []string
	add := func(u string) {
		if u = strings.SplitN(u, "?", 2)[0]; strings.HasPrefix(u, "/files/") {
			out = append(out, u)
		}
	}
	add(l.VideoURL)
	for _, b := range l.Blocks {
		add(b.URL)
		if b.Quiz != nil {
			add(b.Quiz.Image)
		}
	}
	return out
}

// teacherFilesInUse — багшийн бусад хичээл, номын ашиглаж буй файлын замууд (exclude хичээлийг тооцохгүй).
func (s *Server) teacherFilesInUse(ctx context.Context, teacherID, excludeLesson, excludeBook string) map[string]bool {
	used := map[string]bool{}
	courses, _ := s.store.CoursesByTeacher(ctx, teacherID, false)
	for _, c := range courses {
		ls, _ := s.store.LessonsByCourse(ctx, c.ID)
		for i := range ls {
			if ls[i].ID == excludeLesson {
				continue
			}
			for _, p := range lessonFileRefs(&ls[i]) {
				used[p] = true
			}
		}
	}
	books, _ := s.store.BooksByTeacher(ctx, teacherID, false)
	for _, b := range books {
		if b.ID != excludeBook && b.CoverURL != "" {
			used[strings.SplitN(b.CoverURL, "?", 2)[0]] = true
		}
	}
	return used
}

// deleteOrphanFiles — refs дотроос багшийнх бөгөөд өөр хаана ч ашиглагдаагүй файлуудыг дискнээс устгана.
func (s *Server) deleteOrphanFiles(teacherID string, refs []string, used map[string]bool) int {
	n := 0
	seen := map[string]bool{}
	for _, p := range refs {
		if seen[p] || used[p] || !files.OwnedBy(p, teacherID) {
			continue
		}
		seen[p] = true
		parts := strings.Split(strings.TrimPrefix(p, "/files/"), "/") // teacher/vis/name
		if len(parts) != 3 {
			continue
		}
		name, err := url.PathUnescape(parts[2])
		if err != nil {
			continue
		}
		if s.files.Delete(teacherID, parts[1], name) == nil {
			n++
		}
	}
	return n
}

// handleDeleteLesson: DELETE /api/courses/{id}/lessons/{lid} — багш өөрийн хичээлийг устгана,
// зөвхөн энэ хичээлд ашиглагдсан файлууд хамт устна. Суралцагчдын явц, худалдан авалтын түүх хэвээр.
func (s *Server) handleDeleteLesson(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	l, err := s.store.LessonByID(r.Context(), course.ID, r.PathValue("lid"))
	if s.storeErr(w, r, err) {
		return
	}
	if err := s.store.DeleteLesson(r.Context(), course.ID, l.ID); s.storeErr(w, r, err) {
		return
	}
	removed := s.deleteOrphanFiles(c.UID, lessonFileRefs(l), s.teacherFilesInUse(r.Context(), c.UID, l.ID, ""))
	s.courses.Delete(course.ID)
	s.search.markDirty()
	s.profiles.Delete(c.Name)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_files": removed})
}
