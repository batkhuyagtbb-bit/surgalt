package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- чатын мэдүүлэг ба хориг (reports.go) ----

const reportCols = `id, conversation_id, conv_kind, teacher_id, message_id, reporter_key, reporter_id, reporter_name,
	target_key, target_id, target_name, reason, note, context, status, resolution, resolved_by, created_at, resolved_at`

func (c *ClickHouse) writeReport(ctx context.Context, r *Report) error {
	ctxJSON, err := json.Marshal(r.Context)
	if err != nil {
		return err
	}
	return c.insert(ctx, "chat_reports", []string{"id", "conversation_id", "conv_kind", "teacher_id", "message_id",
		"reporter_key", "reporter_id", "reporter_name", "target_key", "target_id", "target_name", "reason", "note", "context",
		"status", "resolution", "resolved_by", "created_at", "resolved_at", "ver"},
		r.ID, r.ConversationID, r.ConvKind, r.TeacherID, r.MessageID, r.ReporterKey, r.ReporterID, r.ReporterName,
		r.TargetKey, r.TargetID, r.TargetName, r.Reason, r.Note, string(ctxJSON), r.Status, r.Resolution, r.ResolvedBy,
		r.CreatedAt.UTC(), nullTime(r.ResolvedAt), ver())
}

func (c *ClickHouse) reportsWhere(ctx context.Context, where string, limit int, args ...any) ([]Report, error) {
	out := []Report{}
	err := c.query(ctx, "SELECT "+reportCols+" FROM chat_reports FINAL WHERE "+where+" ORDER BY created_at DESC LIMIT ?", append(args, limit),
		func(rows driver.Rows) error {
			var r Report
			var ctxJSON string
			if err := rows.Scan(&r.ID, &r.ConversationID, &r.ConvKind, &r.TeacherID, &r.MessageID, &r.ReporterKey, &r.ReporterID,
				&r.ReporterName, &r.TargetKey, &r.TargetID, &r.TargetName, &r.Reason, &r.Note, &ctxJSON, &r.Status, &r.Resolution,
				&r.ResolvedBy, &r.CreatedAt, &r.ResolvedAt); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(ctxJSON), &r.Context); err != nil || r.Context == nil {
				r.Context = []ReportMsg{}
			}
			out = append(out, r)
			return nil
		})
	return out, err
}

func (c *ClickHouse) CreateReport(ctx context.Context, r *Report) error {
	unlock, err := c.lock(ctx, "report:"+r.MessageID+"|"+r.ReporterKey)
	if err != nil {
		return err
	}
	defer unlock()
	n, err := c.count(ctx, "SELECT count() FROM chat_reports FINAL WHERE message_id = ? AND reporter_key = ?", r.MessageID, r.ReporterKey)
	if err != nil {
		return err
	}
	if n > 0 {
		return ErrConflict
	}
	r.ID, r.CreatedAt, r.Status = NewID(), time.Now(), ReportOpen
	return c.writeReport(ctx, r)
}

func (c *ClickHouse) Reports(ctx context.Context, status string, limit int) ([]Report, error) {
	if limit <= 0 {
		limit = 200
	}
	if status == "" {
		return c.reportsWhere(ctx, "1", limit)
	}
	return c.reportsWhere(ctx, "status = ?", limit, status)
}

func (c *ClickHouse) UpdateReport(ctx context.Context, id string, fn func(*Report) error) (*Report, error) {
	unlock, err := c.lock(ctx, "report:"+id)
	if err != nil {
		return nil, err
	}
	defer unlock()
	rs, err := c.reportsWhere(ctx, "id = ?", 1, id)
	if err != nil {
		return nil, err
	}
	if len(rs) == 0 {
		return nil, ErrNotFound
	}
	r := &rs[0]
	if err := fn(r); err != nil {
		return nil, err
	}
	return r, c.writeReport(ctx, r)
}

func (c *ClickHouse) SetConversationBlock(ctx context.Context, convID, side string, on bool) error {
	return c.insert(ctx, "conversation_blocks", []string{"conversation_id", "side", "deleted", "ver"}, convID, side, !on, ver())
}

func (c *ClickHouse) ConversationBlocks(ctx context.Context, convID string) (map[string]bool, error) {
	out := map[string]bool{}
	err := c.query(ctx, "SELECT side FROM conversation_blocks FINAL WHERE conversation_id = ? AND deleted = false", []any{convID},
		func(r driver.Rows) error {
			var side string
			if err := r.Scan(&side); err != nil {
				return err
			}
			out[side] = true
			return nil
		})
	return out, err
}
