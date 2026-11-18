package store

import (
	"context"
	"testing"
)

// Нэгдсэн эрэмбэ — сургалт бүрийн онооны дундаж (олон сургалтад хамрагдсанаар нэмэгдэхгүй).
func TestRankPointsAverage(t *testing.T) {
	m, ctx := NewMemory(), context.Background()
	if avg, _ := m.SaveRankPoints(ctx, "u", "a", 900); avg != 900 {
		t.Fatalf("нэг сургалт: %d", avg)
	}
	if avg, _ := m.SaveRankPoints(ctx, "u", "b", 300); avg != 600 {
		t.Fatalf("(900+300)/2 = 600 байх ёстой: %d", avg)
	}
	if avg, _ := m.SaveRankPoints(ctx, "u", "c", 301); avg != 500 { // 1501/3 = 500.33 → 500
		t.Fatalf("дундаж тоймлол: %d", avg)
	}
	if avg, _ := m.SaveRankPoints(ctx, "u", "b", 600); avg != 600 { // шинэчилсэн оноо: (900+600+301)/3 = 600.33 → 600
		t.Fatalf("шинэчилсэн дундаж: %d", avg)
	}
	if avg, _ := m.SaveRankPoints(ctx, "v", "a", 50); avg != 50 { // өөр суралцагч нөлөөлөхгүй
		t.Fatalf("өөр суралцагч: %d", avg)
	}
}
