package store

import (
	"context"
	"slices"
	"time"
)

// hlMem — суралцагчийн тодруулгууд: (хэрэглэгч, хичээл) → жагсаалт (highlights.go).
type hlMem map[[2]string][]Highlight

func (m *Memory) Highlights(_ context.Context, userID, lessonID string) ([]Highlight, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Highlight{}, m.hl[[2]string{userID, lessonID}]...), nil
}

func (m *Memory) UserHighlights(_ context.Context, userID string) ([]HighlightSet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []HighlightSet{}
	for k, hs := range m.hl {
		if k[0] == userID && len(hs) > 0 {
			out = append(out, HighlightSet{LessonID: k[1], Items: slices.Clone(hs), UpdatedAt: m.hlAt[k]})
		}
	}
	slices.SortFunc(out, func(a, b HighlightSet) int { return b.UpdatedAt.Compare(a.UpdatedAt) })
	return out, nil
}

func (m *Memory) LessonHighlights(_ context.Context, lessonIDs []string) ([]HighlightSet, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	want := map[string]bool{}
	for _, id := range lessonIDs {
		want[id] = true
	}
	out := []HighlightSet{}
	for k, hs := range m.hl {
		if want[k[1]] && len(hs) > 0 {
			out = append(out, HighlightSet{UserID: k[0], LessonID: k[1], Items: slices.Clone(hs), UpdatedAt: m.hlAt[k]})
		}
	}
	return out, nil
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
	if m.hlAt == nil {
		m.hlAt = map[[2]string]time.Time{}
	}
	m.hlAt[k] = time.Now()
	return nil
}
