package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- «үүлэн үг» хэсгийн хариултууд (cloud.go): (хичээл, хэсэг, хэрэглэгч) бүрт нэг мөр ----

func (c *ClickHouse) SetCloudWords(ctx context.Context, lessonID, blockID, userID string, words []string) error {
	if words == nil {
		words = []string{}
	}
	return c.insert(ctx, "cloud_answers", []string{"lesson_id", "block_id", "user_id", "words", "at", "ver"},
		lessonID, blockID, userID, words, time.Now().UTC(), ver())
}

func (c *ClickHouse) CloudWords(ctx context.Context, lessonID, blockID string) (map[string][]string, error) {
	out := map[string][]string{}
	err := c.query(ctx, "SELECT user_id, words FROM cloud_answers FINAL WHERE lesson_id = ? AND block_id = ?", []any{lessonID, blockID}, func(r driver.Rows) error {
		var uid string
		var ws []string
		if err := r.Scan(&uid, &ws); err != nil {
			return err
		}
		out[uid] = ws
		return nil
	})
	return out, err
}
