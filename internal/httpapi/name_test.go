package httpapi

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// Нэрийг шууд засах: зөвхөн нэр солигдож, бусад талбар хэвээр үлдэнэ; профайл хуудас шинэ нэрийг харуулна.
func TestInlineName(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "inlteach", "teacher")
	if code, r := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Хуучин Нэр","headline":"Математикийн багш","bio":"Танилцуулга","location":"Улаанбаатар","links":{"website":"https://example.com"}}`); code != 200 {
		t.Fatalf("профайл: %d %v", code, r)
	}
	if code, _ := call(t, srv, "PUT", "/api/me/name", "", `{"display_name":"Х"}`); code != 401 {
		t.Fatalf("нэвтрээгүй: %d", code)
	}
	for _, bad := range []string{`{"display_name":"   "}`, `{"display_name":"` + strings.Repeat("я", 81) + `"}`} {
		if code, _ := call(t, srv, "PUT", "/api/me/name", tt, bad); code != 400 {
			t.Fatalf("буруу нэр %s: %d", bad, code)
		}
	}
	code, me := call(t, srv, "PUT", "/api/me/name", tt, `{"display_name":"  Бат-Эрдэнэ   Дорж "}`)
	if code != 200 || me["display_name"] != "Бат-Эрдэнэ Дорж" {
		t.Fatalf("нэр солих: %d %v", code, me)
	}
	if me["headline"] != "Математикийн багш" || me["bio"] != "Танилцуулга" || me["location"] != "Улаанбаатар" {
		t.Fatalf("бусад талбар хэвээр байх ёстой: %v", me)
	}
	if l, _ := me["links"].(map[string]any); l["website"] != "https://example.com" {
		t.Fatalf("холбоос хэвээр байх ёстой: %v", me["links"])
	}
	res, err := http.Get(srv.URL + "/t/inlteach")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(page), "Бат-Эрдэнэ Дорж") {
		t.Fatal("профайл хуудас шинэ нэрийг харуулах ёстой (кэш цэвэрлэгдээгүй)")
	}
	st, _ := register(t, srv, "inlstud", "student")
	if code, me := call(t, srv, "PUT", "/api/me/name", st, `{"display_name":"Сараа"}`); code != 200 || me["display_name"] != "Сараа" {
		t.Fatalf("суралцагч нэр солих: %d %v", code, me)
	}
}
