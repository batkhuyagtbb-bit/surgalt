package store

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// eraseByUser — хэрэглэгчийн хувийн мөрүүдийг бүрэн устгах хүснэгтүүд (багана → утга нь ID эсвэл "u:<ID>").
var eraseByUser = []struct{ table, col string }{
	{"activity_events", "user_id"}, {"study_sessions", "user_id"}, {"quiz_logs", "user_id"}, {"reflections", "user_id"},
	{"video_watches", "user_id"}, {"book_events", "user_id"}, {"lesson_progress", "user_id"}, {"exam_attempts", "user_id"},
	{"submissions", "user_id"}, {"comments", "user_id"}, {"likes", "user_id"}, {"rank_points", "user_id"}, {"rank_levels", "user_id"},
	{"notifications", "user_id"}, {"notification_reads", "user_id"}, {"enrollments", "user_id"}, {"lesson_access", "user_id"},
	{"book_access", "user_id"}, {"meeting_access", "user_id"}, {"identities", "user_id"}, {"conversation_members", "user_id"},
	{"page_visits", "visitor_id"},
}

func (c *ClickHouse) DeleteUserData(ctx context.Context, uid string) error {
	unlock, err := c.lock(ctx, "user:"+uid)
	if err != nil {
		return err
	}
	defer unlock()
	u, err := c.UserByID(ctx, uid)
	if err != nil {
		return err
	}
	key := "u:" + uid
	for _, t := range eraseByUser {
		if err := c.conn.Exec(ctx, "DELETE FROM "+t.table+" WHERE "+t.col+" = ?", uid); err != nil {
			return fmt.Errorf("erase %s: %w", t.table, err)
		}
	}
	for _, t := range []string{"message_reactions", "conversation_reads"} {
		if err := c.conn.Exec(ctx, "DELETE FROM "+t+" WHERE user_key = ?", key); err != nil {
			return fmt.Errorf("erase %s: %w", t, err)
		}
	}
	// Чат: өөрийн мессежүүдийг «устгагдсан» болгож (харагдах давхарга), хадгалсан нэр, агуулгыг арилгана.
	var ids []string
	var convs []string
	if err := c.query(ctx, "SELECT id, conversation_id FROM messages WHERE sender_id = ? OR (sender_id = '' AND sender = ? AND visitor_key = ?)",
		[]any{uid, SenderVisitor, key}, func(r driver.Rows) error {
			var id, conv string
			if err := r.Scan(&id, &conv); err != nil {
				return err
			}
			ids, convs = append(ids, id), append(convs, conv)
			return nil
		}); err != nil {
		return err
	}
	if len(ids) > 0 {
		b, err := c.conn.PrepareBatch(ctx, "INSERT INTO message_edits (message_id, conversation_id, body, deleted, edited_at, ver)")
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		for i := range ids {
			if err := b.Append(ids[i], convs[i], "", true, now, ver()); err != nil {
				return err
			}
		}
		if err := b.Send(); err != nil {
			return err
		}
	}
	type stmt struct {
		sql  string
		args []any
	}
	alters := []stmt{
		{"ALTER TABLE conversations UPDATE visitor_name = ?, last_message = '' WHERE user_id = ? AND kind = ''", []any{DeletedName, uid}},
		{"ALTER TABLE slots UPDATE student_name = ?, note = '' WHERE student_id = ?", []any{DeletedName, uid}},
	}
	if len(ids) > 0 { // хадгалсан нэр, агуулга, бусдын хариултад иш татсан хэсэг
		alters = append(alters,
			stmt{"ALTER TABLE messages UPDATE sender_name = ?, body = '', attachment = '', reply_body = '' WHERE has(?, id)", []any{DeletedName, ids}},
			stmt{"ALTER TABLE messages UPDATE reply_body = '', reply_name = ? WHERE has(?, reply_to)", []any{DeletedName, ids}})
	}
	sync := clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{"mutations_sync": 2}))
	for _, q := range alters {
		if err := c.conn.Exec(sync, q.sql, q.args...); err != nil {
			return fmt.Errorf("anonymize: %w", err)
		}
	}
	anonymize(u)
	return c.writeUser(ctx, u, true)
}
