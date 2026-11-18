package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
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
	// LessonCount — сургалтаар хуваалцсан бол бүх хичээлийн тоо (хаалттай нь ч; хүлээн авагч бүгдийг харна).
	LessonCount int        `json:"lesson_count,omitempty"`
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
		writeErr(w, http.StatusBadRequest, "эрхийн төрлөө сонгоно уу")
		return
	case in.Mode == store.ShareEdit && course.TeacherID != c.UID:
		writeErr(w, http.StatusForbidden, "«Хамт хөгжүүлэх» эрхийг зөвхөн сургалтын эзэмшигч өгнө")
		return
	case in.Mode == store.ShareEdit && in.LessonID != "":
		writeErr(w, http.StatusBadRequest, "«Хамт хөгжүүлэх» эрхийг бүхэл сургалтад өгнө — хичээлд зөвхөн «Харах, хадгалах» эрх өгөх боломжтой")
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
		writeErr(w, http.StatusBadRequest, "өөртөө хуваалцах боломжгүй")
		return
	case teaches(course, t.ID):
		writeErr(w, http.StatusConflict, "«"+t.DisplayName+"» энэ сургалтад аль хэдийн хамт хөгжүүлэгчээр нэгдсэн байна")
		return
	}
	existing, err := s.store.SharesOfCourse(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if slices.ContainsFunc(existing, func(x store.Share) bool { return x.ToID == t.ID && x.LessonID == in.LessonID && x.Mode == in.Mode }) {
		writeErr(w, http.StatusConflict, "«"+t.DisplayName+"»-д аль хэдийн хуваалцсан байна")
		return
	}
	if in.Mode == store.ShareEdit {
		if len(course.Teachers) >= maxCourseTeachers {
			writeErr(w, http.StatusBadRequest, "нэг сургалтыг 10 хүртэл багш хамт хөгжүүлэх боломжтой")
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
	body := "«" + title + "» — та үүнийг харж, өөрийн сургалтад хадгалах боломжтой. Студийн «Сургалтууд» → «Надад хуваалцсан» хэсгээс нээнэ үү."
	link := "/me#courses"
	switch {
	case in.Mode == store.ShareEdit:
		body = "«" + title + "» — та хамт хөгжүүлэх эрхтэй боллоо: студийн «Сургалтууд» хэсгээс нээж, хичээл нэмж, засах боломжтой."
		link = "/me#course=" + course.ID
	case lesson == nil:
		body = "«" + title + "» — хичээлүүдийг нь харж, өөрийн сургалт болгон хуулж аваад дараалал, бүлгийг нь өөрөө зохион байгуулах боломжтой."
		link = "/me#shared=" + sh.ID
	}
	s.notify(ctx, &store.Notification{UserID: t.ID, Type: NotifAccount, Count: 1, ActorID: c.UID,
		Title: "🔗 " + refs[c.UID].DisplayName + " танд " + what + " хуваалцлаа", Body: body, Link: link})
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
		} else if ls, err := s.store.LessonsByCourse(ctx, co.ID); err == nil {
			v.LessonCount = len(ls)
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
			writeErr(w, http.StatusForbidden, "хамт хөгжүүлэгчийг зөвхөн сургалтын эзэмшигч хасна")
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
		writeErr(w, http.StatusNotFound, "хуваалцсан сургалт устгагдсан байна")
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
		"course": map[string]any{"id": course.ID, "title": course.Title, "description": course.Description, "published": course.Published, "cover": course.Cover}}
	lid := r.URL.Query().Get("lesson")
	if lid == "" {
		lid = sh.LessonID
	}
	if sh.LessonID == "" {
		ls, err := s.store.LessonsByCourse(ctx, course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		slices.SortStableFunc(ls, func(a, b store.Lesson) int { return a.Position - b.Position })
		fc := s.fileCopier(c.UID, 0)
		rows := make([]map[string]any, 0, len(ls))
		for i := range ls {
			l := &ls[i]
			n, _ := fc.need(nil, lessonFiles(ls[i:i+1])) // хуулахад хэрэгтэй хэмжээ (өөрийн санд)
			rows = append(rows, map[string]any{"id": l.ID, "title": l.Title, "section": l.Section, "kind": lessonKind(l), "hidden": l.Hidden, "is_free": l.IsFree, "blocks": len(l.Blocks), "bytes": n})
		}
		out["lessons"] = rows
	}
	// Хүлээн авагчийн файлын сан: хуулахаас өмнө багтах эсэхийг студи харуулна.
	if u, err := s.store.UserByID(ctx, c.UID); err == nil && s.files != nil {
		if used, err := s.files.Usage(c.UID); err == nil {
			out["storage"] = map[string]int64{"used": used, "quota": s.quotaOf(u)}
		}
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
			writeErr(w, http.StatusInternalServerError, "хичээлийг уншиж чадсангүй — дахин оролдоно уу")
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
		out["lesson_bytes"], _ = s.fileCopier(c.UID, 0).need(nil, lessonFiles([]store.Lesson{*l}))
	}
	writeJSON(w, http.StatusOK, out)
}

// handleSharedSave: POST /api/me/shared/{sid}/save — хуваалцсан хичээл(үүд)ийг өөрийн сургалтад хуулбарлан хадгална.
//
//	{course_id, section}        — байгаа сургалтад (өөрийн эсвэл хамт заадаг); хуулбар «бэлтгэж байна» төлөвтэй.
//	{new_course: true, title}   — шинэ, өөрийн сургалт болгоно: эх сургалтын тайлбар, толгойн зураг, үнэ, тохиргоо,
//	                              бүлэг, дараалал, хичээлийн нээлттэй/хаалттай төлөв хэвээр (сургалт өөрөө ноорог).
//	                              Хүлээн авагч дараа нь студидээ чирж, бүлэглэж, засаж, нэмж зохион байгуулна.
//	{lesson_id} | {lesson_ids}  — нэг эсвэл сонгосон хичээлүүд (эх дарааллаар); хоосон бол бүх хичээл (бүлэг нь хэвээр).
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
		CourseID  string   `json:"course_id"`
		NewCourse bool     `json:"new_course"`
		Title     string   `json:"title"`
		Section   string   `json:"section"`
		LessonID  string   `json:"lesson_id"`
		LessonIDs []string `json:"lesson_ids"`
	}
	if !decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	// Эхлээд эх хичээлүүдийг (алдаатай бол шинэ сургалт хоосон үлдэхгүй).
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
		all, err := s.store.LessonsByCourse(ctx, src.ID)
		if s.storeErr(w, r, err) {
			return
		}
		slices.SortStableFunc(all, func(a, b store.Lesson) int { return a.Position - b.Position })
		srcLessons = all
		if len(in.LessonIDs) > 0 {
			if len(in.LessonIDs) > 2000 {
				writeErr(w, http.StatusBadRequest, "хичээл хэт олон байна")
				return
			}
			want := map[string]bool{}
			for _, id := range in.LessonIDs {
				want[id] = true
			}
			srcLessons = slices.DeleteFunc(slices.Clone(all), func(l store.Lesson) bool { return !want[l.ID] })
			if len(srcLessons) != len(want) {
				writeErr(w, http.StatusNotFound, "сонгосон хичээлийн зарим нь олдсонгүй — хуудсаа сэргээнэ үү")
				return
			}
		}
	}
	if len(srcLessons) == 0 {
		writeErr(w, http.StatusBadRequest, "хадгалах хичээлээ сонгоно уу")
		return
	}
	// Хадгалах сургалтын эзэн — файлууд түүний санд хуулагдаж, багтаамжийг нь ашиглана (шинэ сургалт бол өөрөө).
	var target *store.Course
	ownerID := c.UID
	if !in.NewCourse {
		var err error
		target, err = s.store.CourseByID(ctx, in.CourseID)
		if errors.Is(err, store.ErrNotFound) || (err == nil && !teaches(target, c.UID)) {
			writeErr(w, http.StatusNotFound, "хадгалах сургалт олдсонгүй")
			return
		}
		if s.storeErr(w, r, err) {
			return
		}
		ownerID = target.TeacherID
	}
	if !s.planOpen(w, r, ownerID) {
		return
	}
	owner, err := s.store.UserByID(ctx, ownerID)
	if s.storeErr(w, r, err) {
		return
	}
	// Багтаамжийг урьдчилан шалгана: өмнө нь хуулж явсаар дундуур нь «дотоод алдаа» өгч, хагас хуулбар нь санг
	// дүүргэдэг байв.
	fc := s.fileCopier(ownerID, s.quotaOf(owner))
	var optional []string
	if in.NewCourse && sh.LessonID == "" {
		optional = []string{src.Cover}
	}
	need, err := fc.need(lessonFiles(srcLessons), optional)
	if errors.Is(err, errSharedFile) {
		writeErr(w, http.StatusNotFound, "хуваалцсан хичээлийн файл олдсонгүй (эх багш устгасан байж магадгүй)")
		return
	}
	if need > 0 && s.files != nil {
		used, err := s.files.Usage(ownerID)
		if err != nil {
			s.filesErr(w, err)
			return
		}
		if free := max(0, fc.quota-used); need > free {
			whose := "таны"
			if ownerID != c.UID {
				whose = "сургалтын эзэмшигчийн"
			}
			writeErr(w, http.StatusRequestEntityTooLarge, "Файлын сан хүрэлцэхгүй: сонгосон хичээлүүдийн файл "+mbLabel(need)+", "+whose+
				" санд "+mbLabel(free)+" сул байна (нийт "+mbLabel(fc.quota)+"). Цөөн хичээл сонгох эсвэл «Данс» → «Файлын сан» хэсгээс багтаамжаа нэмнэ үү.")
			return
		}
	}
	if in.NewCourse {
		var ok bool
		if target, ok = s.forkCourse(w, r, c.UID, src, srcLessons, sh, in.Title, fc); !ok {
			fc.rollback()
			return
		}
	}
	saved := []*store.Lesson{}
	for i := range srcLessons {
		section := in.Section
		if section == "" && (len(srcLessons) > 1 || in.NewCourse) {
			section = srcLessons[i].Section
		}
		// Шинэ сургалт ноорог тул эх хичээлийн нээлттэй/хаалттай төлөв хэвээр; байгаа сургалтад хаалттай орно.
		hidden := !in.NewCourse || srcLessons[i].Hidden
		nl, err := s.copyLesson(ctx, target, &srcLessons[i], section, hidden, fc.copy)
		if err != nil {
			var ce *copyErr
			switch {
			case errors.As(err, &ce):
				writeErr(w, ce.status, ce.msg)
			case errors.Is(err, errSharedFile):
				writeErr(w, http.StatusNotFound, "хуваалцсан хичээлийн файл олдсонгүй (эх багш устгасан байж магадгүй)")
			case errors.Is(err, files.ErrQuota), errors.Is(err, files.ErrTooLarge), errors.Is(err, files.ErrType):
				s.filesErr(w, err)
			default:
				s.storeErr(w, r, err)
			}
			// Бүгдийг буцаана: энэ удаа нэмсэн хичээл, хуулсан файл, шинэ сургалт үлдэхгүй.
			bg := context.WithoutCancel(ctx)
			for _, l := range saved {
				if err := s.store.DeleteLesson(bg, target.ID, l.ID); err != nil {
					s.log.Warn("хуулбар хичээлийг буцааж чадсангүй", "course", target.ID, "lesson", l.ID, "err", err)
				}
			}
			if in.NewCourse {
				if err := s.store.DeleteCourse(bg, target.TeacherID, target.ID); err != nil {
					s.log.Warn("хуулбар сургалтыг буцааж чадсангүй", "course", target.ID, "err", err)
				}
				s.profiles.Delete(c.Name)
				s.search.markDirty()
			} else if len(saved) > 0 {
				s.courseEdited(bg, target)
			}
			fc.rollback()
			return
		}
		saved = append(saved, nl)
	}
	s.courseEdited(ctx, target)
	s.log.Info("хуваалцсан хичээл хадгалав", "share", sh.ID, "to", c.UID, "course", target.ID, "new", in.NewCourse, "n", len(saved))
	writeJSON(w, http.StatusCreated, map[string]any{"lessons": saved, "count": len(saved), "course_id": target.ID})
}

// forkCourse — хуваалцсан сургалтаас хүлээн авагчийн өөрийн шинэ (ноорог) сургалтыг үүсгэнэ: гарчиг (өөрөө өөрчилж
// болно), тайлбар, толгойн зураг (файл бол түүний сан руу хуулна), үнэ, дараалал, камер, сертификат, сануулгын
// тохиргоо. Нууц үг, нийтлэх төлөв, байгууллагын хөтөлбөр, хамт заагч багш нар хуулагдахгүй.
func (s *Server) forkCourse(w http.ResponseWriter, r *http.Request, uid string, src *store.Course, srcLessons []store.Lesson, sh *store.Share, title string, fc *fileCopy) (*store.Course, bool) {
	if title = strings.TrimSpace(title); title == "" {
		title = src.Title
		if sh.LessonID != "" && len(srcLessons) == 1 {
			title = srcLessons[0].Title
		}
	}
	in := courseInput{Title: title, Description: src.Description, Price: src.Price, Drip: src.Drip, UnlockAllPaid: src.UnlockAllPaid,
		Camera: src.Camera, Certificate: src.Certificate, MaxWarnings: src.MaxWarnings, BlockHours: src.BlockHours, BlockMinutes: src.BlockMinutes}
	if sh.LessonID != "" { // нэг хичээлээр хуваалцсан бол сургалтын тайлбар, үнийг авахгүй
		in.Description, in.Price = "", 0
	}
	if msg := in.validate(); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return nil, false
	}
	ctx := r.Context()
	course := &store.Course{TeacherID: uid, Title: in.Title, Description: in.Description, Price: in.Price, Drip: in.Drip, UnlockAllPaid: in.UnlockAllPaid,
		Camera: in.Camera, Certificate: in.Certificate, MaxWarnings: in.MaxWarnings, BlockHours: in.BlockHours, BlockMinutes: in.BlockMinutes}
	if cover := src.Cover; cover != "" && sh.LessonID == "" {
		if strings.HasPrefix(cover, "/files/") {
			if np, err := fc.copy(cover); err == nil {
				cover = np
			}
		}
		if coverOK(cover, uid) == "" {
			course.Cover = cover // хуулж чадаагүй (устсан, багтаамж) бол градиент хэвээр
		}
	}
	if err := s.store.CreateCourse(ctx, course); s.storeErr(w, r, err) {
		return nil, false
	}
	return course, true
}

type copyErr struct {
	status int
	msg    string
}

func (e *copyErr) Error() string { return e.msg }

// copyLesson — src хичээлийг target сургалтад хуулбарлана (hidden бол «бэлтгэж байна» төлөвтэй); файлуудыг cpFile-ээр хуулна.
func (s *Server) copyLesson(ctx context.Context, target *store.Course, src *store.Lesson, section string, hidden bool, cpFile func(string) (string, error)) (*store.Lesson, error) {
	var cp store.Lesson // гүн хуулбар: кэшээс ирсэн хичээлийн блок, асуултыг өөрчлөхгүй
	if b, err := json.Marshal(src); err != nil || json.Unmarshal(b, &cp) != nil {
		return nil, &copyErr{http.StatusInternalServerError, "хичээлийг хуулж чадсангүй — дахин оролдоно уу"}
	}
	in := lessonInput{Hidden: hidden, Title: cp.Title, Content: cp.Content, IsFree: cp.IsFree, Price: cp.Price, UnlockAfterH: cp.UnlockAfterH,
		AlwaysOpen: cp.AlwaysOpen, Format: cp.Format, Mode: cp.Mode, Section: section, Blocks: cp.Blocks, ActiveMin: cp.ActiveMin,
		Exam: cp.Exam, Assignment: cp.Assignment, Discussion: cp.Discussion, UnlockRule: cp.UnlockRule}
	// Эх сургалтад «зөвхөн багцаар» нээгддэг хичээл: багц үнэгүй, нууц үггүй сургалтад худалдах аргагүй тул үнэгүй болно
	// (өмнө нь «төлбөртэй хичээлийн үнийг заана уу» алдаагаар хадгалагддаггүй байв). Хуулбарыг багш шалгаад тохируулна.
	if !in.IsFree && in.Price == 0 && target.Price == 0 && !target.Locked {
		in.IsFree = true
	}
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

// fileCopy — өөр багшийн файлын сангийн замыг toID багшийн сан руу хуулж шинэ замыг өгнө (ижил зам нэг л удаа
// хуулагдана). Гадаад холбоос, аль хэдийн өөрийн файл бол өөрчлөхгүй. Хуулсан файлуудыг санаж, алдаа гарвал
// rollback-аар устгана (хагас хуулбар хүлээн авагчийн санг дүүргэхгүй).
type fileCopy struct {
	s       *Server
	toID    string
	quota   int64
	done    map[string]string
	created []string
}

func (s *Server) fileCopier(toID string, quota int64) *fileCopy {
	return &fileCopy{s: s, toID: toID, quota: quota, done: map[string]string{}}
}

// foreign — хуулах шаардлагатай (өөр багшийн файлын сангийн) зам; үгүй бол "".
func (fc *fileCopy) foreign(u string) string {
	u = stripSig(strings.TrimSpace(u))
	if !strings.HasPrefix(u, "/files/") || files.OwnedBy(u, fc.toID) {
		return ""
	}
	return u
}

func (fc *fileCopy) copy(u string) (string, error) {
	src := fc.foreign(u)
	if src == "" {
		return stripSig(strings.TrimSpace(u)), nil
	}
	if v, ok := fc.done[src]; ok {
		return v, nil
	}
	teacher, vis, name, ok := files.SplitPath(src)
	if !ok || fc.s.files == nil {
		return "", errSharedFile
	}
	f, _, err := fc.s.files.Open(teacher, vis, name)
	if err != nil {
		return "", errSharedFile
	}
	defer f.Close()
	orig := name // <санамсаргүй>__<эх нэр> → эх нэрээр
	if i := strings.Index(name, "__"); i >= 0 {
		orig = name[i+2:]
	}
	info, err := fc.s.files.SaveOpts(fc.toID, vis, orig, f, fc.quota, false)
	if err != nil {
		return "", err
	}
	fc.done[src] = info.Path
	fc.created = append(fc.created, info.Path)
	return info.Path, nil
}

// need — эдгээр замуудыг хуулахад хэрэгтэй багтаамж (давхардлыг нэг удаа). Олдохгүй файл байвал errSharedFile;
// optional замууд (толгойн зураг) олдохгүй бол алгасна.
func (fc *fileCopy) need(paths, optional []string) (int64, error) {
	seen := map[string]bool{}
	var total int64
	for i, u := range append(slices.Clone(paths), optional...) {
		src := fc.foreign(u)
		if src == "" || seen[src] {
			continue
		}
		seen[src] = true
		teacher, vis, name, ok := files.SplitPath(src)
		var f *os.File
		var fi fs.FileInfo
		err := errSharedFile
		if ok && fc.s.files != nil {
			f, fi, err = fc.s.files.Open(teacher, vis, name)
		}
		if err != nil {
			if i >= len(paths) {
				continue
			}
			return 0, errSharedFile
		}
		f.Close()
		total += fi.Size()
	}
	return total, nil
}

// rollback — энэ удаа хуулсан файлуудыг устгана.
func (fc *fileCopy) rollback() {
	for _, p := range fc.created {
		if teacher, vis, name, ok := files.SplitPath(p); ok {
			_ = fc.s.files.Delete(teacher, vis, name)
		}
	}
	fc.created = nil
}

// lessonFiles — хичээлүүдийн файлын сангийн замууд (видео, блокийн медиа, текст доторх зураг, асуултын зураг).
func lessonFiles(ls []store.Lesson) []string {
	var out []string
	for i := range ls {
		l := &ls[i]
		out = append(out, l.VideoURL)
		for j := range l.Blocks {
			b := &l.Blocks[j]
			switch b.Type {
			case "image", "audio", "video", "file":
				out = append(out, b.URL)
			case "text":
				out = append(out, filePathRe.FindAllString(b.HTML, -1)...)
			case "quiz":
				if b.Quiz != nil {
					out = append(out, b.Quiz.Image)
				}
			}
		}
	}
	return out
}

// mbLabel — байтыг «12.3 MB» гэх мэт.
func mbLabel(n int64) string {
	switch {
	case n >= 1<<30:
		return strconv.FormatFloat(float64(n)/(1<<30), 'f', 2, 64) + " GB"
	case n >= 1<<20:
		return strconv.FormatFloat(float64(n)/(1<<20), 'f', 1, 64) + " MB"
	}
	return strconv.FormatInt(max(1, n>>10), 10) + " KB"
}
