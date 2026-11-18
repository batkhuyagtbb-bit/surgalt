package httpapi

import (
	"bytes"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
)

// Сургалт бүрийн QR (уншуулбал #join-оор нэвтэрнэ), Facebook-т хуваалцах зураг (og.jpg), хуваалцах цонх.
func TestCourseShareQRAndOG(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "shareteach", "teacher")
	code, r := call(t, srv, "POST", "/api/courses", tt, `{"title":"Физикийн үндэс — механик","price":0,"published":true}`)
	if code != 201 && code != 200 {
		t.Fatalf("сургалт: %d %v", code, r)
	}
	cid := r["id"].(string)
	_, r = call(t, srv, "POST", "/api/courses", tt, `{"title":"Ноорог","price":0,"published":false}`)
	draft := r["id"].(string)

	get := func(path string) (*http.Response, []byte) {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		return res, b
	}

	// Сургалтын хуудас: сургалтын өөрийн OG зураг, «Хуваалцах» товч, QR цонх, Facebook холбоос.
	res, body := get("/c/" + cid)
	html := string(body)
	for _, want := range []string{"/c/" + cid + "/og.jpg?v=", `data-course-share`, `id="courseShareModal"`, `/c/` + cid + `/qr.png?size=512`, `facebook.com/sharer/sharer.php?u=`, `qr.png?size=1024&download=1`} {
		if !strings.Contains(html, want) {
			t.Errorf("сургалтын хуудсанд %q алга", want)
		}
	}
	if strings.Contains(html, "/t/shareteach/og.jpg") {
		t.Error("сургалтын хуудас багшийн профайлын зургийг биш, сургалтынхаа зургийг өгөх ёстой")
	}

	// QR: PNG, хэмжээ хязгаартай, татах үед нэртэй.
	res, b := get("/c/" + cid + "/qr.png?size=99999&download=1")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/png" {
		t.Fatalf("qr.png: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil || cfg.Width > 1024 || cfg.Width < 256 {
		t.Fatalf("QR хэмжээ: %v %+v", err, cfg)
	}
	if !strings.Contains(res.Header.Get("Content-Disposition"), "qr.png") {
		t.Errorf("татах нэр: %q", res.Header.Get("Content-Disposition"))
	}
	if got := courseJoinURL("https://surgalt.mn", cid); got != "https://surgalt.mn/c/"+cid+"#join" {
		t.Errorf("QR холбоос: %q", got)
	}

	// og.jpg: 1200×630 JPEG.
	res, b = get("/c/" + cid + "/og.jpg")
	if res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/jpeg" {
		t.Fatalf("og.jpg: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	jc, err := jpeg.DecodeConfig(bytes.NewReader(b))
	if err != nil || jc.Width != 1200 || jc.Height != 630 {
		t.Fatalf("og зургийн хэмжээ: %v %+v", err, jc)
	}

	// Ноорог, байхгүй сургалт: 404 (QR, зураг аль аль нь).
	for _, p := range []string{"/c/" + draft + "/qr.png", "/c/" + draft + "/og.jpg", "/c/000000000000000000000000/qr.png", "/c/000000000000000000000000/og.jpg"} {
		if res, _ := get(p); res.StatusCode != http.StatusNotFound {
			t.Errorf("%s: %d, 404 байх ёстой", p, res.StatusCode)
		}
	}
}
