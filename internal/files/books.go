package files

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	_ "golang.org/x/image/webp"
)

// Номын хуудсууд: <root>/books/<teacher>/<book>/p00001.img — файлын сангийн жагсаалтад
// харагдахгүй, нээлттэй URL-гүй. Зөвхөн сервер уншигчийн тэмдэгтэйгээр өгнө.

const (
	MaxBookPages     = 3000
	MaxBookPageBytes = 6 << 20
	maxBookPageSide  = 4000
)

var ErrBadPage = errors.New("хуудасны зураг буруу")

func (s *Store) bookDir(teacherID, bookID string) (string, error) {
	if !idRe.MatchString(teacherID) || !idRe.MatchString(bookID) {
		return "", ErrBadName
	}
	return filepath.Join(s.root, "books", teacherID, bookID), nil
}

func pageName(n int) string { return fmt.Sprintf("p%05d.img", n) }

// SaveBookPage нь n-р хуудсын зургийг (JPEG/PNG/WebP) шалгаж хадгална.
func (s *Store) SaveBookPage(teacherID, bookID string, n int, r io.Reader) error {
	if n < 1 || n > MaxBookPages {
		return ErrBadPage
	}
	dir, err := s.bookDir(teacherID, bookID)
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(io.LimitReader(r, MaxBookPageBytes+1))
	if err != nil {
		return err
	}
	if len(raw) > MaxBookPageBytes {
		return fmt.Errorf("%w: хуудас 6MB-аас их", ErrBadPage)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width < 50 || cfg.Height < 50 || cfg.Width > maxBookPageSide || cfg.Height > maxBookPageSide {
		return ErrBadPage
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	tmp := filepath.Join(dir, pageName(n)+".tmp")
	if err := os.WriteFile(tmp, raw, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, pageName(n)))
}

// BookPage нь хадгалсан хуудасны түүхий зургийг буцаана.
func (s *Store) BookPage(teacherID, bookID string, n int) ([]byte, error) {
	dir, err := s.bookDir(teacherID, bookID)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(dir, pageName(n)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return b, err
}

// BookPagesReady нь 1..total хуудас бүгд байгаа эсэхийг шалгаж, илүүдлийг устгана.
func (s *Store) BookPagesReady(teacherID, bookID string, total int) error {
	dir, err := s.bookDir(teacherID, bookID)
	if err != nil {
		return err
	}
	for n := 1; n <= total; n++ {
		if _, err := os.Stat(filepath.Join(dir, pageName(n))); err != nil {
			return fmt.Errorf("%d-р хуудас дутуу байна", n)
		}
	}
	for n := total + 1; n <= MaxBookPages; n++ {
		if os.Remove(filepath.Join(dir, pageName(n))) != nil {
			break
		}
	}
	return nil
}

// DeleteBookPages нь номын бүх хуудсыг устгана.
func (s *Store) DeleteBookPages(teacherID, bookID string) error {
	dir, err := s.bookDir(teacherID, bookID)
	if err != nil {
		return err
	}
	return os.RemoveAll(dir)
}
