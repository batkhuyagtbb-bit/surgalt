package httpapi

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
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
	// Гадаад холбоос: YouTube/Vimeo — ID-г кодолсон "yt:"/"vm:" хэлбэрээр (жинхэнэ холбоос харагдахгүй);
	// бусад сайтын шууд видео/аудио файл — манай серверээр дамжуулж (proxy) холбоосыг нууна.
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if id := youTubeID(path); id != "" {
			return "yt:" + obfuscate(id)
		}
		if id := vimeoID(path); id != "" {
			return "vm:" + obfuscate(id)
		}
		if isStreamPath(path) && strings.HasPrefix(path, "https://") {
			ext := strings.ToLower(path[strings.LastIndexByte(strings.SplitN(path, "?", 2)[0], '.'):])
			kind := "audio"
			if strings.HasPrefix(mimeOf(strings.SplitN(path, "?", 2)[0]), "video/") {
				kind = "video"
			}
			return "/api/media/" + s.files.Ticket(path, uid, mediaTicketTTL) + "/" + kind + ext
		}
		return path
	}
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
	if strings.HasPrefix(path, "https://") { // гадаад видео: proxy (холбоос суралцагчид харагдахгүй)
		if d := r.Header.Get("Sec-Fetch-Dest"); (d != "" && d != "video" && d != "audio" && d != "empty") || r.Header.Get("Sec-Fetch-Mode") == "navigate" {
			writeErr(w, http.StatusForbidden, "видеог зөвхөн хичээл дотроос үзнэ")
			return
		}
		s.proxyMedia(w, r, path)
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

var (
	ytRe = regexp.MustCompile(`(?:youtube\.com/(?:watch\?v=|embed/|shorts/|live/)|youtu\.be/|youtube-nocookie\.com/embed/)([\w-]{11})`)
	vmRe = regexp.MustCompile(`vimeo\.com/(?:video/)?(\d+)`)
)

func youTubeID(u string) string {
	if m := ytRe.FindStringSubmatch(u); m != nil {
		return m[1]
	}
	return ""
}

func vimeoID(u string) string {
	if m := vmRe.FindStringSubmatch(u); m != nil {
		return m[1]
	}
	return ""
}

// obfuscate — ID-г хуулж шууд хайлт хийх боломжгүй хэлбэрт (урвуу + base64url). Нууцлал биш, илт харагдахаас сэргийлнэ.
func obfuscate(id string) string {
	b := []byte(id)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// mediaProxy — гадаад видео татах клиент: дотоод/хувийн хаяг руу хандахыг хориглоно (SSRF).
var mediaProxy = &http.Client{
	Timeout: 0,
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			for _, ip := range ips {
				if ip.IP.IsLoopback() || ip.IP.IsPrivate() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsUnspecified() || ip.IP.IsMulticast() {
					return nil, errors.New("хувийн хаяг руу хандахыг хориглосон")
				}
			}
			var d net.Dialer
			return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
		ResponseHeaderTimeout: 20 * time.Second,
		MaxIdleConnsPerHost:   16,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || req.URL.Scheme != "https" {
			return errors.New("redirect хориглосон")
		}
		return nil
	},
}

// proxyMedia — Range-ийг дамжуулж гадаад видеог урсгалаар өгнө.
func (s *Server) proxyMedia(w http.ResponseWriter, r *http.Request, src string) {
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, src, nil)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "видео нээгдсэнгүй")
		return
	}
	if rg := r.Header.Get("Range"); rg != "" {
		req.Header.Set("Range", rg)
	}
	req.Header.Set("User-Agent", "surgalt-media/1")
	res, err := mediaProxy.Do(req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "видео нээгдсэнгүй")
		return
	}
	defer res.Body.Close()
	ct := res.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "video/") && !strings.HasPrefix(ct, "audio/") && ct != "application/octet-stream" {
		writeErr(w, http.StatusUnsupportedMediaType, "видео биш файл")
		return
	}
	for _, h := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified", "ETag"} {
		if v := res.Header.Get(h); v != "" {
			w.Header().Set(h, v)
		}
	}
	w.Header().Set("Content-Disposition", "inline")
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(3 * time.Hour))
	w.WriteHeader(res.StatusCode)
	_, _ = io.Copy(w, res.Body)
}
