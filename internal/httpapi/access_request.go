package httpapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// Нээлгэх хүсэлт: суралцагч нууц үгтэй сургалт (course.Locked) эсвэл дарааллын «багш гараар нээнэ» хичээлийг
// (UnlockRule "manual") нээж өгөхийг багшаас хүснэ. Багш(ууд)ад мэдэгдэл очиж (Link "grant:<uid>:<сургалт>:<хичээл>"),
// багш «Нээх» дарахад зөвхөн тэр суралцагчид нээгдэнэ: сургалт — нууц үг оруулсантай адил, хичээл — lesson_grants.
// Багш зөвхөн бодит хүсэлтийн (хадгалагдсан мэдэгдэл, 30 хоног) хариуд нээнэ — дурын хэрэглэгчийг элсүүлж, мэдэгдэл
// явуулж болохгүй.

const (
	accessReqWindow  = 12 * time.Hour      // нэг (хэрэглэгч, сургалт, хичээл) дээр нэг хүсэлт
	accessReqKeys    = 100_000             // давхардлын санах ойн дээд хэмжээ (процесс тус бүр)
	accessMsgMax     = 300                 // хүсэлтийн мессеж (тэмдэгт)
	grantProofWindow = 30 * 24 * time.Hour // «Нээх» нь энэ хугацаанд ирсэн хүсэлтэд л хүчинтэй
)

const msgAccessBusy = "хэт олон хүсэлт илгээлээ — түр хүлээгээд дахин оролдоно уу"

// accessDedup — нээлгэх хүсэлтийн давхардал: түлхүүр → хугацаа дуусах мөч (процесс доторх, хуулбар бүрт тусдаа).
type accessDedup struct {
	mu sync.Mutex
	m  map[string]time.Time
}

// sent — энэ түлхүүрээр сүүлийн 12 цагт хүсэлт явсан эсэх.
func (d *accessDedup) sent(key string, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	exp, ok := d.m[key]
	return ok && now.Before(exp)
}

// claim — хүсэлт явуулах эрхийг атомараар авна (false бол саяхан явсан). Дүүрвэл хугацаа нь дууссаныг, тэгээд ч
// дүүрэн бол дурын 10%-ийг хаяна: санах ой хязгаартай, хэрэглэгч бүрийн rate limit спамаас хамгаална.
func (d *accessDedup) claim(key string, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if exp, ok := d.m[key]; ok && now.Before(exp) {
		return false
	}
	if d.m == nil {
		d.m = map[string]time.Time{}
	}
	if len(d.m) >= accessReqKeys {
		for k, exp := range d.m {
			if !now.Before(exp) {
				delete(d.m, k)
			}
		}
		for k := range d.m {
			if len(d.m) < accessReqKeys*9/10 {
				break
			}
			delete(d.m, k)
		}
	}
	d.m[key] = now.Add(accessReqWindow)
	return true
}

// release — хүсэлт хүрээгүй (мэдэгдэл хадгалагдаагүй) бол тэмдгийг буцаана: суралцагч дахин илгээж болно.
func (d *accessDedup) release(key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.m, key)
}

// userDripState — суралцагчийн хувьд энэ хичээлийн дарааллын төлөв (GET хичээл, /access-тай ижил баримтаар).
func (s *Server) userDripState(ctx context.Context, uid string, course *store.Course, l *store.Lesson) (LessonState, error) {
	lessons, err := s.store.LessonsByCourse(ctx, course.ID)
	if err != nil {
		return LessonState{}, err
	}
	lessons = visibleLessons(lessons, false)
	progress, err := s.store.LessonProgress(ctx, uid, course.ID)
	if err != nil {
		return LessonState{}, err
	}
	enrolled, _ := s.store.IsEnrolled(ctx, uid, course.ID)
	return dripState(course, lessons, l, progress, fullAccess(course, uid, enrolled), time.Now(), s.dripFacts(ctx, uid, course.ID, lessons)), nil
}

// courseTeachers — эзэмшигч ба хамтран заагчид (давхардалгүй).
func courseTeachers(course *store.Course) []string {
	out := []string{course.TeacherID}
	for _, t := range course.Teachers {
		if t != "" && !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	return out
}

func grantLink(uid, courseID, lessonID string) string {
	return "grant:" + uid + ":" + courseID + ":" + lessonID
}

// handleAccessRequest: POST /api/courses/{id}/access-request {"lesson_id":"" | "<id>","message":"…"}
// → {"sent":true} | {"sent":false,"already":true} | {"open":true}.
func (s *Server) handleAccessRequest(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		LessonID string `json:"lesson_id"`
		Message  string `json:"message"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Message = strings.Join(strings.Fields(in.Message), " ")
	if utf8.RuneCountInString(in.Message) > accessMsgMax {
		writeErr(w, http.StatusBadRequest, "мессеж 300 тэмдэгтээс хэтрэхгүй байх ёстой")
		return
	}
	ctx := r.Context()
	course, err := s.store.CourseByID(ctx, r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	if !course.Published && !teaches(course, c.UID) { // байхгүй сургалттай адил
		writeErr(w, http.StatusNotFound, "мэдээлэл олдсонгүй — устгагдсан байж магадгүй")
		return
	}
	open := func() { writeJSON(w, http.StatusOK, map[string]any{"open": true}) }
	var lesson *store.Lesson
	if in.LessonID == "" { // нууц үгтэй сургалт
		if !course.Locked {
			writeErr(w, http.StatusBadRequest, "энэ сургалтад нууц үг тавиагүй байна")
			return
		}
		if teaches(course, c.UID) {
			open()
			return
		}
		if ok, err := s.codeUnlocked(ctx, c.UID, course); s.storeErr(w, r, err) {
			return
		} else if ok {
			open()
			return
		}
	} else { // «багш гараар нээнэ» хичээл
		l, err := s.store.LessonByID(ctx, course.ID, in.LessonID)
		if s.storeErr(w, r, err) {
			return
		}
		if hiddenFrom(l, course, c.UID) {
			s.apiErr(w, r, errLessonHidden)
			return
		}
		if !course.Drip || l.UnlockRule != "manual" || l.AlwaysOpen {
			writeErr(w, http.StatusBadRequest, "энэ хичээлийг багш гараар нээдэггүй")
			return
		}
		if teaches(course, c.UID) {
			open()
			return
		}
		// Сургалтад нэвтрэх эрхгүй (төлөөгүй, нууц үгээ оруулаагүй) бол эхлээд тэр эрхээ авна.
		has, err := s.lessonAccess(ctx, c.UID, course, l)
		if s.storeErr(w, r, err) {
			return
		}
		if !has {
			if course.Locked {
				writeJSON(w, http.StatusForbidden, map[string]any{"error": "Эхлээд багшаас авсан нууц үгээ оруулж сургалтыг нээнэ үү", "need_code": true})
				return
			}
			writeErr(w, http.StatusForbidden, "Эхлээд энэ хичээлийг эсвэл сургалтыг худалдаж авна уу")
			return
		}
		st, err := s.userDripState(ctx, c.UID, course, l)
		if s.storeErr(w, r, err) {
			return
		}
		if st.Reason != "manual" {
			open()
			return
		}
		lesson = l
	}
	key := c.UID + "|" + course.ID + "|" + in.LessonID
	now := time.Now()
	already := func() { writeJSON(w, http.StatusOK, map[string]any{"sent": false, "already": true}) }
	if s.accessReqs.sent(key, now) {
		already()
		return
	}
	// Хэрэглэгч бүрт (спам) ба сургалт бүрт (олон бүртгэлээс нэг багшийг дарахгүй) хязгаар.
	if !s.reqLim.Allow(c.UID) || !s.reqCourseLim.Allow(course.ID) {
		writeErr(w, http.StatusTooManyRequests, msgAccessBusy)
		return
	}
	if !s.accessReqs.claim(key, now) { // зэрэг ирсэн давхар товшилт
		already()
		return
	}
	name := s.displayName(ctx, c.UID, c.Name)
	tpl := store.Notification{Type: NotifAccessRequest, ActorID: c.UID, Body: course.Title,
		Title: "🔑 " + name + ": «" + course.Title + "» сургалтыг нээх хүсэлт",
		Link:  grantLink(c.UID, course.ID, in.LessonID)}
	if lesson != nil {
		tpl.Title = "🔐 " + name + ": «" + lesson.Title + "» хичээлийг нээх хүсэлт"
	}
	if in.Message != "" {
		tpl.Body = short(in.Message)
	}
	var ns []*store.Notification
	for _, uid := range courseTeachers(course) {
		if uid != c.UID {
			n := tpl
			n.UserID = uid
			ns = append(ns, &n)
		}
	}
	// Мэдэгдэл хадгалагдаагүй бол хүсэлт хүрээгүй: давхардлын тэмдгийг буцааж, алдааг харуулна (дахин оролдоно).
	dctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.deliverNotifs(dctx, ns...); err != nil {
		s.accessReqs.release(key)
		s.storeErr(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sent": true})
}

// handleGrantAccess: POST /api/me/students/{uid}/grant {"course_id":"…","lesson_id":"" | "<id>"} — тухайн сургалтын
// багш (эзэмшигч эсвэл хамтран заагч) суралцагчид нууц үгтэй сургалтыг эсвэл «багш гараар нээнэ» хичээлийг нээнэ.
// Зөвхөн холбоотой хэрэглэгчид: сургалт — 30 хоногт ирсэн хүсэлт, эсвэл өмнө нь нууц үгээр нээсэн (багш үгээ сольсон);
// хичээл — сургалтад эрхтэй бөгөөд 30 хоногт ирсэн хүсэлт. Бусад үед (байхгүй хэрэглэгч ч) ижил 404.
// → {"granted":true} | {"granted":false,"open":true} (түгжээгүй, эсвэл аль хэдийн нээлттэй).
func (s *Server) handleGrantAccess(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		CourseID string `json:"course_id"`
		LessonID string `json:"lesson_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !s.grantLim.Allow(c.UID) {
		writeErr(w, http.StatusTooManyRequests, msgAccessBusy)
		return
	}
	ctx := r.Context()
	uid := r.PathValue("uid")
	notFound := func() {
		writeErr(w, http.StatusNotFound, "мэдээлэл олдсонгүй — устгагдсан байж магадгүй")
	}
	// fail — ErrNotFound бол ижил 404 (хэрэглэгч, хичээл байгаа эсэхийг илрүүлэхгүй), бусад нь storeErr.
	fail := func(err error) bool {
		if err == nil {
			return false
		}
		if errors.Is(err, store.ErrNotFound) {
			notFound()
			return true
		}
		return s.storeErr(w, r, err)
	}
	course, err := s.store.CourseByID(ctx, in.CourseID)
	if fail(err) {
		return
	}
	if !teaches(course, c.UID) || !course.Published {
		notFound()
		return
	}
	if !s.planOpen(w, r, course.TeacherID) {
		return
	}
	if _, err := s.store.UserByID(ctx, uid); fail(err) {
		return
	}
	alreadyOpen := func() { writeJSON(w, http.StatusOK, map[string]any{"granted": false, "open": true}) }
	if teaches(course, uid) {
		alreadyOpen()
		return
	}
	since := time.Now().Add(-grantProofWindow)
	requested := func(lessonID string) (bool, error) {
		return s.store.HasNotification(ctx, courseTeachers(course), NotifAccessRequest, grantLink(uid, course.ID, lessonID), since)
	}
	n := &store.Notification{UserID: uid, Type: NotifAccessGranted, ActorID: c.UID}
	if in.LessonID == "" { // нууц үгтэй сургалт: нууц үг оруулсантай адил (эрх + элсэлт)
		asked, err := requested("")
		if s.storeErr(w, r, err) {
			return
		}
		if !asked { // багш үгээ сольсон: өмнө нь нууц үгээр (эсвэл багшаар) нээсэн хүнийг хүсэлтгүй дахин нээнэ
			h, err := s.store.CourseUnlock(ctx, uid, course.ID)
			if s.storeErr(w, r, err) {
				return
			}
			if h == "" {
				notFound()
				return
			}
		}
		if !course.Locked {
			alreadyOpen()
			return
		}
		if ok, err := s.codeUnlocked(ctx, uid, course); s.storeErr(w, r, err) {
			return
		} else if ok {
			alreadyOpen()
			return
		}
		if err := s.store.SetCourseUnlock(ctx, uid, course.ID, codeHash(course.AccessCode)); s.storeErr(w, r, err) {
			return
		}
		enrolled, err := s.store.IsEnrolled(ctx, uid, course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		if !enrolled {
			if err := s.store.Enroll(ctx, uid, course); s.storeErr(w, r, err) {
				return
			}
			s.rosterChanged(course.TeacherID)
			s.journals.Delete(course.ID)
		}
		n.Title, n.Body, n.Link = "🔓 Багш «"+course.Title+"» сургалтыг тань нээлээ", "Сургалтын бүх хичээлийг үзэх боломжтой боллоо — амжилт хүсье!", "/c/"+course.ID
	} else { // «багш гараар нээнэ» хичээл
		l, err := s.store.LessonByID(ctx, course.ID, in.LessonID) // өөр сургалтын хичээл бол 404
		if fail(err) {
			return
		}
		has, err := s.lessonAccess(ctx, uid, course, l) // сургалтад эрхгүй хүнд хичээл нээх утгагүй
		if s.storeErr(w, r, err) {
			return
		}
		if !has {
			notFound()
			return
		}
		if asked, err := requested(l.ID); s.storeErr(w, r, err) {
			return
		} else if !asked {
			notFound()
			return
		}
		if !course.Drip || l.UnlockRule != "manual" || l.AlwaysOpen {
			alreadyOpen()
			return
		}
		grants, err := s.store.LessonGrants(ctx, uid, course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		if grants[l.ID] {
			alreadyOpen()
			return
		}
		if err := s.store.GrantLesson(ctx, uid, course.ID, l.ID, c.UID); s.storeErr(w, r, err) {
			return
		}
		s.dripOpen.Delete(uid + "|" + l.ID)
		n.Title, n.Body, n.Link = "🔓 Багш «"+l.Title+"» хичээлийг тань нээлээ", course.Title, "/c/"+course.ID+"#l="+l.ID
	}
	s.notify(ctx, n)
	// Суралцагчийн нээлттэй хуудас (хичээлийн жагсаалт, нууц үгийн цонх) шинэчлэхгүйгээр шууд нээгдэнэ.
	s.hub.SendUsers([]string{uid}, map[string]any{"type": "access_granted", "course_id": course.ID, "lesson_id": in.LessonID})
	writeJSON(w, http.StatusOK, map[string]any{"granted": true})
}
