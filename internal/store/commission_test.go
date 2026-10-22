package store

import "testing"

// Багшийн «Татах боломжтой» = борлуулалт − платформын шимтгэл − шилжүүлсэн − хүлээгдэж буй.
func TestBalanceCommission(t *testing.T) {
	SetTeacherCommission(21)
	defer SetTeacherCommission(0)
	orders := []Order{{Amount: 1000, Status: OrderPaid}, {Amount: 25000, Status: OrderPaid}}
	b := balanceOf(orders, []Payout{{Amount: 5000, Status: PayoutPaid}, {Amount: 1000, Status: PayoutPending}, {Amount: 9999, Status: PayoutRejected}})
	if b.Earned != 26000 || b.Commission != 5460 || b.Net != 20540 || b.Available != 14540 || b.CommissionPct != 21 {
		t.Fatalf("үлдэгдэл: %+v", b)
	}
	// шимтгэлийг хассан дүнгээс илүүг татахгүй
	if _, err := newPayout(&PayoutAccount{TeacherID: "t"}, b, b.Available+1, false); err != ErrConflict {
		t.Fatalf("илүү дүн зөвшөөрөгдлөө: %v", err)
	}
	if p, err := newPayout(&PayoutAccount{TeacherID: "t"}, b, 0, false); err != nil || p.Amount != 14540 {
		t.Fatalf("бүх үлдэгдэл: %+v %v", p, err)
	}
	// өмнө нь их татсан бол сөрөг болохгүй
	if b := balanceOf(orders, []Payout{{Amount: 26000, Status: PayoutPaid}}); b.Available != 0 {
		t.Fatalf("сөрөг үлдэгдэл: %+v", b)
	}
}
