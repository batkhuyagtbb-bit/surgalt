package store

import (
	"testing"
	"time"
)

// Үнэгүй 2 сар (гүйлгээний шимтгэл 2%) → үнэгүй хугацаанд авсан багц түүний араас эхэлнэ (хоног алдагдахгүй); дараагийн
// багц араас; эрхгүй бол одооноос. Хуучин багцыг таньна (хадгалсан эрх), гэхдээ худалдаалахгүй.
func TestAddPlan(t *testing.T) {
	defer func(v, w time.Time) { PlanTrialFrom, PlanTrialShortFrom = v, w }(PlanTrialFrom, PlanTrialShortFrom)
	SetTeacherCommission(21) // хуучин захиалгын шимтгэл (TEACHER_COMMISSION_PERCENT) — үнэгүй хугацаанд хамаарахгүй
	defer SetTeacherCommission(0)
	t0 := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	PlanTrialFrom, PlanTrialShortFrom = t0.AddDate(-1, 0, 0), t0.AddDate(-1, 0, 0)
	u := &User{Role: RoleTeacher, CreatedAt: t0}
	day := 24 * time.Hour
	month, _ := PlanByID("month")
	half, _ := PlanByID("half")
	year, _ := PlanByID("year")
	trialEnd := t0.AddDate(0, PlanTrialMonths, 0)

	if id, pct, ok := u.PlanAt(t0.Add(10 * day)); id != PlanTrial || pct != TransactionFeePct || !ok || !u.PlanUntil().Equal(trialEnd) {
		t.Fatalf("үнэгүй хугацаа: %s %v %v %v", id, pct, ok, u.PlanUntil())
	}
	// 30 дахь өдөр бүтэн жил: үнэгүй хугацаа дуусахад эхэлнэ.
	now := t0.Add(30 * day)
	p := u.AddPlan(year, "o1", now)
	if !p.From.Equal(trialEnd) || !p.Until.Equal(trialEnd.AddDate(1, 0, 0)) {
		t.Fatalf("бүтэн жил үнэгүй хугацааны араас: %+v", p)
	}
	if id, _, _ := u.PlanAt(now.Add(day)); id != PlanTrial {
		t.Fatalf("үнэгүй хугацаа үргэлжилнэ: %s", id)
	}
	if id, pct, _ := u.PlanAt(trialEnd.Add(day)); id != "year" || pct != TransactionFeePct {
		t.Fatalf("бүтэн жилийн шимтгэл: %s %v", id, pct)
	}
	// Дээр нь хагас жил: бүтэн жилийн араас.
	h := u.AddPlan(half, "o2", now.Add(day))
	if !h.From.Equal(p.Until) || !h.Until.Equal(p.Until.AddDate(0, 6, 0)) || !u.PlanUntil().Equal(h.Until) {
		t.Fatalf("хагас жил араас: %+v", h)
	}
	// Бүх эрх дууссаны дараа сарын багц одооноос.
	late := h.Until.Add(40 * day)
	if _, pct, ok := u.PlanAt(late); ok || u.PlanActive(late) || pct != TransactionFeePct {
		t.Fatal("эрх дууссан байх ёстой")
	}
	m := u.AddPlan(month, "o3", late)
	if !m.From.Equal(late) || !m.Until.Equal(late.AddDate(0, 1, 0)) || !u.PlanActive(late.Add(day)) {
		t.Fatalf("дууссаны дараа: %+v", m)
	}
	// Үнэ: сар 29,900, хагас жил 99,000, бүтэн жил 159,000; хуучин багц таньна, худалдаалахгүй, хувь авахгүй.
	if len(Plans) != 3 || Plans[0].Price != 29900 || Plans[1].Price != 99000 || Plans[2].Price != 159000 {
		t.Fatalf("үнийн жагсаалт: %+v", Plans)
	}
	if pl, ok := PlanByID("pro"); !ok || pl.Pct != TransactionFeePct {
		t.Fatalf("хуучин багц: %+v %v", pl, ok)
	}
	if _, ok := PlanOnSale(nil, "pro"); ok {
		t.Fatal("хуучин багц худалдаалахгүй")
	}
	// Хадгалах/унших
	got := decodePlan(u.Plan.encode())
	if len(got.Periods) != len(u.Plan.Periods) || got.Periods[0].Plan != u.Plan.Periods[0].Plan {
		t.Fatalf("JSON: %+v", got)
	}
	if (&User{Role: RoleStudent}).PlanActive(late) != true {
		t.Fatal("суралцагчид хамаарахгүй")
	}
}

// Үнэгүй хугацаа: шинэ багшид 2 сар; PlanTrialShortFrom-оос өмнө багш болсон хүнд амласан 3 сар хэвээр.
func TestTrialLength(t *testing.T) {
	defer func(v, w time.Time) { PlanTrialFrom, PlanTrialShortFrom = v, w }(PlanTrialFrom, PlanTrialShortFrom)
	cut := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	PlanTrialFrom, PlanTrialShortFrom = cut.AddDate(-1, 0, 0), cut
	old := &User{Role: RoleTeacher, CreatedAt: cut.Add(-time.Hour)}
	if !old.TrialUntil().Equal(old.CreatedAt.AddDate(0, 3, 0)) {
		t.Fatalf("өмнө бүртгүүлсэн багш 3 сар: %v", old.TrialUntil())
	}
	nw := &User{Role: RoleTeacher, CreatedAt: cut.Add(time.Hour)}
	if PlanTrialMonths != 2 || !nw.TrialUntil().Equal(nw.CreatedAt.AddDate(0, 2, 0)) {
		t.Fatalf("шинэ багш 2 сар: %v", nw.TrialUntil())
	}
}

// Шимтгэл захиалга бүрээр: төлөгдөх үеийн хувь; хуучин захиалга (0) — TEACHER_COMMISSION_PERCENT; платформын захиалгаас авахгүй.
func TestOrderCommission(t *testing.T) {
	SetTeacherCommission(21)
	defer SetTeacherCommission(0)
	cases := []struct {
		o    Order
		want int64
	}{
		{Order{Kind: OrderKindCourse, Amount: 10000}, 2100},
		{Order{Kind: OrderKindCourse, Amount: 10000, CommissionPct: 8}, 800},
		{Order{Kind: OrderKindLesson, Amount: 999, CommissionPct: 12}, 120},
		{Order{Kind: OrderKindPlan, Amount: 500000}, 0},
		{Order{Kind: OrderKindStorage, Amount: 5000}, 0},
	}
	for _, c := range cases {
		if got := OrderCommission(&c.o); got != c.want {
			t.Fatalf("%+v: %d != %d", c.o, got, c.want)
		}
	}
	// Шинэ журам: үнэгүй хугацаанд, багцтай үед гүйлгээний шимтгэл 2%.
	trial := &User{Role: RoleTeacher, CreatedAt: time.Now()}
	if got := OrderCommission(&Order{Kind: OrderKindCourse, Amount: 10000, CommissionPct: saleCommissionPct(trial, time.Now())}); got != 200 {
		t.Fatalf("гүйлгээний шимтгэл 2%%: %d", got)
	}
	b := balanceOf([]Order{{Amount: 10000}, {Amount: 10000, CommissionPct: 8}}, nil)
	if b.Commission != 2900 || b.Net != 17100 || b.CommissionPct != 14.5 {
		t.Fatalf("холимог үлдэгдэл: %+v", b)
	}
}

// Багцад багтсан файлын сан: үнэгүй хэмжээнээс их бол түүнийг орлож, нэмэлт худалдсан сан дээр нэмэгдэнэ; эрх дуусвал үнэгүй хэмжээ.
func TestPlanStorage(t *testing.T) {
	SetTeacherCommission(21)
	defer SetTeacherCommission(0)
	now := time.Now()
	free := int64(100 << 20)
	u := &User{Role: RoleTeacher, CreatedAt: now}
	u.Plan.TrialFrom = &now
	if q := u.StorageQuota(free, now); q != free {
		t.Fatalf("үнэгүй хугацаанд: %d", q)
	}
	u.Plan.Periods = []PlanPeriod{{Plan: "year", From: now.Add(-time.Hour), Until: now.AddDate(1, 0, 0)}}
	if q := u.StorageQuota(free, now); q != 1024<<20 {
		t.Fatalf("бүтэн жил: %d", q)
	}
	exp := now.Add(24 * time.Hour)
	u.StorageExtraBytes, u.StorageExpiresAt = 500<<20, &exp
	if q := u.StorageQuota(free, now); q != (1024+500)<<20 {
		t.Fatalf("бүтэн жил + нэмэлт 500MB: %d", q)
	}
	if q := u.StorageQuota(free, u.PlanUntil().Add(time.Hour)); q != free {
		t.Fatalf("эрх, нэмэлт дууссаны дараа: %d", q)
	}
	if m, _ := PlanByID("month"); m.StorageMB != 300 || m.Title() != "Сарын" {
		t.Fatalf("сарын багц: %+v", m)
	}
	if h, _ := PlanByID("half"); h.StorageMB != 500 || h.Months != 6 {
		t.Fatalf("хагас жил: %+v", h)
	}
	if st, _ := PlanByID("start"); st.Title() != "Starter (Эхлэл)" {
		t.Fatalf("хуучин багцын нэр: %+v", st)
	}
}

// Байгууллага (анхдагч 1%): үнэгүй хугацаатай, дараа нь байгууллагын багц; гэрээний сан багцынхаас их бол түүнийг орлоно.
// Хянагч өөр шимтгэл тавьсан бол гэрээт: багц авахгүй, хугацаагүй; 0% гэрээ «хуучин захиалга» (21%) болохгүй.
func TestOrgPlan(t *testing.T) {
	SetTeacherCommission(21)
	defer SetTeacherCommission(0)
	defer func(v, w time.Time) { PlanTrialFrom, PlanTrialShortFrom = v, w }(PlanTrialFrom, PlanTrialShortFrom)
	now := time.Now()
	PlanTrialFrom, PlanTrialShortFrom = now.AddDate(-1, 0, 0), now.AddDate(-1, 0, 0)
	u := &User{Role: RoleTeacher, CreatedAt: now, Org: &Org{Pct: OrgFeePct, StorageMB: 3000}}
	if id, pct, ok := u.PlanAt(now); id != PlanTrial || pct != OrgFeePct || u.Org.Contract() || !ok || !u.PlanUntil().Equal(now.AddDate(0, PlanTrialMonths, 0)) {
		t.Fatalf("байгууллагын үнэгүй хугацаа: %s %v %v %v", id, pct, ok, u.PlanUntil())
	}
	if u.StorageBase(10<<20, now) != 3000<<20 {
		t.Fatalf("гэрээний сан: %d", u.StorageBase(10<<20, now))
	}
	// Байгууллагад зөвхөн байгууллагын багц: сар 129,000 (2GB), хагас жил 500,000 (5GB), жил 1,000,000 (5GB).
	if _, ok := PlanOnSale(u, "year"); ok {
		t.Fatal("байгууллагад багшийн багц худалдахгүй")
	}
	yr, ok := PlanOnSale(u, "org_year")
	if !ok || yr.Price != 1000000 || yr.StorageMB != 5120 || len(PlansFor(u)) != 3 || PlansFor(u)[0].Price != 129000 || PlansFor(u)[1].Price != 500000 {
		t.Fatalf("байгууллагын багц: %+v %v", yr, PlansFor(u))
	}
	late := u.PlanUntil().Add(24 * time.Hour)
	if _, _, ok := u.PlanAt(late); ok || u.PlanActive(late) {
		t.Fatal("үнэгүй хугацааны дараа эрхгүй")
	}
	p := u.AddPlan(yr, "o1", now.Add(time.Hour)) // үнэгүй хугацааны араас
	if !p.From.Equal(u.TrialUntil()) || !u.PlanActive(late) {
		t.Fatalf("жилийн багц араас: %+v", p)
	}
	if id, pct, _ := u.PlanAt(late); id != "org_year" || pct != OrgFeePct || u.StorageBase(10<<20, late) != 5120<<20 {
		t.Fatalf("багцын үе: %s %v %d", id, pct, u.StorageBase(10<<20, late))
	}
	o := &Order{Amount: 10000, CommissionPct: saleCommissionPct(u, now)}
	if OrderCommission(o) != 100 {
		t.Fatalf("1%% шимтгэл: %d", OrderCommission(o))
	}
	// Гэрээт (7.5%): багц авахгүй, хугацаагүй, гэрээний сан.
	u.Org.Pct = 7.5
	if id, pct, ok := u.PlanAt(late.AddDate(5, 0, 0)); id != PlanContract || pct != 7.5 || !ok || !u.PlanUntil().Equal(ContractUntil) || len(PlansFor(u)) != 0 {
		t.Fatalf("гэрээт байгууллага: %s %v %v", id, pct, ok)
	}
	if u.StorageBase(10<<20, now) != 3000<<20 {
		t.Fatalf("гэрээт байгууллагын сан: %d", u.StorageBase(10<<20, now))
	}
	o = &Order{Amount: 10000, CommissionPct: saleCommissionPct(u, now)}
	if OrderCommission(o) != 750 {
		t.Fatalf("7.5%% шимтгэл: %d", OrderCommission(o))
	}
	u.Org.Pct = 0
	o = &Order{Amount: 10000, CommissionPct: saleCommissionPct(u, now)}
	if OrderCommission(o) != 0 {
		t.Fatalf("0%% гэрээнд шимтгэл авлаа: %d", OrderCommission(o))
	}
	if OrderCommission(&Order{Amount: 10000}) != 2100 { // хуучин захиалга хэвээр
		t.Fatal("хуучин захиалгын шимтгэл өөрчлөгдлөө")
	}
	if decodeOrg(u.Org.encode()) == nil || decodeOrg("") != nil {
		t.Fatal("org JSON")
	}
}
