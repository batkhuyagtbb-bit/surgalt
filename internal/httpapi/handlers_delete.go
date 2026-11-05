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
		for _, p := range richFileRefs(b.HTML) { // текст доторх зураг
			add(p)
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
		if b.ID == excludeBook {
			continue
		}
		for _, u := range []string{b.CoverURL, b.FileURL} {
			if u != "" {
				used[strings.SplitN(u, "?", 2)[0]] = true
			}
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
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
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
	removed := s.deleteOrphanFiles(course.TeacherID, lessonFileRefs(l), s.teacherFilesInUse(r.Context(), course.TeacherID, l.ID, ""))
	s.courseEdited(r.Context(), course)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_files": removed})
}

// handleDeleteCourse: DELETE /api/courses/{id} — багш өөрийн сургалтыг устгана. Суралцагчтай (элссэн эсвэл хичээл
// худалдаж авсан) сургалтыг устгахгүй — төлсөн суралцагчийн эрх алдагдана; тэр үед ноорог болгож нуухыг санал болгоно.
// Зөвхөн энэ сургалтын хичээлүүдэд ашиглагдсан файлууд хамт устна.
func (s *Server) handleDeleteCourse(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	ctx := r.Context()
	students, err := s.store.CourseStudents(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	n := 0
	for _, st := range students {
		if st.UserID != c.UID {
			n++
		}
	}
	if n > 0 {
		s.apiErr(w, r, apiError(http.StatusConflict, "Энэ сургалтад "+itoa(n)+" суралцагч байгаа тул устгах боломжгүй — «Тохиргоо»-оос ноорог болгож нуугаарай.", map[string]any{"students": n}))
		return
	}
	lessons, _ := s.store.LessonsByCourse(ctx, course.ID)
	if err := s.store.DeleteCourse(ctx, c.UID, course.ID); s.storeErr(w, r, err) {
		return
	}
	var refs []string
	for i := range lessons {
		refs = append(refs, lessonFileRefs(&lessons[i])...)
	}
	removed := s.deleteOrphanFiles(c.UID, refs, s.teacherFilesInUse(ctx, c.UID, "", ""))
	s.courses.Delete(course.ID)
	s.rosters.Delete(c.UID)
	s.search.markDirty()
	s.profiles.Delete(c.Name)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted_files": removed})
}
