package store

import (
	"context"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func (c *ClickHouse) StudentRanks(ctx context.Context, warn []string) (map[string]*StudentRank, error) {
	out := map[string]*StudentRank{}
	at := func(id string) *StudentRank {
		r := out[id]
		if r == nil {
			r = &StudentRank{ID: id}
			out[id] = r
		}
		return r
	}
	// Сургалт, хичээлийн багш — өөрийн сургалт дахь үйлдлийг хасахад.
	courseT, lessonT := map[string]string{}, map[string]string{}
	if err := c.query(ctx, "SELECT id, teacher_id FROM courses FINAL", nil, func(r driver.Rows) error {
		var id, t string
		if err := r.Scan(&id, &t); err != nil {
			return err
		}
		courseT[id] = t
		return nil
	}); err != nil {
		return nil, err
	}
	if err := c.query(ctx, "SELECT id, course_id FROM lessons FINAL", nil, func(r driver.Rows) error {
		var id, cid string
		if err := r.Scan(&id, &cid); err != nil {
			return err
		}
		lessonT[id] = courseT[cid]
		return nil
	}); err != nil {
		return nil, err
	}
	// (хэрэглэгч, түлхүүр, тоо) мөрүүдийг уншиж, түлхүүрийн багш нь өөрөө биш бол add-аар нэмнэ.
	rows3 := func(q string, args []any, teacherOf func(string) string, add func(*StudentRank, int64)) error {
		return c.query(ctx, q, args, func(r driver.Rows) error {
			var uid, key string
			var n uint64
			if err := r.Scan(&uid, &key, &n); err != nil {
				return err
			}
			if uid != "" && teacherOf(key) != uid {
				add(at(uid), int64(n))
			}
			return nil
		})
	}
	self := func(t string) string { return t }
	byCourse := func(id string) string { return courseT[id] }
	byLesson := func(id string) string { return lessonT[id] }
	// Зөвхөн төлсөн эрхтэй (үзээгүй) мөр сургалтад суралцсанд тооцогдохгүй (LessonProgress.Viewed).
	if err := rows3("SELECT user_id, course_id, toUInt64(1) FROM lesson_progress FINAL WHERE "+progressViewedCond+" GROUP BY user_id, course_id", nil, byCourse,
		func(r *StudentRank, n int64) { r.Courses += n }); err != nil {
		return nil, err
	}
	pts := map[string][2]int64{}
	if err := rows3("SELECT user_id, course_id, toUInt64(points) FROM rank_points FINAL WHERE points > 0", nil, byCourse,
		func(r *StudentRank, n int64) { v := pts[r.ID]; pts[r.ID] = [2]int64{v[0] + n, v[1] + 1} }); err != nil {
		return nil, err
	}
	creative := func(r *StudentRank, n int64) { r.Creative += n }
	for _, q := range []struct {
		sql string
		of  func(string) string
	}{
		{"SELECT user_id, teacher_id, count() FROM reflections FINAL GROUP BY user_id, teacher_id", self},
		{"SELECT user_id, teacher_id, count() FROM submissions FINAL GROUP BY user_id, teacher_id", self},
		{"SELECT user_id, lesson_id, count() FROM cloud_answers FINAL GROUP BY user_id, lesson_id", byLesson},
		{"SELECT user_id, lesson_id, toUInt64(JSONLength(data)) FROM highlights FINAL WHERE data != ''", byLesson},
	} {
		if err := rows3(q.sql, nil, q.of, creative); err != nil {
			return nil, err
		}
	}
	// Хичээлийн идэвхтэй хугацаа: сесс бүрийн тоолуурт сануулгын төрөл байхгүй бол «сануулгагүй».
	if err := c.query(ctx, "SELECT user_id, teacher_id, active_sec, counts FROM study_sessions FINAL", nil, func(r driver.Rows) error {
		var uid, tid string
		var sec int32
		var counts map[string]int32
		if err := r.Scan(&uid, &tid, &sec, &counts); err != nil {
			return err
		}
		if uid == "" || uid == tid {
			return nil
		}
		s, clean := at(uid), true
		s.ActiveMin += int64(sec)
		for k, n := range counts {
			if n > 0 && isWarn(k, warn) {
				clean = false
			}
		}
		if clean {
			s.CleanMin += int64(sec)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if len(warn) > 0 {
		if err := rows3("SELECT user_id, teacher_id, count() FROM activity_events WHERE has(?, type) GROUP BY user_id, teacher_id", []any{warn}, self,
			func(r *StudentRank, n int64) { r.Warnings += n }); err != nil {
			return nil, err
		}
	}
	talk := func(r *StudentRank, n int64) { r.Talks += n }
	if err := rows3("SELECT user_id, teacher_id, count() FROM comments FINAL WHERE NOT deleted GROUP BY user_id, teacher_id", nil, self, talk); err != nil {
		return nil, err
	}
	if err := rows3("SELECT sender_id, teacher_id, count() FROM messages WHERE sender != 'teacher' AND sender_id != '' GROUP BY sender_id, teacher_id", nil, self, talk); err != nil {
		return nil, err
	}
	for u, v := range pts {
		at(u).Points = v[0] / v[1]
	}
	for _, r := range out {
		r.ActiveMin /= 60
		r.CleanMin /= 60
	}
	return out, nil
}
