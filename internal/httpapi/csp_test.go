package httpapi

import (
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// Хуудас бүр CSP-тэй, inline скрипт бүр хэшээрээ зөвшөөрөгдсөн, inline event handler (onclick=…) алга; админ —
// frame-ancestors 'none'; embed — sandbox; API-д CSP тавихгүй.
func TestCSPPages(t *testing.T) {
	srv, s, _ := newPlanServer(t)
	tt, _ := register(t, srv, "cspteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"CSP","price":5000,"published":true}`)
	cid := c["id"].(string)
	su, _ := register(t, srv, "cspstud", "student")
	_, d := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", su, "")
	oid := d["order"].(map[string]any)["id"].(string)

	scriptRe := regexp.MustCompile(`(?s)<script([^>]*)>(.*?)</script>`)
	handlerRe := regexp.MustCompile(`<[a-zA-Z][^>]*\son[a-z]+\s*=`)
	pages := []string{"/", "/login", "/me", "/privacy", "/terms", "/reset", "/t/cspteach", "/c/" + cid, "/intro2",
		"/pay/" + oid + "?t=" + s.payMAC(oid), "/c/000000000000000000000000", "/admin"}
	for _, p := range pages {
		res, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		csp := res.Header.Get("Content-Security-Policy")
		if csp == "" {
			t.Errorf("%s (%d): CSP алга", p, res.StatusCode)
			continue
		}
		if want := map[bool]string{true: "frame-ancestors 'none'", false: "frame-ancestors 'self'"}[p == "/admin"]; !strings.Contains(csp, want) {
			t.Errorf("%s: %s алга", p, want)
		}
		html := string(b)
		for _, m := range scriptRe.FindAllStringSubmatch(html, -1) {
			attrs, body := m[1], m[2]
			if strings.Contains(attrs, "src=") || strings.Contains(attrs, "application/ld+json") {
				continue
			}
			if !strings.Contains(csp, scriptHash(body)) {
				t.Errorf("%s: inline скриптийн хэш CSP-д алга: %.80q", p, body)
			}
		}
		if m := handlerRe.FindString(html); m != "" {
			t.Errorf("%s: inline event handler: %.100q", p, m)
		}
	}

	res, err := http.Get(srv.URL + "/embed-frame")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if csp := res.Header.Get("Content-Security-Policy"); !strings.HasPrefix(csp, "sandbox allow-scripts") || strings.Contains(csp, "allow-same-origin") {
		t.Errorf("embed-frame CSP: %q", csp)
	}
	res, _ = http.Get(srv.URL + "/api/me")
	res.Body.Close()
	if res.Header.Get("Content-Security-Policy") != "" {
		t.Error("API хариунд CSP тавих шаардлагагүй")
	}
	res, _ = http.Post(srv.URL+"/api/csp-report", "application/csp-report",
		strings.NewReader(`{"csp-report":{"document-uri":"https://surgalt.mn/c/1?x","effective-directive":"script-src-elem","blocked-uri":"https://evil.example/x.js","line-number":3}}`))
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("csp-report: %d", res.StatusCode)
	}
}

// Static JS, intro2.html дахь CDN-ийн скрипт бүр cspScripts-д (сангийн хувилбар солиход CSP-г мартахгүй).
func TestCSPExternalScripts(t *testing.T) {
	urlRe := regexp.MustCompile(`https://(?:cdnjs\.cloudflare\.com|cdn\.jsdelivr\.net|www\.youtube\.com/iframe_api)[^"'\x60\s)]*`)
	_ = fs.WalkDir(staticFS, "static", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !(strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".html")) || strings.Contains(p, "/vendor/") {
			return err
		}
		b, _ := staticFS.ReadFile(p)
		for _, u := range urlRe.FindAllString(string(b), -1) {
			ok := false
			for _, pre := range cspScripts {
				if strings.HasPrefix(u, pre) || strings.HasPrefix(pre, u) { // PDFJS = ".../pdf.js/4.10.38/" — тогтмол угтвар
					ok = true
				}
			}
			if !ok {
				t.Errorf("%s: %s — cspScripts-д нэмнэ үү", p, u)
			}
		}
		return nil
	})
}

// Бүх JS/CSS алдаагүй minify хийгдэнэ; app.js-ийн гадагш гаргасан window.SG, intro.js-ийн export хэвээр.
func TestMinifyAssets(t *testing.T) {
	sub, _ := fs.Sub(staticFS, "static")
	_ = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !minifiable(p) {
			return err
		}
		raw, _ := fs.ReadFile(sub, p)
		out, err := minifyAsset(p, raw)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			return nil
		}
		if len(out) >= len(raw) {
			t.Errorf("%s: багассангүй (%d → %d)", p, len(raw), len(out))
		}
		switch p {
		case "app.js":
			if !strings.Contains(string(out), "window.SG=") {
				t.Error("app.js: window.SG алга")
			}
		case "intro.js":
			if !strings.Contains(string(out), "runIntro") {
				t.Error("intro.js: export runIntro алга")
			}
		}
		return nil
	})
	mfs := newMinFS(sub, nil)
	f, err := mfs.Open("app.js")
	if err != nil {
		t.Fatal(err)
	}
	st, _ := f.Stat()
	b, _ := io.ReadAll(f)
	f.Close()
	raw, _ := fs.ReadFile(sub, "app.js")
	if int64(len(b)) != st.Size() || len(b) >= len(raw) {
		t.Errorf("minFS: хэмжээ %d/%d, эх %d", len(b), st.Size(), len(raw))
	}
}
