package store

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"
)

// learnMem — шалгалт, сесс, логийн санах ойн хадгалалт (тест ба хөгжүүлэлтэд).
type learnMem struct {
	mu       sync.RWMutex
	attempts map[string]*ExamAttempt
	sessions map[string]*StudySession
	events   []ActivityEvent
	rankPts  map[[2]string]int // (user, course) → оноо
	rankLvl  map[string]int    // user → олгосон түвшин
	subs     map[string]*Submission
}

func cloneSub(s *Submission) Submission {
	c := *s
	c.Files = append([]string{}, s.Files...)
	c.Links = append([]string{}, s.Links...)
	if s.Score != nil {
		v := *s.Score
		c.Score = &v
	}
	return c
}

func (m *Memory) SaveSubmission(_ context.Context, sub *Submission) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.subs == nil {
		l.subs = map[string]*Submission{}
	}
	sub.ID = sub.UserID + ":" + sub.LessonID
	c := cloneSub(sub)
	l.subs[sub.ID] = &c
	return nil
}

func (m *Memory) SubmissionFor(_ context.Context, userID, lessonID string) (*Submission, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	s, ok := l.subs[userID+":"+lessonID]
	if !ok {
		return nil, ErrNotFound
	}
	c := cloneSub(s)
	return &c, nil
}

func (m *Memory) Submissions(_ context.Context, lessonID string) ([]Submission, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := []Submission{}
	for _, s := range l.subs {
		if s.LessonID == lessonID {
			out = append(out, cloneSub(s))
		}
	}
	slices.SortFunc(out, func(a, b Submission) int { return b.SubmittedAt.Compare(a.SubmittedAt) })
	return out, nil
}

func (m *Memory) GradeSubmission(_ context.Context, lessonID, userID string, score int, feedback string) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	s, ok := l.subs[userID+":"+lessonID]
	if !ok {
		return ErrNotFound
	}
	now := time.Now()
	s.Score, s.Feedback, s.GradedAt = &score, feedback, &now
	return nil
}

func (m *Memory) RankCourseIDs(_ context.Context, userID string) ([]string, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	var out []string
	for k, v := range l.rankPts {
		if k[0] == userID && v > 0 {
			out = append(out, k[1])
		}
	}
	slices.Sort(out)
	return out, nil
}

func (m *Memory) SaveRankPoints(_ context.Context, userID, courseID string, points int) (int, error) {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.rankPts == nil {
		l.rankPts = map[[2]string]int{}
	}
	l.rankPts[[2]string{userID, courseID}] = points
	total := 0
	for k, v := range l.rankPts {
		if k[0] == userID {
			total += v
		}
	}
	return total, nil
}

func (m *Memory) UserRankLevel(_ context.Context, userID string) (int, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.rankLvl[userID], nil
}

func (m *Memory) SetUserRankLevel(_ context.Context, userID string, level int) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.rankLvl == nil {
		l.rankLvl = map[string]int{}
	}
	l.rankLvl[userID] = level
	return nil
}

func (f ActivityFilter) match(teacher, course, user, lesson, kind string, at time.Time) bool {
	return (f.TeacherID == "" || f.TeacherID == teacher) && (f.CourseID == "" || f.CourseID == course) &&
		(f.UserID == "" || f.UserID == user) && (f.LessonID == "" || f.LessonID == lesson) &&
		(f.Kind == "" || f.Kind == kind) && (f.Since.IsZero() || !at.Before(f.Since))
}

func cloneAttempt(a *ExamAttempt) ExamAttempt {
	c := *a
	c.Results = maps.Clone(a.Results)
	c.Order = slices.Clone(a.Order)
	return c
}

func (m *Memory) CreateExamAttempt(_ context.Context, a *ExamAttempt) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.attempts == nil {
		l.attempts = map[string]*ExamAttempt{}
	}
	a.ID = m.next()
	c := cloneAttempt(a)
	l.attempts[a.ID] = &c
	return nil
}

func (m *Memory) ExamAttemptByID(_ context.Context, id string) (*ExamAttempt, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	a, ok := l.attempts[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := cloneAttempt(a)
	return &c, nil
}

func (m *Memory) ExamAttempts(_ context.Context, f ActivityFilter) ([]ExamAttempt, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := []ExamAttempt{}
	for _, a := range l.attempts {
		if f.match(a.TeacherID, a.CourseID, a.UserID, a.LessonID, "", a.StartedAt) {
			out = append(out, cloneAttempt(a))
		}
	}
	slices.SortFunc(out, func(a, b ExamAttempt) int { return b.StartedAt.Compare(a.StartedAt) })
	return out, nil
}

func (m *Memory) FinishExamAttempt(_ context.Context, a *ExamAttempt) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	cur, ok := l.attempts[a.ID]
	if !ok {
		return ErrNotFound
	}
	if cur.Status != AttemptActive {
		return ErrConflict
	}
	cur.Status, cur.Reason, cur.FinishedAt = a.Status, a.Reason, a.FinishedAt
	cur.Score, cur.Max, cur.Pct, cur.Passed, cur.Results = a.Score, a.Max, a.Pct, a.Passed, maps.Clone(a.Results)
	return nil
}

func (m *Memory) AddAttemptViolation(_ context.Context, id string) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	if a, ok := l.attempts[id]; ok {
		a.Violations++
		return nil
	}
	return ErrNotFound
}

func (m *Memory) CreateSession(_ context.Context, s *StudySession) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sessions == nil {
		l.sessions = map[string]*StudySession{}
	}
	s.ID = m.next()
	if s.Counts == nil {
		s.Counts = map[string]int{}
	}
	c := *s
	c.Counts = maps.Clone(s.Counts)
	l.sessions[s.ID] = &c
	return nil
}

func (m *Memory) SessionByID(_ context.Context, id string) (*StudySession, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	s, ok := l.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := *s
	c.Counts = maps.Clone(s.Counts)
	return &c, nil
}

func (m *Memory) AddSessionBeat(_ context.Context, id string, b SessionBeat) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	s, ok := l.sessions[id]
	if !ok {
		return ErrNotFound
	}
	s.ActiveSec += b.ActiveSec
	s.IdleSec += b.IdleSec
	s.AwaySec += b.AwaySec
	s.FocusSum += b.FocusSum
	s.FocusN += b.FocusN
	s.Camera = s.Camera || b.Camera
	for k, v := range b.Counts {
		s.Counts[k] += v
	}
	s.LastAt = time.Now()
	if b.End != "" {
		s.Ended, s.EndReason = true, b.End
	}
	return nil
}

func (m *Memory) Sessions(_ context.Context, f ActivityFilter, limit int) ([]StudySession, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := []StudySession{}
	for _, s := range l.sessions {
		if f.match(s.TeacherID, s.CourseID, s.UserID, s.LessonID, s.Kind, s.LastAt) {
			c := *s
			c.Counts = maps.Clone(s.Counts)
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b StudySession) int { return b.LastAt.Compare(a.LastAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) AddActivityEvents(_ context.Context, evs []ActivityEvent) error {
	l := &m.learn
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, e := range evs {
		e.ID = m.next()
		if e.At.IsZero() {
			e.At = time.Now()
		}
		l.events = append(l.events, e)
	}
	return nil
}

func (m *Memory) ActivityEvents(_ context.Context, f ActivityFilter, limit int) ([]ActivityEvent, error) {
	l := &m.learn
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := []ActivityEvent{}
	for i := len(l.events) - 1; i >= 0 && (limit <= 0 || len(out) < limit); i-- {
		e := l.events[i]
		if !f.BeforeAt.IsZero() && !(e.At.Before(f.BeforeAt) || (e.At.Equal(f.BeforeAt) && e.ID < f.BeforeID)) {
			continue
		}
		if f.match(e.TeacherID, e.CourseID, e.UserID, e.LessonID, "", e.At) {
			out = append(out, e)
		}
	}
	return out, nil
}
