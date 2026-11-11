package httpapi

import (
	"testing"
)

// Модулийг (бүлгийг) нээх/хаахад бүх хичээл нь хамт нээгдэж/хаагдана: нэг түгжээн дор атомараар,
// хаалттай модульд нэмсэн эсвэл зөөж оруулсан хичээл ч хаалттай болно, бусад модуль хөндөгдөхгүй.
func TestSectionVisibility(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "secvis", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Модульт сургалт","price":0,"published":true}`)
	cid := c["id"].(string)
	mk := func(title, sec string) string {
		_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"`+title+`","is_free":true,"section":"`+sec+`"}`)
		return l["id"].(string)
	}
	a1, a2, a3 := mk("A1", "Модуль 1"), mk("A2", "Модуль 1"), mk("A3", "Модуль 1")
	b1 := mk("B1", "Модуль 2")
	state := func() map[string]bool {
		_, r := call(t, srv, "GET", "/api/me/courses/"+cid, tt, "")
		out := map[string]bool{}
		for _, x := range r["lessons"].([]any) {
			l := x.(map[string]any)
			h, _ := l["hidden"].(bool)
			out[l["id"].(string)] = h
		}
		return out
	}

	// 1) Модуль 1-ийг хаахад 3 хичээл бүгд хамт хаагдана, Модуль 2 хөндөгдөхгүй.
	code, r := call(t, srv, "PUT", "/api/courses/"+cid+"/section-visibility", tt, `{"section":"Модуль 1","hidden":true}`)
	if code != 200 || r["changed"].(float64) != 3 {
		t.Fatalf("модуль хаах: %d %v", code, r)
	}
	st := state()
	if !st[a1] || !st[a2] || !st[a3] || st[b1] {
		t.Fatalf("хаасны дараа: %v", st)
	}

	// 2) Хаалттай модульд шинэ хичээл нэмэхэд хаалттай үүснэ; нээлттэй модульд нэмбэл нээлттэй.
	a4, b2 := mk("A4", "Модуль 1"), mk("B2", "Модуль 2")
	if st = state(); !st[a4] || st[b2] {
		t.Fatalf("шинэ хичээл: A4 хаалттай, B2 нээлттэй байх ёстой: %v", st)
	}

	// 3) Нээлттэй хичээлийг хаалттай модуль руу чирж оруулбал хамт хаагдана.
	order := `{"items":[{"id":"` + a1 + `","section":"Модуль 1"},{"id":"` + a2 + `","section":"Модуль 1"},{"id":"` + a3 + `","section":"Модуль 1"},{"id":"` + a4 + `","section":"Модуль 1"},{"id":"` + b2 + `","section":"Модуль 1"},{"id":"` + b1 + `","section":"Модуль 2"}]}`
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lesson-order", tt, order); code != 200 {
		t.Fatalf("дараалал: %d", code)
	}
	if st = state(); !st[b2] || st[b1] {
		t.Fatalf("хаалттай модуль руу зөөсөн B2 хаагдах ёстой: %v", st)
	}

	// 4) Хичээлийн тохиргоогоор бүлгийг нь сольж хаалттай модуль руу оруулбал хаагдана.
	b3 := mk("B3", "Модуль 2")
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+b3, tt, `{"title":"B3","is_free":true,"section":"Модуль 1"}`); code != 200 {
		t.Fatalf("засах: %d", code)
	}
	if st = state(); !st[b3] {
		t.Fatalf("бүлэг сольж хаалттай модульд орсон B3 хаагдах ёстой: %v", st)
	}

	// 5) Модулийг нээхэд бүх (одоо 6) хичээл хамт нээгдэнэ.
	code, r = call(t, srv, "PUT", "/api/courses/"+cid+"/section-visibility", tt, `{"section":"Модуль 1","hidden":false}`)
	if code != 200 || r["changed"].(float64) != 6 {
		t.Fatalf("модуль нээх: %d %v", code, r)
	}
	for id, h := range state() {
		if h {
			t.Fatalf("нээсний дараа хаалттай хичээл үлдсэн: %s", id)
		}
	}

	// 6) Суралцагч, өөр багш модуль нээх/хаах эрхгүй.
	other, _ := register(t, srv, "secvis2", "teacher")
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/section-visibility", other, `{"section":"Модуль 1","hidden":true}`); code == 200 {
		t.Fatalf("өөр багш модуль хааж чадах ёсгүй")
	}
}
