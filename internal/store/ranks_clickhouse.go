package store

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func (c *ClickHouse) TeacherRanks(ctx context.Context) (map[string]*TeacherRank, error) {
	out := map[string]*TeacherRank{}
	at := func(id string) *TeacherRank {
		r := out[id]
		if r == nil {
			r = &TeacherRank{ID: id}
			out[id] = r
		}
		return r
	}
	scan2 := func(q string, set func(*TeacherRank, int64)) error {
		return c.query(ctx, q, nil, func(rows driver.Rows) error {
			var id string
			var n uint64
			if err := rows.Scan(&id, &n); err != nil {
				return err
			}
			if id != "" {
				set(at(id), int64(n))
			}
			return nil
		})
	}
	if err := c.query(ctx, `SELECT id, toUInt64(greatest(sum(views), 0)) FROM counters WHERE entity = 'user' GROUP BY id`, nil, func(rows driver.Rows) error {
		var id string
		var n uint64
		if err := rows.Scan(&id, &n); err != nil {
			return err
		}
		at(id).Views = int64(n)
		return nil
	}); err != nil {
		return nil, err
	}
	if err := scan2(`SELECT teacher_id, count() FROM courses FINAL WHERE published AND NOT deleted GROUP BY teacher_id`,
		func(r *TeacherRank, n int64) { r.Courses = n }); err != nil {
		return nil, err
	}
	if err := c.query(ctx, `SELECT teacher_id, uniqExact(user_id), count() FROM enrollments FINAL GROUP BY teacher_id`, nil, func(rows driver.Rows) error {
		var id string
		var students, enrolls uint64
		if err := rows.Scan(&id, &students, &enrolls); err != nil {
			return err
		}
		if id != "" {
			r := at(id)
			r.Students, r.Enrolls = int64(students), int64(enrolls)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	// Харилцаа: багшид чатаар бичсэн хүн ба хичээлд нь сэтгэгдэл үлдээсэн хүн (давтагдаагүй).
	if err := scan2(`SELECT teacher_id, uniqExact(who) FROM (
			SELECT teacher_id, if(sender_id != '', sender_id, visitor_key) AS who FROM messages WHERE sender != 'teacher'
			UNION ALL
			SELECT teacher_id, user_id AS who FROM comments FINAL WHERE NOT deleted AND user_id != teacher_id
		) WHERE who != '' GROUP BY teacher_id`,
		func(r *TeacherRank, n int64) { r.Talks = n }); err != nil {
		return nil, err
	}
	return out, nil
}
