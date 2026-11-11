package store

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"time"
)

// ---- Багшийн системийн эрх (багц) ----
// Шинэ багш, байгууллагад 2 сар үнэгүй (багшид файлын сан STORAGE_FREE_MB, анхдагч 100MB). Дараа нь профайлын «Данс»
// хэсгээс QPay-ээр нэг профайлын багц (сар, хагас жил, бүтэн жил; байгууллагад OrgPlans) авч сунгана. Борлуулалтаас
// хувь авахгүй — зөвхөн гүйлгээний шимтгэл (багшид TransactionFeePct, байгууллагад Org.Pct) хасагдаж, үлдсэн нь данс руу.
// Эрх дууссан багш сургалтаа удирдах, шинээр борлуулах боломжгүй (худалдаж авсан суралцагчдын эрх хэвээр).

// Plan — худалдаж авах багц: хугацаа (сар), үнэ, борлуулалтаас авах шимтгэл (%).
type Plan struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`       // Бүтэн жил
	Label     string  `json:"label"`      // нэмэлт тайлбар (хуучин багцад: Эхлэл)
	StorageMB int64   `json:"storage_mb"` // багцад багтсан файлын сан (үнэгүй хэмжээнээс их бол түүнийг орлоно)
	Months    int     `json:"months"`
	Price     int64   `json:"price"`
	Pct       float64 `json:"commission_pct"`
}

// Title — «Бүтэн жил», хуучин багцад «Starter (Эхлэл)»: захиалга, мэдэгдэлд.
func (p Plan) Title() string {
	if p.Label == "" {
		return p.Name
	}
	return p.Name + " (" + p.Label + ")"
}

// TransactionFeePct — гүйлгээний шимтгэл (%): багшийн борлуулалт бүрээс хасагдана (үнэгүй хугацаанд ч). Хувь авахгүй.
const TransactionFeePct = 2.0

// Plans — худалдаалж буй багцууд (хямдаас үнэтэй рүү). ID-г хадгалсан эрх, захиалгад ашигладаг тул солихгүй.
var Plans = []Plan{
	{ID: "month", Name: "Сарын", StorageMB: 300, Months: 1, Price: 29900, Pct: TransactionFeePct},
	{ID: "half", Name: "Хагас жил", StorageMB: 500, Months: 6, Price: 99000, Pct: TransactionFeePct},
	{ID: "year", Name: "Бүтэн жил", StorageMB: 1024, Months: 12, Price: 159000, Pct: TransactionFeePct},
}

// OrgFeePct — байгууллагын гүйлгээний шимтгэлийн анхдагч (%): хянагч бүртгэхэд Org.Pct-д тавина. Хянагч өөр хувь
// тогтоовол гэрээт байгууллага (Org.Contract) болно.
const OrgFeePct = 1.0

// Contract — гэрээт байгууллага: хянагч анхдагчаас (OrgFeePct) өөр шимтгэл тогтоосон — багц авахгүй, эрх хугацаагүй,
// нөхцөлийг хянагч гэрээгээр тогтооно.
func (o *Org) Contract() bool { return o != nil && o.Pct != OrgFeePct }

// OrgPlans — байгууллагын багцууд (байгууллагын профайлд; холбосон багш, удирдагчид нэмэлт төлбөргүй).
var OrgPlans = []Plan{
	{ID: "org_month", Name: "Сарын", Label: "Байгууллага", StorageMB: 2048, Months: 1, Price: 129000, Pct: OrgFeePct},
	{ID: "org_half", Name: "Хагас жил", Label: "Байгууллага", StorageMB: 5120, Months: 6, Price: 500000, Pct: OrgFeePct},
	{ID: "org_year", Name: "Бүтэн жил", Label: "Байгууллага", StorageMB: 5120, Months: 12, Price: 1000000, Pct: OrgFeePct},
}

// legacyPlans — 2026.11 хүртэлх багцууд: худалдаалахгүй, хадгалсан эрх, төлөгдөөгүй захиалгыг таних л зорилготой.
// Хувь авахгүй журмаар шимтгэл нь гүйлгээний шимтгэлтэй тэнцүү.
var legacyPlans = []Plan{
	{ID: "start", Name: "Starter", Label: "Эхлэл", StorageMB: 300, Months: 3, Price: 69000, Pct: TransactionFeePct},
	{ID: "standard", Name: "Standard", Label: "Стандарт", StorageMB: 500, Months: 6, Price: 99000, Pct: TransactionFeePct},
	{ID: "major", Name: "Advanced", Label: "Ахисан түвшин", StorageMB: 1024, Months: 12, Price: 189000, Pct: TransactionFeePct},
	{ID: "pro", Name: "Enterprise", Label: "Бизнес / Энтерпрайз", StorageMB: 2048, Months: 12, Price: 500000, Pct: TransactionFeePct},
}

// PlanUnlimited — хянагчийн олгодог хязгааргүй эрх: хугацаагүй (ContractUntil), файлын сан UnlimitedMB. Худалдаалахгүй.
const PlanUnlimited = "unlimited"

// UnlimitedMB — «хязгааргүй» файлын сан (1TB): quota-гийн тооцоонд энэ хэмжээ, харагдацад «хязгааргүй».
const UnlimitedMB int64 = 1 << 20

// adminPlans — зөвхөн хянагч олгодог багцууд (PlanByID таньдаг, худалдаанд байхгүй).
var adminPlans = []Plan{
	{ID: PlanUnlimited, Name: "Хязгааргүй", Label: "Хянагч олгосон", StorageMB: UnlimitedMB, Months: 0, Pct: TransactionFeePct},
}

// Until — from-оос эхэлсэн багцын төгсгөл; хугацаагүй (Months 0) багцад ContractUntil.
func (p Plan) Until(from time.Time) time.Time {
	if p.Months <= 0 {
		return ContractUntil
	}
	return from.AddDate(0, p.Months, 0)
}

// Үнэгүй хугацааны (болон эрхгүй үеийн) шимтгэл = TransactionFeePct. Хуучин захиалга (commission_pct 0) л
// TEACHER_COMMISSION_PERCENT-ээр тооцогдоно.
const (
	PlanTrial       = "trial"    // үнэгүй хугацаа (багш, байгууллага)
	PlanContract    = "contract" // гэрээт байгууллага (Org.Contract): хугацаагүй
	PlanTrialMonths = 2          // PlanTrialShortFrom-оос өмнө бүртгүүлсэн хүнд planTrialMonthsBefore (амласан 3 сар)
)

func (o *Org) encode() string {
	if o == nil {
		return ""
	}
	b, _ := json.Marshal(o)
	return string(b)
}

func decodeOrg(s string) *Org {
	if s == "" {
		return nil
	}
	var o Org
	if json.Unmarshal([]byte(s), &o) != nil {
		return nil
	}
	if o.Programs == nil {
		o.Programs = []Program{}
	}
	return &o
}

// Clone — гүн хуулбар (санах ойн сан, кэш дуудагчийн өөрчлөлтөөс хамгаална).
func (o *Org) Clone() *Org {
	if o == nil {
		return nil
	}
	c := *o
	c.Programs = append([]Program{}, o.Programs...)
	if len(o.Managers) > 0 {
		c.Managers = append([]string{}, o.Managers...)
	}
	return &c
}

// PlanTrialFrom — журам эхэлсэн өдөр: үүнээс өмнө бүртгүүлсэн багшийн үнэгүй хугацаа энэ өдрөөс тоологдоно.
var PlanTrialFrom = time.Date(2026, 11, 3, 0, 0, 0, 0, time.FixedZone("ULAT", 8*3600))

// PlanTrialShortFrom — үнэгүй хугацаа 3 сараас 2 сар болсон өдөр: үүнээс өмнө багш болсон хүнд амласан 3 сар хэвээр.
var PlanTrialShortFrom = time.Date(2026, 11, 7, 0, 0, 0, 0, time.FixedZone("ULAT", 8*3600))

const planTrialMonthsBefore = 3

// PlanByID — багц (байгууллагын, хуучин багцыг ч таньна: хадгалсан эрх, захиалга).
func PlanByID(id string) (Plan, bool) {
	for _, list := range [][]Plan{Plans, OrgPlans, legacyPlans, adminPlans} {
		for _, p := range list {
			if p.ID == id {
				return p, true
			}
		}
	}
	return Plan{}, false
}

// ContractUntil — гэрээт байгууллагын эрхийн «дуусах» хугацаа (хугацаагүй).
var ContractUntil = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// PlansFor — хэрэглэгчид худалдаалж буй багцууд: байгууллагад OrgPlans, гэрээт байгууллагад байхгүй, багшид Plans.
func PlansFor(u *User) []Plan {
	switch {
	case u == nil || u.Org == nil:
		return Plans
	case u.Org.Contract():
		return []Plan{}
	}
	return OrgPlans
}

// PlanOnSale — u-д худалдаалж буй багц (худалдан авах, хянагч олгоход).
func PlanOnSale(u *User, id string) (Plan, bool) {
	for _, p := range PlansFor(u) {
		if p.ID == id {
			return p, true
		}
	}
	return Plan{}, false
}

// FeePct — хэрэглэгчийн гүйлгээний шимтгэл: байгууллагад хянагчийн тогтоосон (Org.Pct), багшид TransactionFeePct.
func (u *User) FeePct() float64 {
	if u.Org != nil {
		return u.Org.Pct
	}
	return TransactionFeePct
}

// PlanTerm — «3 сар», «1 жил».
func PlanTerm(months int) string {
	if months <= 0 {
		return "хугацаагүй"
	}
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
	months := PlanTrialMonths
	if from.Before(PlanTrialShortFrom) {
		months = planTrialMonthsBefore
	}
	if from.Before(PlanTrialFrom) {
		from = PlanTrialFrom
	}
	return from.AddDate(0, months, 0)
}

// PlanAt — t үеийн эрх: багцын ID (эсвэл PlanTrial), шимтгэлийн хувь, эрхтэй эсэх.
// Байгууллагын шимтгэл багцаас үл хамааран хянагчийн тогтоосон Org.Pct.
func (u *User) PlanAt(t time.Time) (string, float64, bool) {
	if u.Org.Contract() {
		return PlanContract, u.Org.Pct, true
	}
	for _, p := range u.Plan.Periods {
		if !t.Before(p.From) && t.Before(p.Until) {
			if pl, ok := PlanByID(p.Plan); ok {
				if u.Org != nil {
					return pl.ID, u.Org.Pct, true
				}
				return pl.ID, pl.Pct, true
			}
		}
	}
	if t.Before(u.TrialUntil()) {
		return PlanTrial, u.FeePct(), true
	}
	return "", u.FeePct(), false
}

// PlanStorageBytes — t үеийн идэвхтэй багцад багтсан файлын сан (үнэгүй хугацаа, эрхгүй үед 0); байгууллагад
// хянагчийн гэрээгээр тогтоосон сантай (Org.StorageMB) харьцуулж ихийг нь.
func (u *User) PlanStorageBytes(t time.Time) int64 {
	var b int64
	id, _, ok := u.PlanAt(t)
	if pl, found := PlanByID(id); ok && found {
		b = pl.StorageMB << 20
	}
	if u.Org != nil {
		b = max(b, u.Org.StorageMB<<20)
	}
	return b
}

// PlanUntil — эрх дуусах хугацаа (үнэгүй хугацаа, дараалсан бүх багцыг оруулаад). Ирээдүйд завсар үүсдэггүй.
func (u *User) PlanUntil() time.Time {
	if u.Org.Contract() {
		return ContractUntil
	}
	end := u.TrialUntil()
	for _, p := range u.Plan.Periods {
		if p.Until.After(end) {
			end = p.Until
		}
	}
	return end
}

// PlanActive — системийн эрхтэй эсэх (багш, байгууллагад хамаарна).
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
	case pl.Months <= 0: // хязгааргүй: одооноос хугацаагүй; бусад багцын үлдэгдэл хэрэггүй (өнгөрсөн хэсэг нь үлдэнэ)
		out := make([]PlanPeriod, 0, len(u.Plan.Periods)+1)
		for _, p := range u.Plan.Periods {
			if p.From.Before(now) {
				if p.Until.After(now) {
					p.Until = now
				}
				out = append(out, p)
			}
		}
		np = PlanPeriod{Plan: pl.ID, From: now, Until: ContractUntil, Order: order}
		u.Plan.Periods = append(out, np)
	case !ok:
		np = PlanPeriod{Plan: pl.ID, From: now, Until: pl.Until(now), Order: order}
		u.Plan.Periods = append(u.Plan.Periods, np)
	case pl.Pct < pct && u.Org == nil: // байгууллагын шимтгэл багцаас үл хамаарна — араас нь
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
		np = PlanPeriod{Plan: pl.ID, From: start, Until: pl.Until(start), Order: order}
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
	switch {
	case o.CommissionPct > 0:
		return o.CommissionPct
	case o.CommissionPct < 0: // шимтгэлгүй гэрээ (0%) — 0 нь «хуучин захиалга» гэсэн утгатай тул zeroCommission-оор тэмдэглэнэ
		return 0
	}
	return teacherCommission
}

// zeroCommission — 0%-ийн шимтгэлийг (байгууллагын гэрээ) захиалгад хадгалах утга.
const zeroCommission = -1

// OrderCommission — нэг борлуулалтаас авах платформын шимтгэл (бүхэл төгрөг рүү тоймлоно).
func OrderCommission(o *Order) int64 {
	if o.Amount <= 0 || o.Platform() {
		return 0
	}
	return int64(math.Round(float64(o.Amount) * o.CommissionRate() / 100))
}

// saleCommissionPct — багшийн одоогийн шимтгэл (үнэгүй хугацаанд, эрхгүй үед гүйлгээний шимтгэл).
func saleCommissionPct(teacher *User, now time.Time) float64 {
	if teacher == nil {
		return TransactionFeePct
	}
	_, pct, _ := teacher.PlanAt(now)
	if pct <= 0 {
		return zeroCommission
	}
	return pct
}
