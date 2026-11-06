package store

import (
	"context"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- Давтах сануулга (мартах муруй): хичээлийг анх үзсэнээс 1, 3, 7, 14, 30 хоногийн дараа суралцагчид ----

// ReviewItem — суралцагчийн давтах хичээл: анх үзсэн цаг (lesson_progress.viewed_at), үе шат (хоног).
type ReviewItem struct {
	UserID   string
	CourseID string
	LessonID string
	ViewedAt time.Time
	Stage    int
}

func (r ReviewItem) key() string { return r.UserID + "|" + r.LessonID + "|" + strconv.Itoa(r.Stage) }

type ReviewStore interface {
	// ReviewDue — from ≤ viewed_at < to хооронд анх үзсэн хичээлүүд (Stage-ийг дуудагч тавина).
	ReviewDue(ctx context.Context, from, to time.Time) ([]ReviewItem, error)
	// ClaimReviews — өмнө нь сануулаагүй (хэрэглэгч, хичээл, үе шат)-ыг тэмдэглээд буцаана. Түгжээтэй тул олон хуулбар,
	// дахин эхлүүлэхэд давхар сануулахгүй.
	ClaimReviews(ctx context.Context, items []ReviewItem) ([]ReviewItem, error)
}

func (c *ClickHouse) ReviewDue(ctx context.Context, from, to time.Time) ([]ReviewItem, error) {
	var out []ReviewItem
	err := c.query(ctx, "SELECT user_id, course_id, lesson_id, viewed_at FROM lesson_progress FINAL WHERE viewed_at >= ? AND viewed_at < ?",
		[]any{from.UTC(), to.UTC()}, func(r driver.Rows) error {
			var it ReviewItem
			if err := r.Scan(&it.UserID, &it.CourseID, &it.LessonID, &it.ViewedAt); err != nil {
				return err
			}
			out = append(out, it)
			return nil
		})
	return out, err
}

func (c *ClickHouse) ClaimReviews(ctx context.Context, items []ReviewItem) ([]ReviewItem, error) {
	if len(items) == 0 {
		return nil, nil
	}
	unlock, err := c.lock(ctx, "reviews")
	if err != nil {
		return nil, err
	}
	defer unlock()
	users, seen := []string{}, map[string]bool{}
	for _, it := range items {
		if !seen[it.UserID] {
			seen[it.UserID] = true
			users = append(users, it.UserID)
		}
	}
	sent := map[string]bool{}
	if err := c.query(ctx, "SELECT user_id, lesson_id, stage FROM review_reminders FINAL WHERE has(?, user_id)", []any{users},
		func(r driver.Rows) error {
			var it ReviewItem
			var st int32
			if err := r.Scan(&it.UserID, &it.LessonID, &st); err != nil {
				return err
			}
			it.Stage = int(st)
			sent[it.key()] = true
			return nil
		}); err != nil {
		return nil, err
	}
	fresh := freshReviews(items, sent)
	if len(fresh) == 0 {
		return nil, nil
	}
	b, err := c.conn.PrepareBatch(ctx, "INSERT INTO review_reminders (user_id, lesson_id, stage, course_id, sent_at, ver)")
	if err != nil {
		return nil, err
	}
	now, v := time.Now().UTC(), ver()
	for _, it := range fresh {
		if err := b.Append(it.UserID, it.LessonID, int32(it.Stage), it.CourseID, now, v); err != nil {
			return nil, err
		}
	}
	return fresh, b.Send()
}

// freshReviews — sent-д байхгүй (мөн жагсаалтад давхардаагүй) зүйлс; sent-ийг шинэчилнэ.
func freshReviews(items []ReviewItem, sent map[string]bool) []ReviewItem {
	var out []ReviewItem
	for _, it := range items {
		if k := it.key(); !sent[k] {
			sent[k] = true
			out = append(out, it)
		}
	}
	return out
}

func (m *Memory) ReviewDue(_ context.Context, from, to time.Time) ([]ReviewItem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	courseOf := map[string]string{}
	for cid, ls := range m.lessons {
		for _, l := range ls {
			courseOf[l.ID] = cid
		}
	}
	var out []ReviewItem
	for k, p := range m.progress {
		if !p.ViewedAt.Before(from) && p.ViewedAt.Before(to) {
			out = append(out, ReviewItem{UserID: k[0], CourseID: courseOf[k[1]], LessonID: k[1], ViewedAt: p.ViewedAt})
		}
	}
	return out, nil
}

func (m *Memory) ClaimReviews(_ context.Context, items []ReviewItem) ([]ReviewItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.reviews == nil {
		m.reviews = map[string]bool{}
	}
	return freshReviews(items, m.reviews), nil
}
