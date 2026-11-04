package httpapi

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"testing"
)

// get — Accept-Encoding-ийг гараар тавьж (Go клиент автоматаар задлахгүй) түүхий хариуг авна.
func getRaw(t *testing.T, url, enc string) (*http.Response, []byte) {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	if enc != "" {
		req.Header.Set("Accept-Encoding", enc)
	}
	res, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res, b
}

func gunzip(t *testing.T, b []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestCompression(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()

	// Static: урьдчилан шахсан хувилбар, задлахад эх файлтай ижил.
	res, raw := getRaw(t, srv.URL+"/static/app.js", "")
	if res.Header.Get("Content-Encoding") != "" || len(raw) < 100_000 {
		t.Fatalf("gzip хүсээгүй бол шахахгүй: enc=%q len=%d", res.Header.Get("Content-Encoding"), len(raw))
	}
	res, gz := getRaw(t, srv.URL+"/static/app.js", "gzip, br")
	if res.Header.Get("Content-Encoding") != "gzip" || len(gz) >= len(raw)/2 {
		t.Fatalf("app.js шахагдаагүй: enc=%q %d/%d", res.Header.Get("Content-Encoding"), len(gz), len(raw))
	}
	if string(gunzip(t, gz)) != string(raw) {
		t.Fatal("шахсан app.js эхтэйгээ таарахгүй")
	}
	if !strings.Contains(res.Header.Get("Cache-Control"), "immutable") || !strings.Contains(res.Header.Get("Content-Type"), "javascript") {
		t.Fatalf("static толгой: %v", res.Header)
	}

	// JSON (1KB-аас том) шахагдана.
	_, tr := call(t, srv, "POST", "/api/auth/register", "", `{"username":"gzt","email":"g@x.mn","password":"password1","role":"teacher","accept_terms":true}`)
	tt := tr["token"].(string)
	desc := strings.Repeat("Сургалтын тайлбар. ", 120)
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Курс","description":"`+desc+`","published":true}`)
	cid := c["id"].(string)
	res, body := getRaw(t, srv.URL+"/api/courses/"+cid, "gzip")
	if res.Header.Get("Content-Encoding") != "gzip" || !strings.Contains(res.Header.Get("Content-Type"), "json") {
		t.Fatalf("JSON шахагдаагүй: %v", res.Header)
	}
	if !strings.Contains(string(gunzip(t, body)), "Сургалтын тайлбар") {
		t.Fatal("шахсан JSON задрахгүй")
	}
	// HTML хуудас шахагдана.
	if res, _ := getRaw(t, srv.URL+"/login", "gzip"); res.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("HTML шахагдаагүй: %v", res.Header)
	}
}

func TestFriendly404(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	res, b := getRaw(t, srv.URL+"/bairgui/huudas", "")
	if res.StatusCode != 404 || !strings.Contains(res.Header.Get("Content-Type"), "text/html") {
		t.Fatalf("404 HTML хүлээсэн: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	for _, want := range []string{"Энэ хуудас олдсонгүй", "/bairgui/huudas", "Нүүр хуудас", "app.css"} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("404 хуудсанд %q алга", want)
		}
	}
	if strings.Contains(string(b), "404 page not found") {
		t.Fatal("Go-ийн энгийн 404 текст хэвээр")
	}
	// Олдоогүй багш — мөн найрсаг хуудас.
	res, b = getRaw(t, srv.URL+"/t/hen-ch-bish", "")
	if res.StatusCode != 404 || !strings.Contains(string(b), "Энэ хуудас олдсонгүй") {
		t.Fatalf("олдоогүй профайл: %d", res.StatusCode)
	}
	// API-ийн олдоогүй хаяг HTML буцаахгүй.
	res, b = getRaw(t, srv.URL+"/api/bairgui", "")
	if res.StatusCode != 404 || strings.Contains(string(b), "<html") {
		t.Fatalf("API 404 HTML болсон: %d", res.StatusCode)
	}
	// Буруу method (POST-оор нүүр хуудас) 404 хуудас болохгүй.
	req, _ := http.NewRequest("POST", srv.URL+"/login", nil)
	if res, err := http.DefaultClient.Do(req); err != nil || res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("405 хүлээсэн: %v %v", res, err)
	}
}

func TestHomeLite(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	_, tr := call(t, srv, "POST", "/api/auth/register", "", `{"username":"liteteach","email":"lt@x.mn","password":"password1","role":"teacher","accept_terms":true}`)
	tt := tr["token"].(string)
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Үнэгүй","published":true}`)
	cid := c["id"].(string)
	_, sr := call(t, srv, "POST", "/api/auth/register", "", `{"username":"litestud","email":"ls@x.mn","password":"password1","accept_terms":true}`)
	st := sr["token"].(string)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", st, ""); code != 200 && code != 201 {
		t.Fatalf("элсэлт %d", code)
	}
	code, full := call(t, srv, "GET", "/api/me/home", st, "")
	if code != 200 {
		t.Fatalf("home %d", code)
	}
	code, lite := call(t, srv, "GET", "/api/me/home?lite=1", st, "")
	if code != 200 {
		t.Fatalf("home lite %d", code)
	}
	// Чатын самбарт хэрэгтэй талбарууд ижил.
	for _, k := range []string{"teachers", "chats", "courses"} {
		if len(full[k].([]any)) != len(lite[k].([]any)) {
			t.Fatalf("%s: бүтэн %d, lite %d", k, len(full[k].([]any)), len(lite[k].([]any)))
		}
	}
	if len(lite["teachers"].([]any)) != 1 {
		t.Fatalf("элссэн сургалтын багш харагдах ёстой: %v", lite["teachers"])
	}
	// Багшийн самбар lite-д тооцогдохгүй.
	_, tfull := call(t, srv, "GET", "/api/me/home", tt, "")
	_, tlite := call(t, srv, "GET", "/api/me/home?lite=1", tt, "")
	if tfull["teacher"] == nil || tlite["teacher"] != nil {
		t.Fatalf("багшийн самбар: бүтэн=%v lite=%v", tfull["teacher"] != nil, tlite["teacher"] != nil)
	}
}
