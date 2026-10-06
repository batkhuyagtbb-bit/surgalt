package store

import (
	"context"
	"slices"
	"sync"
	"time"
)

// discussMem — хэлэлцүүлгийн санах ойн хадгалалт.
type discussMem struct {
	mu       sync.RWMutex
	comments map[string]*Comment
	likes    map[[3]string]bool // (target, target_id, user) → liked
}

func (m *Memory) dm() *discussMem {
	m.discussOnce.Do(func() { m.discuss = &discussMem{comments: map[string]*Comment{}, likes: map[[3]string]bool{}} })
	return m.discuss
}

func (m *Memory) AddComment(_ context.Context, cm *Comment) error {
	d := m.dm()
	d.mu.Lock()
	defer d.mu.Unlock()
	cm.ID, cm.CreatedAt = NewID(), time.Now()
	c := *cm
	d.comments[cm.ID] = &c
	return nil
}

func (m *Memory) Comments(_ context.Context, lessonID string, limit int) ([]Comment, error) {
	d := m.dm()
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := []Comment{}
	for _, c := range d.comments {
		if c.LessonID == lessonID && !c.Deleted {
			out = append(out, *c)
		}
	}
	slices.SortFunc(out, func(a, b Comment) int { return a.CreatedAt.Compare(b.CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) CommentByID(_ context.Context, id string) (*Comment, error) {
	d := m.dm()
	d.mu.RLock()
	defer d.mu.RUnlock()
	c, ok := d.comments[id]
	if !ok || c.Deleted {
		return nil, ErrNotFound
	}
	cc := *c
	return &cc, nil
}

func (m *Memory) DeleteComment(_ context.Context, id string) error {
	d := m.dm()
	d.mu.Lock()
	defer d.mu.Unlock()
	c, ok := d.comments[id]
	if !ok {
		return ErrNotFound
	}
	c.Deleted = true
	return nil
}

func (m *Memory) CommentCounts(_ context.Context, lessonIDs []string) (map[string]int, error) {
	d := m.dm()
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := map[string]int{}
	for _, c := range d.comments {
		if !c.Deleted && slices.Contains(lessonIDs, c.LessonID) {
			out[c.LessonID]++
		}
	}
	return out, nil
}

func (m *Memory) ToggleLike(_ context.Context, userID, target, targetID string) (bool, int, error) {
	d := m.dm()
	d.mu.Lock()
	defer d.mu.Unlock()
	k := [3]string{target, targetID, userID}
	d.likes[k] = !d.likes[k]
	n := 0
	for kk, v := range d.likes {
		if v && kk[0] == target && kk[1] == targetID {
			n++
		}
	}
	return d.likes[k], n, nil
}

func (m *Memory) Likes(_ context.Context, userID, target string, targetIDs []string) (map[string]int, map[string]bool, error) {
	d := m.dm()
	d.mu.RLock()
	defer d.mu.RUnlock()
	counts, mine := map[string]int{}, map[string]bool{}
	for k, v := range d.likes {
		if v && k[0] == target && slices.Contains(targetIDs, k[1]) {
			counts[k[1]]++
			if k[2] == userID {
				mine[k[1]] = true
			}
		}
	}
	return counts, mine, nil
}
