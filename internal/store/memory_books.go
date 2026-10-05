package store

import (
	"context"
	"slices"
	"sync"
	"time"
)

type bookMem struct {
	mu      sync.RWMutex
	books   map[string]*Book
	access  map[[2]string]bool   // (user, book)
	pending map[[2]string]string // (user, book) -> order
	granted map[string]bool      // хэрэгжсэн захиалга
	events  []BookEvent
}

func (b *bookMem) init() {
	if b.books == nil {
		b.books, b.access, b.pending, b.granted = map[string]*Book{}, map[[2]string]bool{}, map[[2]string]string{}, map[string]bool{}
	}
}

// grant нь Memory.mu түгжээтэй үед дуудагдана (bookMem өөрийн түгжээтэй, эсрэг дараалал байхгүй).
func (b *bookMem) grant(userID, bookID, orderID string, amount int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.init()
	delete(b.pending, [2]string{userID, bookID})
	if b.granted[orderID] {
		return
	}
	b.granted[orderID] = true
	b.access[[2]string{userID, bookID}] = true
	if bk := b.books[bookID]; bk != nil {
		bk.Revenue += amount
	}
}

func (m *Memory) CreateBook(_ context.Context, bk *Book) error {
	b := &m.books
	b.mu.Lock()
	defer b.mu.Unlock()
	b.init()
	bk.ID, bk.CreatedAt, bk.UpdatedAt = m.next(), time.Now(), time.Now()
	c := *bk
	b.books[bk.ID] = &c
	return nil
}

func (m *Memory) UpdateBook(_ context.Context, bk *Book) error {
	b := &m.books
	b.mu.Lock()
	defer b.mu.Unlock()
	b.init()
	cur, ok := b.books[bk.ID]
	if !ok || cur.TeacherID != bk.TeacherID {
		return ErrNotFound
	}
	cur.Kind, cur.Title, cur.Author, cur.Description, cur.CoverURL = bk.Kind, bk.Title, bk.Author, bk.Description, bk.CoverURL
	cur.Price, cur.Published, cur.PreviewN, cur.UpdatedAt = bk.Price, bk.Published, bk.PreviewN, time.Now()
	*bk = *cur
	return nil
}

func (m *Memory) SetBookPages(_ context.Context, id, teacherID string, pages int) error {
	b := &m.books
	b.mu.Lock()
	defer b.mu.Unlock()
	b.init()
	cur, ok := b.books[id]
	if !ok || cur.TeacherID != teacherID {
		return ErrNotFound
	}
	cur.Pages, cur.UpdatedAt = pages, time.Now()
	return nil
}

func (m *Memory) DeleteBook(_ context.Context, id, teacherID string) error {
	b := &m.books
	b.mu.Lock()
	defer b.mu.Unlock()
	b.init()
	cur, ok := b.books[id]
	if !ok || cur.TeacherID != teacherID {
		return ErrNotFound
	}
	delete(b.books, id)
	return nil
}

func (m *Memory) BookByID(_ context.Context, id string) (*Book, error) {
	b := &m.books
	b.mu.RLock()
	defer b.mu.RUnlock()
	cur, ok := b.books[id]
	if !ok {
		return nil, ErrNotFound
	}
	c := *cur
	return &c, nil
}

func (m *Memory) BooksByTeacher(_ context.Context, teacherID string, onlyPublished bool) ([]Book, error) {
	b := &m.books
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := []Book{}
	for _, bk := range b.books {
		if bk.TeacherID == teacherID && (!onlyPublished || bk.Published) {
			out = append(out, *bk)
		}
	}
	slices.SortFunc(out, func(a, b Book) int { return b.CreatedAt.Compare(a.CreatedAt) })
	return out, nil
}

func (m *Memory) HasBookAccess(_ context.Context, userID, bookID string) (bool, error) {
	b := &m.books
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.access[[2]string{userID, bookID}], nil
}

func (m *Memory) CreateOrGetPendingBookOrder(_ context.Context, userID string, bk *Book) (*Order, error) {
	b := &m.books
	b.mu.Lock()
	b.init()
	k := [2]string{userID, bk.ID}
	id, ok := b.pending[k]
	b.mu.Unlock()
	m.mu.Lock()
	defer m.mu.Unlock()
	if ok {
		if o := m.orders[id]; o != nil && o.Status == OrderPending {
			o.Amount = bk.Price
			oc := *o
			return &oc, nil
		}
	}
	o := &Order{ID: m.next(), Kind: OrderKindBook, Title: bk.Title, UserID: userID, BookID: bk.ID, TeacherID: bk.TeacherID, Amount: bk.Price, Status: OrderPending, CreatedAt: time.Now()}
	m.orders[o.ID] = o
	b.mu.Lock()
	b.pending[k] = o.ID
	b.mu.Unlock()
	oc := *o
	return &oc, nil
}

func (m *Memory) AddBookEvent(_ context.Context, e BookEvent) error {
	b := &m.books
	b.mu.Lock()
	defer b.mu.Unlock()
	b.init()
	e.ID = m.next()
	if e.At.IsZero() {
		e.At = time.Now()
	}
	b.events = append(b.events, e)
	if bk := b.books[e.BookID]; bk != nil {
		switch bookCounter[e.Type] {
		case "views":
			bk.Views++
		case "previews":
			bk.Previews++
		case "reads":
			bk.Reads++
		case "interest":
			bk.Interest++
		case "sales":
			bk.Sales++
		}
	}
	return nil
}

func (m *Memory) BookEvents(_ context.Context, teacherID, bookID string, limit int) ([]BookEvent, error) {
	b := &m.books
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := []BookEvent{}
	for i := len(b.events) - 1; i >= 0 && (limit <= 0 || len(out) < limit); i-- {
		e := b.events[i]
		if e.TeacherID == teacherID && (bookID == "" || e.BookID == bookID) {
			out = append(out, e)
		}
	}
	return out, nil
}
