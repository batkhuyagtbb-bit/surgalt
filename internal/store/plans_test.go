package store

import (
	"testing"
	"time"
)

// Үнэгүй 3 сар → эрт авсан багцад үлдэгдэл нэмэгдэнэ; дээд багц шууд эхэлж, доод нь араас; эрхгүй бол одооноос.
func TestAddPlan(t *testing.T) {
	defer func(v time.Time) { PlanTrialFrom = v }(PlanTrialFrom)
	SetTeacherCommission(21) // үнэгүй хугацааны шимтгэл (TEACHER_COMMISSION_PERCENT)
	defer SetTeacherCommission(0)
	t0 := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	PlanTrialFrom = t0.AddDate(-1, 0, 0)
	u := &User{Role: RoleTeacher, CreatedAt: t0}
	day := 24 * time.Hour
	start, _ := PlanByID("start")
	pro, _ := PlanByID("pro")
	std, _ := PlanByID("standard")

	if id, pct, ok := u.PlanAt(t0.Add(10 * day)); id != PlanTrial || pct != 21 || !ok || !u.PlanUntil().Equal(t0.AddDate(0, 3, 0)) {
		t.Fatalf("үнэгүй хугацаа: %s %v %v %v", id, pct, ok, u.PlanUntil())
	}
	// 30 дахь өдөр Эхлэл (3 сар): шууд 20%, үнэгүй хугацааны үлдсэн 2 сар орчим нэмэгдэнэ.
	now := t0.Add(30 * day)
	p := u.AddPlan(start, "o1", now)
	if !p.From.Equal(now) || !p.Until.Equal(now.AddDate(0, 3, 0).Add(t0.AddDate(0, 3, 0).Sub(now))) {
		t.Fatalf("Эхлэл: %+v", p)
	}
	if id, pct, _ := u.PlanAt(now.Add(day)); id != "start" || pct != 20 {
		t.Fatalf("Эхлэлийн шимтгэл: %s %v", id, pct)
	}
	end1 := p.Until
	// Эхлэл дээр Pro (дээд): одооноос Pro, Эхлэлийн үлдэгдэл Pro-гийн дараа.
	now2 := now.Add(10 * day)
	pp := u.AddPlan(pro, "o2", now2)
	if !pp.From.Equal(now2) || !pp.Until.Equal(now2.AddDate(1, 0, 0)) {
		t.Fatalf("Pro: %+v", pp)
	}
	if id, pct, _ := u.PlanAt(now2.Add(day)); id != "pro" || pct != 8 {
		t.Fatalf("Pro шимтгэл: %s %v", id, pct)
	}
	wantEnd := pp.Until.Add(end1.Sub(now2))
	if !u.PlanUntil().Equal(wantEnd) {
		t.Fatalf("Эхлэлийн үлдэгдэл хадгалагдах: %v != %v (%+v)", u.PlanUntil(), wantEnd, u.Plan.Periods)
	}
	if id, _, _ := u.PlanAt(pp.Until.Add(day)); id != "start" {
		t.Fatalf("Pro дууссаны дараа Эхлэлийн үлдэгдэл: %s", id)
	}
	// Pro дээр Standart (доод): одоогийн эрх дуусахад эхэлнэ.
	ps := u.AddPlan(std, "o3", now2.Add(day))
	if !ps.From.Equal(wantEnd) || !ps.Until.Equal(wantEnd.AddDate(0, 6, 0)) {
		t.Fatalf("Standart араас: %+v", ps)
	}
	if id, _, _ := u.PlanAt(now2.Add(2 * day)); id != "pro" {
		t.Fatalf("доод багц одоогийнхыг солихгүй: %s", id)
	}
	// Бүх эрх дууссаны дараа авбал одооноос.
	late := ps.Until.Add(40 * day)
	if _, _, ok := u.PlanAt(late); ok || u.PlanActive(late) {
		t.Fatal("эрх дууссан байх ёстой")
	}
	pl := u.AddPlan(start, "o4", late)
	if !pl.From.Equal(late) || !u.PlanActive(late.Add(day)) {
		t.Fatalf("дууссаны дараа: %+v", pl)
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
	free := int64(300 << 20)
	u := &User{Role: RoleTeacher, CreatedAt: now}
	u.Plan.TrialFrom = &now
	if q := u.StorageQuota(free, now); q != free {
		t.Fatalf("үнэгүй хугацаанд: %d", q)
	}
	adv, _ := PlanByID("major")
	u.AddPlan(adv, "o1", now)
	if q := u.StorageQuota(free, now); q != 1024<<20 {
		t.Fatalf("Advanced: %d", q)
	}
	exp := now.Add(24 * time.Hour)
	u.StorageExtraBytes, u.StorageExpiresAt = 500<<20, &exp
	if q := u.StorageQuota(free, now); q != (1024+500)<<20 {
		t.Fatalf("Advanced + нэмэлт 500MB: %d", q)
	}
	if q := u.StorageQuota(free, u.PlanUntil().Add(time.Hour)); q != free {
		t.Fatalf("эрх, нэмэлт дууссаны дараа: %d", q)
	}
	if st, _ := PlanByID("start"); st.StorageMB != 300 || st.Title() != "Starter (Эхлэл)" {
		t.Fatalf("Starter: %+v", st)
	}
}
