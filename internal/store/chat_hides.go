package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ChatHideStore — «Чат устгах» (зөвхөн өөрт): хэрэглэгч яриаг жагсаалтаасаа устгасан мөчийг хадгална.
// Тэр мөчөөс өмнөх мессеж түүнд харагдахгүй; шинэ мессеж ирвэл яриа жагсаалтад дахин гарна (зөвхөн шинэ мессежтэй).
// Нөгөө тал (багш/суралцагч) яриагаа хэвээр харна.
type ChatHideStore interface {
	ChatHides(ctx context.Context, userID string) (map[string]time.Time, error)
	ChatHideAt(ctx context.Context, userID, conversationID string) (time.Time, error)
	HideChat(ctx context.Context, userID, conversationID string, at time.Time) error
}

func (c *ClickHouse) ChatHides(ctx context.Context, userID string) (map[string]time.Time, error) {
	out := map[string]time.Time{}
	err := c.query(ctx, "SELECT conversation_id, at FROM chat_hides FINAL WHERE user_id = ?", []any{userID},
		func(r driver.Rows) error {
			var id string
			var at time.Time
			if err := r.Scan(&id, &at); err != nil {
				return err
			}
			out[id] = at
			return nil
		})
	return out, err
}

func (c *ClickHouse) ChatHideAt(ctx context.Context, userID, conversationID string) (time.Time, error) {
	var at time.Time
	err := c.query(ctx, "SELECT at FROM chat_hides FINAL WHERE user_id = ? AND conversation_id = ? LIMIT 1", []any{userID, conversationID},
		func(r driver.Rows) error { return r.Scan(&at) })
	return at, err
}

func (c *ClickHouse) HideChat(ctx context.Context, userID, conversationID string, at time.Time) error {
	return c.insert(ctx, "chat_hides", []string{"user_id", "conversation_id", "at", "ver"}, userID, conversationID, at.UTC(), ver())
}

func (m *Memory) ChatHides(_ context.Context, userID string) (map[string]time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]time.Time{}
	for k, at := range m.chatHides {
		if k[0] == userID {
			out[k[1]] = at
		}
	}
	return out, nil
}

func (m *Memory) ChatHideAt(_ context.Context, userID, conversationID string) (time.Time, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.chatHides[[2]string{userID, conversationID}], nil
}

func (m *Memory) HideChat(_ context.Context, userID, conversationID string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.chatHides == nil {
		m.chatHides = map[[2]string]time.Time{}
	}
	m.chatHides[[2]string{userID, conversationID}] = at
	return nil
}
