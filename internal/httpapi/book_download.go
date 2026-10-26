package httpapi

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Ном хоёр янзаар зарагдана: онлайн унших эрх (хуудас бүр уншигчийн нэртэй зураг) ба татаж авах эрх
// (эх файл + онлайн унших). Онлайн унших эрхтэй хүн татах эрх рүү зөрүүг төлж шилжинэ.

// bookDownload — эх файлыг татаж болох эсэх (багш өөрөө, үнэгүй татах, татах эрх худалдаж авсан).
func (s *Server) bookDownload(r *http.Request, b *store.Book, uid string) bool {
	switch {
	case uid == "" || b.FileURL == "":
		return false
	case uid == b.TeacherID:
		return true
	case !b.Download:
		return false
	case b.DownloadPrice == 0:
		return true
	}
	ok, err := s.store.HasBookDownload(r.Context(), uid, b.ID)
	return err == nil && ok
}

// bookDownloadPrice — татах эрхийн төлөх дүн: онлайн унших эрхээ худалдаж авсан бол зөрүү.
func (s *Server) bookDownloadPrice(r *http.Request, b *store.Book, uid string) int64 {
	if !b.Download {
		return 0
	}
	if uid != "" && b.Price > 0 {
		if ok, err := s.store.HasBookAccess(r.Context(), uid, b.ID); err == nil && ok {
			return max(0, b.DownloadPrice-b.Price)
		}
	}
	return b.DownloadPrice
}

// handleBookDownload: GET /api/books/{id}/download — татах эрхтэй хүнд эх файл (нэвтрэлтийн токентой авна,
// тиймээс холбоос нь тараагдахгүй). Лог: «Эх файлыг татсан».
func (s *Server) handleBookDownload(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	b, err := s.store.BookByID(r.Context(), r.PathValue("id"))
	if err != nil || (!b.Published && b.TeacherID != c.UID) {
		writeErr(w, http.StatusNotFound, "ном олдсонгүй")
		return
	}
	if !s.bookDownload(r, b, c.UID) {
		writeErr(w, http.StatusPaymentRequired, "Эх файлыг татахын тулд «Татаж авах эрх» худалдаж авна уу")
		return
	}
	if c.UID != b.TeacherID && !bookLim.allow(c.UID+"|dl|"+b.ID) {
		writeErr(w, http.StatusTooManyRequests, "Хэт олон удаа татлаа — түр хүлээгээд дахин оролдоно уу")
		return
	}
	teacher, vis, name, ok := files.SplitPath(b.FileURL)
	if !ok {
		writeErr(w, http.StatusNotFound, "файл олдсонгүй")
		return
	}
	f, fi, err := s.files.Open(teacher, vis, name)
	if err != nil {
		writeErr(w, http.StatusNotFound, "файл олдсонгүй")
		return
	}
	defer f.Close()
	s.logBook(r, b, c.UID, c.Name, "download", "", time.Hour)
	ext := strings.ToLower(path.Ext(name))
	fn := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>|`, r) || r < 32 {
			return '_'
		}
		return r
	}, b.Title) + ext
	h := w.Header()
	h.Set("Content-Disposition", `attachment; filename="book`+ext+`"; filename*=UTF-8''`+url.PathEscape(fn))
	h.Set("Cache-Control", "private, no-store")
	h.Set("X-Robots-Tag", "noindex")
	http.ServeContent(w, r, fn, fi.ModTime(), f)
}
