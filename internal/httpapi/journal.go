package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"surgalt/internal/store"
)

// Сургалтын журнал (багшид): суралцагч × хичээлийн хүснэгт — хичээл бүрийн оноо, дууссан эсэх, асуулга,
// шалгалтын хувь, даалгаврын дүн. Нүд, мөр дээр дарахад дэлгэрэнгүй (идэвхтэй хугацаа, видео, дүгнэлт,
// таб солилт, үндэслэл). Нэг сургалтын бүх нотолгоог нэг хүсэлтээр.

type JournalLesson struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Position int    `json:"position"`
	Section  string `json:"section,omitempty"`
	Kind     string `json:"kind"` // lesson | exam | assignment
	Hidden   bool   `json:"hidden,omitempty"`
	IsFree   bool   `json:"is_free,omitempty"`
	// Rubric — рубрикаар үнэлдэг даалгаврын хүснэгт (журналд шалгуур бүрийн дүнг харуулахад).
	Rubric *store.Rubric `json:"rubric,omitempty"`
}

type JournalExam struct {
	Best       int  `json:"best"`
	Passed     bool `json:"passed"`
	Tries      int  `json:"tries"`
	Terminated int  `json:"terminated,omitempty"`
}

type JournalAsg struct {
	Score    *int           `json:"score,omitempty"`
	Max      int            `json:"max"`
	Late     bool           `json:"late,omitempty"`
	At       time.Time      `json:"at"`
	Feedback string         `json:"feedback,omitempty"`
	Rubric   map[string]int `json:"rubric,omitempty"` // шалгуур бүрээс сонгосон түвшин
}

type JournalCell struct {
	Points    int          `json:"pts"`
	Viewed    bool         `json:"viewed,omitempty"`
	Done      bool         `json:"done,omitempty"`
	Disq      bool         `json:"disq,omitempty"`
	Improved  bool         `json:"improved,omitempty"` // дахин судалж оноо сайжирсан (rank.go lessonPoints)
	QuizOK    int          `json:"q_ok,omitempty"`
	QuizN     int          `json:"q_n,omitempty"`
	Active    int          `json:"active,omitempty"` // секунд
	Total     int          `json:"total,omitempty"`
	Sessions  int          `json:"sessions,omitempty"`
	Video     int          `json:"video,omitempty"` // %
	HasVideo  bool         `json:"has_video,omitempty"`
	Reflected bool         `json:"refl,omitempty"`
	Tabs      int          `json:"tabs,omitempty"`
	Reasons   []string     `json:"reasons,omitempty"`
	Exam      *JournalExam `json:"exam,omitempty"`
	Asg       *JournalAsg  `json:"asg,omitempty"`
	Last      *time.Time   `json:"last,omitempty"`
}

type JournalStudent struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"` // багшийн өгсөн нэр (alias) байвал тэр, эс бөгөөс бүртгэлийн нэр
	// AccountName — бүртгэлийн нэр (DisplayName) үргэлж. Кэшлэсэн нэгтгэлд хоосон; хуудсыг хуулахад бөглөнө.
	AccountName string `json:"account_name"`
	// Alias, Group, Archived — багшийн журналын мэдээлэл (journal_meta.go); суралцагчид хэзээ ч харагдахгүй.
	Alias      string     `json:"alias,omitempty"`
	Group      string     `json:"group,omitempty"`
	Archived   bool       `json:"archived,omitempty"`
	ArchivedAt *time.Time `json:"archived_at,omitempty"`
	Username   string     `json:"username,omitempty"`
	Avatar     string     `json:"avatar_url,omitempty"`
	Enrolled   bool       `json:"enrolled"`
	Rank       RankInfo   `json:"rank"`
	Done       int        `json:"done"`
	Avg        int        `json:"avg"`    // үнэлэгдсэн хичээлийн дундаж оноо
	Active     int        `json:"active"` // нийт идэвхтэй секунд
	Last       *time.Time `json:"last,omitempty"`
	Recent     bool       `json:"recent,omitempty"` // сүүлийн 7 хоногт хичээллэсэн
}

// recentDays — журнал, тэмдэглэлд «идэвхтэй» гэж тооцох хугацаа.
const recentDays = 7

// sortJournal — идэвхтэй суралцагчид дээрээ: сүүлийн 7 хоногт хичээллэсэн → өмнө нь хичээллэсэн
// (оноо, дууссан хичээл, идэвхтэй хугацаагаар) → огт эхлээгүй (цагаан толгойн дарааллаар).
func sortJournal(out []JournalStudent) {
	tier := func(js *JournalStudent) int {
		switch {
		case js.Recent:
			return 0
		case js.Last != nil || js.Rank.Points > 0 || js.Done > 0 || js.Active > 0:
			return 1
		}
		return 2
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := &out[i], &out[j]
		if ta, tb := tier(a), tier(b); ta != tb {
			return ta < tb
		}
		if a.Rank.Points != b.Rank.Points {
			return a.Rank.Points > b.Rank.Points
		}
		if a.Done != b.Done {
			return a.Done > b.Done
		}
		if a.Active != b.Active {
			return a.Active > b.Active
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}

// handleCourseJournal: GET /api/me/courses/{id}/journal?q=&group=&nogroup=1&archived=1
func (s *Server) handleCourseJournal(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	ctx := r.Context()
	jr, err := s.journal(ctx, course)
	if s.storeErr(w, r, err) {
		return
	}
	// Багшийн өгсөн нэр, бүлэг, архив — 30 секундын нэгтгэлийн гадна хүсэлт бүрт (засвар бүх хуулбарт шууд харагдана).
	metas, err := s.store.CourseStudentMetas(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	// Хуудас: нэрээр хайж, бүлэг/архиваар шүүж, эхний journalStudents-ийг (сүүлд идэвхтэй байснаас) нүднүүдийнх нь хамт.
	qv := r.URL.Query()
	v := journalPage(jr, metas, journalFilter{
		q:        strings.ToLower(strings.TrimSpace(qv.Get("q"))),
		group:    journalName(qv.Get("group")),
		noGroup:  qv.Get("nogroup") == "1",
		archived: qv.Get("archived") == "1",
	}, journalStudents)
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"course":         map[string]any{"id": course.ID, "title": course.Title},
		"lessons":        jr.lessons,
		"students":       v.students,
		"cells":          v.cells,
		"students_total": v.total,
		"groups":         v.groups,
		"ungrouped":      v.ungrouped,
		"archived":       v.archived,
	})
}

// journal — сургалтын нэгтгэл (30 секунд кэштэй).
func (s *Server) journal(ctx context.Context, course *store.Course) (*journalAgg, error) {
	return s.journals.GetOrLoad(ctx, course.ID, func(ctx context.Context) (*journalAgg, error) { return s.buildJournal(ctx, course) })
}

// journalStudents — журналд нэг дор өгөх суралцагчийн дээд тоо (бусдыг нэрээр хайна).
const journalStudents = 300

// journalAgg — нэг сургалтын бүтэн журнал (бүх суралцагч × хичээл). 30 000 суралцагчтай сургалтад тооцоо хүнд тул
// 30 секунд кэшилж, хариуд нь зөвхөн хуудсыг өгнө. Бүх хүсэлт хуваалцдаг тул ЗӨВХӨН УНШИНА (journalPage хуулбарлана).
type journalAgg struct {
	lessons  []JournalLesson
	students []JournalStudent // sortJournal-ийн дарааллаар
	cells    map[string]map[string]*JournalCell
	lower    [][2]string // students-тэй ижил дарааллаар: жижиг үсгээр нэр, username (хайлтад)
	blank    RankInfo    // эхлээгүй суралцагчийн эрэмбэ (өгөгдөл нь устсан суралцагчийн хоосон мөрөнд)
	built    time.Time   // тооцоо эхэлсэн мөч — үүнээс хойш өгөгдөл нь устсан суралцагчийн мөр хуучирсан
}

// journalFilter — журналын хуудасны шүүлт (бүгд AND). q нь жижиг үсгээр.
type journalFilter struct {
	q, group          string
	noGroup, archived bool
}

// JournalGroup — сургалтын бүлэг ба архивлаагүй суралцагчийн тоо.
type JournalGroup struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// journalView — нэг хариуны хуудас: шүүсэн суралцагчид (≤ limit), нүднүүд, шүүлтийн дараах нийт тоо, бүх суралцагчаар
// тоолсон бүлэг/бүлэггүй/архив.
type journalView struct {
	students  []JournalStudent
	cells     map[string]map[string]*JournalCell
	total     int
	groups    []JournalGroup
	ungrouped int
	archived  int
}

// journalPage — кэшлэсэн нэгтгэлийг журналын мэдээлэлтэй нэг давталтаар нийлүүлнэ. jr-ийг өөрчлөхгүй: зөвхөн хуудсанд
// орох мөрүүдийг хуулна. Бүлэг, бүлэггүй, архивын тоо нь шүүлтээс үл хамааран бүх суралцагчаар. Мэдээлэл нь нэгтгэлээс
// хойш устгагдсан (ResetAt > built) суралцагчийн мөр хуучирсан тул элссэн бол хоосон мөр (нүдгүй), элсээгүй бол хасна.
func journalPage(jr *journalAgg, metas map[string]store.CourseStudentMeta, f journalFilter, limit int) journalView {
	v := journalView{students: make([]JournalStudent, 0, max(0, min(limit, len(jr.students)))), cells: map[string]map[string]*JournalCell{}}
	type grp struct {
		name string
		n    int
	}
	groups := map[string]*grp{}
	for i := range jr.students {
		js := &jr.students[i]
		m := metas[js.UserID]
		stale := m.ResetAt != nil && m.ResetAt.After(jr.built)
		if stale && !js.Enrolled {
			continue
		}
		switch {
		case m.Archived:
			v.archived++
		case m.Group == "":
			v.ungrouped++
		default:
			k := strings.ToLower(m.Group)
			if g := groups[k]; g != nil {
				g.n++
			} else {
				groups[k] = &grp{name: m.Group, n: 1}
			}
		}
		if m.Archived != f.archived || (f.group != "" && !strings.EqualFold(m.Group, f.group)) || (f.noGroup && m.Group != "") {
			continue
		}
		if f.q != "" {
			ln, lu := jr.lowerAt(i)
			if !strings.Contains(ln, f.q) && !strings.Contains(lu, f.q) && (m.Alias == "" || !strings.Contains(strings.ToLower(m.Alias), f.q)) {
				continue
			}
		}
		v.total++
		if len(v.students) >= limit {
			continue
		}
		cp := *js
		if stale {
			cp = JournalStudent{UserID: js.UserID, Name: js.Name, Username: js.Username, Avatar: js.Avatar, Enrolled: js.Enrolled, Rank: jr.blank}
		} else if row := jr.cells[js.UserID]; row != nil {
			v.cells[js.UserID] = row
		}
		cp.AccountName = js.Name
		if m.Alias != "" {
			cp.Name = m.Alias
		}
		cp.Alias, cp.Group, cp.Archived, cp.ArchivedAt = m.Alias, m.Group, m.Archived, m.ArchivedAt
		v.students = append(v.students, cp)
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	v.groups = make([]JournalGroup, len(keys))
	for i, k := range keys {
		v.groups[i] = JournalGroup{Name: groups[k].name, Count: groups[k].n}
	}
	return v
}

// lowerAt — i-р суралцагчийн нэр, username-ийг жижиг үсгээр (нэгтгэлд бэлдсэн; байхгүй бол тооцно).
func (jr *journalAgg) lowerAt(i int) (string, string) {
	if i < len(jr.lower) {
		return jr.lower[i][0], jr.lower[i][1]
	}
	return strings.ToLower(jr.students[i].Name), strings.ToLower(jr.students[i].Username)
}

func (s *Server) buildJournal(ctx context.Context, course *store.Course) (*journalAgg, error) {
	built := time.Now() // уншилт эхлэхээс өмнө: энэ мөчөөс хойших устгал нэгтгэлд тусаагүй байж болно
	lessons, err := s.store.LessonsByCourse(ctx, course.ID)
	if err != nil {
		return nil, err
	}
	prog, err := s.store.CourseProgress(ctx, course.ID)
	if err != nil {
		return nil, err
	}
	f := store.ActivityFilter{CourseID: course.ID}
	ss, _ := s.store.Sessions(ctx, f, 50000)
	refs, _ := s.store.Reflections(ctx, f, 50000)
	watches, _ := s.store.VideoWatches(ctx, f)
	atts, _ := s.store.ExamAttempts(ctx, f)
	students, _ := s.store.CourseStudents(ctx, course.ID)

	users, enrolled, names := map[string]bool{}, map[string]bool{}, map[string]string{}
	for _, st := range students {
		users[st.UserID] = true
		enrolled[st.UserID] = st.EnrolledAt != nil || len(st.LessonIDs) > 0
	}
	for uid := range prog {
		users[uid] = true
	}
	ev := map[string]*rankEvidence{}
	evOf := func(uid string) *rankEvidence {
		if ev[uid] == nil {
			ev[uid] = newRankEvidence()
		}
		return ev[uid]
	}
	for i := range ss {
		users[ss[i].UserID] = true
		names[ss[i].UserID] = ss[i].UserName
		evOf(ss[i].UserID).addSession(&ss[i])
	}
	for _, rf := range refs {
		evOf(rf.UserID).reflect[rf.LessonID] = true
	}
	for i := range watches {
		evOf(watches[i].UserID).addWatch(&watches[i])
	}

	workAt := map[string]time.Time{} // шалгалт, даалгаврын сүүлийн хугацаа (идэвхээр эрэмбэлэхэд)
	touch := func(uid string, t time.Time) {
		if t.After(workAt[uid]) {
			workAt[uid] = t
		}
	}
	cells := map[string]map[string]*JournalCell{}
	cell := func(uid, lid string) *JournalCell {
		if cells[uid] == nil {
			cells[uid] = map[string]*JournalCell{}
		}
		if cells[uid][lid] == nil {
			cells[uid][lid] = &JournalCell{}
		}
		return cells[uid][lid]
	}
	for _, a := range atts {
		if a.Status == store.AttemptActive {
			continue
		}
		users[a.UserID] = true
		touch(a.UserID, a.StartedAt)
		if a.FinishedAt != nil {
			touch(a.UserID, *a.FinishedAt)
		}
		cl := cell(a.UserID, a.LessonID)
		if cl.Exam == nil {
			cl.Exam = &JournalExam{Best: -1}
		}
		cl.Exam.Tries++
		cl.Exam.Best = max(cl.Exam.Best, a.Pct)
		cl.Exam.Passed = cl.Exam.Passed || a.Passed
		if a.Status == store.AttemptTerminated {
			cl.Exam.Terminated++
		}
	}
	jl := make([]JournalLesson, len(lessons))
	for i, l := range lessons {
		kind := "lesson"
		if l.Exam != nil {
			kind = "exam"
		} else if l.Assignment != nil {
			kind = "assignment"
			if subs, err := s.store.Submissions(ctx, l.ID); err == nil {
				for _, sb := range subs {
					users[sb.UserID] = true
					touch(sb.UserID, sb.SubmittedAt)
					cell(sb.UserID, l.ID).Asg = &JournalAsg{Score: sb.Score, Max: l.Assignment.MaxScore, Late: sb.Late, At: sb.SubmittedAt, Feedback: sb.Feedback, Rubric: sb.Rubric}
				}
			}
		}
		jl[i] = JournalLesson{ID: l.ID, Title: l.Title, Position: l.Position, Section: l.Section, Kind: kind, Hidden: l.Hidden, IsFree: l.IsFree}
		if l.Assignment != nil && l.Assignment.Grading == "rubric" {
			jl[i].Rubric = l.Assignment.Rubric
		}
	}
	delete(users, course.TeacherID)
	delete(users, "")

	visible := visibleLessons(lessons, false)
	out := make([]JournalStudent, 0, len(users))
	for uid := range users {
		ranks, info := computeRanks(visible, prog[uid], evOf(uid))
		js := JournalStudent{UserID: uid, Name: names[uid], Enrolled: enrolled[uid], Rank: info}
		sum := 0
		for _, lr := range ranks {
			cl := cell(uid, lr.LessonID)
			cl.Points, cl.Viewed, cl.Done, cl.Disq, cl.Improved = lr.Points, true, lr.Completed, lr.Disqualified, lr.Improved
			cl.QuizOK, cl.QuizN, cl.Active, cl.Total, cl.Sessions = lr.QuizCorrect, lr.QuizTotal, lr.ActiveSec, lr.TotalSec, lr.Sessions
			cl.Video, cl.HasVideo, cl.Reflected, cl.Tabs, cl.Reasons, cl.Last = lr.VideoPct, lr.HasVideo, lr.Reflected, lr.TabSwitches, lr.Reasons, lr.LastAt
			if lr.Completed {
				js.Done++
			}
			sum += lr.Points
			js.Active += lr.ActiveSec
			if lr.LastAt != nil && (js.Last == nil || lr.LastAt.After(*js.Last)) {
				js.Last = lr.LastAt
			}
		}
		if len(ranks) > 0 {
			js.Avg = sum / len(ranks)
		}
		if t, ok := workAt[uid]; ok && (js.Last == nil || t.After(*js.Last)) {
			js.Last = &t
		}
		js.Recent = js.Last != nil && time.Since(*js.Last) <= recentDays*24*time.Hour
		out = append(out, js)
	}
	ids := make([]string, 0, len(out))
	for _, js := range out {
		ids = append(ids, js.UserID)
	}
	if us, err := s.store.UsersByIDs(ctx, ids); err == nil {
		byID := make(map[string]store.User, len(us))
		for _, u := range us {
			byID[u.ID] = u
		}
		for i := range out {
			if u, ok := byID[out[i].UserID]; ok {
				out[i].Name, out[i].Username, out[i].Avatar = u.DisplayName, u.Username, u.AvatarURL
			}
		}
	}
	for i := range out {
		if out[i].Name == "" {
			out[i].Name = "Суралцагч"
		}
	}
	sortJournal(out)
	lower := make([][2]string, len(out))
	for i := range out {
		lower[i] = [2]string{strings.ToLower(out[i].Name), strings.ToLower(out[i].Username)}
	}
	_, blank := computeRanks(visible, nil, newRankEvidence())
	return &journalAgg{lessons: jl, students: out, cells: cells, lower: lower, blank: blank, built: built}, nil
}
