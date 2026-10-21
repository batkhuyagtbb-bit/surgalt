package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"testing"

	"surgalt/internal/auth"
)

// Хянагчийн самбар: тусдаа зам, нууц үгээр нэвтрэх, cookie, өөр сайтын хүсэлт, хандалтын тоолол.
func TestAdminPanel(t *testing.T) {
	hash, err := auth.HashPassword(context.Background(), "admin-secret-pass")
	if err != nil {
		t.Fatal(err)
	}
	srv, _ := newTestServerWith(t, func(c *Config) {
		c.AdminPath, c.AdminLogin, c.AdminPassHash = "/sg-control-x1", "boss", hash
	})
	defer srv.Close()
	jar, _ := cookiejar.New(nil)
	cl := &http.Client{Jar: jar}
	do := func(method, path, body string, hdr map[string]string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
		res, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { res.Body.Close() })
		return res
	}
	if r := do("GET", "/admin", "", nil); r.StatusCode != http.StatusNotFound {
		t.Fatalf("далд зам тохируулсан үед /admin 404 байх ёстой: %d", r.StatusCode)
	}
	if r := do("GET", "/sg-control-x1", "", nil); r.StatusCode != 200 || r.Header.Get("X-Robots-Tag") == "" {
		t.Fatalf("самбарын хуудас: %d %q", r.StatusCode, r.Header.Get("X-Robots-Tag"))
	}
	if r := do("GET", "/api/admin/overview", "", nil); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй: %d", r.StatusCode)
	}
	if r := do("POST", "/api/admin/login", `{"login":"boss","password":"wrong"}`, nil); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("буруу нууц үг: %d", r.StatusCode)
	}
	if r := do("POST", "/api/admin/login", `{"login":"boss","password":"admin-secret-pass"}`, map[string]string{"Origin": "https://evil.example"}); r.StatusCode != http.StatusForbidden {
		t.Fatalf("өөр сайтаас нэвтрэх: %d", r.StatusCode)
	}
	r := do("POST", "/api/admin/login", `{"login":"boss","password":"admin-secret-pass"}`, nil)
	if r.StatusCode != 200 {
		t.Fatalf("нэвтрэх: %d", r.StatusCode)
	}
	var ck *http.Cookie
	for _, c := range r.Cookies() {
		if c.Name == adminCookie {
			ck = c
		}
	}
	if ck == nil || !ck.HttpOnly || ck.SameSite != http.SameSiteStrictMode || ck.Path != "/api/admin" {
		t.Fatalf("cookie: %+v", ck)
	}
	// хандалт: бот тоологдохгүй, хянагчийн хуудас тоологдохгүй
	ua := map[string]string{"User-Agent": "Mozilla/5.0 (Windows NT 10.0) Chrome/129.0"}
	do("POST", "/api/hit", `{"p":"/","vid":"visitor-aaa","ref":"www.facebook.com"}`, ua)
	do("POST", "/api/hit", `{"p":"/login","vid":"visitor-aaa","ref":"self"}`, ua)
	do("POST", "/api/hit", `{"p":"/","vid":"visitor-bbb","ref":""}`, ua)
	do("POST", "/api/hit", `{"p":"/sg-control-x1","vid":"visitor-ccc","ref":""}`, ua)
	do("POST", "/api/hit", `{"p":"/","vid":"visitor-ddd","ref":""}`, map[string]string{"User-Agent": "Googlebot/2.1"})
	r = do("GET", "/api/admin/overview?days=7", "", nil)
	if r.StatusCode != 200 {
		t.Fatalf("тойм: %d", r.StatusCode)
	}
	var ov struct {
		Traffic struct {
			Today struct{ Views, Unique int64 }
		}
	}
	_ = json.NewDecoder(r.Body).Decode(&ov)
	if ov.Traffic.Today.Views != 3 || ov.Traffic.Today.Unique != 2 {
		t.Fatalf("өнөөдрийн хандалт: %+v", ov.Traffic.Today)
	}
	for _, p := range []string{"/api/admin/teachers", "/api/admin/students", "/api/admin/transactions?status=all", "/api/admin/finance"} {
		if r := do("GET", p, "", nil); r.StatusCode != 200 {
			t.Fatalf("%s: %d", p, r.StatusCode)
		}
	}
	// зардал: өөр сайтаас бол татгалзана, өөрөөс бол бүртгэнэ
	exp := `{"date":"2026-01-05","category":"Сервер","amount":50000,"note":"VPS"}`
	if r := do("POST", "/api/admin/expenses", exp, map[string]string{"Origin": "https://evil.example"}); r.StatusCode != http.StatusForbidden {
		t.Fatalf("өөр сайтаас зардал: %d", r.StatusCode)
	}
	if r := do("POST", "/api/admin/expenses", exp, map[string]string{"Origin": srv.URL}); r.StatusCode != http.StatusCreated {
		t.Fatalf("зардал: %d", r.StatusCode)
	}
	do("POST", "/api/admin/logout", "", nil)
	if r := do("GET", "/api/admin/overview", "", nil); r.StatusCode != http.StatusUnauthorized {
		t.Fatalf("гарсны дараа: %d", r.StatusCode)
	}
}

func TestValidAdminPath(t *testing.T) {
	for p, want := range map[string]bool{"/sg-control-7f3a": true, "/admin": true, "/login": false, "/a/b": false, "admin": false, "/ab": false} {
		if got := ValidAdminPath(p); got != want {
			t.Errorf("%q: %v, хүлээсэн %v", p, got, want)
		}
	}
}
