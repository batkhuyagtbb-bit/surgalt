package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Багш нар хооронд хуваалцана (store.Share): сургалт, хичээл (шалгалт, даалгавар ч хичээл) тус бүр дээр багшийн
// имэйл (эсвэл @нэр) бичиж илгээнэ.
//   - «Харах, хадгалах» (ShareView): хүлээн авагч студидээ агуулгыг урьдчилан харж, өөрийн сургалтад хуулбарлан
//     хадгална (блок, асуулт, шалгалт, даалгавар, тохиргоо); файлууд нь түүний файлын сан руу хуулагдана —
//     багтаамжийг нь ашиглана, эх багш хожим файлаа устгасан ч нөлөөлөхгүй. Хуулбар «бэлтгэж байна» төлөвтэй үүснэ.
//   - «Хамт хөгжүүлэх» (ShareEdit, зөвхөн сургалтаар, зөвхөн эзэмшигч): хүлээн авагч Course.Teachers-т орж сургалтыг
//     студидээ нээж хичээл оруулж, засна (байгууллагын холбосон багштай ижил эрх). Хуваалцалтыг буцаахад гарна.

type shareView struct {
	ID          string     `json:"id"`
	Mode        string     `json:"mode"`
	CourseID    string     `json:"course_id"`
	CourseTitle string     `json:"course_title"`
	LessonID    string     `json:"lesson_id,omitempty"`
	LessonTitle string     `json:"lesson_title,omitempty"`
	LessonKind  string     `json:"lesson_kind,omitempty"` // lesson | exam | assignment
	From        TeacherRef `json:"from"`
	To          TeacherRef `json:"to"`
	CreatedAt   any        `json:"created_at"`
}

func lessonKind(l *store.Lesson) string {
	switch {
	case l.Exam != nil:
		return "exam"
	case l.Assignment != nil:
		return "assignment"
	}
	return "lesson"
}

// filePathRe — текстийн HTML доторх файлын сангийн замууд (зураг г.м).
var filePathRe = regexp.MustCompile(`/files/[0-9a-f]+/(?:public|private)/[^"'\s<>)]+`)

var errSharedFile = errors.New("хуваалцсан хичээлийн файл олдсонгүй")

// handleShareCourse: POST /api/courses/{id}/share {who, mode, lesson_id} — багшид хуваалцана.
func (s *Server) handleShareCourse(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Who      string `json:"who"`
		Mode     string `json:"mode"`
		LessonID string `json:"lesson_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	if in.Mode == "" {
		in.Mode = store.ShareView
	}
	var lesson *store.Lesson
	switch {
	case in.Mode != store.ShareView && in.Mode != store.ShareEdit:
		writeErr(w, http.StatusBadRequest, "эрх буруу")
		return
	case in.Mode == store.ShareEdit && course.TeacherID != c.UID:
		writeErr(w, http.StatusForbidden, "«Хамт хөгжүүлэх» эрхийг зөвхөн сургалтын эзэмшигч өгнө")
		return
	case in.Mode == store.ShareEdit && in.LessonID != "":
		writeErr(w, http.StatusBadRequest, "«Хамт хөгжүүлэх» эрх сургалтаар өгөгдөнө — хичээлээр «Харах, хадгалах» эрх өгнө")
		return
	case in.LessonID != "":
		var err error
		if lesson, err = s.store.LessonByID(ctx, course.ID, in.LessonID); s.storeErr(w, r, err) {
			return
		}
	}
	t, err := s.resolveTeacher(ctx, in.Who)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusNotFound, "ийм багш олдсонгүй — surgalt.mn-д бүртгэлтэй имэйл эсвэл @нэрийг бичнэ үү")
		return
	case s.storeErr(w, r, err):
		return
	case t.ID == c.UID:
		writeErr(w, http.StatusBadRequest, "өөртөө хуваалцахгүй")
		return
	case teaches(course, t.ID):
		writeErr(w, http.StatusConflict, "«"+t.DisplayName+"» энэ сургалтыг аль хэдийн хамт заадаг")
		return
	}
	existing, err := s.store.SharesOfCourse(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if slices.ContainsFunc(existing, func(x store.Share) bool { return x.ToID == t.ID && x.LessonID == in.LessonID && x.Mode == in.Mode }) {
		writeErr(w, http.StatusConflict, "«"+t.DisplayName+"»-д аль хэдийн хуваалцсан")
		return
	}
	if in.Mode == store.ShareEdit {
		if len(course.Teachers) >= maxCourseTeachers {
			writeErr(w, http.StatusBadRequest, "нэг сургалтыг 10 хүртэл багш хамт хөгжүүлнэ")
			return
		}
		updated, err := s.store.SetCourseOrg(ctx, course.TeacherID, course.ID, course.ProgramID, append(slices.Clone(course.Teachers), t.ID))
		if s.storeErr(w, r, err) {
			return
		}
		s.courseEdited(ctx, updated)
	}
	sh := &store.Share{FromID: c.UID, ToID: t.ID, CourseID: course.ID, LessonID: in.LessonID, Mode: in.Mode}
	if err := s.store.CreateShare(ctx, sh); s.storeErr(w, r, err) {
		return
	}
	refs := s.peopleRefs(ctx, []string{c.UID, t.ID})
	what, title := "сургалт", course.Title
	if lesson != nil {
		what, title = map[string]string{"exam": "шалгалт", "assignment": "даалгавар", "lesson": "хичээл"}[lessonKind(lesson)], lesson.Title
	}
	body := "«" + title + "» — харах, өөрийн сургалтад хадгалах эрхтэй. Студийн «Сургалтууд» → «Надад хуваалцсан» хэсэгт."
	if in.Mode == store.ShareEdit {
		body = "«" + title + "» — хамт хөгжүүлэх эрхтэй: студийн «Сургалтууд» хэсэгт нээж хичээл оруулж, засна."
	}
	s.notify(ctx, &store.Notification{UserID: t.ID, Type: NotifAccount, Count: 1, ActorID: c.UID,
		Title: "🔗 " + refs[c.UID].DisplayName + " танд " + what + " хуваалцлаа", Body: body, Link: "/me#courses"})
	s.hub.SendUsers([]string{t.ID}, map[string]any{"type": "share"})
	s.log.Info("хуваалцлаа", "from", c.UID, "to", t.ID, "course", course.ID, "lesson", in.LessonID, "mode", in.Mode)
	writeJSON(w, http.StatusCreated, s.shareViews(ctx, []store.Share{*sh}, course)[0])
}

// shareViews — хуваалцалтуудыг нэр, гарчигтай харагдац болгоно (course өгвөл бүгд тэр сургалтынх).
func (s *Server) shareViews(ctx context.Context, list []store.Share, course *store.Course) []shareView {
	ids := []string{}
	courses := map[string]*store.Course{}
	for _, sh := range list {
		ids = append(ids, sh.FromID, sh.ToID)
		if course != nil && sh.CourseID == course.ID {
			courses[sh.CourseID] = course
		} else if _, ok := courses[sh.CourseID]; !ok {
			if co, err := s.store.CourseByID(ctx, sh.CourseID); err == nil {
				courses[sh.CourseID] = co
			}
		}
	}
	refs := s.peopleRefs(ctx, ids)
	out := make([]shareView, 0, len(list))
	for _, sh := range list {
		co := courses[sh.CourseID]
		if co == nil {
			continue // сургалт устсан
		}
		v := shareView{ID: sh.ID, Mode: sh.Mode, CourseID: co.ID, CourseTitle: co.Title, LessonID: sh.LessonID, From: refs[sh.FromID], To: refs[sh.ToID], CreatedAt: sh.CreatedAt}
		if sh.LessonID != "" {
			l, err := s.store.LessonByID(ctx, co.ID, sh.LessonID)
			if err != nil {
				continue // хичээл устсан
			}
			v.LessonTitle, v.LessonKind = l.Title, lessonKind(l)
		}
		out = append(out, v)
	}
	return out
}

// handleCourseShares: GET /api/courses/{id}/shares — энэ сургалтын хуваалцалтууд (багш нарт).
func (s *Server) handleCourseShares(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	list, err := s.store.SharesOfCourse(r.Context(), course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shares": s.shareViews(r.Context(), list, course)})
}

// handleDeleteShare: DELETE /api/courses/{id}/shares/{sid} — хуваалцалтыг буцаана (хамт хөгжүүлэгч багш гарна).
func (s *Server) handleDeleteShare(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	ctx := r.Context()
	sh, err := s.store.ShareByID(ctx, r.PathValue("sid"))
	if errors.Is(err, store.ErrNotFound) || (err == nil && sh.CourseID != course.ID) {
		writeErr(w, http.StatusNotFound, "хуваалцалт олдсонгүй")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	if sh.Mode == store.ShareEdit {
		if course.TeacherID != c.UID {
			writeErr(w, http.StatusForbidden, "хамт хөгжүүлэгчийг зөвхөн эзэмшигч хасна")
			return
		}
		if slices.Contains(course.Teachers, sh.ToID) {
			updated, err := s.store.SetCourseOrg(ctx, course.TeacherID, course.ID, course.ProgramID, slices.DeleteFunc(slices.Clone(course.Teachers), func(id string) bool { return id == sh.ToID }))
			if s.storeErr(w, r, err) {
				return
			}
			s.courseEdited(ctx, updated)
		}
	}
	if err := s.store.DeleteShare(ctx, sh.ID); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleMyShared: GET /api/me/shared — надад хуваалцсан сургалт, хичээлүүд.
func (s *Server) handleMyShared(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	list, err := s.store.SharesTo(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shares": s.shareViews(r.Context(), list, nil)})
}

// myShare — надад хуваалцсан нэг хуваалцалт ба түүний сургалт.
func (s *Server) myShare(w http.ResponseWriter, r *http.Request, uid string) (*store.Share, *store.Course, bool) {
	sh, err := s.store.ShareByID(r.Context(), r.PathValue("sid"))
	if errors.Is(err, store.ErrNotFound) || (err == nil && sh.ToID != uid) {
		writeErr(w, http.StatusNotFound, "хуваалцалт олдсонгүй")
		return nil, nil, false
	}
	if s.storeErr(w, r, err) {
		return nil, nil, false
	}
	course, err := s.store.CourseByID(r.Context(), sh.CourseID)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "хуваалцсан сургалт устсан")
		return nil, nil, false
	}
	if s.storeErr(w, r, err) {
		return nil, nil, false
	}
	return sh, course, true
}

// sharedLesson — хуваалцалтад багтах хичээл (сургалтаар бол дурын, хичээлээр бол зөвхөн тэр).
func (s *Server) sharedLesson(ctx context.Context, sh *store.Share, lid string) (*store.Lesson, error) {
	if sh.LessonID != "" && lid != sh.LessonID {
		return nil, store.ErrNotFound
	}
	return s.store.LessonByID(ctx, sh.CourseID, lid)
}

// handleSharedView: GET /api/me/shared/{sid}?lesson= — урьдчилан харах: сургалтын хичээлийн жагсаалт, нэг хичээлийн
// бүрэн агуулга (медиа нь гарын үсэгтэй холбоосоор — багш студидээ харна).
func (s *Server) handleSharedView(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	sh, course, ok := s.myShare(w, r, c.UID)
	if !ok {
		return
	}
	ctx := r.Context()
	out := map[string]any{"share": s.shareViews(ctx, []store.Share{*sh}, course)[0],
		"course": map[string]any{"id": course.ID, "title": course.Title, "description": course.Description, "published": course.Published}}
	lid := r.URL.Query().Get("lesson")
	if lid == "" {
		lid = sh.LessonID
	}
	if sh.LessonID == "" {
		ls, err := s.store.LessonsByCourse(ctx, course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		rows := make([]map[string]any, 0, len(ls))
		for i := range ls {
			l := &ls[i]
			rows = append(rows, map[string]any{"id": l.ID, "title": l.Title, "section": l.Section, "kind": lessonKind(l), "hidden": l.Hidden, "is_free": l.IsFree, "blocks": len(l.Blocks)})
		}
		out["lessons"] = rows
	}
	if lid != "" {
		l, err := s.sharedLesson(ctx, sh, lid)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "хичээл олдсонгүй")
			return
		}
		if s.storeErr(w, r, err) {
			return
		}
		var cp store.Lesson
		if b, err := json.Marshal(l); err != nil || json.Unmarshal(b, &cp) != nil {
			writeErr(w, http.StatusInternalServerError, "хичээлийг уншиж чадсангүй")
			return
		}
		cp.VideoURL = s.media(cp.VideoURL)
		for j := range cp.Blocks {
			b := &cp.Blocks[j]
			if b.URL != "" {
				if parts := s.files.Parts(b.URL); b.Type == "video" && len(parts) > 1 {
					b.Parts = make([]string, len(parts))
					for k, p := range parts {
						b.Parts[k] = s.media(p)
					}
				}
				b.URL = s.media(b.URL)
			}
			if b.Quiz != nil && b.Quiz.Image != "" {
				q := *b.Quiz
				q.Image = s.media(q.Image)
				b.Quiz = &q
			}
			b.HTML = s.richMedia(b.HTML)
		}
		out["lesson"] = cp
	}
	writeJSON(w, http.StatusOK, out)
}

// handleSharedSave: POST /api/me/shared/{sid}/save {course_id, section, lesson_id} — хуваалцсан хичээл(үүд)ийг өөрийн
// сургалтад хуулбарлан хадгална. Сургалтаар хуваалцсан бол lesson_id хоосон → бүх хичээл (бүлэг нь хэвээр).
func (s *Server) handleSharedSave(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	sh, src, ok := s.myShare(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		CourseID string `json:"course_id"`
		Section  string `json:"section"`
		LessonID string `json:"lesson_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	target, err := s.store.CourseByID(ctx, in.CourseID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && !teaches(target, c.UID)) {
		writeErr(w, http.StatusNotFound, "хадгалах сургалт олдсонгүй")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	if !s.planOpen(w, r, target.TeacherID) {
		return
	}
	var srcLessons []store.Lesson
	switch {
	case in.LessonID != "" || sh.LessonID != "":
		lid := in.LessonID
		if lid == "" {
			lid = sh.LessonID
		}
		l, err := s.sharedLesson(ctx, sh, lid)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "хичээл олдсонгүй")
			return
		}
		if s.storeErr(w, r, err) {
			return
		}
		srcLessons = []store.Lesson{*l}
	default:
		if srcLessons, err = s.store.LessonsByCourse(ctx, src.ID); s.storeErr(w, r, err) {
			return
		}
	}
	owner, err := s.store.UserByID(ctx, target.TeacherID)
	if s.storeErr(w, r, err) {
		return
	}
	cpFile := s.fileCopier(target.TeacherID, s.quotaOf(owner))
	saved := []*store.Lesson{}
	for i := range srcLessons {
		section := in.Section
		if section == "" && len(srcLessons) > 1 {
			section = srcLessons[i].Section
		}
		nl, err := s.copyLesson(ctx, target, &srcLessons[i], section, cpFile)
		if err != nil {
			var ce *copyErr
			if errors.As(err, &ce) {
				writeErr(w, ce.status, ce.msg)
			} else if errors.Is(err, errSharedFile) {
				writeErr(w, http.StatusNotFound, "хуваалцсан хичээлийн файл олдсонгүй (эх багш устгасан байж магадгүй)")
			} else if s.storeErr(w, r, err) {
				// файлын сангийн алдаа (багтаамж г.м) storeErr-т танигдахгүй бол доор
			} else {
				s.filesErr(w, err)
			}
			if len(saved) > 0 {
				s.courseEdited(ctx, target)
			}
			return
		}
		saved = append(saved, nl)
	}
	s.courseEdited(ctx, target)
	s.log.Info("хуваалцсан хичээл хадгалав", "share", sh.ID, "to", c.UID, "course", target.ID, "n", len(saved))
	writeJSON(w, http.StatusCreated, map[string]any{"lessons": saved, "count": len(saved)})
}

type copyErr struct {
	status int
	msg    string
}

func (e *copyErr) Error() string { return e.msg }

// copyLesson — src хичээлийг target сургалтад «бэлтгэж байна» төлөвтэй хуулбарлана; файлуудыг cpFile-ээр хуулна.
func (s *Server) copyLesson(ctx context.Context, target *store.Course, src *store.Lesson, section string, cpFile func(string) (string, error)) (*store.Lesson, error) {
	var cp store.Lesson // гүн хуулбар: кэшээс ирсэн хичээлийн блок, асуултыг өөрчлөхгүй
	if b, err := json.Marshal(src); err != nil || json.Unmarshal(b, &cp) != nil {
		return nil, &copyErr{http.StatusInternalServerError, "хичээлийг хуулж чадсангүй"}
	}
	in := lessonInput{Hidden: true, Title: cp.Title, Content: cp.Content, IsFree: cp.IsFree, Price: cp.Price, UnlockAfterH: cp.UnlockAfterH,
		AlwaysOpen: cp.AlwaysOpen, Format: cp.Format, Mode: cp.Mode, Section: section, Blocks: cp.Blocks, ActiveMin: cp.ActiveMin,
		Exam: cp.Exam, Assignment: cp.Assignment, Discussion: cp.Discussion, UnlockRule: cp.UnlockRule}
	var err error
	if in.VideoURL, err = cpFile(cp.VideoURL); err != nil {
		return nil, err
	}
	for i := range in.Blocks {
		b := &in.Blocks[i]
		b.Parts = nil
		switch b.Type {
		case "image", "audio", "video", "file":
			if b.URL, err = cpFile(b.URL); err != nil {
				return nil, err
			}
		case "text":
			if strings.Contains(b.HTML, "/files/") {
				var ferr error
				b.HTML = filePathRe.ReplaceAllStringFunc(b.HTML, func(p string) string {
					np, err := cpFile(p)
					if err != nil {
						ferr = err
						return p
					}
					return np
				})
				if ferr != nil {
					return nil, ferr
				}
			}
		case "quiz":
			if b.Quiz != nil && b.Quiz.Image != "" {
				if b.Quiz.Image, err = cpFile(b.Quiz.Image); err != nil {
					return nil, err
				}
			}
		}
	}
	if msg := in.validate(target.TeacherID, target); msg != "" {
		return nil, &copyErr{http.StatusBadRequest, "хуваалцсан хичээл: " + msg}
	}
	nl := lessonFromInput(in, target.ID)
	if err := s.store.CreateLesson(ctx, nl); err != nil {
		return nil, err
	}
	return nl, nil
}

// fileCopier — өөр багшийн файлын сангийн замыг toID багшийн сан руу хуулж шинэ замыг өгнө (ижил зам нэг л удаа
// хуулагдана). Гадаад холбоос, аль хэдийн өөрийн файл бол өөрчлөхгүй.
func (s *Server) fileCopier(toID string, quota int64) func(string) (string, error) {
	done := map[string]string{}
	return func(u string) (string, error) {
		u = stripSig(strings.TrimSpace(u))
		if !strings.HasPrefix(u, "/files/") || files.OwnedBy(u, toID) {
			return u, nil
		}
		if v, ok := done[u]; ok {
			return v, nil
		}
		teacher, vis, name, ok := files.SplitPath(u)
		if !ok || s.files == nil {
			return "", errSharedFile
		}
		f, _, err := s.files.Open(teacher, vis, name)
		if err != nil {
			return "", errSharedFile
		}
		defer f.Close()
		orig := name // <санамсаргүй>__<эх нэр> → эх нэрээр
		if i := strings.Index(name, "__"); i >= 0 {
			orig = name[i+2:]
		}
		info, err := s.files.SaveOpts(toID, vis, orig, f, quota, false)
		if err != nil {
			return "", err
		}
		done[u] = info.Path
		return info.Path, nil
	}
}
