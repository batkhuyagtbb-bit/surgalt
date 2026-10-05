package store

import (
	"context"
	"time"
)

// Book — багшийн зардаг ном эсвэл өгүүлэл. Эх файл уншигчид хэзээ ч очдоггүй: хуудас бүр
// зураг болж хадгалагдаж, сервер уншигчийн тэмдэгтэйгээр нэг нэгээр нь өгнө.
type Book struct {
	ID          string    `json:"id" bson:"_id"`
	TeacherID   string    `json:"teacher_id" bson:"teacher_id"`
	Kind        string    `json:"kind" bson:"kind"` // book | article
	Title       string    `json:"title" bson:"title"`
	Author      string    `json:"author" bson:"author"`
	Description string    `json:"description" bson:"description"`
	CoverURL    string    `json:"cover_url" bson:"cover_url"`
	Price       int64     `json:"price" bson:"price"` // 0 = үнэгүй
	Published   bool      `json:"published" bson:"published"`
	Pages       int       `json:"pages" bson:"pages"`             // бэлтгэсэн хуудасны тоо
	PreviewN    int       `json:"preview_pages" bson:"preview_n"` // төлөөгүй хүнд харагдах хуудас
	Views       int64     `json:"views" bson:"views"`
	Previews    int64     `json:"previews" bson:"previews"`
	Reads       int64     `json:"reads" bson:"reads"`
	Interest    int64     `json:"interest" bson:"interest"`
	Sales       int64     `json:"sales" bson:"sales"`
	Revenue     int64     `json:"revenue" bson:"revenue"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

// BookEvent — номын лог: view (хуудас үзсэн), preview (үнэгүй хэсгийг уншсан), read (бүтэн уншиж эхэлсэн),
// interest (сонирхсон), paywall (төлбөрийн хананд хүрсэн), purchase (худалдаж авсан), limit (хэт хурдан татсан).
type BookEvent struct {
	ID        string    `json:"id" bson:"_id"`
	BookID    string    `json:"book_id" bson:"book_id"`
	TeacherID string    `json:"teacher_id" bson:"teacher_id"`
	UserID    string    `json:"user_id,omitempty" bson:"user_id,omitempty"`
	UserName  string    `json:"user_name" bson:"user_name"`
	Type      string    `json:"type" bson:"type"`
	Detail    string    `json:"detail,omitempty" bson:"detail,omitempty"`
	IP        string    `json:"ip,omitempty" bson:"ip,omitempty"`
	At        time.Time `json:"at" bson:"at"`
}

const OrderKindBook = "book" // ном, өгүүлэл

type BookStore interface {
	CreateBook(ctx context.Context, b *Book) error
	// UpdateBook нь зөвхөн b.TeacherID эзэмшдэг номыг шинэчилнэ (тоолуурт хүрэхгүй).
	UpdateBook(ctx context.Context, b *Book) error
	SetBookPages(ctx context.Context, id, teacherID string, pages int) error
	DeleteBook(ctx context.Context, id, teacherID string) error
	BookByID(ctx context.Context, id string) (*Book, error)
	BooksByTeacher(ctx context.Context, teacherID string, onlyPublished bool) ([]Book, error)
	HasBookAccess(ctx context.Context, userID, bookID string) (bool, error)
	CreateOrGetPendingBookOrder(ctx context.Context, userID string, b *Book) (*Order, error)
	// AddBookEvent нь логт бичиж, номын тоолуурыг (views, reads …) нэмнэ.
	AddBookEvent(ctx context.Context, e BookEvent) error
	BookEvents(ctx context.Context, teacherID, bookID string, limit int) ([]BookEvent, error)
}

// bookCounter — үйл явдлын төрөл → номын тоолуурын талбар.
var bookCounter = map[string]string{"view": "views", "preview": "previews", "read": "reads", "interest": "interest", "purchase": "sales"}
