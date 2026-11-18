package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// LessonGrantStore — дарааллын «багш гараар нээнэ» (UnlockRule "manual") хичээлийг багш тодорхой суралцагчид
// нээсэн эрх. Эрх тул журналын «бүрмөсөн устгах» (course_reset.go) үүнийг хөндөхгүй; бүртгэл устгахад (user_id)
// болон сургалт устгахад (course_id) устна.
type LessonGrantStore interface {
	// GrantLesson — суралцагчид тухайн хичээлийг нээнэ (давтан дуудвал хэвээр, сүүлийн багш, цаг хадгалагдана).
	GrantLesson(ctx context.Context, userID, courseID, lessonID, by string) error
	// LessonGrants — суралцагчид энэ сургалтад багш гараар нээсэн хичээлүүд (lessonID → true; байхгүй бол хоосон).
	LessonGrants(ctx context.Context, userID, courseID string) (map[string]bool, error)
}

func (c *ClickHouse) GrantLesson(ctx context.Context, userID, courseID, lessonID, by string) error {
	return c.insert(ctx, "lesson_grants", []string{"user_id", "course_id", "lesson_id", "granted_by", "at", "ver"},
		userID, courseID, lessonID, by, time.Now().UTC(), ver())
}

func (c *ClickHouse) LessonGrants(ctx context.Context, userID, courseID string) (map[string]bool, error) {
	out := map[string]bool{}
	err := c.query(ctx, "SELECT lesson_id FROM lesson_grants FINAL WHERE user_id = ? AND course_id = ?", []any{userID, courseID},
		func(r driver.Rows) error {
			var id string
			if err := r.Scan(&id); err != nil {
				return err
			}
			out[id] = true
			return nil
		})
	return out, err
}

func (m *Memory) GrantLesson(_ context.Context, userID, courseID, lessonID, by string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.grants == nil {
		m.grants = map[[2]string]map[string]string{}
	}
	k := [2]string{userID, courseID}
	if m.grants[k] == nil {
		m.grants[k] = map[string]string{}
	}
	m.grants[k][lessonID] = by
	return nil
}

func (m *Memory) LessonGrants(_ context.Context, userID, courseID string) (map[string]bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := map[string]bool{}
	for lid := range m.grants[[2]string{userID, courseID}] {
		out[lid] = true
	}
	return out, nil
}
