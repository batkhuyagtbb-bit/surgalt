package httpapi

import (
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
}

type JournalExam struct {
	Best       int  `json:"best"`
	Passed     bool `json:"passed"`
	Tries      int  `json:"tries"`
	Terminated int  `json:"terminated,omitempty"`
}

type JournalAsg struct {
	Score    *int      `json:"score,omitempty"`
	Max      int       `json:"max"`
	Late     bool      `json:"late,omitempty"`
	At       time.Time `json:"at"`
	Feedback string    `json:"feedback,omitempty"`
}

type JournalCell struct {
	Points    int          `json:"pts"`
	Viewed    bool         `json:"viewed,omitempty"`
	Done      bool         `json:"done,omitempty"`
	Disq      bool         `json:"disq,omitempty"`
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
	UserID   string     `json:"user_id"`
	Name     string     `json:"name"`
	Username string     `json:"username,omitempty"`
	Avatar   string     `json:"avatar_url,omitempty"`
	Enrolled bool       `json:"enrolled"`
	Rank     RankInfo   `json:"rank"`
	Done     int        `json:"done"`
	Avg      int        `json:"avg"`    // үнэлэгдсэн хичээлийн дундаж оноо
	Active   int        `json:"active"` // нийт идэвхтэй секунд
	Last     *time.Time `json:"last,omitempty"`
}

// handleCourseJournal: GET /api/me/courses/{id}/journal
func (s *Server) handleCourseJournal(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	course, ok := s.ownCourse(w, r, c.UID)
	if !ok {
		return
	}
	ctx := r.Context()
	lessons, err := s.store.LessonsByCourse(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	prog, err := s.store.CourseProgress(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
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
					cell(sb.UserID, l.ID).Asg = &JournalAsg{Score: sb.Score, Max: l.Assignment.MaxScore, Late: sb.Late, At: sb.SubmittedAt, Feedback: sb.Feedback}
				}
			}
		}
		jl[i] = JournalLesson{ID: l.ID, Title: l.Title, Position: l.Position, Section: l.Section, Kind: kind, Hidden: l.Hidden, IsFree: l.IsFree}
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
			cl.Points, cl.Viewed, cl.Done, cl.Disq = lr.Points, true, lr.Completed, lr.Disqualified
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
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) }) // журнал шиг цагаан толгойн дарааллаар
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]any{
		"course":   map[string]any{"id": course.ID, "title": course.Title},
		"lessons":  jl,
		"students": out,
		"cells":    cells,
	})
}
