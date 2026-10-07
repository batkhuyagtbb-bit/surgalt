package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

const maxPrice = 100_000_000 // ₮

type courseInput struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	Price         int64  `json:"price"`
	Published     bool   `json:"published"`
	Drip          bool   `json:"drip"`            // хичээлүүд дарааллаар нээгдэнэ
	UnlockAllPaid bool   `json:"unlock_all_paid"` // багц төлсөн бол бүгд шууд
	Camera        string `json:"camera"`          // off | optional | required
	Certificate   bool   `json:"certificate"`     // сертификат олгоно
	MaxWarnings   int    `json:"max_warnings"`    // таб солих сануулгын тоо (1-20, 0 = 3)
	BlockHours    int    `json:"block_hours"`     // (хуучин) хориг, цаг
	BlockMinutes  int    `json:"block_minutes"`   // хэтэрвэл хэдэн минутын дараа автоматаар нээгдэх (0 = анхдагч 30 мин, -1 = багш нээтэл)
}

func (in *courseInput) validate() string {
	in.Title = strings.TrimSpace(in.Title)
	switch {
	case in.Title == "" || utf8.RuneCountInString(in.Title) > 200:
		return "гарчиг 1-200 тэмдэгт"
	case utf8.RuneCountInString(in.Description) > 20000:
		return "тайлбар хэт урт"
	case in.Price < 0 || in.Price > maxPrice:
		return "үнэ 0-100,000,000₮"
	}
	if in.MaxWarnings < 0 || in.MaxWarnings > 20 {
		return "сануулгын тоо 1-20"
	}
	if in.BlockHours < 0 || in.BlockHours > 24*30 {
		return "хоригийн хугацаа 0-720 цаг"
	}
	if in.BlockMinutes < blockManual || in.BlockMinutes > 60*24*30 {
		return "хоригийн хугацаа 0-43200 минут (−1 = багш нээх хүртэл)"
	}
	if in.BlockMinutes == 0 && in.BlockHours > 0 { // хуучин талбараар ирвэл минут руу хөрвүүлнэ
		in.BlockMinutes = in.BlockHours * 60
	}
	in.BlockHours = 0
	switch in.Camera {
	case "":
		in.Camera = "optional"
	case "off", "optional", "required":
	default:
		return "камерын горим: off, optional, required"
	}
	return ""
}

// invalidateTeacher нь багшийн профайл болон сургалтуудын кэшийг цэвэрлэнэ.
func (s *Server) invalidateTeacher(r *http.Request, teacherID, username string) {
	s.profiles.Delete(username)
	if cs, err := s.store.CoursesByTeacher(r.Context(), teacherID, false); err == nil {
		for _, c := range cs {
			s.courses.Delete(c.ID)
			s.search.markDirty()
		}
	}
}

func (s *Server) handleCreateCourse(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	var in courseInput
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	course := &store.Course{TeacherID: c.UID, Title: in.Title, Description: in.Description, Price: in.Price, Published: in.Published, Drip: in.Drip, UnlockAllPaid: in.UnlockAllPaid, Camera: in.Camera, Certificate: in.Certificate, MaxWarnings: in.MaxWarnings, BlockHours: in.BlockHours, BlockMinutes: in.BlockMinutes}
	if err := s.store.CreateCourse(r.Context(), course); s.storeErr(w, r, err) {
		return
	}
	s.profiles.Delete(c.Name)
	s.search.markDirty()
	writeJSON(w, http.StatusCreated, course)
}

func (s *Server) handleUpdateCourse(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	var in courseInput
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	course := &store.Course{ID: r.PathValue("id"), TeacherID: c.UID, Title: in.Title, Description: in.Description, Price: in.Price, Published: in.Published, Drip: in.Drip, UnlockAllPaid: in.UnlockAllPaid, Camera: in.Camera, Certificate: in.Certificate, MaxWarnings: in.MaxWarnings, BlockHours: in.BlockHours, BlockMinutes: in.BlockMinutes}
	if err := s.store.UpdateCourse(r.Context(), course); s.storeErr(w, r, err) {
		return
	}
	s.courses.Delete(course.ID)
	s.search.markDirty()
	s.profiles.Delete(c.Name)
	writeJSON(w, http.StatusOK, course)
}

// ownCourse нь сургалтыг ачаалж, дуудагч эзэмшигч мөн эсэхийг шалгана.
func (s *Server) ownCourse(w http.ResponseWriter, r *http.Request, uid string) (*store.Course, bool) {
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return nil, false
	}
	if course.TeacherID != uid {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return nil, false
	}
	return course, true
}

type lessonInput struct {
	Hidden       bool              `json:"hidden"` // шинээр үүсгэхэд: бэлтгэж дуусаагүй — суралцагчдад харагдахгүй
	Title        string            `json:"title"`
	Content      string            `json:"content"`
	VideoURL     string            `json:"video_url"`
	IsFree       bool              `json:"is_free"`
	Price        int64             `json:"price"`          // төлбөртэй хичээлийн дангаар худалдах үнэ (₮)
	UnlockAfterH int               `json:"unlock_after_h"` // өмнөх хичээлийг үзснээс хойш хэдэн цагийн дараа (0 = шууд)
	AlwaysOpen   bool              `json:"always_open"`    // дарааллаас үл хамааран нээлттэй
	Format       string            `json:"format"`         // lecture | seminar | practice | lab | ""
	Mode         string            `json:"mode"`           // classroom | online | blended | ""
	Section      string            `json:"section"`        // хичээлийн бүлэг (модуль), хоосон = бүлэггүй
	Blocks       []store.Block     `json:"blocks"`         // дэлгэрэнгүй агуулга (текст, зураг, дуу, видео, файл, асуулт)
	ActiveMin    int               `json:"active_min"`     // идэвхтэй суралцах ёстой минут
	Exam         *store.Exam       `json:"exam"`           // хоосон биш бол шалгалт
	Assignment   *store.Assignment `json:"assignment"`     // хоосон биш бол даалгавар
	Discussion   bool              `json:"discussion"`     // хэлэлцүүлэгтэй (лайк, сэтгэгдэл)
	UnlockRule   string            `json:"unlock_rule"`    // дараалалд нээгдэх нөхцөл (store.UnlockRules)
}

const maxUnlockHours = 24 * 365

// validate: агуулга бүр үнэгүй эсвэл төлбөртэй; төлбөртэй бол үнээ заана
// (сургалтад багц үнэ байвал 0 = "зөвхөн багцаар").
func (in *lessonInput) validate(teacherID string, course *store.Course) string {
	in.Title = strings.TrimSpace(in.Title)
	in.Section = strings.Join(strings.Fields(in.Section), " ")
	if in.IsFree {
		in.Price = 0
	}
	switch {
	case in.Title == "" || utf8.RuneCountInString(in.Title) > 200:
		return "гарчиг 1-200 тэмдэгт"
	case len(in.Content) > 200_000:
		return "агуулга хэт урт"
	case in.VideoURL != "" && !validURL(in.VideoURL) && !files.OwnedBy(in.VideoURL, teacherID):
		return "медиа: http(s) холбоос эсвэл өөрийн файлын сангийн файл"
	case in.Price < 0 || in.Price > maxPrice:
		return "үнэ 0-100,000,000₮"
	case !in.IsFree && in.Price == 0 && course.Price == 0:
		return "төлбөртэй хичээлийн үнийг заана уу (эсвэл сургалтад багц үнэ тавина уу)"
	case in.UnlockAfterH < 0 || in.UnlockAfterH > maxUnlockHours:
		return "нээгдэх хугацаа 0-8760 цаг"
	case in.Format != "" && store.LessonFormats[in.Format] == "":
		return "сургалтын хэлбэр: лекц, семинар, дадлага, лаборатори"
	case in.Mode != "" && store.LessonModes[in.Mode] == "":
		return "заах аргын төрөл: танхимын, цахим, холимог"
	case utf8.RuneCountInString(in.Section) > 80:
		return "бүлгийн нэр 80 тэмдэгтээс хэтрэхгүй"
	case store.UnlockRules[in.UnlockRule] == "":
		return "нээгдэх нөхцөл буруу"
	}
	if in.ActiveMin < 0 || in.ActiveMin > 600 {
		return "идэвхтэй суралцах хугацаа 0-600 минут"
	}
	if in.Exam != nil && in.Assignment != nil {
		return "нэг хичээл шалгалт эсвэл даалгаврын аль нэг нь байна"
	}
	if msg := validateExam(in.Exam); msg != "" {
		return msg
	}
	if msg := validateAssignment(in.Assignment); msg != "" {
		return msg
	}
	return validateBlocks(in.Blocks, teacherID)
}

func lessonFromInput(in lessonInput, courseID string) *store.Lesson {
	return &store.Lesson{CourseID: courseID, Title: in.Title, Content: in.Content, VideoURL: in.VideoURL, IsFree: in.IsFree, Price: in.Price,
		UnlockAfterH: in.UnlockAfterH, AlwaysOpen: in.AlwaysOpen, Format: in.Format, Mode: in.Mode, Section: in.Section, Blocks: in.Blocks, ActiveMin: in.ActiveMin, Exam: in.Exam, Assignment: in.Assignment, Discussion: in.Discussion, UnlockRule: in.UnlockRule, Hidden: in.Hidden}
}

func (s *Server) handleCreateLesson(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in lessonInput
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(c.UID, course); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	l := lessonFromInput(in, course.ID)
	if err := s.store.CreateLesson(r.Context(), l); s.storeErr(w, r, err) {
		return
	}
	s.courses.Delete(course.ID)
	s.search.markDirty()
	s.profiles.Delete(c.Name)
	writeJSON(w, http.StatusCreated, l)
}

func (s *Server) handleUpdateLesson(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in lessonInput
	if !decode(w, r, &in) {
		return
	}
	if msg := in.validate(c.UID, course); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	l := lessonFromInput(in, course.ID)
	l.ID = r.PathValue("lid")
	if err := s.store.UpdateLesson(r.Context(), l); s.storeErr(w, r, err) {
		return
	}
	s.courses.Delete(course.ID)
	s.search.markDirty()
	s.profiles.Delete(c.Name)
	writeJSON(w, http.StatusOK, l)
}

// handleReorderLessons: багш хичээлүүдийг чирж зөөсний дараа бүх дараалал ба бүлгийг нэг дор хадгална.
func (s *Server) handleReorderLessons(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Items []store.LessonOrder `json:"items"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Items) > 2000 {
		writeErr(w, http.StatusBadRequest, "хэт олон хичээл")
		return
	}
	for i := range in.Items {
		in.Items[i].Section = strings.Join(strings.Fields(in.Items[i].Section), " ")
		if utf8.RuneCountInString(in.Items[i].Section) > 80 {
			writeErr(w, http.StatusBadRequest, "бүлгийн нэр 80 тэмдэгтээс хэтрэхгүй")
			return
		}
	}
	if err := s.store.ReorderLessons(r.Context(), course.ID, in.Items); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusBadRequest, "хичээлийн жагсаалт таарахгүй байна — хуудсаа сэргээнэ үү")
			return
		}
		s.storeErr(w, r, err)
		return
	}
	s.courses.Delete(course.ID)
	s.search.markDirty()
	s.profiles.Delete(c.Name)
	lessons, err := s.store.LessonsByCourse(r.Context(), course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, lessons)
}

// lessonAccess: үнэгүй, багш, дангаар худалдаж авсан, эсвэл үнэтэй багцад элссэн.
func (s *Server) lessonAccess(ctx context.Context, uid string, course *store.Course, l *store.Lesson) (bool, error) {
	if l.IsFree || course.TeacherID == uid {
		return true, nil
	}
	if course.Price > 0 {
		if ok, err := s.store.IsEnrolled(ctx, uid, course.ID); err != nil || ok {
			return ok, err
		}
	}
	bought, err := s.store.PurchasedLessons(ctx, uid, course.ID)
	if err != nil {
		return false, err
	}
	for _, id := range bought {
		if id == l.ID {
			return true, nil
		}
	}
	return false, nil
}

// ---- дараалсан нээлт (drip) ----

// LessonState нь суралцагчид нэг хичээл одоо нээлттэй юу, үгүй бол яагаад, хэзээ нээгдэхийг хэлнэ.
type LessonState struct {
	Open       bool       `json:"open"`
	Reason     string     `json:"reason,omitempty"` // "prev" — өмнөхөө үзээгүй, "quiz" — өмнөхийн асуултуудад бүрэн зөв хариулаагүй, "timer" — хугацаа болоогүй
	PrevTitle  string     `json:"prev_title,omitempty"`
	UnlockAt   *time.Time `json:"unlock_at,omitempty"`
	QuizLeft   int        `json:"quiz_left,omitempty"`   // quiz: хэдэн асуулт зөв хариулагдаагүй үлдсэн
	QuizTotal  int        `json:"quiz_total,omitempty"`  // quiz: өмнөх хичээлийн нийт асуулт
	ActiveLeft int        `json:"active_left,omitempty"` // active: өмнөх хичээлд дутуу идэвхтэй минут
	ActiveMin  int        `json:"active_min,omitempty"`
}

// quizBlockIDs — хичээл доторх өөрийгөө сорих асуултуудын ID (шалгалтын хичээлд хамаарахгүй: тэр тусдаа тэнцдэг).
func quizBlockIDs(l *store.Lesson) []string {
	if l.Exam != nil {
		return nil
	}
	var ids []string
	for _, b := range l.Blocks {
		if b.Type == "quiz" && b.Quiz != nil {
			ids = append(ids, b.ID)
		}
	}
	return ids
}

// quizMastery — хичээлийн асуултуудаас хэд нь зөв хариулагдсан, бүгд зөв үү, хэзээ дууссан бэ.
// QuizDoneAt хадгалагдаагүй хуучин явцад "бүгд зөв" байвал үзсэн цагаар тооцно.
func quizMastery(l *store.Lesson, p *store.LessonProgress) (correct, total int, done bool, at time.Time) {
	ids := quizBlockIDs(l)
	total = len(ids)
	if p == nil {
		return 0, total, total == 0, time.Time{}
	}
	for _, id := range ids {
		if p.Quiz[id] {
			correct++
		}
	}
	done = correct == total
	at = p.ViewedAt
	if p.QuizDoneAt != nil {
		done, at = true, *p.QuizDoneAt
	}
	return
}

// dripExtra — дарааллын шалгалтад хэрэгтэй нэмэлт баримт: хичээл бүрийн идэвхтэй секунд, тэнцсэн шалгалт.
type dripExtra struct {
	active map[string]int
	passed map[string]bool
}

func (s *Server) dripFacts(ctx context.Context, uid, courseID string) dripExtra {
	x := dripExtra{active: map[string]int{}, passed: map[string]bool{}}
	if ss, err := s.store.Sessions(ctx, store.ActivityFilter{UserID: uid, CourseID: courseID}, 5000); err == nil {
		for _, v := range ss {
			x.active[v.LessonID] += v.ActiveSec
		}
	}
	if atts, err := s.store.ExamAttempts(ctx, store.ActivityFilter{UserID: uid, CourseID: courseID}); err == nil {
		for _, a := range atts {
			if a.Passed {
				x.passed[a.LessonID] = true
			}
		}
	}
	return x
}

// dripState — дараагийн хичээл нээгдэх дүрэм:
//  1. Өмнөх хичээлийг үзсэн байх.
//  2. Өмнөх хичээлийн идэвхтэй суралцах хугацааг (ActiveMin) гүйцээсэн байх (эсвэл "дууслаа" гэсэн).
//  3. Өмнөх нь шалгалт бол тэнцсэн, асуулттай бол бүгдэд нь зөв хариулсан байх — тэгвэл цаг, өдрөөс
//     үл хамааран ШУУД нээгдэнэ (таймер хамаарахгүй).
//     Идэвхтэй хугацаатай хичээлийн минутыг бүрэн үзсэн бол мөн шууд нээгдэнэ.
//  4. Асуулт, шалгалт, идэвхтэй хугацаа аль нь ч үгүй хичээлд л хүлээх хугацаа (UnlockAfterH) үйлчилнэ.
func dripState(course *store.Course, lessons []store.Lesson, l *store.Lesson, progress map[string]store.LessonProgress, fullAccess bool, now time.Time, x dripExtra) LessonState {
	if !course.Drip || l.AlwaysOpen || fullAccess {
		return LessonState{Open: true}
	}
	var prev *store.Lesson
	for i := range lessons {
		if lessons[i].Position < l.Position && (prev == nil || lessons[i].Position > prev.Position) {
			prev = &lessons[i]
		}
	}
	if prev == nil {
		return LessonState{Open: true}
	}
	if l.UnlockRule == "manual" { // багш гараар нээнэ ("Шууд нээлттэй болгох" = AlwaysOpen)
		return LessonState{Reason: "manual", PrevTitle: prev.Title}
	}
	p, ok := progress[prev.ID]
	if !ok {
		return LessonState{Reason: "prev", PrevTitle: prev.Title}
	}
	if l.UnlockRule != "" {
		return ruleState(l, prev, &p, now, x)
	}
	if need := prev.ActiveMin * 60; need > 0 && p.CompletedAt == nil && x.active[prev.ID] < need {
		left := (need - x.active[prev.ID] + 59) / 60
		return LessonState{Reason: "active", PrevTitle: prev.Title, ActiveLeft: left, ActiveMin: prev.ActiveMin}
	}
	if prev.Exam != nil {
		if x.passed[prev.ID] {
			return LessonState{Open: true}
		}
		return LessonState{Reason: "exam", PrevTitle: prev.Title}
	}
	correct, total, done, _ := quizMastery(prev, &p)
	if total > 0 {
		if done {
			return LessonState{Open: true} // асуултуудад бүгдэд нь зөв → цаг, өдөр хамаагүй шууд
		}
		return LessonState{Reason: "quiz", PrevTitle: prev.Title, QuizLeft: total - correct, QuizTotal: total}
	}
	if prev.ActiveMin > 0 {
		return LessonState{Open: true} // идэвхтэй минутаа бүрэн үзсэн → цаг, өдөр хамаагүй шууд
	}
	base := p.ViewedAt
	if p.CompletedAt != nil {
		base = *p.CompletedAt
	}
	at := base.Add(time.Duration(l.UnlockAfterH) * time.Hour)
	if now.Before(at) {
		return LessonState{Reason: "timer", PrevTitle: prev.Title, UnlockAt: &at}
	}
	return LessonState{Open: true}
}

// fullAccess: багш, эсвэл үнэтэй багцад элссэн бөгөөд сургалт "багц төлсөн бол бүгд нээлттэй" тохиргоотой.
func fullAccess(course *store.Course, uid string, enrolled bool) bool {
	return course.TeacherID == uid || (course.UnlockAllPaid && enrolled && course.Price > 0)
}

// handleCompleteLesson: суралцагч хичээлээ дууссан гэж тэмдэглэнэ (явцын хэсэгт харагдана).
func (s *Server) handleCompleteLesson(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	l, err := s.store.LessonByID(r.Context(), course.ID, r.PathValue("lid"))
	if s.storeErr(w, r, err) {
		return
	}
	if hiddenFrom(l, course, c.UID) {
		s.apiErr(w, r, errLessonHidden)
		return
	}
	has, err := s.lessonAccess(r.Context(), c.UID, course, l)
	if s.storeErr(w, r, err) {
		return
	}
	if !has {
		writeErr(w, http.StatusForbidden, "энэ хичээлд эрхгүй")
		return
	}
	if l.Exam != nil && course.TeacherID != c.UID {
		writeErr(w, http.StatusConflict, "шалгалтыг өгч тэнцсэнээр дуусна")
		return
	}
	if l.Assignment != nil && course.TeacherID != c.UID {
		if _, err := s.store.SubmissionFor(r.Context(), c.UID, l.ID); err != nil {
			writeErr(w, http.StatusConflict, "даалгаврын хариугаа илгээснээр дуусна")
			return
		}
	}
	if l.ActiveMin > 0 && course.TeacherID != c.UID {
		done, err := s.lessonActiveSec(r.Context(), c.UID, l.ID)
		if s.storeErr(w, r, err) {
			return
		}
		if done < l.ActiveMin*60 {
			writeJSON(w, http.StatusConflict, map[string]any{"error": fmt.Sprintf("Идэвхтэй суралцах хугацаа хүрээгүй: %d/%d мин", done/60, l.ActiveMin), "active_done_sec": done, "active_min": l.ActiveMin})
			return
		}
	}
	if err := s.store.MarkLessonCompleted(r.Context(), c.UID, course.ID, l.ID); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"completed": true})
}

// handleBuyLesson: нэг хичээлийг дангаар нь худалдаж авах захиалга.
func (s *Server) handleBuyLesson(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	if !course.Published {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	l, err := s.store.LessonByID(r.Context(), course.ID, r.PathValue("lid"))
	if s.storeErr(w, r, err) {
		return
	}
	if hiddenFrom(l, course, c.UID) {
		s.apiErr(w, r, errLessonHidden)
		return
	}
	has, err := s.lessonAccess(r.Context(), c.UID, course, l)
	if s.storeErr(w, r, err) {
		return
	}
	if has {
		writeJSON(w, http.StatusOK, map[string]any{"unlocked": true})
		return
	}
	if l.Price <= 0 {
		writeErr(w, http.StatusBadRequest, "энэ хичээл зөвхөн сургалтын багцаар нээгдэнэ")
		return
	}
	o, err := s.store.CreateOrGetPendingLessonOrder(r.Context(), c.UID, course, l)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"unlocked": false, "order": o,
		"payment": s.paymentInfo(r, o),
	})
}

func (s *Server) handleMyCourses(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	cs, err := s.store.CoursesByTeacher(r.Context(), c.UID, false)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (s *Server) handleMyCourse(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	lessons, err := s.store.LessonsByCourse(r.Context(), course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	for i := range lessons {
		lessons[i].VideoURL = s.media(lessons[i].VideoURL)
		for j := range lessons[i].Blocks { // багш өөрийн хувийн файлыг засварлагч дээр урьдчилан харна
			if u := lessons[i].Blocks[j].URL; u != "" {
				if parts := s.files.Parts(u); lessons[i].Blocks[j].Type == "video" && len(parts) > 1 {
					lessons[i].Blocks[j].Parts = make([]string, len(parts))
					for k, p := range parts {
						lessons[i].Blocks[j].Parts[k] = s.media(p)
					}
				}
				lessons[i].Blocks[j].URL = s.media(u)
			}
			if q := lessons[i].Blocks[j].Quiz; q != nil && q.Image != "" {
				qc := *q
				qc.Image = s.media(q.Image)
				lessons[i].Blocks[j].Quiz = &qc
			}
		}
	}
	ids := make([]string, len(lessons))
	for i := range lessons {
		ids[i] = lessons[i].ID
	}
	counts, _ := s.store.CommentCounts(r.Context(), ids) // хэлэлцүүлгийн сэтгэгдлийн тоо (жагсаалтад)
	writeJSON(w, http.StatusOK, map[string]any{"course": course, "lessons": lessons, "comments": counts})
}

func (s *Server) handleMyEnrollments(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	cs, err := s.store.EnrolledCourses(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, cs)
}

func (s *Server) handleMySales(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	sum, err := s.store.TeacherSales(r.Context(), c.UID, limit)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) handleCourseAccess(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	owner := course.TeacherID == c.UID
	enrolled := owner
	if !owner {
		if enrolled, err = s.store.IsEnrolled(r.Context(), c.UID, course.ID); s.storeErr(w, r, err) {
			return
		}
	}
	bought, err := s.store.PurchasedLessons(r.Context(), c.UID, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	lessons, err := s.store.LessonsByCourse(r.Context(), course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	lessons = visibleLessons(lessons, owner) // хаалттай хичээл суралцагчид огт харагдахгүй
	progress, err := s.store.LessonProgress(r.Context(), c.UID, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	full := fullAccess(course, c.UID, enrolled)
	facts := s.dripFacts(r.Context(), c.UID, course.ID)
	states := make(map[string]LessonState, len(lessons))
	done := 0
	for i := range lessons {
		states[lessons[i].ID] = dripState(course, lessons, &lessons[i], progress, full, time.Now(), facts)
		if p, ok := progress[lessons[i].ID]; ok && p.CompletedAt != nil {
			done++
		}
	}
	// Цэргийн цол: энэ сургалт дахь хичээл бүрийн үнэлгээ ба нийлбэр.
	ranks, rank := s.courseRanks(r.Context(), c.UID, course.ID, lessons, progress)
	rankBy := make(map[string]LessonRank, len(ranks))
	for _, lr := range ranks {
		rankBy[lr.LessonID] = lr
	}
	var total RankInfo
	awarded := false
	if !owner && len(ranks) > 0 {
		total, awarded = s.autoAward(r.Context(), c.UID, course.ID, "/c/"+course.ID, rank.Points) // систем өөрөө олгоно
	}
	w.Header().Set("Cache-Control", "private, no-store")
	// all: бүх хичээл төлбөрийн хувьд нээлттэй (багш эсвэл үнэтэй багц худалдаж авсан).
	writeJSON(w, http.StatusOK, map[string]any{"enrolled": enrolled, "owner": owner,
		"all": owner || (enrolled && course.Price > 0), "lessons": bought,
		"states": states, "progress": progress, "done": done, "total": len(lessons),
		"ranks": rankBy, "rank": rank, "rank_total": total, "rank_awarded": awarded,
		"blocks":       s.lessonBlocks(r.Context(), c.UID, store.ActivityFilter{CourseID: course.ID}, func(string) int { return blockMinutes(course) }),
		"max_warnings": maxWarnings(course)})
}

// handleGetLesson: үнэгүй хичээлийг хэн ч, бусдыг зөвхөн элссэн хүн эсвэл багш үзнэ.
func (s *Server) handleGetLesson(w http.ResponseWriter, r *http.Request) {
	cid, lid := r.PathValue("id"), r.PathValue("lid")
	pc, err := s.publicCourse(r.Context(), cid)
	if err == nil {
		found := false
		for _, l := range pc.Data.Lessons {
			if l.ID == lid {
				if l.IsFree {
					if p, ok := s.principal(r); !ok || p.IsGuest() || p.UID != pc.Data.Course.TeacherID { // багш өөрийн хичээлээ үзвэл тоолохгүй
						s.trackView(r, pc.Data.Course.TeacherID, NotifLessonView, l.ID, l.Title, "/c/"+cid)
					}
					if p, ok := s.principal(r); ok && !p.IsGuest() { // нэвтэрсэн бол явцад тэмдэглэнэ (дараалалд хэрэгтэй)
						if co, err := s.store.CourseByID(r.Context(), cid); err == nil {
							if b := s.lessonBlocked(r.Context(), p.UID, co, l.ID); b != nil {
								b.Title = l.Title
								s.apiErr(w, r, blockedErr(b))
								return
							}
						}
						_ = s.store.MarkLessonViewed(r.Context(), p.UID, cid, l.ID)
					}
					writeJSON(w, http.StatusOK, l)
					return
				}
				found = true
				break
			}
		}
		if !found { // нийтийн жагсаалтад байхгүй (устгасан эсвэл хаалттай) — багш л цааш (хаалттай хичээлээ ч үзнэ)
			if p, ok := s.principal(r); !ok || p.IsGuest() || p.UID != pc.Data.Course.TeacherID {
				writeErr(w, http.StatusNotFound, "хичээл олдсонгүй")
				return
			}
		}
	} else if err != store.ErrNotFound {
		s.storeErr(w, r, err)
		return
	}
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, err := s.store.CourseByID(r.Context(), cid)
	if s.storeErr(w, r, err) {
		return
	}
	if course.TeacherID != c.UID && !course.Published {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	l, err := s.store.LessonByID(r.Context(), cid, lid)
	if s.storeErr(w, r, err) {
		return
	}
	if hiddenFrom(l, course, c.UID) {
		s.apiErr(w, r, errLessonHidden)
		return
	}
	has, err := s.lessonAccess(r.Context(), c.UID, course, l)
	if s.storeErr(w, r, err) {
		return
	}
	if !has {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{"error": "энэ хичээл төлбөртэй", "lesson_price": l.Price, "course_price": course.Price})
		return
	}
	// Дараалсан нээлт: өмнөх хичээлээ үзсэн, хүлээх хугацаа нь өнгөрсөн байх ёстой.
	if course.Drip && !l.AlwaysOpen && course.TeacherID != c.UID {
		lessons, err := s.store.LessonsByCourse(r.Context(), course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		lessons = visibleLessons(lessons, false) // хаалттай хичээл дараалалд тооцогдохгүй
		progress, err := s.store.LessonProgress(r.Context(), c.UID, course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		enrolled, _ := s.store.IsEnrolled(r.Context(), c.UID, course.ID)
		if st := dripState(course, lessons, l, progress, fullAccess(course, c.UID, enrolled), time.Now(), s.dripFacts(r.Context(), c.UID, course.ID)); !st.Open {
			msg := "Эхлээд «" + st.PrevTitle + "» хичээлийг үзнэ үү"
			if st.Reason == "quiz" {
				msg = fmt.Sprintf("Эхлээд «%s» хичээлийн асуултуудад бүгдэд нь зөв хариулна уу (%d/%d үлдсэн)", st.PrevTitle, st.QuizLeft, st.QuizTotal)
			}
			if st.Reason == "active" {
				msg = fmt.Sprintf("Эхлээд «%s» хичээлийн идэвхтэй суралцах хугацааг гүйцээнэ үү (%d мин дутуу)", st.PrevTitle, st.ActiveLeft)
			}
			if st.Reason == "exam" {
				msg = "Эхлээд «" + st.PrevTitle + "» шалгалтад тэнцэнэ үү"
			}
			if st.Reason == "complete" {
				msg = "Эхлээд «" + st.PrevTitle + "» хичээлийг дуусгана уу"
			}
			if st.Reason == "manual" {
				msg = "Энэ хичээлийг багш нээх хүртэл хүлээнэ үү"
			}
			if st.Reason == "timer" {
				msg = "Энэ хичээл " + st.UnlockAt.Local().Format("01/02 15:04") + "-д нээгдэнэ"
			}
			writeJSON(w, http.StatusLocked, map[string]any{"error": msg, "state": st})
			return
		}
	}
	if b := s.lessonBlocked(r.Context(), c.UID, course, l.ID); b != nil {
		b.Title = l.Title
		s.apiErr(w, r, blockedErr(b))
		return
	}
	if course.TeacherID != c.UID {
		_ = s.store.MarkLessonViewed(r.Context(), c.UID, course.ID, l.ID)
	}
	l.VideoURL = s.viewerMedia(l.VideoURL, c.UID)
	l.Blocks = s.viewerBlocks(examIntro(l))
	writeJSON(w, http.StatusOK, l)
}

// handleEnroll: үнэгүй бол шууд элсүүлнэ, төлбөртэй бол захиалга үүсгэнэ.
func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	if !course.Published {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	if course.TeacherID == c.UID {
		writeErr(w, http.StatusBadRequest, "өөрийн сургалтыг худалдан авах боломжгүй")
		return
	}
	enrolled, err := s.store.IsEnrolled(r.Context(), c.UID, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if enrolled {
		writeJSON(w, http.StatusOK, map[string]any{"enrolled": true})
		return
	}
	if course.Price == 0 {
		if err := s.store.Enroll(r.Context(), c.UID, course); s.storeErr(w, r, err) {
			return
		}
		s.notify(r.Context(), &store.Notification{UserID: course.TeacherID, Type: NotifEnroll, Count: 1,
			Title: "🎓 Шинэ суралцагч элслээ", Body: c.Name + " — " + course.Title, Link: "/c/" + course.ID})
		writeJSON(w, http.StatusCreated, map[string]any{"enrolled": true})
		return
	}
	order, err := s.store.CreateOrGetPendingOrder(r.Context(), c.UID, course)
	if s.storeErr(w, r, err) {
		return
	}
	// Энд төлбөрийн үйлчилгээ (QPay, SocialPay г.м)-нд invoice үүсгэж,
	// QR/deeplink-ийг хариунд оруулна. Төлөгдсөний дараа тэд webhook руу мэдэгдэнэ.
	writeJSON(w, http.StatusCreated, map[string]any{
		"enrolled": false,
		"order":    order,
		"payment":  s.paymentInfo(r, order),
	})
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	o, err := s.store.OrderByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	if o.UserID != c.UID {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	o = s.syncPayment(r.Context(), o, false) // QPay-ээс төлөгдсөн эсэхийг шалгана (callback ирээгүй ч)
	writeJSON(w, http.StatusOK, o)
}

// handleDevPay нь зөвхөн DEV_PAYMENTS=1 үед: төлбөрийн системгүйгээр урсгалыг турших.
func (s *Server) handleDevPay(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.DevPayments {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	id := r.PathValue("id")
	byQR := constantEq(r.URL.Query().Get("t"), s.payMAC(id)) // демо QR-ыг утсаар уншуулсан: нэвтрэлтгүй
	uid := ""
	if !byQR {
		c, ok := s.requireUser(w, r)
		if !ok {
			return
		}
		uid = c.UID
	}
	o, err := s.store.OrderByID(r.Context(), id)
	if s.storeErr(w, r, err) {
		return
	}
	if !byQR && o.UserID != uid {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	wasPaid := o.Status == store.OrderPaid
	o, err = s.store.MarkOrderPaid(r.Context(), o.ID, o.Amount)
	if s.storeErr(w, r, err) {
		return
	}
	if !wasPaid {
		s.notifyPaid(r.Context(), o)
	}
	writeJSON(w, http.StatusOK, o)
}

// handlePaymentWebhook: төлбөрийн үйлчилгээ X-Webhook-Secret толгойтой дуудна.
// Идемпотент — ижил мэдэгдэл олон ирсэн ч нэг л удаа элсүүлнэ.
func (s *Server) handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.WebhookSecret == "" || !constantEq(r.Header.Get("X-Webhook-Secret"), s.cfg.WebhookSecret) {
		writeErr(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var in struct {
		OrderID string `json:"order_id"`
		Amount  int64  `json:"amount"`
	}
	if !decode(w, r, &in) {
		return
	}
	before, err := s.store.OrderByID(r.Context(), in.OrderID)
	if s.storeErr(w, r, err) {
		return
	}
	o, err := s.store.MarkOrderPaid(r.Context(), in.OrderID, in.Amount)
	if s.storeErr(w, r, err) {
		return
	}
	if before.Status != store.OrderPaid {
		s.notifyPaid(r.Context(), o)
	}
	writeJSON(w, http.StatusOK, o)
}

// ruleState — багшийн гараар сонгосон нөхцөлөөр (нөхцөл биелсэн мөчөөс хүлээх хугацаа тоологдоно).
func ruleState(l, prev *store.Lesson, p *store.LessonProgress, now time.Time, x dripExtra) LessonState {
	base := p.ViewedAt
	switch l.UnlockRule {
	case "complete":
		if p.CompletedAt == nil {
			return LessonState{Reason: "complete", PrevTitle: prev.Title}
		}
		base = *p.CompletedAt
	case "active": // хугацааг бүрэн судалсан бол шууд
		if need := prev.ActiveMin * 60; need > 0 && p.CompletedAt == nil && x.active[prev.ID] < need {
			return LessonState{Reason: "active", PrevTitle: prev.Title, ActiveLeft: (need - x.active[prev.ID] + 59) / 60, ActiveMin: prev.ActiveMin}
		}
		return LessonState{Open: true}
	case "quiz": // асуултуудад бүрэн зөв хариулсан бол шууд
		correct, total, done, _ := quizMastery(prev, p)
		if total > 0 && !done {
			return LessonState{Reason: "quiz", PrevTitle: prev.Title, QuizLeft: total - correct, QuizTotal: total}
		}
		return LessonState{Open: true}
	case "quiz_active": // асуулт бүрэн зөв БА хугацаа бүрэн
		if need := prev.ActiveMin * 60; need > 0 && p.CompletedAt == nil && x.active[prev.ID] < need {
			return LessonState{Reason: "active", PrevTitle: prev.Title, ActiveLeft: (need - x.active[prev.ID] + 59) / 60, ActiveMin: prev.ActiveMin}
		}
		if correct, total, done, _ := quizMastery(prev, p); total > 0 && !done {
			return LessonState{Reason: "quiz", PrevTitle: prev.Title, QuizLeft: total - correct, QuizTotal: total}
		}
		return LessonState{Open: true}
	case "exam": // шалгалтад тэнцсэн бол шууд
		if prev.Exam != nil && !x.passed[prev.ID] {
			return LessonState{Reason: "exam", PrevTitle: prev.Title}
		}
		return LessonState{Open: true}
	}
	at := base.Add(time.Duration(l.UnlockAfterH) * time.Hour)
	if now.Before(at) {
		return LessonState{Reason: "timer", PrevTitle: prev.Title, UnlockAt: &at}
	}
	return LessonState{Open: true}
}
