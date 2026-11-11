package store

import (
	"context"
	"slices"
	"time"
)

// ---- Багш хооронд хуваалцалт (санах ойн store) ----

func (m *Memory) CreateShare(_ context.Context, sh *Share) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	sh.ID, sh.CreatedAt = m.next(), time.Now()
	c := *sh
	m.shares = append(m.shares, &c)
	return nil
}

func (m *Memory) ShareByID(_ context.Context, id string) (*Share, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sh := range m.shares {
		if sh.ID == id {
			c := *sh
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (m *Memory) sharesWhere(keep func(*Share) bool) []Share {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Share{}
	for _, sh := range m.shares {
		if keep(sh) {
			out = append(out, *sh)
		}
	}
	slices.SortFunc(out, func(a, b Share) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out
}

func (m *Memory) SharesTo(_ context.Context, toID string) ([]Share, error) {
	return m.sharesWhere(func(sh *Share) bool { return sh.ToID == toID }), nil
}

func (m *Memory) SharesOfCourse(_ context.Context, courseID string) ([]Share, error) {
	return m.sharesWhere(func(sh *Share) bool { return sh.CourseID == courseID }), nil
}

func (m *Memory) DeleteShare(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, sh := range m.shares {
		if sh.ID == id {
			m.shares = append(m.shares[:i], m.shares[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}
