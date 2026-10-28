package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// CourseUnlockStore — нууц үгтэй сургалт: хэрэглэгч аль нууц үгээр (хэшээр) нээснийг хадгална.
// Багш нууц үгээ солибол хэш таарахгүй тул хүн бүр шинэ үгийг багшаас авч дахин оруулна.
type CourseUnlockStore interface {
	CourseUnlock(ctx context.Context, userID, courseID string) (string, error)
	SetCourseUnlock(ctx context.Context, userID, courseID, codeHash string) error
}

func (c *ClickHouse) CourseUnlock(ctx context.Context, userID, courseID string) (string, error) {
	var h string
	err := c.query(ctx, "SELECT code_hash FROM course_unlocks FINAL WHERE user_id = ? AND course_id = ? LIMIT 1", []any{userID, courseID},
		func(r driver.Rows) error { return r.Scan(&h) })
	return h, err
}

func (c *ClickHouse) SetCourseUnlock(ctx context.Context, userID, courseID, codeHash string) error {
	return c.insert(ctx, "course_unlocks", []string{"user_id", "course_id", "code_hash", "at", "ver"}, userID, courseID, codeHash, time.Now().UTC(), ver())
}

func (m *Memory) CourseUnlock(_ context.Context, userID, courseID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.unlocks[[2]string{userID, courseID}], nil
}

func (m *Memory) SetCourseUnlock(_ context.Context, userID, courseID, codeHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unlocks == nil {
		m.unlocks = map[[2]string]string{}
	}
	m.unlocks[[2]string{userID, courseID}] = codeHash
	return nil
}
