package httpapi

import (
	"bytes"
	"image/jpeg"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/oauth"
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

// Зураггүй хуудсыг (нүүр) Facebook-т хуваалцахад сайтын зураг /og.jpg, fb:app_id; номын хавтсыг robots.txt хаахгүй;
// Facebook доторх хөтөчөөс Google нэвтрэлт Google-ийн 403 биш, тайлбартай нэвтрэх хуудас руу.
func TestSiteOGAndInApp(t *testing.T) {
	srv, _ := newTestServerWith(t, func(c *Config) { c.PublicURL, c.FacebookAppID = "https://surgalt.mn", "1430100012007117" })
	defer srv.Close()
	defer func() { siteURL, siteFBApp = "", "" }()
	get := func(path, ua string) (*http.Response, []byte) {
		req, _ := http.NewRequest("GET", srv.URL+path, nil)
		if ua != "" {
			req.Header.Set("User-Agent", ua)
		}
		res, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		return res, b
	}
	_, body := get("/", "")
	html := string(body)
	for _, want := range []string{`property="og:image" content="https://surgalt.mn/og.jpg"`, `og:image:width" content="1200"`, `property="fb:app_id" content="1430100012007117"`, `twitter:card" content="summary_large_image"`} {
		if !strings.Contains(html, want) {
			t.Errorf("нүүр хуудсанд %q алга", want)
		}
	}
	if strings.Count(html, `property="og:image"`) != 1 {
		t.Errorf("og:image нэг л удаа байх ёстой")
	}
	res, img := get("/og.jpg", "facebookexternalhit/1.1")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("og.jpg: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	if cfg, err := jpeg.DecodeConfig(bytes.NewReader(img)); err != nil || cfg.Width != 1200 || cfg.Height != 630 {
		t.Fatalf("сайтын зураг: %v %+v", err, cfg)
	}
	if _, rb := get("/robots.txt", ""); !strings.Contains(string(rb), "Allow: /api/books/*/cover\nDisallow: /api/") {
		t.Errorf("robots.txt: %s", rb)
	}

	for ua, want := range map[string]bool{
		"Mozilla/5.0 (Linux; Android 13; wv) Chrome/129.0 Mobile Safari/537.36 [FB_IAB/FB4A;FBAV/485.0.0.42.109;]": true,
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) Mobile/15E148 [FBAN/MessengerForiOS;FBAV/478.0]":    true,
		"Mozilla/5.0 (iPhone) Mobile/15E148 Instagram 350.0.0":                                                      true,
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/130.0 Safari/537.36":                   false,
	} {
		if inAppBrowser(ua) != want {
			t.Errorf("inAppBrowser(%q) != %v", ua, want)
		}
	}
}

func TestGoogleInAppRedirect(t *testing.T) {
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{TrustProxy: true}, newTestStore(t), auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.OAuth = oauth.NewRegistry("https://surgalt.mn", oauth.Credentials{ClientID: "g", ClientSecret: "gs"}, oauth.Credentials{}, oauth.Credentials{ClientID: "f", ClientSecret: "fs"}, oauth.Credentials{})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	start := func(provider, ua string) string {
		req, _ := http.NewRequest("GET", srv.URL+"/auth/"+provider+"/start", nil)
		req.Header.Set("User-Agent", ua)
		res, err := http.DefaultTransport.RoundTrip(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.Header.Get("Location")
	}
	fb := "Mozilla/5.0 (Linux; Android 13; wv) Chrome/129.0 Mobile Safari/537.36 [FB_IAB/FB4A;FBAV/485.0.0.42.109;]"
	if loc := start("google", fb); !strings.HasPrefix(loc, "/login#error=") {
		t.Fatalf("Facebook доторх хөтөчөөс Google: %s", loc)
	}
	if loc := start("facebook", fb); !strings.HasPrefix(loc, "https://www.facebook.com/") {
		t.Fatalf("Facebook доторх хөтөчөөс Facebook нэвтрэлт хэвээр: %s", loc)
	}
	if loc := start("google", "Mozilla/5.0 (Windows NT 10.0) Chrome/130.0 Safari/537.36"); !strings.HasPrefix(loc, "https://accounts.google.com/") {
		t.Fatalf("энгийн хөтөч: %s", loc)
	}
}
