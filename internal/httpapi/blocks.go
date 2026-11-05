package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"surgalt/internal/store"
)

// Сануулгын хязгаар хэтэрсэн (auto_block) суралцагч тэр хичээл рүү дахин орж чадахгүй:
// багш "хоригийг цуцлах" (teacher_unblock) хүртэл, эсвэл сургалтын тохиргооны BlockHours өнгөрөх хүртэл.

const (
	defaultMaxWarnings  = 3
	defaultBlockMinutes = 30 // анхны тохиргоо: хаагдсан хичээл 30 минутын дараа автоматаар нээгдэнэ
	blockManual         = -1 // багш "Дахин нээх" дарах хүртэл хаалттай
)

// blockMinutes — хориг хэдэн минутын дараа автоматаар нээгдэх (0 = багш нээтэл). Багш сонгоогүй бол 30 минут.
func blockMinutes(c *store.Course) int {
	switch {
	case c.BlockMinutes == blockManual:
		return 0
	case c.BlockMinutes > 0:
		return c.BlockMinutes
	case c.BlockHours > 0:
		return c.BlockHours * 60
	}
	return defaultBlockMinutes
}

func maxWarnings(c *store.Course) int {
	if c.MaxWarnings <= 0 {
		return defaultMaxWarnings
	}
	return c.MaxWarnings
}

// LessonBlock — нэг хичээлийн хоригийн төлөв.
type LessonBlock struct {
	LessonID string     `json:"lesson_id"`
	Title    string     `json:"title,omitempty"`
	At       time.Time  `json:"at"`
	Until    *time.Time `json:"until,omitempty"` // хоосон бол багш нээтэл
	Reason   string     `json:"reason,omitempty"`
	Count    int        `json:"count"` // энэ сургалтад хэд дэх удаагаа хаагдсан (давтан зөрчил)
}

// lessonBlocks — суралцагчийн тухайн сургалт (courseID хоосон бол багшийн бүх сургалт) дахь идэвхтэй хоригууд.
func (s *Server) lessonBlocks(ctx context.Context, uid string, f store.ActivityFilter, blockMins func(courseID string) int) map[string]LessonBlock {
	f.UserID = uid
	evs, err := s.store.ActivityEvents(ctx, f, 2000)
	out := map[string]LessonBlock{}
	if err != nil {
		return out
	}
	lastBlock, lastUnblock := map[string]store.ActivityEvent{}, map[string]time.Time{}
	courseOf := map[string]string{}
	perCourse := map[string]int{} // сургалт бүрт нийт хэдэн удаа хаагдсан
	for _, e := range evs {
		switch e.Type {
		case "auto_block":
			perCourse[e.CourseID]++
			if cur, ok := lastBlock[e.LessonID]; !ok || e.At.After(cur.At) {
				lastBlock[e.LessonID] = e
				courseOf[e.LessonID] = e.CourseID
			}
		case "teacher_unblock":
			if e.At.After(lastUnblock[e.LessonID]) {
				lastUnblock[e.LessonID] = e.At
			}
		}
	}
	now := time.Now()
	for lid, b := range lastBlock {
		if !lastUnblock[lid].IsZero() && !lastUnblock[lid].Before(b.At) {
			continue
		}
		n := perCourse[courseOf[lid]]
		lb := LessonBlock{LessonID: lid, At: b.At, Reason: b.Detail, Count: n}
		if m := blockMins(courseOf[lid]); m > 0 {
			// Ухаалаг: давтан зөрчилд хугацаа 2 дахин уртасна (1, 2, 4… дахин), дээд тал нь 30 хоног.
			mult := 1 << min(max(n-1, 0), 5)
			d := min(time.Duration(m*mult)*time.Minute, 30*24*time.Hour)
			until := b.At.Add(d)
			if !now.Before(until) {
				continue
			}
			lb.Until = &until
		}
		out[lid] = lb
	}
	return out
}

// lessonBlocked — суралцагч энэ хичээлд хоригтой эсэх (багш өөрөө хэзээ ч хоригдохгүй). Хичээлийн бараг бүх хүсэлт
// (явц, үүлэн үг, тодруулга…) үүнийг шалгадаг тул 15 секунд кэшилнэ; хориг/цуцлалт бичигдэхэд шууд шинэчлэгдэнэ
// (blocksChanged).
func (s *Server) lessonBlocked(ctx context.Context, uid string, course *store.Course, lessonID string) *LessonBlock {
	if uid == "" || teaches(course, uid) {
		return nil
	}
	b, _ := s.blocks.GetOrLoad(ctx, uid+"|"+lessonID, func(ctx context.Context) (*LessonBlock, error) {
		bs := s.lessonBlocks(ctx, uid, store.ActivityFilter{CourseID: course.ID, LessonID: lessonID}, func(string) int { return blockMinutes(course) })
		if b, ok := bs[lessonID]; ok {
			return &b, nil
		}
		return nil, nil
	})
	if b == nil || (b.Until != nil && !time.Now().Before(*b.Until)) { // хугацаат хориг кэшэд байхдаа дуусч болно
		return nil
	}
	cp := *b // дуудагч гарчиг нэмдэг
	return &cp
}

// blocksChanged — хориг (auto_block) эсвэл цуцлалт (teacher_unblock) бичигдсэн хичээлүүдийн кэшийг хүчингүй болгоно.
func (s *Server) blocksChanged(evs []store.ActivityEvent) {
	for _, e := range evs {
		if e.Type == "auto_block" || e.Type == "teacher_unblock" {
			s.blocks.Delete(e.UserID + "|" + e.LessonID)
			s.dripOpen.Delete(e.UserID + "|" + e.LessonID)
		}
	}
}

func blockedErr(b *LessonBlock) error {
	what := "Та сануулгын хязгаарыг хэтрүүлсэн тул энэ хичээл"
	if strings.HasPrefix(b.Reason, "Шалгалт") {
		what = "Шалгалт зөрчлөөр хаагдсан тул энэ шалгалт"
	}
	msg := what + " түр хаагдсан — багш тань нээх хүртэл хүлээнэ үү"
	if b.Until != nil {
		msg = what + " " + b.Until.Local().Format("01/02 15:04") + " хүртэл хаагдсан"
	}
	return apiError(http.StatusLocked, msg, map[string]any{"blocked": b})
}

// handleUnblockStudent: POST /api/me/students/{uid}/unblock {lesson_id?} — багш хоригийг цуцална
// (lesson_id хоосон бол өөрийн бүх сургалт дахь бүх хоригийг).
func (s *Server) handleUnblockStudent(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	var in struct {
		LessonID string `json:"lesson_id"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&in) // хоосон бие ч болно
	uid := r.PathValue("uid")
	ctx := r.Context()
	// Өөрийн сургалтууд (системийн эрхтэй бол) ба байгууллагаас холбосон сургалтууд (эрхийг байгууллагаар).
	courses, err := s.store.CoursesByTeacher(ctx, c.UID, false)
	if s.storeErr(w, r, err) {
		return
	}
	if until, err := s.planUntil(ctx, c.UID); err == nil && !time.Now().Before(until) {
		courses = nil
	}
	shared, err := s.store.CoursesTaughtBy(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	courses = append(courses, shared...)
	if len(courses) == 0 && !s.planOpen(w, r, c.UID) { // эрх дууссан, холбосон сургалтгүй
		return
	}
	hours, ownerOf := map[string]int{}, map[string]string{}
	for _, co := range courses {
		hours[co.ID], ownerOf[co.ID] = blockMinutes(&co), co.TeacherID
	}
	courseOf, lessonOf := map[string]string{}, map[string]store.Lesson{}
	for _, co := range courses {
		ls, _ := s.store.LessonsByCourse(ctx, co.ID)
		for _, l := range ls {
			courseOf[l.ID], lessonOf[l.ID] = co.ID, l
		}
	}
	// Үйл явдал сургалтын эзэмшигчийн ID-гаар бичигддэг тул эзэмшигч бүрээр уншиж, зөвхөн өөрийн заадаг хичээлийг авна.
	bs := map[string]LessonBlock{}
	owners := map[string]bool{}
	for _, o := range ownerOf {
		owners[o] = true
	}
	for o := range owners {
		for lid, b := range s.lessonBlocks(ctx, uid, store.ActivityFilter{TeacherID: o, LessonID: in.LessonID}, func(cid string) int { return hours[cid] }) {
			if _, mine := courseOf[lid]; mine {
				bs[lid] = b
			}
		}
	}
	targets := make(map[string]bool, len(bs))
	for lid := range bs {
		targets[lid] = true
	}
	// Шалгалтыг багш гараар нээвэл: хориг автоматаар дууссан ч (эсвэл ердийн журмаар унасан ч) оролдлого нь
	// дууссан байж болно — үргэлж нэг нэмэлт оролдлого өгнө (teacher_unblock = +1 оролдлого, хоригийг ч цуцална).
	exam := ""
	if in.LessonID != "" {
		l, ok := lessonOf[in.LessonID]
		if !ok {
			writeErr(w, http.StatusNotFound, "олдсонгүй")
			return
		}
		if l.Exam != nil {
			// Нэг ч оролдлого хийгээгүй бол нээх юм алга (дурын хэрэглэгчид мэдэгдэл явуулахаас сэргийлнэ).
			atts, err := s.store.ExamAttempts(r.Context(), store.ActivityFilter{UserID: uid, LessonID: l.ID})
			if s.storeErr(w, r, err) {
				return
			}
			if len(atts) > 0 || targets[l.ID] {
				targets[l.ID], exam = true, l.Title
			}
		}
	}
	if len(targets) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"unblocked": 0})
		return
	}
	name := s.displayName(r.Context(), uid, "")
	var evs []store.ActivityEvent
	link := ""
	for lid := range targets {
		detail := "багш хоригийг цуцлав"
		if lessonOf[lid].Exam != nil {
			detail = "багш шалгалтыг дахин нээв (+1 оролдлого)"
		}
		evs = append(evs, store.ActivityEvent{UserID: uid, UserName: name, CourseID: courseOf[lid], LessonID: lid, TeacherID: ownerOf[courseOf[lid]], Type: "teacher_unblock", Detail: detail, At: time.Now()})
		if len(targets) == 1 {
			link = "/c/" + courseOf[lid] + "#l=" + lid
		}
	}
	if err := s.store.AddActivityEvents(r.Context(), evs); s.storeErr(w, r, err) {
		return
	}
	s.blocksChanged(evs)
	for lid := range targets {
		s.journals.Delete(courseOf[lid]) // журнал шинэ төлвийг шууд харуулна
	}
	n := &store.Notification{UserID: uid, Type: "unblock", ActorID: c.UID, Title: "🔓 Багш хичээлийг тань дахин нээлээ", Body: "Анхааралтай, бусад цонхоо хаагаад үзээрэй.", Link: link}
	if exam != "" {
		n.Title, n.Body = "🔓 Багш шалгалтыг тань дахин нээлээ", "«"+exam+"» — дахин өгөх боломжтой (нэмэлт оролдлого). Бусад цонхоо хаагаад, анхааралтай өгөөрэй."
	}
	s.notify(r.Context(), n)
	// Суралцагчийн нээлттэй хуудас (хичээлийн жагсаалт, хаагдсан цонх) шинэчлэхгүйгээр шууд нээгдэнэ.
	opened := make([]map[string]string, 0, len(targets))
	for lid := range targets {
		opened = append(opened, map[string]string{"course_id": courseOf[lid], "lesson_id": lid})
	}
	s.hub.SendUsers([]string{uid}, map[string]any{"type": "lesson_unblocked", "lessons": opened})
	writeJSON(w, http.StatusOK, map[string]any{"unblocked": len(evs), "exam": exam != ""})
}
