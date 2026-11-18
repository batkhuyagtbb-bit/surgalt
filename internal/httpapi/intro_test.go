package httpapi

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// Танилцуулга видео дискнээс (IntroDir): Range-ээр хэсэгчлэн, video/mp4; буруу нэр, хавтаснаас гарах оролдлого дискийг уншихгүй.
func TestIntroDisk(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tour-hd.mp4"), []byte("0123456789abcdef"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(dir), "secret.mp4"), []byte("x"), 0o644); err == nil {
		defer os.Remove(filepath.Join(filepath.Dir(dir), "secret.mp4"))
	}
	h := staticHandler(false, dir, slog.Default())
	req := httptest.NewRequest("GET", "/static/intro/tour-hd.mp4?v=3", nil)
	req.Header.Set("Range", "bytes=4-7")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusPartialContent || rec.Body.String() != "4567" || rec.Header().Get("Content-Type") != "video/mp4" || rec.Header().Get("Content-Range") != "bytes 4-7/16" {
		t.Fatalf("range: %d %q %q %q", rec.Code, rec.Body.String(), rec.Header().Get("Content-Type"), rec.Header().Get("Content-Range"))
	}
	for _, p := range []string{"/static/intro/..%2fsecret.mp4", "/static/intro/../secret.mp4", "/static/intro/Tour.MP4", "/static/intro/none.mp4"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code == http.StatusOK || rec.Code == http.StatusPartialContent {
			t.Fatalf("%s: %d — дискнээс уншиж болохгүй", p, rec.Code)
		}
	}
	// embed-ийн бусад файл хэвээр
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/static/ar.js", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("ar.js: %d", rec.Code)
	}
}
