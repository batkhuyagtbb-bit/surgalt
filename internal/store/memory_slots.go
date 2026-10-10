package store

import (
	"context"
	"sort"
	"time"
)

func (m *Memory) AddSlots(_ context.Context, slots []*Slot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range slots {
		s.ID, s.CreatedAt = m.next(), time.Now()
		cp := *s
		m.slots[s.ID] = &cp
	}
	return nil
}

func (m *Memory) SlotByID(_ context.Context, id string) (*Slot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.slots[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func (m *Memory) slotsWhere(keep func(*Slot) bool) []Slot {
	out := []Slot{}
	for _, s := range m.slots {
		if keep(s) {
			out = append(out, *s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartsAt.Before(out[j].StartsAt) })
	return out
}

func (m *Memory) Slots(_ context.Context, teacherID string, from, to time.Time) ([]Slot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.slotsWhere(func(s *Slot) bool {
		return s.TeacherID == teacherID && !s.StartsAt.Before(from) && s.StartsAt.Before(to)
	}), nil
}

func (m *Memory) StudentSlots(_ context.Context, userID string, from time.Time) ([]Slot, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.slotsWhere(func(s *Slot) bool { return s.StudentID == userID && !s.StartsAt.Before(from) }), nil
}

func (m *Memory) UpdateSlot(_ context.Context, id string, fn func(*Slot) error) (*Slot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.slots[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *s
	if err := fn(&cp); err != nil {
		return nil, err
	}
	m.slots[id] = &cp
	out := cp
	return &out, nil
}

func (m *Memory) DeleteSlot(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.slots[id]; !ok {
		return ErrNotFound
	}
	delete(m.slots, id)
	return nil
}

func (m *Memory) CreateOrGetPendingSlotOrder(_ context.Context, userID string, s *Slot, title string) (*Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, o := range m.orders {
		if o.Kind == OrderKindSlot && o.UserID == userID && o.SlotID == s.ID && o.Status == OrderPending {
			o.Amount = s.Price
			oc := *o
			return &oc, nil
		}
	}
	o := &Order{ID: m.next(), Kind: OrderKindSlot, Title: title, UserID: userID, SlotID: s.ID, TeacherID: s.TeacherID,
		Amount: s.Price, Status: OrderPending, CreatedAt: time.Now()}
	m.orders[o.ID] = o
	oc := *o
	return &oc, nil
}

func (m *Memory) DeleteMeeting(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, mt := range m.meetings {
		if mt.ID == id {
			m.meetings = append(m.meetings[:i], m.meetings[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
