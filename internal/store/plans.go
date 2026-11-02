package store

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"time"
)

// ---- Багшийн системийн эрх (багц) ----
// Шинэ багшид 3 сар үнэгүй (борлуулалтын шимтгэл 21%). Дараа нь профайлын «Данс» хэсгээс QPay-ээр багц авч сунгана.
// Эрх дууссан багш сургалтаа удирдах, шинээр борлуулах боломжгүй (худалдаж авсан суралцагчдын эрх хэвээр).

// Plan — худалдаж авах багц: хугацаа (сар), үнэ, борлуулалтаас авах платформын шимтгэл (%).
type Plan struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Months int     `json:"months"`
	Price  int64   `json:"price"`
	Pct    float64 `json:"commission_pct"`
}

// Plans — үнийн жагсаалт (хямдаас үнэтэй рүү).
var Plans = []Plan{
	{ID: "start", Name: "Эхлэл", Months: 3, Price: 69000, Pct: 20},
	{ID: "standard", Name: "Standart", Months: 6, Price: 99000, Pct: 16},
	{ID: "major", Name: "Major", Months: 12, Price: 189000, Pct: 12},
	{ID: "pro", Name: "Pro", Months: 12, Price: 500000, Pct: 8},
}

// Үнэгүй хугацааны (болон эрхгүй үеийн) шимтгэл = TEACHER_COMMISSION_PERCENT (анхдагч 21%).
const (
	PlanTrial       = "trial" // үнэгүй хугацаа
	PlanTrialMonths = 3
)

// PlanTrialFrom — журам эхэлсэн өдөр: үүнээс өмнө бүртгүүлсэн багшийн үнэгүй 3 сар энэ өдрөөс тоологдоно.
var PlanTrialFrom = time.Date(2026, 11, 3, 0, 0, 0, 0, time.FixedZone("ULAT", 8*3600))

func PlanByID(id string) (Plan, bool) {
	for _, p := range Plans {
		if p.ID == id {
			return p, true
		}
	}
	return Plan{}, false
}

// PlanTerm — «3 сар», «1 жил».
func PlanTerm(months int) string {
	if months%12 == 0 {
		return strconv.Itoa(months/12) + " жил"
	}
	return strconv.Itoa(months) + " сар"
}

// PlanPeriod — нэг багцын хугацаа [From, Until).
type PlanPeriod struct {
	Plan  string    `json:"plan"`
	From  time.Time `json:"from"`
	Until time.Time `json:"until"`
	Order string    `json:"order,omitempty"` // захиалга (хянагч олгосон бол хоосон)
}

// PlanState — users.plan баганад JSON-оор хадгална.
type PlanState struct {
	TrialFrom *time.Time   `json:"trial_from,omitempty"` // багш болсон өдөр (хоосон бол бүртгүүлсэн өдөр)
	Periods   []PlanPeriod `json:"periods,omitempty"`
	Notice    string       `json:"notice,omitempty"` // сүүлд илгээсэн сануулга (давтахгүй)
}

func (p PlanState) encode() string {
	if p.TrialFrom == nil && len(p.Periods) == 0 && p.Notice == "" {
		return ""
	}
	b, _ := json.Marshal(p)
	return string(b)
}

func decodePlan(s string) PlanState {
	var p PlanState
	if s != "" {
		_ = json.Unmarshal([]byte(s), &p)
	}
	return p
}

// TrialUntil — үнэгүй хугацааны төгсгөл.
func (u *User) TrialUntil() time.Time {
	from := u.CreatedAt
	if u.Plan.TrialFrom != nil {
		from = *u.Plan.TrialFrom
	}
	if from.Before(PlanTrialFrom) {
		from = PlanTrialFrom
	}
	return from.AddDate(0, PlanTrialMonths, 0)
}

// PlanAt — t үеийн эрх: багцын ID (эсвэл PlanTrial), шимтгэлийн хувь, эрхтэй эсэх.
func (u *User) PlanAt(t time.Time) (string, float64, bool) {
	for _, p := range u.Plan.Periods {
		if !t.Before(p.From) && t.Before(p.Until) {
			if pl, ok := PlanByID(p.Plan); ok {
				return pl.ID, pl.Pct, true
			}
		}
	}
	if t.Before(u.TrialUntil()) {
		return PlanTrial, teacherCommission, true
	}
	return "", teacherCommission, false
}

// PlanUntil — эрх дуусах хугацаа (үнэгүй хугацаа, дараалсан бүх багцыг оруулаад). Ирээдүйд завсар үүсдэггүй.
func (u *User) PlanUntil() time.Time {
	end := u.TrialUntil()
	for _, p := range u.Plan.Periods {
		if p.Until.After(end) {
			end = p.Until
		}
	}
	return end
}

// PlanActive — системийн эрхтэй эсэх (зөвхөн багшид хамаарна).
func (u *User) PlanActive(now time.Time) bool {
	return u.Role != RoleTeacher || now.Before(u.PlanUntil())
}

// AddPlan — багц нэмнэ:
//   - эрхгүй бол одооноос эхэлнэ;
//   - одоогийнхоос шимтгэл багатай (дээд) багц бол шууд эхэлж, одоогийн багцын үлдэгдэл араас нь үргэлжилнэ
//     (үнэгүй хугацааны үлдэгдэл шинэ багцад нэмэгдэнэ — эрт авсан хүн хохирохгүй);
//   - ижил эсвэл доод багц бол одоогийн эрх дуусахад эхэлнэ.
func (u *User) AddPlan(pl Plan, order string, now time.Time) PlanPeriod {
	cur, pct, ok := u.PlanAt(now)
	u.Plan.Periods = append([]PlanPeriod(nil), u.Plan.Periods...) // хуулбар дээр (дуудагчийн slice-ийг өөрчлөхгүй)
	var np PlanPeriod
	switch {
	case !ok:
		np = PlanPeriod{Plan: pl.ID, From: now, Until: now.AddDate(0, pl.Months, 0), Order: order}
		u.Plan.Periods = append(u.Plan.Periods, np)
	case pl.Pct < pct:
		end := now.AddDate(0, pl.Months, 0)
		if cur == PlanTrial {
			end = end.Add(u.TrialUntil().Sub(now))
		}
		shift := end.Sub(now)
		out := make([]PlanPeriod, 0, len(u.Plan.Periods)+2)
		for _, p := range u.Plan.Periods {
			switch {
			case !p.Until.After(now):
				out = append(out, p)
			case p.From.Before(now): // одоогийнх: өнгөрсөн хэсэг үлдэж, үлдэгдэл нь шинэ багцын дараа
				out = append(out, PlanPeriod{Plan: p.Plan, From: p.From, Until: now, Order: p.Order},
					PlanPeriod{Plan: p.Plan, From: end, Until: p.Until.Add(shift), Order: p.Order})
			default:
				out = append(out, PlanPeriod{Plan: p.Plan, From: p.From.Add(shift), Until: p.Until.Add(shift), Order: p.Order})
			}
		}
		np = PlanPeriod{Plan: pl.ID, From: now, Until: end, Order: order}
		u.Plan.Periods = append(out, np)
	default:
		start := u.PlanUntil()
		np = PlanPeriod{Plan: pl.ID, From: start, Until: start.AddDate(0, pl.Months, 0), Order: order}
		u.Plan.Periods = append(u.Plan.Periods, np)
	}
	sort.Slice(u.Plan.Periods, func(i, j int) bool { return u.Plan.Periods[i].From.Before(u.Plan.Periods[j].From) })
	// 2 жилээс хуучин, хоосон хугацааг хаяна.
	keep := u.Plan.Periods[:0]
	for _, p := range u.Plan.Periods {
		if p.Until.After(p.From) && p.Until.After(now.AddDate(-2, 0, 0)) {
			keep = append(keep, p)
		}
	}
	u.Plan.Periods = keep
	u.Plan.Notice = ""
	return np
}

// applyPlanOrder — төлөгдсөн багцын захиалгыг хэрэглэгчид хэрэгжүүлнэ.
func applyPlanOrder(u *User, o *Order, now time.Time) {
	if pl, ok := PlanByID(o.Plan); ok {
		u.AddPlan(pl, o.ID, now)
	}
}

// ---- Борлуулалтын шимтгэл (захиалга бүрээр) ----

// Platform — платформын өөрийн борлуулалт (багшийн орлогод орохгүй): файлын сан, системийн эрх.
func (o *Order) Platform() bool { return o.Kind == OrderKindStorage || o.Kind == OrderKindPlan }

// CommissionRate — борлуулалтын шимтгэлийн хувь: төлөгдөх үеийн багцын хувь; хуучин захиалгад TEACHER_COMMISSION_PERCENT.
func (o *Order) CommissionRate() float64 {
	if o.CommissionPct > 0 {
		return o.CommissionPct
	}
	return teacherCommission
}

// OrderCommission — нэг борлуулалтаас авах платформын шимтгэл (бүхэл төгрөг рүү тоймлоно).
func OrderCommission(o *Order) int64 {
	if o.Amount <= 0 || o.Platform() {
		return 0
	}
	return int64(math.Round(float64(o.Amount) * o.CommissionRate() / 100))
}

// saleCommissionPct — багшийн одоогийн шимтгэл (эрхгүй бол үнэгүй хугацааных).
func saleCommissionPct(teacher *User, now time.Time) float64 {
	if teacher == nil {
		return teacherCommission
	}
	_, pct, _ := teacher.PlanAt(now)
	return pct
}
