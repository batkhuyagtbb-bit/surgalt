package store

import (
	"testing"
	"time"
)

// Хязгааргүй эрх: хянагч олгоход одооноос хугацаагүй, өмнөх багцын үлдэгдэл хасагдана, файлын сан UnlimitedMB.
func TestUnlimitedPlan(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	u := &User{Role: RoleTeacher, CreatedAt: now.AddDate(0, -6, 0)}
	year, _ := PlanByID("year")
	u.AddPlan(year, "o1", now.AddDate(0, -1, 0))
	ul, ok := PlanByID(PlanUnlimited)
	if !ok || ul.StorageMB != UnlimitedMB || ul.Months != 0 {
		t.Fatalf("хязгааргүй багц: %+v %v", ul, ok)
	}
	if _, ok := PlanOnSale(u, PlanUnlimited); ok {
		t.Fatal("хязгааргүй багц худалдаанд байх ёсгүй")
	}
	np := u.AddPlan(ul, "", now)
	if !np.From.Equal(now) || !np.Until.Equal(ContractUntil) {
		t.Fatalf("хугацаа: %v → %v", np.From, np.Until)
	}
	id, _, active := u.PlanAt(now.AddDate(50, 0, 0))
	if id != PlanUnlimited || !active || !u.PlanUntil().Equal(ContractUntil) {
		t.Fatalf("50 жилийн дараа: %s %v %v", id, active, u.PlanUntil())
	}
	if got := u.PlanStorageBytes(now); got != UnlimitedMB<<20 {
		t.Fatalf("файлын сан: %d", got)
	}
	for _, p := range u.Plan.Periods {
		if p.Plan == "year" && p.Until.After(now) {
			t.Fatalf("өмнөх багцын үлдэгдэл хасагдаагүй: %+v", p)
		}
	}
	if PlanTerm(0) != "хугацаагүй" {
		t.Fatal("PlanTerm(0)")
	}
}
