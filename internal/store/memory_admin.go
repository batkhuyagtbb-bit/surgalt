package store

import (
	"context"
	"slices"
	"sort"
	"sync"
	"time"
)

// adminMem — сайтын хандалт ба зардлын бүртгэл (санах ойд; тэг утгаараа ашиглагдана).
type adminMem struct {
	mu       sync.RWMutex
	hits     []SiteHit
	expenses map[string]*Expense
}

func (m *Memory) AdminUsers(_ context.Context) ([]User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]User, 0, len(m.users))
	for _, u := range m.users {
		out = append(out, *u)
	}
	slices.SortFunc(out, func(a, b User) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (m *Memory) AdminCourses(_ context.Context) ([]Course, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Course, 0, len(m.courses))
	for _, c := range m.courses {
		out = append(out, m.courseCopyLocked(c))
	}
	slices.SortFunc(out, func(a, b Course) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (m *Memory) AdminBooks(_ context.Context) ([]Book, error) {
	b := &m.books
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]Book, 0, len(b.books))
	for _, bk := range b.books {
		out = append(out, *bk)
	}
	slices.SortFunc(out, func(a, b Book) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (m *Memory) AdminOrders(_ context.Context, since time.Time) ([]Order, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Order{}
	for _, o := range m.orders {
		if !o.CreatedAt.Before(since) {
			out = append(out, *o)
		}
	}
	slices.SortFunc(out, func(a, b Order) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (m *Memory) AdminEnrollments(_ context.Context) ([]Enrollment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := []Enrollment{}
	teacher := func(courseID string) string {
		if c := m.courses[courseID]; c != nil {
			return c.TeacherID
		}
		return ""
	}
	for k, at := range m.enrollments {
		out = append(out, Enrollment{UserID: k[0], CourseID: k[1], TeacherID: teacher(k[1]), At: at})
	}
	for k, courseID := range m.lessonAcc {
		out = append(out, Enrollment{UserID: k[0], CourseID: courseID, TeacherID: teacher(courseID), Lesson: true})
	}
	return out, nil
}

func (m *Memory) AddSiteHits(_ context.Context, hs []SiteHit) error {
	a := &m.admin
	a.mu.Lock()
	defer a.mu.Unlock()
	a.hits = append(a.hits, hs...)
	return nil
}

func (m *Memory) SiteTraffic(_ context.Context, q TrafficQuery) (*SiteTraffic, error) {
	a := &m.admin
	a.mu.RLock()
	defer a.mu.RUnlock()
	type set = map[string]bool
	win := func(from time.Time) TrafficWindow {
		u, n := set{}, int64(0)
		for _, h := range a.hits {
			if !h.At.Before(from) {
				n++
				u[h.VisitorID] = true
			}
		}
		return TrafficWindow{Views: n, Unique: int64(len(u))}
	}
	out := &SiteTraffic{Online: win(q.Online), Today: win(q.Today), Week: win(q.Week), Month: win(q.Month), Period: win(q.Since)}
	type day struct {
		views    int64
		uniq, mm set
	}
	days := map[string]*day{}
	pages, refs, devs := map[string]int64{}, map[string]int64{}, map[string]int64{}
	off := time.Duration(q.TZOffset) * time.Second
	for _, h := range a.hits {
		if h.At.Before(q.Since) {
			continue
		}
		k := h.At.UTC().Add(off).Format("2006-01-02")
		d := days[k]
		if d == nil {
			d = &day{uniq: set{}, mm: set{}}
			days[k] = d
		}
		d.views++
		d.uniq[h.VisitorID] = true
		if h.UserID != "" {
			d.mm[h.UserID] = true
		}
		pages[h.Path]++
		refs[h.Referrer]++
		devs[h.Device]++
	}
	for k, d := range days {
		out.Daily = append(out.Daily, TrafficDay{Day: k, Views: d.views, Unique: int64(len(d.uniq)), Members: int64(len(d.mm))})
	}
	sort.Slice(out.Daily, func(i, j int) bool { return out.Daily[i].Day < out.Daily[j].Day })
	out.Pages, out.Referrers, out.Devices = topCounts(pages, 12), topCounts(refs, 8), topCounts(devs, 3)
	return out, nil
}

// topCounts — тоогоор буурахаар эрэмбэлж эхний limit-ийг авна.
func topCounts(m map[string]int64, limit int) []NameCount {
	out := make([]NameCount, 0, len(m))
	for k, v := range m {
		out = append(out, NameCount{k, v})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Count > out[j].Count || (out[i].Count == out[j].Count && out[i].Name < out[j].Name)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func (m *Memory) SaveExpense(_ context.Context, e *Expense) error {
	a := &m.admin
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.expenses == nil {
		a.expenses = map[string]*Expense{}
	}
	if e.ID == "" {
		e.ID, e.CreatedAt = NewID(), time.Now()
	}
	cp := *e
	a.expenses[e.ID] = &cp
	return nil
}

func (m *Memory) DeleteExpense(_ context.Context, id string) error {
	a := &m.admin
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.expenses[id] == nil {
		return ErrNotFound
	}
	delete(a.expenses, id)
	return nil
}

func (m *Memory) Expenses(_ context.Context, since time.Time) ([]Expense, error) {
	a := &m.admin
	a.mu.RLock()
	defer a.mu.RUnlock()
	out := []Expense{}
	for _, e := range a.expenses {
		if !e.Date.Before(since) {
			out = append(out, *e)
		}
	}
	slices.SortFunc(out, func(a, b Expense) int {
		if c := b.Date.Compare(a.Date); c != 0 {
			return c
		}
		return b.CreatedAt.Compare(a.CreatedAt)
	})
	return out, nil
}
