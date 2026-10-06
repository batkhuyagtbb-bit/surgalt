package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- хэлэлцүүлэг: сэтгэгдэл ----

const commentCols = `id, course_id, lesson_id, teacher_id, user_id, user_name, parent_id, body, created_at, deleted`

func (c *ClickHouse) writeComment(ctx context.Context, cm *Comment) error {
	return c.insert(ctx, "comments", []string{"id", "course_id", "lesson_id", "teacher_id", "user_id", "user_name", "parent_id", "body", "created_at", "deleted", "ver"},
		cm.ID, cm.CourseID, cm.LessonID, cm.TeacherID, cm.UserID, cm.UserName, cm.ParentID, cm.Body, cm.CreatedAt.UTC(), cm.Deleted, ver())
}

func (c *ClickHouse) commentsWhere(ctx context.Context, where, order string, args ...any) ([]Comment, error) {
	out := []Comment{}
	err := c.query(ctx, "SELECT "+commentCols+" FROM comments FINAL WHERE "+where+" "+order, args, func(r driver.Rows) error {
		var cm Comment
		if err := r.Scan(&cm.ID, &cm.CourseID, &cm.LessonID, &cm.TeacherID, &cm.UserID, &cm.UserName, &cm.ParentID, &cm.Body, &cm.CreatedAt, &cm.Deleted); err != nil {
			return err
		}
		if !cm.Deleted {
			out = append(out, cm)
		}
		return nil
	})
	return out, err
}

func (c *ClickHouse) AddComment(ctx context.Context, cm *Comment) error {
	cm.ID, cm.CreatedAt = NewID(), time.Now()
	return c.writeComment(ctx, cm)
}

func (c *ClickHouse) Comments(ctx context.Context, lessonID string, limit int) ([]Comment, error) {
	return c.commentsWhere(ctx, "lesson_id = ?", "ORDER BY id ASC"+limitClause(limit), lessonID)
}

func (c *ClickHouse) CommentByID(ctx context.Context, id string) (*Comment, error) {
	cs, err := c.commentsWhere(ctx, "id = ?", "LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	if len(cs) == 0 {
		return nil, ErrNotFound
	}
	return &cs[0], nil
}

func (c *ClickHouse) DeleteComment(ctx context.Context, id string) error {
	cm, err := c.CommentByID(ctx, id)
	if err != nil {
		return err
	}
	cm.Deleted = true
	return c.writeComment(ctx, cm)
}

func (c *ClickHouse) CommentCounts(ctx context.Context, lessonIDs []string) (map[string]int, error) {
	out := map[string]int{}
	if len(lessonIDs) == 0 {
		return out, nil
	}
	err := c.query(ctx, "SELECT lesson_id, toInt64(count()) FROM comments FINAL WHERE has(?, lesson_id) AND deleted = false GROUP BY lesson_id", []any{lessonIDs}, func(r driver.Rows) error {
		var id string
		var n int64
		if err := r.Scan(&id, &n); err != nil {
			return err
		}
		out[id] = int(n)
		return nil
	})
	return out, err
}

// ---- лайк ----

func (c *ClickHouse) ToggleLike(ctx context.Context, userID, target, targetID string) (bool, int, error) {
	unlock, err := c.lock(ctx, "like:"+target+":"+targetID+":"+userID)
	if err != nil {
		return false, 0, err
	}
	defer unlock()
	var cur bool
	_ = c.query(ctx, "SELECT liked FROM likes FINAL WHERE target = ? AND target_id = ? AND user_id = ? LIMIT 1", []any{target, targetID, userID},
		func(r driver.Rows) error { return r.Scan(&cur) })
	liked := !cur
	if err := c.insert(ctx, "likes", []string{"target", "target_id", "user_id", "liked", "ver"}, target, targetID, userID, liked, ver()); err != nil {
		return false, 0, err
	}
	n, err := c.count(ctx, "SELECT count() FROM likes FINAL WHERE target = ? AND target_id = ? AND liked = true", target, targetID)
	return liked, int(n), err
}

func (c *ClickHouse) Likes(ctx context.Context, userID, target string, targetIDs []string) (map[string]int, map[string]bool, error) {
	counts, mine := map[string]int{}, map[string]bool{}
	if len(targetIDs) == 0 {
		return counts, mine, nil
	}
	err := c.query(ctx, "SELECT target_id, toInt64(countIf(liked)), max(liked AND user_id = ?) FROM likes FINAL WHERE target = ? AND has(?, target_id) GROUP BY target_id",
		[]any{userID, target, targetIDs}, func(r driver.Rows) error {
			var id string
			var n int64
			var me bool
			if err := r.Scan(&id, &n, &me); err != nil {
				return err
			}
			counts[id], mine[id] = int(n), me
			return nil
		})
	return counts, mine, err
}
