package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- суралцагчийн тодруулгууд (highlights.go): (хэрэглэгч, хичээл) бүрт нэг мөр, JSON ----

func (c *ClickHouse) Highlights(ctx context.Context, userID, lessonID string) ([]Highlight, error) {
	out := []Highlight{}
	err := c.query(ctx, "SELECT data FROM highlights FINAL WHERE user_id = ? AND lesson_id = ?", []any{userID, lessonID}, func(r driver.Rows) error {
		var data string
		if err := r.Scan(&data); err != nil {
			return err
		}
		return json.Unmarshal([]byte(data), &out)
	})
	return out, err
}

func (c *ClickHouse) UserHighlights(ctx context.Context, userID string) ([]HighlightSet, error) {
	out := []HighlightSet{}
	err := c.query(ctx, "SELECT lesson_id, data, updated_at FROM highlights FINAL WHERE user_id = ? ORDER BY updated_at DESC", []any{userID}, func(r driver.Rows) error {
		var hs HighlightSet
		var data string
		if err := r.Scan(&hs.LessonID, &data, &hs.UpdatedAt); err != nil {
			return err
		}
		if json.Unmarshal([]byte(data), &hs.Items) == nil && len(hs.Items) > 0 {
			out = append(out, hs)
		}
		return nil
	})
	return out, err
}

func (c *ClickHouse) SetHighlights(ctx context.Context, userID, lessonID string, hs []Highlight) error {
	if hs == nil {
		hs = []Highlight{}
	}
	data, err := json.Marshal(hs)
	if err != nil {
		return err
	}
	return c.insert(ctx, "highlights", []string{"user_id", "lesson_id", "data", "updated_at", "ver"},
		userID, lessonID, string(data), time.Now().UTC(), ver())
}
