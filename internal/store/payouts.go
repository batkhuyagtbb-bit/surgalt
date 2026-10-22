package store

import (
	"context"
	"math"
	"time"
)

// ---- Багшийн данс: орлого хүлээн авах банкны данс, татах хуваарь, татах хүсэлт ----
// Худалдан авагчийн төлбөр (QPay) платформын нэг merchant дансанд ордог. Багшийн үлдэгдэл = төлөгдсөн
// борлуулалт − (хүлээгдэж буй + шилжүүлсэн) татах хүсэлт. Хүсэлтийг хянагч банкаар шилжүүлээд «шилжүүлсэн» гэж тэмдэглэнэ;
// татгалзсан хүсэлтийн дүн үлдэгдэлд буцна.

const (
	PayoutManual    = "manual"    // зөвхөн багш өөрөө «Одоо татах» дарна
	PayoutInstant   = "instant"   // борлуулалт бүрийн дараа шууд
	PayoutDaily     = "daily"     // өдөрт нэг удаа
	PayoutWeekly    = "weekly"    // долоо хоногт нэг удаа
	PayoutMonthly   = "monthly"   // сард нэг удаа
	PayoutQuarterly = "quarterly" // улиралд нэг удаа

	PayoutPending  = "pending"  // шилжүүлэхийг хүлээж буй
	PayoutPaid     = "paid"     // данс руу шилжүүлсэн
	PayoutRejected = "rejected" // татгалзсан (дүн үлдэгдэлд буцна)
)

// PayoutAccount — багшийн орлого хүлээн авах данс ба автомат татах хуваарь.
type PayoutAccount struct {
	TeacherID   string `json:"-"`
	Bank        string `json:"bank"`
	AccountNo   string `json:"account_no"`
	AccountName string `json:"account_name"`
	Schedule    string `json:"schedule"`
	// ScheduleAt — хуваарийг сүүлд сольсон мөч: шинэ хуваарийн эхний автомат татах энэ мөчөөс хойш тооцогдоно.
	ScheduleAt time.Time `json:"schedule_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Payout — нэг татах хүсэлт; дансны мэдээллийг хүсэлт үүсгэх үеийнхээр хадгална (дараа нь данс сольсон ч).
type Payout struct {
	ID          string     `json:"id"`
	TeacherID   string     `json:"teacher_id"`
	Amount      int64      `json:"amount"`
	Bank        string     `json:"bank"`
	AccountNo   string     `json:"account_no"`
	AccountName string     `json:"account_name"`
	Status      string     `json:"status"`
	Auto        bool       `json:"auto"` // хуваарийн дагуу автоматаар үүссэн
	Note        string     `json:"note,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
	ProcessedBy string     `json:"processed_by,omitempty"`
}

// teacherCommission — багшийн борлуулалтаас платформын авах шимтгэл (%). main.go эхлэхэд тохируулна (анхдагч 0).
var teacherCommission float64

// SetTeacherCommission — шимтгэлийн хувийг (0–100) тохируулна; сервер эхлэхэд нэг удаа.
func SetTeacherCommission(pct float64) { teacherCommission = math.Min(100, math.Max(0, pct)) }

// TeacherCommission — одоогийн шимтгэлийн хувь.
func TeacherCommission() float64 { return teacherCommission }

// Commission — борлуулалтын дүнгээс авах платформын шимтгэл (бүхэл төгрөг рүү тоймлоно).
func Commission(gross int64) int64 {
	if teacherCommission <= 0 || gross <= 0 {
		return 0
	}
	return int64(math.Round(float64(gross) * teacherCommission / 100))
}

// Balance — багшийн орлогын үлдэгдэл.
type Balance struct {
	Earned int64 `json:"earned"` // нийт төлөгдсөн борлуулалт
	// Commission — борлуулалтаас платформын шимтгэл (CommissionPct %); Net = Earned − Commission (багшийн цэвэр орлого).
	Commission    int64   `json:"commission"`
	CommissionPct float64 `json:"commission_pct"`
	Net           int64   `json:"net"`
	Paid          int64   `json:"paid"`      // данс руу шилжүүлсэн
	Pending       int64   `json:"pending"`   // шилжүүлэхийг хүлээж буй
	Available     int64   `json:"available"` // одоо татах боломжтой
}

type PayoutStore interface {
	// PayoutAccount — багшийн данс; тохируулаагүй бол ErrNotFound.
	PayoutAccount(ctx context.Context, teacherID string) (*PayoutAccount, error)
	SetPayoutAccount(ctx context.Context, a *PayoutAccount) error
	// PayoutAccounts — бүх багшийн данс (автомат хуваарьт).
	PayoutAccounts(ctx context.Context) ([]PayoutAccount, error)
	// TeacherPaidOrders — багшийн төлөгдсөн борлуулалт (багтаамжийнхаас бусад), шинээс хуучин.
	TeacherPaidOrders(ctx context.Context, teacherID string) ([]Order, error)
	TeacherBalance(ctx context.Context, teacherID string) (*Balance, error)
	// RequestPayout — данс руу татах хүсэлт: amount <= 0 бол бүх үлдэгдэл. Данс тохируулаагүй бол ErrNotFound,
	// үлдэгдэл хүрэлцэхгүй (эсвэл 0) бол ErrConflict. Багш бүрт түгжээтэй тул давхар татахгүй.
	RequestPayout(ctx context.Context, teacherID string, amount int64, auto bool) (*Payout, error)
	// Payouts — teacherID хоосон бол бүх багшийн, status хоосон бол бүх төлөв; шинэ нь эхэндээ.
	Payouts(ctx context.Context, teacherID, status string, limit int) ([]Payout, error)
	// UpdatePayout — түгжээтэй уншиж fn-ээр өөрчилнө; fn алдаа буцаавал хадгалахгүй.
	UpdatePayout(ctx context.Context, id string, fn func(*Payout) error) (*Payout, error)
}

// earning — багшийн орлогод тооцогдох захиалга эсэх (файлын сангийн багтаамжийг багш өөрөө худалдаж авдаг).
func earning(o *Order, teacherID string) bool {
	return o.TeacherID == teacherID && o.Status == OrderPaid && o.Kind != OrderKindStorage
}

func balanceOf(orders []Order, payouts []Payout) *Balance {
	b := &Balance{}
	for _, o := range orders {
		b.Earned += o.Amount
	}
	for _, p := range payouts {
		switch p.Status {
		case PayoutPaid:
			b.Paid += p.Amount
		case PayoutPending:
			b.Pending += p.Amount
		}
	}
	b.Commission, b.CommissionPct = Commission(b.Earned), teacherCommission
	b.Net = b.Earned - b.Commission
	b.Available = max(0, b.Net-b.Paid-b.Pending) // шимтгэлийг хасаад л татна
	return b
}

// newPayout — үлдэгдлээс татах хүсэлтийг бэлдэнэ (хадгалахгүй).
func newPayout(a *PayoutAccount, bal *Balance, amount int64, auto bool) (*Payout, error) {
	if amount <= 0 {
		amount = bal.Available
	}
	if amount <= 0 || amount > bal.Available {
		return nil, ErrConflict
	}
	return &Payout{ID: NewID(), TeacherID: a.TeacherID, Amount: amount, Bank: a.Bank, AccountNo: a.AccountNo,
		AccountName: a.AccountName, Status: PayoutPending, Auto: auto, CreatedAt: time.Now()}, nil
}
