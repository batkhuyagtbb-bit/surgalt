package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- Багш хооронд хуваалцалт (shares) ----

const shareCols = "id, from_id, to_id, course_id, lesson_id, mode, created_at"

func scanShare(r driver.Rows) (Share, error) {
	var sh Share
	err := r.Scan(&sh.ID, &sh.FromID, &sh.ToID, &sh.CourseID, &sh.LessonID, &sh.Mode, &sh.CreatedAt)
	return sh, err
}

func (c *ClickHouse) writeShare(ctx context.Context, sh *Share, deleted bool) error {
	return c.insert(ctx, "shares", []string{"id", "from_id", "to_id", "course_id", "lesson_id", "mode", "created_at", "ver", "deleted"},
		sh.ID, sh.FromID, sh.ToID, sh.CourseID, sh.LessonID, sh.Mode, sh.CreatedAt.UTC(), ver(), deleted)
}

func (c *ClickHouse) CreateShare(ctx context.Context, sh *Share) error {
	sh.ID, sh.CreatedAt = NewID(), time.Now()
	return c.writeShare(ctx, sh, false)
}

func (c *ClickHouse) shares(ctx context.Context, where string, args ...any) ([]Share, error) {
	out := []Share{}
	err := c.query(ctx, "SELECT "+shareCols+" FROM shares FINAL WHERE deleted = false AND "+where+" ORDER BY created_at DESC", args, func(r driver.Rows) error {
		sh, err := scanShare(r)
		if err == nil {
			out = append(out, sh)
		}
		return err
	})
	return out, err
}

func (c *ClickHouse) ShareByID(ctx context.Context, id string) (*Share, error) {
	list, err := c.shares(ctx, "id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNotFound
	}
	return &list[0], nil
}

func (c *ClickHouse) SharesTo(ctx context.Context, toID string) ([]Share, error) {
	return c.shares(ctx, "to_id = ?", toID)
}

func (c *ClickHouse) SharesOfCourse(ctx context.Context, courseID string) ([]Share, error) {
	return c.shares(ctx, "course_id = ?", courseID)
}

// DeleteShare — устгасан тэмдэгтэй мөр бичнэ (ReplacingMergeTree: хамгийн сүүлийн ver үлдэнэ).
func (c *ClickHouse) DeleteShare(ctx context.Context, id string) error {
	sh, err := c.ShareByID(ctx, id)
	if err != nil {
		return err
	}
	return c.writeShare(ctx, sh, true)
}
