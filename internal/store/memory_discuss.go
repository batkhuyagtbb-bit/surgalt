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

// ---- чат: хариулах, реакц, уншсан (санах ой) ----

type chatExtraMem struct {
	mu        sync.RWMutex
	reactions map[string]map[string]ReactUser // message → userKey → реакц (Name, эможи нь Key талбарт биш, тусад нь)
	emoji     map[[2]string]string            // (message, userKey) → эможи
	reads     map[[2]string]string            // (conversation, userKey) → last_id
}

func (m *Memory) cx() *chatExtraMem {
	m.chatXOnce.Do(func() {
		m.chatX = &chatExtraMem{reactions: map[string]map[string]ReactUser{}, emoji: map[[2]string]string{}, reads: map[[2]string]string{}}
	})
	return m.chatX
}

func (m *Memory) MessageByID(_ context.Context, conversationID, id string) (*Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, msg := range m.messages[conversationID] {
		if msg.ID == id {
			c := *msg
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *Memory) ReactMessage(_ context.Context, _ string, messageID, userKey, name, emoji string) error {
	x := m.cx()
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.reactions[messageID] == nil {
		x.reactions[messageID] = map[string]ReactUser{}
	}
	x.reactions[messageID][userKey] = ReactUser{Key: userKey, Name: name}
	x.emoji[[2]string{messageID, userKey}] = emoji
	return nil
}

func (m *Memory) MessageReactions(_ context.Context, ids []string) (map[string]map[string][]ReactUser, error) {
	x := m.cx()
	x.mu.RLock()
	defer x.mu.RUnlock()
	out := map[string]map[string][]ReactUser{}
	for _, id := range ids {
		for key, u := range x.reactions[id] {
			e := x.emoji[[2]string{id, key}]
			if e == "" {
				continue
			}
			if out[id] == nil {
				out[id] = map[string][]ReactUser{}
			}
			out[id][e] = append(out[id][e], u)
		}
	}
	return out, nil
}

func (m *Memory) MarkRead(_ context.Context, conversationID, userKey, lastID string) error {
	x := m.cx()
	x.mu.Lock()
	defer x.mu.Unlock()
	if cur := x.reads[[2]string{conversationID, userKey}]; lastID > cur {
		x.reads[[2]string{conversationID, userKey}] = lastID
	}
	return nil
}

func (m *Memory) ConversationReads(_ context.Context, conversationID string) (map[string]string, error) {
	x := m.cx()
	x.mu.RLock()
	defer x.mu.RUnlock()
	out := map[string]string{}
	for k, v := range x.reads {
		if k[0] == conversationID {
			out[k[1]] = v
		}
	}
	return out, nil
}
