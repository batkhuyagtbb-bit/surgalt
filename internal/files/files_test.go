package files

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
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

// 6 минутаар хуваагдсан видео: эхний хэсэг жагсаалтад (нийт хэмжээ, хэсгийн тоотой), бусад нь нуугдмал;
// Parts бүх хэсгийн замыг өгнө; устгахад бүх хэсэг устана.
func TestVideoParts(t *testing.T) {
	s, err := New(t.TempDir(), []byte("k"), 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	tid := "6ac3c8afa622636405bd89b5"
	dir := filepath.Join(s.root, "teachers", tid, Private)
	_ = os.MkdirAll(dir, 0o755)
	names := []string{"0123456789abcdef__lec.webm", "0123456789abcdef__lec.p02.webm", "0123456789abcdef__lec.p03.webm"}
	for i, n := range names {
		_ = os.WriteFile(filepath.Join(dir, n), make([]byte, 100*(i+1)), 0o644)
	}
	b, _ := json.Marshal(names)
	_ = os.WriteFile(filepath.Join(dir, ".0123456789abcdef__lec.webm.parts"), b, 0o644)
	list, _ := s.List(tid)
	if len(list) != 1 || list[0].Parts != 3 || list[0].Size != 600 {
		t.Fatalf("жагсаалтад нэг мөр, 3 хэсэг, нийт 600 байт: %+v", list)
	}
	ps := s.Parts(list[0].Path)
	if len(ps) != 3 || !strings.HasSuffix(ps[2], "lec.p03.webm") {
		t.Fatalf("Parts: %v", ps)
	}
	if s.Parts("/files/"+tid+"/private/0123456789abcdef__other.webm") != nil {
		t.Fatal("хуваагдаагүй видео nil")
	}
	if err := s.Delete(tid, Private, names[0]); err != nil {
		t.Fatal(err)
	}
	left, _ := os.ReadDir(dir)
	if len(left) != 0 {
		t.Fatalf("бүх хэсэг устах ёстой: %v", left)
	}
}

// Хөрвүүлэх дүрэм: хуучин Office (.ppt/.doc …) үргэлж PDF; .pptx/.docx эх хэвээр (хичээл дотор шууд
// харагдана), ном оруулахад (officePDF) л PDF; Excel хэзээ ч хөрвүүлэхгүй (хүснэгт харагч).
func TestOfficeConversionRules(t *testing.T) {
	tc := &Transcoder{soffice: "/usr/bin/soffice"}
	cases := []struct {
		ext  string
		book bool
		want string
	}{
		{".ppt", false, ".pdf"}, {".doc", false, ".pdf"}, {".odp", false, ".pdf"},
		{".pptx", false, ""}, {".docx", false, ""}, {".xlsx", false, ""},
		{".pptx", true, ".pdf"}, {".docx", true, ".pdf"}, {".xlsx", true, ""},
	}
	for _, c := range cases {
		if got := tc.targetExt(c.ext, c.book); got != c.want {
			t.Fatalf("%s (ном=%v): %q, хүлээсэн %q", c.ext, c.book, got, c.want)
		}
	}
	if (&Transcoder{}).targetExt(".ppt", false) != "" {
		t.Fatal("LibreOffice байхгүй бол хөрвүүлэхгүй")
	}
}
