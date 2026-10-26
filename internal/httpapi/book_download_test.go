package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Ном хоёр янзаар: онлайн унших эрх ба татаж авах эрх (эх файл). Унших эрхтэй хүн зөрүүг төлж татах эрх авна.
func TestBookReadOrDownload(t *testing.T) {
	bookLim = &bookLimiter{win: map[string]*[2]int64{}, seen: map[string]time.Time{}}
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "bteach", "teacher")
	pdf := []byte("%PDF-1.4\n% surgalt test book\n%%EOF")
	file := uploadFile(t, srv, tt, "Бодлого.pdf", pdf)

	for _, bad := range []string{
		`{"title":"Ном","price":5000,"download":true,"download_price":4000,"file_url":"` + file + `"}`, // татах үнэ ≤ унших үнэ
		`{"title":"Ном","price":5000,"download":true,"download_price":9000}`,                           // файлгүй
		`{"title":"Ном","price":5000,"download":true,"download_price":9000,"file_url":"/files/other/private/0123456789abcdef__x.pdf"}`,
	} {
		if code, r := call(t, srv, "POST", "/api/me/books", tt, bad); code != http.StatusBadRequest {
			t.Fatalf("буруу ном хадгалагдлаа: %s → %d %v", bad, code, r)
		}
	}
	code, b := call(t, srv, "POST", "/api/me/books", tt, `{"title":"Бодлогын ном","price":5000,"preview_pages":1,"download":true,"download_price":12000,"file_url":"`+file+`?exp=1&sig=x"}`)
	if code != 201 || b["file_url"] != file || b["download_price"].(float64) != 12000 {
		t.Fatalf("ном: %d %v", code, b)
	}
	bid := b["id"].(string)
	for p := 1; p <= 2; p++ {
		rawCall(t, srv, "POST", "/api/me/books/"+bid+"/pages/"+string(rune('0'+p)), tt, testJPEG(t))
	}
	call(t, srv, "POST", "/api/me/books/"+bid+"/pages-done", tt, `{"total":2}`)
	call(t, srv, "PUT", "/api/me/books/"+bid, tt, `{"title":"Бодлогын ном","price":5000,"preview_pages":1,"download":true,"download_price":12000,"file_url":"`+file+`","published":true}`)

	// Нээлттэй мэдээлэлд эх файлын зам гарахгүй.
	_, pub := call(t, srv, "GET", "/api/books/"+bid, "", "")
	if raw, _ := json.Marshal(pub); strings.Contains(string(raw), "/files/") || pub["book"].(map[string]any)["download"] != true {
		t.Fatalf("нээлттэй мэдээлэл: %v", pub)
	}
	s1, _ := register(t, srv, "breader", "student")
	if res, _ := rawCall(t, srv, "GET", "/api/books/"+bid+"/download", s1, nil); res.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("эрхгүй татах: %d", res.StatusCode)
	}
	// 1) Онлайн унших эрх (5000₮): уншина, татахгүй.
	_, buy := call(t, srv, "POST", "/api/books/"+bid+"/buy", s1, "")
	if buy["order"].(map[string]any)["amount"].(float64) != 5000 {
		t.Fatalf("унших эрх: %v", buy)
	}
	call(t, srv, "POST", "/api/orders/"+buy["order"].(map[string]any)["id"].(string)+"/dev-pay", s1, "")
	if res, _ := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/2", s1, nil); res.StatusCode != 200 {
		t.Fatalf("унших эрхтэй: %d", res.StatusCode)
	}
	if res, _ := rawCall(t, srv, "GET", "/api/books/"+bid+"/download", s1, nil); res.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("унших эрхээр татах ёсгүй: %d", res.StatusCode)
	}
	_, info := call(t, srv, "GET", "/api/books/"+bid, s1, "")
	if acc := info["access"].(map[string]any); acc["full"] != true || acc["download"] != false || acc["download_price"].(float64) != 7000 {
		t.Fatalf("зөрүү 7000 байх ёстой: %v", acc)
	}
	// 2) Татах эрх рүү шилжих — зөрүү 7000₮.
	_, up := call(t, srv, "POST", "/api/books/"+bid+"/buy?mode=download", s1, "")
	o := up["order"].(map[string]any)
	if o["amount"].(float64) != 7000 || o["kind"] != "book_dl" {
		t.Fatalf("татах эрхийн захиалга: %v", up)
	}
	call(t, srv, "POST", "/api/orders/"+o["id"].(string)+"/dev-pay", s1, "")
	res, body := rawCall(t, srv, "GET", "/api/books/"+bid+"/download", s1, nil)
	if res.StatusCode != 200 || !bytes.Equal(body, pdf) || !strings.Contains(res.Header.Get("Content-Disposition"), "attachment") || !strings.Contains(res.Header.Get("Content-Disposition"), ".pdf") {
		t.Fatalf("татах: %d %v %q", res.StatusCode, res.Header, body)
	}
	// 3) Шууд татах эрх (12000₮) — өөр уншигч; унших эрх агуулна.
	s2, _ := register(t, srv, "breader2", "student")
	_, d2 := call(t, srv, "POST", "/api/books/"+bid+"/buy?mode=download", s2, "")
	if d2["order"].(map[string]any)["amount"].(float64) != 12000 {
		t.Fatalf("бүтэн татах эрх: %v", d2)
	}
	call(t, srv, "POST", "/api/orders/"+d2["order"].(map[string]any)["id"].(string)+"/dev-pay", s2, "")
	if res, _ := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/2", s2, nil); res.StatusCode != 200 {
		t.Fatalf("татах эрх унших эрхийг агуулна: %d", res.StatusCode)
	}
	if res, _ := rawCall(t, srv, "GET", "/api/books/"+bid+"/download", s2, nil); res.StatusCode != 200 {
		t.Fatalf("татах эрхтэй: %d", res.StatusCode)
	}
	_, evb := rawCall(t, srv, "GET", "/api/me/book-events", tt, nil)
	if !strings.Contains(string(evb), `"download"`) || !strings.Contains(string(evb), "татах эрх") {
		t.Fatalf("лог: %s", evb)
	}
	res2, bl := rawCall(t, srv, "GET", "/api/me/books", tt, nil)
	if res2.StatusCode != 200 || !strings.Contains(string(bl), `"revenue":24000`) || !strings.Contains(string(bl), `"sales":3`) {
		t.Fatalf("орлого 5000+7000+12000: %s", bl)
	}
}

// Хянагч суралцагчийг багш болгоно: хуучин токеноор ч багшийн API ажиллана.
func TestAdminMakeTeacher(t *testing.T) {
	srv, _ := newTestServerWith(t, func(c *Config) { c.AdminEmails = []string{"boss@x.mn"} })
	defer srv.Close()
	boss, _ := register(t, srv, "boss", "student")
	s1, sid := register(t, srv, "learner", "student")
	if code, _ := call(t, srv, "GET", "/api/me/books", s1, ""); code != http.StatusForbidden {
		t.Fatalf("суралцагч багшийн API: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/admin/students/"+sid+"/make-teacher", s1, ""); code != http.StatusForbidden && code != http.StatusUnauthorized {
		t.Fatalf("хянагч биш: %d", code)
	}
	if code, r := call(t, srv, "POST", "/api/admin/students/"+sid+"/make-teacher", boss, ""); code != 200 || r["role"] != "teacher" {
		t.Fatalf("багш болгох: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/admin/students/"+sid+"/make-teacher", boss, ""); code != http.StatusConflict {
		t.Fatalf("дахин: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/me/books", s1, ""); code != 200 {
		t.Fatalf("хуучин токеноор багшийн API: %d", code)
	}
	if code, r := call(t, srv, "POST", "/api/courses", s1, `{"title":"Шинэ багшийн сургалт","price":0}`); code != 201 && code != 200 {
		t.Fatalf("шинэ багш сургалт үүсгэх: %d %v", code, r)
	}
}
