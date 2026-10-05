package httpapi

import (
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"

	"surgalt/internal/files"
)

func (s *Server) filesErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, files.ErrType), errors.Is(err, files.ErrBadName), errors.Is(err, files.ErrBadVisibility):
		writeErr(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, files.ErrQuota), errors.Is(err, files.ErrTooLarge):
		writeErr(w, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, files.ErrNotFound):
		writeErr(w, http.StatusNotFound, err.Error())
	default:
		s.log.Error("files", "err", err)
		writeErr(w, http.StatusInternalServerError, "файлын алдаа")
	}
}

// handleUpload: multipart "file" талбар, ?visibility=private|public.
// Том видео ч урсгалаар шууд дискэнд бичигдэнэ.
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Now().Add(2 * time.Hour))
	_ = rc.SetWriteDeadline(time.Now().Add(2 * time.Hour))
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	quota := s.quotaOf(u)
	visibility := r.URL.Query().Get("visibility")
	if visibility == "" {
		visibility = files.Private
	}
	mr, err := r.MultipartReader()
	if err != nil {
		writeErr(w, http.StatusBadRequest, "multipart/form-data хэлбэрээр илгээнэ үү")
		return
	}
	for {
		part, err := mr.NextPart()
		if err != nil {
			writeErr(w, http.StatusBadRequest, `"file" талбар олдсонгүй`)
			return
		}
		if part.FormName() != "file" || part.FileName() == "" {
			part.Close()
			continue
		}
		info, err := s.files.Save(c.UID, visibility, part.FileName(), part, quota)
		part.Close()
		if err != nil {
			s.filesErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, info)
		return
	}
}

func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	list, err := s.files.List(c.UID)
	if err != nil {
		s.filesErr(w, err)
		return
	}
	u, err := s.store.UserByID(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	used, _ := s.files.Usage(c.UID)
	writeJSON(w, http.StatusOK, map[string]any{"files": list, "used": used, "quota": s.quotaOf(u)})
}

func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	if err := s.files.Delete(c.UID, r.PathValue("visibility"), r.PathValue("name")); err != nil {
		s.filesErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleServeFile: /files/{teacher}/{visibility}/{name}
// public — хэн ч; private — зөвхөн хүчинтэй exp+sig-тэй. Range дэмжинэ (видео гүйлгэх).
func (s *Server) handleServeFile(w http.ResponseWriter, r *http.Request) {
	teacher, vis, name := r.PathValue("teacher"), r.PathValue("visibility"), r.PathValue("name")
	if vis == files.Private {
		q := r.URL.Query()
		if !s.files.Verify(r.URL.EscapedPath(), q.Get("exp"), q.Get("sig")) {
			writeErr(w, http.StatusForbidden, "холбоосны хугацаа дууссан эсвэл хүчингүй")
			return
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=86400")
	}
	f, fi, err := s.files.Open(teacher, vis, name)
	if err != nil {
		writeErr(w, http.StatusNotFound, "файл олдсонгүй")
		return
	}
	defer f.Close()
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(2 * time.Hour))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !strings.HasPrefix(mimeOf(name), "video/") && !strings.HasPrefix(mimeOf(name), "audio/") &&
		!strings.HasPrefix(mimeOf(name), "image/") && mimeOf(name) != "application/pdf" {
		w.Header().Set("Content-Disposition", "attachment")
	}
	http.ServeContent(w, r, name, fi.ModTime(), f)
}

// media нь хичээлийн дотоод файлын замыг хугацаатай URL болгоно.
func (s *Server) media(path string) string { return s.files.Resolve(path, 6*time.Hour) }

func mimeOf(name string) string {
	i := strings.LastIndexByte(name, '.')
	if i < 0 {
		return ""
	}
	return mimeByExt(strings.ToLower(name[i:]))
}

func mimeByExt(ext string) string { return mime.TypeByExtension(ext) }
