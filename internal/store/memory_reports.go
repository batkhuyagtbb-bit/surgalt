package store

import (
	"context"
	"sort"
	"time"
)

func cloneReport(r *Report) *Report {
	cp := *r
	cp.Context = append([]ReportMsg(nil), r.Context...)
	return &cp
}

func (m *Memory) CreateReport(_ context.Context, r *Report) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, x := range m.reports {
		if x.MessageID == r.MessageID && x.ReporterKey == r.ReporterKey {
			return ErrConflict
		}
	}
	r.ID, r.CreatedAt, r.Status = m.next(), time.Now(), ReportOpen
	m.reports[r.ID] = cloneReport(r)
	return nil
}

func (m *Memory) Reports(_ context.Context, status string, limit int) ([]Report, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Report{}
	for _, r := range m.reports {
		if status == "" || r.Status == status {
			out = append(out, *cloneReport(r))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *Memory) UpdateReport(_ context.Context, id string, fn func(*Report) error) (*Report, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reports[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := cloneReport(r)
	if err := fn(cp); err != nil {
		return nil, err
	}
	m.reports[id] = cp
	return cloneReport(cp), nil
}

func (m *Memory) SetConversationBlock(_ context.Context, convID, side string, on bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if on {
		m.blocks[[2]string{convID, side}] = true
	} else {
		delete(m.blocks, [2]string{convID, side})
	}
	return nil
}

func (m *Memory) ConversationBlocks(_ context.Context, convID string) (map[string]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]bool{}
	for k := range m.blocks {
		if k[0] == convID {
			out[k[1]] = true
		}
	}
	return out, nil
}
