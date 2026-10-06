package httpapi

import (
	"context"
	"sort"
	"time"

	"surgalt/internal/store"
)

// Суралцагчийн өөрийн хуудас ("Миний хэсэг"): цол, зөвлөмж, сургалт бүрийн цол, даалгавар ба шалгалтын
// хугацаа, төлөв, дүн — бүгдийг нэг дор.

// HomeCourseRank — нэг сургалт дахь цол.
type HomeCourseRank struct {
	CourseID string       `json:"course_id"`
	Title    string       `json:"title"`
	Rank     RankInfo     `json:"rank"`
	Lessons  []LessonRank `json:"lessons"`
}

// HomeTask — даалгавар эсвэл шалгалт: хугацаа, миний төлөв.
type HomeTask struct {
	CourseID    string         `json:"course_id"`
	CourseTitle string         `json:"course_title"`
	LessonID    string         `json:"lesson_id"`
	Title       string         `json:"title"`
	Kind        string         `json:"kind"` // exam | assignment
	Due         DueState       `json:"due"`
	Status      string         `json:"status"` // open | submitted | graded | passed | failed | need_pay | closed
	Score       *int           `json:"score,omitempty"`
	MaxScore    int            `json:"max_score,omitempty"`
	Feedback    string         `json:"feedback,omitempty"`
	ExamBest    int            `json:"exam_best,omitempty"`
	ExamCount   int            `json:"exam_count,omitempty"`
	SubmittedAt *time.Time     `json:"submitted_at,omitempty"`
	Extra       map[string]any `json:"extra,omitempty"`
}

// myProgress — суралцагчийн бүх сургалтаар цол ба даалгавар/шалгалтын төлөвийг нэгтгэнэ.
// Цолыг системээр дахин тооцож (autoAward) шинэ түвшин олгогдсон бол awarded=true.
func (s *Server) myProgress(ctx context.Context, uid string, courses []HomeCourse) (rank RankInfo, byCourse []HomeCourseRank, tasks []HomeTask, awarded bool) {
	var all []LessonRank
	now := time.Now()
	for _, hc := range courses {
		c := hc.Course
		lessons, err := s.store.LessonsByCourse(ctx, c.ID)
		if err != nil {
			continue
		}
		progress, err := s.store.LessonProgress(ctx, uid, c.ID)
		if err != nil {
			continue
		}
		ranks, info := s.courseRanksCtx(ctx, uid, c.ID, lessons, progress)
		if len(ranks) > 0 {
			_, aw := s.autoAward(ctx, uid, c.ID, "/c/"+c.ID, info.Points)
			awarded = awarded || aw
			all = append(all, ranks...)
			byCourse = append(byCourse, HomeCourseRank{CourseID: c.ID, Title: c.Title, Rank: info, Lessons: ranks})
		}
		for i := range lessons {
			l := &lessons[i]
			if l.Exam == nil && l.Assignment == nil {
				continue
			}
			t := HomeTask{CourseID: c.ID, CourseTitle: c.Title, LessonID: l.ID, Title: l.Title, Kind: "assignment"}
			var d store.Due
			if l.Exam != nil {
				t.Kind, d = "exam", l.Exam.Due
			} else {
				d, t.MaxScore = l.Assignment.Due, l.Assignment.MaxScore
			}
			t.Due = dueStateOf(d, progress[l.ID].Quiz[store.LatePassKey], now)
			switch {
			case t.Due.Closed:
				t.Status = "closed"
			case t.Due.NeedPay:
				t.Status = "need_pay"
			default:
				t.Status = "open"
			}
			if l.Assignment != nil {
				if sub, err := s.store.SubmissionFor(ctx, uid, l.ID); err == nil {
					t.SubmittedAt, t.Score, t.Feedback = &sub.SubmittedAt, sub.Score, sub.Feedback
					t.Status = "submitted"
					if sub.Score != nil {
						t.Status = "graded"
					}
				}
			} else if atts, err := s.store.ExamAttempts(ctx, store.ActivityFilter{UserID: uid, LessonID: l.ID}); err == nil && len(atts) > 0 {
				t.ExamBest = -1
				for _, a := range atts {
					if a.Status == store.AttemptActive {
						continue
					}
					t.ExamCount++
					if a.Pct > t.ExamBest {
						t.ExamBest = a.Pct
					}
					if a.Passed {
						t.Status = "passed"
					}
				}
				if t.Status != "passed" && t.ExamCount > 0 {
					t.Status = "failed"
				}
			}
			tasks = append(tasks, t)
		}
	}
	// Хугацаа ойрхон, дуусаагүй нь эхэнд.
	sort.SliceStable(tasks, func(i, j int) bool {
		di, dj := tasks[i].Due.At, tasks[j].Due.At
		switch {
		case di == nil && dj == nil:
			return false
		case di == nil:
			return false
		case dj == nil:
			return true
		}
		return di.Before(*dj)
	})
	total := 0
	for _, r := range byCourse {
		total += r.Rank.Points
	}
	rank = summarizeRank(all, total)
	return
}

// courseRanksCtx — courseRanks-ийн context хувилбар (нүүр хуудсанд).
func (s *Server) courseRanksCtx(ctx context.Context, userID, courseID string, lessons []store.Lesson, progress map[string]store.LessonProgress) ([]LessonRank, RankInfo) {
	return s.courseRanks(ctx, userID, courseID, lessons, progress)
}
