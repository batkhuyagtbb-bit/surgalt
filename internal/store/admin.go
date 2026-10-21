package store

import (
	"context"
	"time"
)

// ---- Хянагчийн самбар: платформын хэмжээний өгөгдөл (Go-д нэгтгэнэ), сайтын хандалт, зардлын бүртгэл ----

// SiteHit — сайтын аль нэг хуудсыг нэг удаа нээсэн нь (хөтчөөс илгээнэ: JS ажилласан жинхэнэ зочин).
type SiteHit struct {
	At        time.Time
	VisitorID string // "u:<хэрэглэгч>" | "a:<хөтчийн ID>" | "ip:<hmac>"
	UserID    string
	Path      string
	Referrer  string // refName-ээр ойлгомжтой болгосон эх сурвалж
	Device    string // mobile | tablet | desktop
}

// TrafficQuery — хандалтын статистикийн хугацааны хил (серверийн цагийн бүсээр тооцсон).
type TrafficQuery struct {
	Since    time.Time // өдөр бүрийн графикийн эхлэл
	Today    time.Time // өнөөдрийн 00:00
	Week     time.Time // сүүлийн 7 хоногийн эхлэл
	Month    time.Time // сүүлийн 30 хоногийн эхлэл
	Online   time.Time // «одоо сайтад» (сүүлийн хэдэн минут)
	TZOffset int       // UTC-ээс зөрөх секунд (өдрөөр бүлэглэхэд)
}

type TrafficDay struct {
	Day     string `json:"day"` // 2006-01-02
	Views   int64  `json:"views"`
	Unique  int64  `json:"unique"`
	Members int64  `json:"members"` // нэвтэрсэн давтагдаагүй хэрэглэгч
}

type TrafficWindow struct {
	Views  int64 `json:"views"`
	Unique int64 `json:"unique"`
}

type NameCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type SiteTraffic struct {
	Daily     []TrafficDay  `json:"daily"` // хоосон өдөр алга — handler нөхнө
	Online    TrafficWindow `json:"online"`
	Today     TrafficWindow `json:"today"`
	Week      TrafficWindow `json:"week"`
	Month     TrafficWindow `json:"month"`
	Period    TrafficWindow `json:"period"` // Since-ээс хойш
	Pages     []NameCount   `json:"pages"`
	Referrers []NameCount   `json:"referrers"`
	Devices   []NameCount   `json:"devices"`
}

// Enrollment — суралцагч сургалтад элссэн (Lesson=false) эсвэл нэг хичээл худалдаж авсан (Lesson=true).
type Enrollment struct {
	UserID    string
	CourseID  string
	TeacherID string
	At        time.Time
	Lesson    bool
}

// Expense — платформын зардал (сервер, сурталчилгаа, цалин г.м); хянагч гараар бүртгэнэ.
// Багшид шилжүүлсэн орлого (payouts) тусдаа тооцогдоно.
type Expense struct {
	ID        string    `json:"id"`
	Date      time.Time `json:"date"`
	Category  string    `json:"category"`
	Amount    int64     `json:"amount"`
	Note      string    `json:"note"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type AdminStore interface {
	// AdminUsers — устгагдаагүй бүх хэрэглэгч, шинэ нь эхэндээ.
	AdminUsers(ctx context.Context) ([]User, error)
	// AdminCourses — бүх сургалт (нийтлээгүй ч), шинэ нь эхэндээ.
	AdminCourses(ctx context.Context) ([]Course, error)
	AdminBooks(ctx context.Context) ([]Book, error)
	// AdminOrders — since-ээс хойш үүссэн бүх захиалга (бүх төлөв), шинэ нь эхэндээ.
	AdminOrders(ctx context.Context, since time.Time) ([]Order, error)
	AdminEnrollments(ctx context.Context) ([]Enrollment, error)

	AddSiteHits(ctx context.Context, hs []SiteHit) error
	SiteTraffic(ctx context.Context, q TrafficQuery) (*SiteTraffic, error)

	SaveExpense(ctx context.Context, e *Expense) error
	DeleteExpense(ctx context.Context, id string) error
	// Expenses — since-ээс хойших (Date) зардал, шинэ нь эхэндээ.
	Expenses(ctx context.Context, since time.Time) ([]Expense, error)
}
