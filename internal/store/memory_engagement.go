package store

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"
)

// engageMem — асуултын хурд, дүгнэлт, видео үзэлтийн зураглалын санах ойн хадгалалт.
type engageMem struct {
	mu      sync.RWMutex
	quiz    []QuizLog
	refl    map[string]*Reflection // id
	watches map[string]*VideoWatch // id
}

func (m *Memory) AddQuizLog(_ context.Context, l QuizLog) error {
	e := &m.engage
	e.mu.Lock()
	defer e.mu.Unlock()
	l.ID = m.next()
	if l.At.IsZero() {
		l.At = time.Now()
	}
	e.quiz = append(e.quiz, l)
	return nil
}

func (m *Memory) QuizLogs(_ context.Context, f ActivityFilter, limit int) ([]QuizLog, error) {
	e := &m.engage
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := []QuizLog{}
	for i := len(e.quiz) - 1; i >= 0 && (limit <= 0 || len(out) < limit); i-- {
		q := e.quiz[i]
		if f.match(q.TeacherID, q.CourseID, q.UserID, q.LessonID, "", q.At) {
			out = append(out, q)
		}
	}
	return out, nil
}

func (m *Memory) SaveReflection(_ context.Context, r *Reflection) error {
	e := &m.engage
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.refl == nil {
		e.refl = map[string]*Reflection{}
	}
	r.ID = r.UserID + ":" + r.LessonID
	if r.At.IsZero() {
		r.At = time.Now()
	}
	c := *r
	e.refl[r.ID] = &c
	return nil
}

func (m *Memory) Reflections(_ context.Context, f ActivityFilter, limit int) ([]Reflection, error) {
	e := &m.engage
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := []Reflection{}
	for _, r := range e.refl {
		if f.match(r.TeacherID, r.CourseID, r.UserID, r.LessonID, "", r.At) {
			out = append(out, *r)
		}
	}
	slices.SortFunc(out, func(a, b Reflection) int { return b.At.Compare(a.At) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) AddVideoWatch(_ context.Context, v VideoWatch) error {
	e := &m.engage
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.watches == nil {
		e.watches = map[string]*VideoWatch{}
	}
	id := v.UserID + ":" + v.LessonID + ":" + v.BlockID
	cur := e.watches[id]
	if cur == nil {
		v.ID, v.At = id, time.Now()
		c := v
		c.Buckets = maps.Clone(v.Buckets)
		e.watches[id] = &c
		return nil
	}
	if v.Duration > cur.Duration {
		cur.Duration = v.Duration
	}
	for k, n := range v.Buckets {
		cur.Buckets[k] += n
	}
	cur.At = time.Now()
	return nil
}

func (m *Memory) VideoWatches(_ context.Context, f ActivityFilter) ([]VideoWatch, error) {
	e := &m.engage
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := []VideoWatch{}
	for _, v := range e.watches {
		if f.match(v.TeacherID, v.CourseID, v.UserID, v.LessonID, "", v.At) {
			c := *v
			c.Buckets = maps.Clone(v.Buckets)
			out = append(out, c)
		}
	}
	return out, nil
}
