package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ---- Хянагчийн самбар (admin.go) ----

func (c *ClickHouse) AdminUsers(ctx context.Context) ([]User, error) {
	out := []User{}
	err := c.query(ctx, "SELECT "+userCols+" FROM users FINAL ORDER BY created_at DESC", nil, func(r driver.Rows) error {
		u, deleted, err := scanUser(r)
		if err != nil {
			return err
		}
		if !deleted {
			out = append(out, *u)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(out))
	for i := range out {
		ids[i] = out[i].ID
	}
	cs, err := c.counters(ctx, "user", ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		out[i].ProfileViews = cs[out[i].ID].views
	}
	return out, nil
}

func (c *ClickHouse) AdminCourses(ctx context.Context) ([]Course, error) {
	return c.coursesWhere(ctx, "1 = 1", "ORDER BY created_at DESC")
}

func (c *ClickHouse) AdminBooks(ctx context.Context) ([]Book, error) {
	return c.booksWhere(ctx, "1 = 1", "ORDER BY created_at DESC")
}

func (c *ClickHouse) AdminOrders(ctx context.Context, since time.Time) ([]Order, error) {
	rows, err := c.ordersWhere(ctx, "created_at >= ?", "ORDER BY created_at DESC", since.UTC())
	if err != nil {
		return nil, err
	}
	out := make([]Order, len(rows))
	for i := range rows {
		out[i] = rows[i].Order
	}
	return out, nil
}

func (c *ClickHouse) AdminEnrollments(ctx context.Context) ([]Enrollment, error) {
	out := []Enrollment{}
	err := c.query(ctx, `SELECT user_id, course_id, teacher_id, created_at, false FROM enrollments FINAL
		UNION ALL SELECT user_id, course_id, teacher_id, created_at, true FROM lesson_access FINAL`, nil, func(r driver.Rows) error {
		var e Enrollment
		if err := r.Scan(&e.UserID, &e.CourseID, &e.TeacherID, &e.At, &e.Lesson); err != nil {
			return err
		}
		out = append(out, e)
		return nil
	})
	return out, err
}

func (c *ClickHouse) AddSiteHits(ctx context.Context, hs []SiteHit) error {
	if len(hs) == 0 {
		return nil
	}
	b, err := c.conn.PrepareBatch(ctx, "INSERT INTO site_hits (at, visitor_id, user_id, path, referrer, device)")
	if err != nil {
		return err
	}
	for _, h := range hs {
		if err := b.Append(h.At.UTC(), h.VisitorID, h.UserID, h.Path, h.Referrer, h.Device); err != nil {
			return err
		}
	}
	return b.Send()
}

func (c *ClickHouse) SiteTraffic(ctx context.Context, q TrafficQuery) (*SiteTraffic, error) {
	out := &SiteTraffic{Daily: []TrafficDay{}}
	from := q.Since
	for _, t := range []time.Time{q.Month, q.Week, q.Today} {
		if t.Before(from) {
			from = t
		}
	}
	var w [10]uint64
	err := c.query(ctx, `SELECT countIf(at >= ?), uniqExactIf(visitor_id, at >= ?), countIf(at >= ?), uniqExactIf(visitor_id, at >= ?),
		countIf(at >= ?), uniqExactIf(visitor_id, at >= ?), countIf(at >= ?), uniqExactIf(visitor_id, at >= ?),
		countIf(at >= ?), uniqExactIf(visitor_id, at >= ?) FROM site_hits WHERE at >= ?`,
		[]any{q.Online.UTC(), q.Online.UTC(), q.Today.UTC(), q.Today.UTC(), q.Week.UTC(), q.Week.UTC(), q.Month.UTC(), q.Month.UTC(),
			q.Since.UTC(), q.Since.UTC(), from.UTC()},
		func(r driver.Rows) error {
			return r.Scan(&w[0], &w[1], &w[2], &w[3], &w[4], &w[5], &w[6], &w[7], &w[8], &w[9])
		})
	if err != nil {
		return nil, err
	}
	win := func(i int) TrafficWindow { return TrafficWindow{Views: int64(w[i]), Unique: int64(w[i+1])} }
	out.Online, out.Today, out.Week, out.Month, out.Period = win(0), win(2), win(4), win(6), win(8)
	err = c.query(ctx, `SELECT toString(toDate(at + toIntervalSecond(?))) AS d, count(), uniqExact(visitor_id), uniqExactIf(user_id, user_id != '')
		FROM site_hits WHERE at >= ? GROUP BY d ORDER BY d`, []any{q.TZOffset, q.Since.UTC()}, func(r driver.Rows) error {
		var d TrafficDay
		var v, u, m uint64
		if err := r.Scan(&d.Day, &v, &u, &m); err != nil {
			return err
		}
		d.Views, d.Unique, d.Members = int64(v), int64(u), int64(m)
		out.Daily = append(out.Daily, d)
		return nil
	})
	if err != nil {
		return nil, err
	}
	top := func(col string, limit int) ([]NameCount, error) {
		res := []NameCount{}
		err := c.query(ctx, "SELECT "+col+" AS k, count() AS n FROM site_hits WHERE at >= ? GROUP BY k ORDER BY n DESC, k LIMIT ?",
			[]any{q.Since.UTC(), limit}, func(r driver.Rows) error {
				var nc NameCount
				var n uint64
				if err := r.Scan(&nc.Name, &n); err != nil {
					return err
				}
				nc.Count = int64(n)
				res = append(res, nc)
				return nil
			})
		return res, err
	}
	if out.Pages, err = top("path", 12); err != nil {
		return nil, err
	}
	if out.Referrers, err = top("referrer", 8); err != nil {
		return nil, err
	}
	if out.Devices, err = top("device", 3); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *ClickHouse) writeExpense(ctx context.Context, e *Expense, deleted bool) error {
	return c.insert(ctx, "admin_expenses", []string{"id", "date", "category", "amount", "note", "created_by", "created_at", "ver", "deleted"},
		e.ID, e.Date.UTC(), e.Category, e.Amount, e.Note, e.CreatedBy, e.CreatedAt.UTC(), ver(), deleted)
}

func (c *ClickHouse) SaveExpense(ctx context.Context, e *Expense) error {
	if e.ID == "" {
		e.ID, e.CreatedAt = NewID(), time.Now()
	}
	return c.writeExpense(ctx, e, false)
}

func (c *ClickHouse) expensesWhere(ctx context.Context, where string, args ...any) ([]Expense, error) {
	out := []Expense{}
	err := c.query(ctx, "SELECT id, date, category, amount, note, created_by, created_at FROM admin_expenses FINAL WHERE deleted = false AND "+where+
		" ORDER BY date DESC, created_at DESC", args, func(r driver.Rows) error {
		var e Expense
		if err := r.Scan(&e.ID, &e.Date, &e.Category, &e.Amount, &e.Note, &e.CreatedBy, &e.CreatedAt); err != nil {
			return err
		}
		out = append(out, e)
		return nil
	})
	return out, err
}

func (c *ClickHouse) DeleteExpense(ctx context.Context, id string) error {
	es, err := c.expensesWhere(ctx, "id = ?", id)
	if err != nil {
		return err
	}
	if len(es) == 0 {
		return ErrNotFound
	}
	return c.writeExpense(ctx, &es[0], true)
}

func (c *ClickHouse) Expenses(ctx context.Context, since time.Time) ([]Expense, error) {
	return c.expensesWhere(ctx, "date >= ?", since.UTC())
}
