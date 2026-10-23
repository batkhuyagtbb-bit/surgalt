package files

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	"image/draw"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
)

// «Цагаан самбар»-т оруулсан зураг: <root>/boards/<хичээл>/<32 hex>.webp.
// Нэр нь таамаглах боломжгүй тул нийтэд (удаан кэштэй) үйлчилнэ.
const (
	BoardImageSide = 2000 // урт талын дээд хэмжээ
	boardMaxImages = 3000 // нэг хичээлийн самбаруудад
)

var (
	boardNameRe   = regexp.MustCompile(`^[0-9a-f]{32}\.webp$`)
	ErrBoardFull  = errors.New("энэ хичээлийн самбарт хэт олон зураг орсон")
	errBoardImage = errors.New("самбарын зургийн нэр буруу")
)

func (s *Store) boardDir(lessonID string) (string, error) {
	if !idRe.MatchString(lessonID) {
		return "", ErrBadName
	}
	return filepath.Join(s.root, "boards", lessonID), nil
}

// SaveBoardImage нь зургийг (PNG/JPEG/WebP/GIF) шалгаж, урт талыг BoardImageSide хүртэл багасгаад
// тунгалаг хэсгийг (дэвсгэр арилгасан) хадгалсан WebP болгоно. Мета өгөгдөл (GPS г.м) үлдэхгүй.
func (s *Store) SaveBoardImage(lessonID string, raw []byte) (name string, w, h int, err error) {
	dir, err := s.boardDir(lessonID)
	if err != nil {
		return "", 0, 0, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
		return "", 0, 0, ErrImage
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", 0, 0, ErrImage
	}
	src = applyOrientation(src, jpegOrientation(raw))
	b := src.Bounds()
	w, h = b.Dx(), b.Dy()
	if m := max(w, h); m > BoardImageSide {
		w, h = max(1, w*BoardImageSide/m), max(1, h*BoardImageSide/m)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)

	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", 0, 0, err
	}
	if ents, err := os.ReadDir(dir); err == nil && len(ents) >= boardMaxImages {
		return "", 0, 0, ErrBoardFull
	}
	var rb [16]byte
	if _, err := rand.Read(rb[:]); err != nil {
		return "", 0, 0, err
	}
	name = hex.EncodeToString(rb[:]) + ".webp"
	tmp := filepath.Join(dir, name+".tmp")
	out, err := os.Create(tmp)
	if err != nil {
		return "", 0, 0, err
	}
	err = webp.Encode(out, dst, webp.Options{Quality: 88, Method: 4})
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, filepath.Join(dir, name))
	}
	if err != nil {
		os.Remove(tmp)
		return "", 0, 0, err
	}
	return name, w, h, nil
}

// OpenBoardImage нь самбарын зургийг нээнэ.
func (s *Store) OpenBoardImage(lessonID, name string) (*os.File, fs.FileInfo, error) {
	dir, err := s.boardDir(lessonID)
	if err != nil {
		return nil, nil, err
	}
	if !boardNameRe.MatchString(name) {
		return nil, nil, errBoardImage
	}
	f, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return nil, nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, fi, nil
}
