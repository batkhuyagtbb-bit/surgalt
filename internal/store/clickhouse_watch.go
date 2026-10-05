package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Олон app хуулбарт чат, мэдэгдлийг түгээх: ClickHouse өөрөө "bus" болно. Хуулбар бүр шинэ
// мөрүүдийг (id > сүүлд үзсэн id; ID цаг хугацаагаар өсдөг) богино хугацаанд санамж авч өөрийн
// WebSocket hub руу хүргэнэ. Нэмэлт брокер (Redis, Kafka) хэрэггүй.

// WatchMessages нь every тутамд шинэ мессежийг deliver-т өгнө (ctx дуустал).
func (c *ClickHouse) WatchMessages(ctx context.Context, every time.Duration, log *slog.Logger, deliver func(Message)) error {
	last, err := c.maxID(ctx, "messages")
	if err != nil {
		return err
	}
	go c.poll(ctx, every, log, func(ctx context.Context) error {
		return c.query(ctx, `SELECT id, conversation_id, teacher_id, visitor_key, sender, sender_id, sender_name, body, created_at
			FROM messages WHERE id > ? ORDER BY id LIMIT 500`, []any{last}, func(r driver.Rows) error {
			var m Message
			if err := r.Scan(&m.ID, &m.ConversationID, &m.TeacherID, &m.VisitorKey, &m.Sender, &m.SenderID, &m.SenderName, &m.Body, &m.CreatedAt); err != nil {
				return err
			}
			last = m.ID
			deliver(m)
			return nil
		})
	})
	return nil
}

// WatchNotifications нь шинэ мэдэгдлийг deliver-т өгнө.
func (c *ClickHouse) WatchNotifications(ctx context.Context, every time.Duration, log *slog.Logger, deliver func(Notification)) error {
	last, err := c.maxID(ctx, "notifications")
	if err != nil {
		return err
	}
	go c.poll(ctx, every, log, func(ctx context.Context) error {
		return c.query(ctx, `SELECT id, user_id, type, title, body, link, count, created_at
			FROM notifications WHERE id > ? ORDER BY id LIMIT 500`, []any{last}, func(r driver.Rows) error {
			var n Notification
			if err := r.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Link, &n.Count, &n.CreatedAt); err != nil {
				return err
			}
			last = n.ID
			deliver(n)
			return nil
		})
	})
	return nil
}

func (c *ClickHouse) maxID(ctx context.Context, table string) (string, error) {
	var id string
	err := c.query(ctx, "SELECT max(id) FROM "+table, nil, func(r driver.Rows) error { return r.Scan(&id) })
	return id, err
}

func (c *ClickHouse) poll(ctx context.Context, every time.Duration, log *slog.Logger, step func(context.Context) error) {
	wait, failing := every, false
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := step(sctx)
		cancel()
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			// ClickHouse түр тасарвал лог дүүргэхгүй: нэг удаа мэдэгдээд 5 сек хүртэл хойшлуулна.
			if !failing && log != nil {
				log.Warn("clickhouse watch: тасарлаа, дахин оролдоно", "err", err)
			}
			failing = true
			if wait = wait * 2; wait > 5*time.Second {
				wait = 5 * time.Second
			}
			continue
		}
		if failing && log != nil {
			log.Info("clickhouse watch: холболт сэргэлээ")
		}
		failing, wait = false, every
	}
}
