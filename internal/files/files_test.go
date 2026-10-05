package files

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

const tid = "0123456789abcdef01234567"

func newStore(t *testing.T) *Store {
	s, err := New(t.TempDir(), []byte("key"), 50<<20)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestImageBecomesWebPMax1000(t *testing.T) {
	s := newStore(t)
	img := image.NewRGBA(image.Rect(0, 0, 3000, 1500))
	for y := 0; y < 1500; y += 7 {
		for x := 0; x < 3000; x++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 99, 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	info, err := s.Save(tid, Public, "Миний зураг.png", &buf, 100<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(info.Name, ".webp") || info.ContentType != "image/webp" {
		t.Fatalf("webp болоогүй: %+v", info)
	}
	f, _, err := s.Open(tid, Public, info.Name)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, format, err := image.DecodeConfig(f)
	if err != nil || format != "webp" || cfg.Width != 1000 || cfg.Height != 500 {
		t.Fatalf("got %s %dx%d %v", format, cfg.Width, cfg.Height, err)
	}
}

func TestQuotaTypeAndTraversal(t *testing.T) {
	s := newStore(t)
	if _, err := s.Save(tid, Private, "a.exe", strings.NewReader("x"), 1<<20); !errors.Is(err, ErrType) {
		t.Fatal("exe хүлээн авагдлаа")
	}
	if _, err := s.Save(tid, Private, "big.pdf", bytes.NewReader(make([]byte, 2<<20)), 1<<20); !errors.Is(err, ErrQuota) {
		t.Fatalf("квот шалгагдаагүй: %v", err)
	}
	if _, err := s.Save("../../etc", Private, "a.pdf", strings.NewReader("x"), 1<<20); err == nil {
		t.Fatal("буруу teacherID")
	}
	if _, _, err := s.Open(tid, Private, "../../../etc/passwd"); err == nil {
		t.Fatal("path traversal")
	}
	entries, _ := os.ReadDir(s.root + "/teachers/" + tid + "/private")
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".upload") {
			t.Fatal("түр файл үлдсэн")
		}
	}
}

func TestSignedURL(t *testing.T) {
	s := newStore(t)
	info, err := s.Save(tid, Private, "Лекц 1.pdf", strings.NewReader("%PDF-1.4"), 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(info.URL)
	q := u.Query()
	if !s.Verify(u.EscapedPath(), q.Get("exp"), q.Get("sig")) {
		t.Fatal("хүчинтэй гарын үсэг татгалзагдлаа")
	}
	if s.Verify(u.EscapedPath()+"x", q.Get("exp"), q.Get("sig")) {
		t.Fatal("өөр зам хүлээн авагдлаа")
	}
	old := s.Resolve(info.Path, -time.Minute)
	ou, _ := url.Parse(old)
	if s.Verify(ou.EscapedPath(), ou.Query().Get("exp"), ou.Query().Get("sig")) {
		t.Fatal("хугацаа дууссан URL хүлээн авагдлаа")
	}
	if s.Resolve("https://youtube.com/x", time.Hour) != "https://youtube.com/x" {
		t.Fatal("гадаад URL өөрчлөгдсөн")
	}
}

func TestJPEGOrientation(t *testing.T) {
	// Хамгийн бага EXIF (Orientation=6) бүхий JPEG толгой
	exif := []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x22, 'E', 'x', 'i', 'f', 0, 0,
		'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0, 0, 0}
	if o := jpegOrientation(exif); o != 6 {
		t.Fatalf("orientation %d", o)
	}
	src := image.NewNRGBA(image.Rect(0, 0, 4, 2))
	if b := applyOrientation(src, 6).Bounds(); b.Dx() != 2 || b.Dy() != 4 {
		t.Fatal("90° эргээгүй")
	}
}
