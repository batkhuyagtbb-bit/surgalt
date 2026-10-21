package store

import (
	"context"
	"slices"
)

// cloudMem — (хичээл, хэсэг, хэрэглэгч) → үгс (cloud.go).
type cloudMem map[[3]string][]string

func (m *Memory) SetCloudWords(_ context.Context, lessonID, blockID, userID string, words []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cloud == nil {
		m.cloud = cloudMem{}
	}
	m.cloud[[3]string{lessonID, blockID, userID}] = slices.Clone(words)
	return nil
}

func (m *Memory) CloudWords(_ context.Context, lessonID, blockID string) (map[string][]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string][]string{}
	for k, ws := range m.cloud {
		if k[0] == lessonID && k[1] == blockID {
			out[k[2]] = slices.Clone(ws)
		}
	}
	return out, nil
}
