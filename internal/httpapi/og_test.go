package httpapi

import (
	"bytes"
	"image/jpeg"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Профайлыг Facebook-т хуваалцахад: 1200×630 зураг, Open Graph тагууд.
func TestProfileOG(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "ogteach", "teacher")
	if code, r := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Өнөрбаяр Үүрцайх","headline":"Математикийн багш 🎓","bio":"Олимпиадын бэлтгэл","avatar_url":"","cover_url":"","subjects":["Математик"],"location":"Улаанбаатар","links":{}}`); code != 200 {
		t.Fatalf("профайл: %d %v", code, r)
	}
	call(t, srv, "POST", "/api/courses", tt, `{"title":"Алгебр","price":0,"published":true}`)

	res, err := http.Get(srv.URL + "/t/ogteach")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	for _, want := range []string{`property="og:image" content="`, `/t/ogteach/og.jpg?v=`, `og:image:width" content="1200"`, `og:image:height" content="630"`, `twitter:card" content="summary_large_image"`, `og:type" content="profile"`, `og:description" content="Математикийн багш`, `1 сургалт`} {
		if !strings.Contains(html, want) {
			t.Errorf("HTML-д %q алга", want)
		}
	}
	res, err = http.Get(srv.URL + "/t/ogteach/og.jpg")
	if err != nil {
		t.Fatal(err)
	}
	img, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("og.jpg: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(img))
	if err != nil || cfg.Width != 1200 || cfg.Height != 630 {
		t.Fatalf("зургийн хэмжээ: %v %+v", err, cfg)
	}
	if res, _ := http.Get(srv.URL + "/t/nobody404/og.jpg"); res.StatusCode != http.StatusNotFound {
		t.Fatalf("байхгүй профайл: %d", res.StatusCode)
	}
	bold, _ := ogFonts()
	if got := newOGText(bold, 20).clean("Багш 🎓 Өө Үү"); got != "Багш  Өө Үү" && got != "Багш Өө Үү" {
		t.Errorf("emoji хасагдах ёстой: %q", got)
	}
}
