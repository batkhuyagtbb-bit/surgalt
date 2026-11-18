package httpapi

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// «Систем хэрхэн ажилладаг вэ?» (/ar): хуудас ба скрипт нь манай серверээс, inline скриптгүй (CSP-д хэш шаардахгүй).
func TestARPage(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	get := func(p string) (int, string) {
		res, err := http.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}
	for _, p := range []string{"/ar", "/ar.html"} {
		code, body := get(p)
		// Скрипт хувилбартай (?v=assetVersion): /static нь immutable кэштэй — өмнө орсон утас шинэ ar.js-ийг татна.
		if code != 200 || !strings.Contains(body, `src="/static/ar.js?v=`+assetVersion+`"`) || !strings.Contains(body, `src="/static/vendor/three-r128.min.js?v=`+assetVersion+`"`) {
			t.Fatalf("%s: %d", p, code)
		}
		if regexp.MustCompile(`<script>`).MatchString(body) || strings.Contains(body, " onclick=") {
			t.Fatalf("%s: inline скрипт байж болохгүй (CSP)", p)
		}
	}
	if code, js := get("/static/ar.js"); code != 200 || !strings.Contains(js, "Мартах муруй") {
		t.Fatalf("ar.js: %d", code)
	}
	if ValidAdminPath("/ar") {
		t.Fatal("/ar нь хянагчийн замд давхцахгүй")
	}
}
