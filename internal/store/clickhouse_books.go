package store

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const bookCols = `id, teacher_id, kind, title, author, description, cover_url, price, published, pages, preview_n,
	created_at, updated_at, deleted`

func scanBook(r driver.Rows) (Book, bool, error) {
	var b Book
	var deleted bool
	var pages, preview int32
	err := r.Scan(&b.ID, &b.TeacherID, &b.Kind, &b.Title, &b.Author, &b.Description, &b.CoverURL, &b.Price, &b.Published,
		&pages, &preview, &b.CreatedAt, &b.UpdatedAt, &deleted)
	b.Pages, b.PreviewN = int(pages), int(preview)
	return b, deleted, err
}

func (c *ClickHouse) writeBook(ctx context.Context, b *Book, deleted bool) error {
	return c.insert(ctx, "books", []string{"id", "teacher_id", "kind", "title", "author", "description", "cover_url", "price",
		"published", "pages", "preview_n", "created_at", "updated_at", "ver", "deleted"},
		b.ID, b.TeacherID, b.Kind, b.Title, b.Author, b.Description, b.CoverURL, b.Price, b.Published, int32(b.Pages),
		int32(b.PreviewN), b.CreatedAt.UTC(), b.UpdatedAt.UTC(), ver(), deleted)
}

// enrichBooks — тоолуурууд (үзэлт, уншилт, борлуулалт, орлого) counters-оос.
func (c *ClickHouse) enrichBooks(ctx context.Context, bs []Book) error {
	if len(bs) == 0 {
		return nil
	}
	ids := make([]string, len(bs))
	for i := range bs {
		ids[i] = bs[i].ID
	}
	cs, err := c.counters(ctx, "book", ids)
	if err != nil {
		return err
	}
	for i := range bs {
		cr := cs[bs[i].ID]
		bs[i].Views, bs[i].Previews, bs[i].Reads, bs[i].Interest, bs[i].Sales, bs[i].Revenue = cr.views, cr.previews, cr.reads, cr.interest, cr.sales, cr.revenue
	}
	return nil
}

func (c *ClickHouse) booksWhere(ctx context.Context, where, order string, args ...any) ([]Book, error) {
	out := []Book{}
	err := c.query(ctx, "SELECT "+bookCols+" FROM books FINAL WHERE "+where+" "+order, args, func(r driver.Rows) error {
		b, deleted, err := scanBook(r)
		if err != nil {
			return err
		}
		if !deleted {
			out = append(out, b)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, c.enrichBooks(ctx, out)
}

func (c *ClickHouse) CreateBook(ctx context.Context, b *Book) error {
	now := time.Now()
	b.ID, b.CreatedAt, b.UpdatedAt = NewID(), now, now
	return c.writeBook(ctx, b, false)
}

func (c *ClickHouse) ownedBook(ctx context.Context, id, teacherID string) (*Book, error) {
	b, err := c.BookByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b.TeacherID != teacherID {
		return nil, ErrNotFound
	}
	return b, nil
}

func (c *ClickHouse) UpdateBook(ctx context.Context, b *Book) error {
	unlock, err := c.lock(ctx, "book:"+b.ID)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.ownedBook(ctx, b.ID, b.TeacherID)
	if err != nil {
		return err
	}
	cur.Kind, cur.Title, cur.Author, cur.Description, cur.CoverURL = b.Kind, b.Title, b.Author, b.Description, b.CoverURL
	cur.Price, cur.Published, cur.PreviewN, cur.UpdatedAt = b.Price, b.Published, b.PreviewN, time.Now()
	if err := c.writeBook(ctx, cur, false); err != nil {
		return err
	}
	*b = *cur
	return nil
}

func (c *ClickHouse) SetBookPages(ctx context.Context, id, teacherID string, pages int) error {
	unlock, err := c.lock(ctx, "book:"+id)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.ownedBook(ctx, id, teacherID)
	if err != nil {
		return err
	}
	cur.Pages, cur.UpdatedAt = pages, time.Now()
	return c.writeBook(ctx, cur, false)
}

func (c *ClickHouse) DeleteBook(ctx context.Context, id, teacherID string) error {
	unlock, err := c.lock(ctx, "book:"+id)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := c.ownedBook(ctx, id, teacherID)
	if err != nil {
		return err
	}
	return c.writeBook(ctx, cur, true)
}

func (c *ClickHouse) BookByID(ctx context.Context, id string) (*Book, error) {
	bs, err := c.booksWhere(ctx, "id = ?", "LIMIT 1", id)
	if err != nil {
		return nil, err
	}
	if len(bs) == 0 {
		return nil, ErrNotFound
	}
	return &bs[0], nil
}

func (c *ClickHouse) BooksByTeacher(ctx context.Context, teacherID string, onlyPublished bool) ([]Book, error) {
	where := "teacher_id = ?"
	if onlyPublished {
		where += " AND published = true"
	}
	return c.booksWhere(ctx, where, "ORDER BY created_at DESC LIMIT 500", teacherID)
}

func (c *ClickHouse) HasBookAccess(ctx context.Context, userID, bookID string) (bool, error) {
	n, err := c.count(ctx, "SELECT count() FROM book_access FINAL WHERE user_id = ? AND book_id = ?", userID, bookID)
	return n > 0, err
}

// grantBook — захиалга төлөгдөхөд (MarkOrderPaid-ийн түгжээн дотор, applied тэмдгээр нэг л удаа).
func (c *ClickHouse) grantBook(ctx context.Context, userID, bookID string, amount int64) error {
	ok, err := c.HasBookAccess(ctx, userID, bookID)
	if err != nil || ok {
		return err
	}
	if err := c.insert(ctx, "book_access", []string{"user_id", "book_id", "created_at", "ver"}, userID, bookID, time.Now().UTC(), ver()); err != nil {
		return err
	}
	return c.counterAdd(ctx, "book", bookID, "revenue", amount)
}

func (c *ClickHouse) CreateOrGetPendingBookOrder(ctx context.Context, userID string, b *Book) (*Order, error) {
	return c.pendingOrder(ctx, userID+":b:"+b.ID, "user_id = ? AND book_id = ? AND kind = ?", []any{userID, b.ID, OrderKindBook},
		&Order{Kind: OrderKindBook, Title: b.Title, UserID: userID, BookID: b.ID, TeacherID: b.TeacherID, Amount: b.Price})
}

func (c *ClickHouse) AddBookEvent(ctx context.Context, e BookEvent) error {
	e.ID = NewID()
	if e.At.IsZero() {
		e.At = time.Now()
	}
	if err := c.insert(ctx, "book_events", []string{"id", "book_id", "teacher_id", "user_id", "user_name", "type", "detail", "ip", "at"},
		e.ID, e.BookID, e.TeacherID, e.UserID, e.UserName, e.Type, e.Detail, e.IP, e.At.UTC()); err != nil {
		return err
	}
	if col := bookCounter[e.Type]; col != "" {
		return c.counterAdd(ctx, "book", e.BookID, col, 1)
	}
	return nil
}

func (c *ClickHouse) BookEvents(ctx context.Context, teacherID, bookID string, limit int) ([]BookEvent, error) {
	where, args := "teacher_id = ?", []any{teacherID}
	if bookID != "" {
		where += " AND book_id = ?"
		args = append(args, bookID)
	}
	out := []BookEvent{}
	err := c.query(ctx, "SELECT id, book_id, teacher_id, user_id, user_name, type, detail, ip, at FROM book_events WHERE "+where+
		" ORDER BY at DESC, id DESC"+limitClause(limit), args, func(r driver.Rows) error {
		var e BookEvent
		if err := r.Scan(&e.ID, &e.BookID, &e.TeacherID, &e.UserID, &e.UserName, &e.Type, &e.Detail, &e.IP, &e.At); err != nil {
			return err
		}
		out = append(out, e)
		return nil
	})
	return out, err
}
