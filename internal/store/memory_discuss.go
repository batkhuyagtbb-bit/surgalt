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

// ---- сурагч хоорондын яриа ба бүлэг (санах ой) ----

func (m *Memory) CreateDM(_ context.Context, a, b, bName string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, cv := range m.convs {
		if cv.Kind == ConvDM && ((cv.TeacherID == a && cv.UserID == b) || (cv.TeacherID == b && cv.UserID == a)) {
			cc := *cv
			return &cc, nil
		}
	}
	now := time.Now()
	cv := &Conversation{ID: m.next(), TeacherID: a, VisitorKey: "u:" + b, VisitorName: bName, UserID: b, Kind: ConvDM, CreatedAt: now, LastMessageAt: now}
	m.convs[cv.ID] = cv
	cc := *cv
	return &cc, nil
}

func (m *Memory) CreateTeam(_ context.Context, creator, title string, members []string) (*Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	id := m.next()
	cv := &Conversation{ID: id, TeacherID: creator, VisitorKey: TeamVisitorKey(id), VisitorName: title, Kind: ConvTeam, CreatedAt: now, LastMessageAt: now}
	m.convs[id] = cv
	if m.teamMembers == nil {
		m.teamMembers = map[string]map[string]bool{}
	}
	m.teamMembers[id] = map[string]bool{creator: true}
	for _, u := range members {
		if u != "" {
			m.teamMembers[id][u] = true
		}
	}
	cc := *cv
	return &cc, nil
}

func (m *Memory) TeamMembers(_ context.Context, convID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []string{}
	for u := range m.teamMembers[convID] {
		out = append(out, u)
	}
	slices.Sort(out)
	return out, nil
}

func (m *Memory) IsTeamMember(_ context.Context, convID, userID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.teamMembers[convID][userID], nil
}

func (m *Memory) UserDMConversations(_ context.Context, userID string, limit int) ([]Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Conversation{}
	for _, cv := range m.convs {
		if cv.Kind == ConvDM && (cv.TeacherID == userID || cv.UserID == userID) {
			out = append(out, *cv)
		}
	}
	slices.SortFunc(out, func(a, b Conversation) int { return b.LastMessageAt.Compare(a.LastMessageAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) UserTeamConversations(_ context.Context, userID string, limit int) ([]Conversation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Conversation{}
	for id, members := range m.teamMembers {
		if members[userID] {
			if cv := m.convs[id]; cv != nil {
				out = append(out, *cv)
			}
		}
	}
	slices.SortFunc(out, func(a, b Conversation) int { return b.LastMessageAt.Compare(a.LastMessageAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
