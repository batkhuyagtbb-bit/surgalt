package store

import (
	"context"
	"slices"
)

// hlMem — суралцагчийн тодруулгууд: (хэрэглэгч, хичээл) → жагсаалт (highlights.go).
type hlMem map[[2]string][]Highlight

func (m *Memory) Highlights(_ context.Context, userID, lessonID string) ([]Highlight, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Highlight{}, m.hl[[2]string{userID, lessonID}]...), nil
}

func (m *Memory) SetHighlights(_ context.Context, userID, lessonID string, hs []Highlight) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hl == nil {
		m.hl = hlMem{}
	}
	k := [2]string{userID, lessonID}
	if len(hs) == 0 {
		delete(m.hl, k)
		return nil
	}
	m.hl[k] = slices.Clone(hs)
	return nil
}
