package httpapi

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// Нүүр зургийн байрлал: анх 50%, эзэмшигч чирж хадгална (0-100), нийтийн профайл, HTML тэр байрлалаар; шинэ зураг
// тавихад голдоо буцна.
func TestCoverPos(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tok, _ := register(t, srv, "coverteach", "teacher")
	profile := func(cover string) {
		t.Helper()
		if code, r := call(t, srv, "PUT", "/api/me/profile", tok, `{"display_name":"Багш","cover_url":"`+cover+`"}`); code != 200 {
			t.Fatalf("профайл: %d %v", code, r)
		}
	}
	profile("https://example.com/a.jpg")
	if _, me := call(t, srv, "GET", "/api/me", tok, ""); me["cover_y"].(float64) != 50 {
		t.Fatalf("анхны байрлал: %v", me["cover_y"])
	}
	if code, _ := call(t, srv, "PUT", "/api/me/cover-pos", tok, `{"y":150}`); code != http.StatusBadRequest {
		t.Fatalf("100-аас их байрлал: %d", code)
	}
	if code, r := call(t, srv, "PUT", "/api/me/cover-pos", tok, `{"y":22.4}`); code != 200 || r["cover_y"].(float64) != 22 {
		t.Fatalf("байрлал хадгалах: %d %v", code, r)
	}
	if _, p := call(t, srv, "GET", "/api/teachers/coverteach", "", ""); p["teacher"].(map[string]any)["cover_y"].(float64) != 22 {
		t.Fatalf("нийтийн профайл: %v", p["teacher"])
	}
	res, err := http.Get(srv.URL + "/t/coverteach")
	if err != nil {
		t.Fatal(err)
	}
	html, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(html), "object-position:50% 22%") {
		t.Fatal("HTML-д нүүр зургийн байрлал алга")
	}
	profile("https://example.com/a.jpg") // зураг солигдоогүй — байрлал хэвээр
	if _, me := call(t, srv, "GET", "/api/me", tok, ""); me["cover_y"].(float64) != 22 {
		t.Fatalf("бусад талбар хадгалахад байрлал алдагдлаа: %v", me["cover_y"])
	}
	profile("https://example.com/b.jpg")
	if _, me := call(t, srv, "GET", "/api/me", tok, ""); me["cover_y"].(float64) != 50 {
		t.Fatalf("шинэ зурагт байрлал голдоо буцах ёстой: %v", me["cover_y"])
	}
}
