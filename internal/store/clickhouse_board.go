package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- «Цагаан самбар» (board.go): (хичээл, хэсэг, өрөө) бүрт нэг мөр ----

func (c *ClickHouse) SaveBoardRooms(ctx context.Context, rows []BoardRoomData) error {
	if len(rows) == 0 {
		return nil
	}
	b, err := c.conn.PrepareBatch(ctx, "INSERT INTO board_rooms (lesson_id, block_id, room, data, updated_at, ver)")
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, r := range rows {
		if err := b.Append(r.LessonID, r.BlockID, r.Room, string(r.Data), now, ver()); err != nil {
			return err
		}
	}
	return b.Send()
}

func (c *ClickHouse) BoardRooms(ctx context.Context, lessonID, blockID string) (map[string][]byte, error) {
	out := map[string][]byte{}
	err := c.query(ctx, "SELECT room, data FROM board_rooms FINAL WHERE lesson_id = ? AND block_id = ?", []any{lessonID, blockID}, func(r driver.Rows) error {
		var room, data string
		if err := r.Scan(&room, &data); err != nil {
			return err
		}
		out[room] = []byte(data)
		return nil
	})
	return out, err
}
