package httpapi

import (
	"net/http"
	"strings"
	"time"

	"surgalt/internal/files"
)

// Суралцагчид очих видео, аудионы холбоос: бодит /files/... зам харагдахгүй, тасалбар (3 цаг, хэрэглэгчид
// холбосон) бөгөөд зөвхөн хуудсан доторх <video>/<audio> элементээс (Sec-Fetch-Dest: video|audio|empty)
// нээгдэнэ — холбоосыг хуулж шинэ таб, татагч программд оруулахад ажиллахгүй.

const mediaTicketTTL = 3 * time.Hour

func isStreamPath(p string) bool {
	m := mimeOf(strings.SplitN(p, "?", 2)[0])
	return strings.HasPrefix(m, "video/") || strings.HasPrefix(m, "audio/")
}

// viewerMedia — суралцагчид (эзэмшигч биш) очих медиа холбоос. Видео/аудио → тасалбар, бусад → гарын үсэгтэй URL.
func (s *Server) viewerMedia(path, uid string) string {
	if !strings.HasPrefix(path, "/files/") || !strings.Contains(path, "/"+files.Private+"/") || !isStreamPath(path) {
		return s.media(path)
	}
	ext := path[strings.LastIndexByte(path, '.'):]
	kind := "audio"
	if strings.HasPrefix(mimeOf(path), "video/") {
		kind = "video"
	}
	return "/api/media/" + s.files.Ticket(path, uid, mediaTicketTTL) + "/" + kind + ext
}

// handleMedia: GET /api/media/{ticket}/{name}
func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "private, no-store")
	path, _, ok := s.files.ResolveTicket(r.PathValue("ticket"))
	if !ok {
		writeErr(w, http.StatusForbidden, "холбоосны хугацаа дууссан эсвэл хүчингүй")
		return
	}
	// Хуудас болгон нээх (шинэ таб, хаягийн мөр) → хориглоно; <video>/<audio>-оос ирсэн хүсэлт л үйлчилнэ.
	if d := r.Header.Get("Sec-Fetch-Dest"); d != "" && d != "video" && d != "audio" && d != "empty" {
		writeErr(w, http.StatusForbidden, "видеог зөвхөн хичээл дотроос үзнэ")
		return
	}
	if r.Header.Get("Sec-Fetch-Mode") == "navigate" {
		writeErr(w, http.StatusForbidden, "видеог зөвхөн хичээл дотроос үзнэ")
		return
	}
	teacher, vis, name, ok := files.SplitPath(path)
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
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(2 * time.Hour))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Content-Type", mimeOf(name))
	http.ServeContent(w, r, "", fi.ModTime(), f)
}
