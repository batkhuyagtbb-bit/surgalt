package httpapi

import (
	"bytes"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/klauspost/compress/gzhttp"
	"github.com/klauspost/compress/gzip"
)

// ---- Шахалт ----
// IIS proxy нь application/json-ийг шахдаггүй тул API-ийн хариу 3–5 дахин том явж байв. JSON, HTML, текстийг энд
// gzip-ээр шахна (1KB-аас бага бол шахахгүй). Static файлыг staticHandler нэг удаа шахаж санах ойд хадгалдаг;
// Content-Encoding тавигдсан хариуг gzhttp дахин шахахгүй. Медиа (видео, аудио, PDF, зураг) шахагдахгүй.

var compressTypes = []string{
	"application/json", "text/html", "text/plain", "text/css", "text/csv", "text/xml", "application/xml",
	"application/javascript", "text/javascript", "image/svg+xml", "application/manifest+json",
}

func compressMW(next http.Handler) http.Handler {
	wrap, err := gzhttp.NewWrapper(gzhttp.MinSize(1024), gzhttp.CompressionLevel(5), gzhttp.ContentTypes(compressTypes))
	if err != nil {
		return next
	}
	gz := wrap(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "" { // WebSocket — шууд дамжуулна
			next.ServeHTTP(w, r)
			return
		}
		gz.ServeHTTP(w, r)
	})
}

// gzStatic — static файлын шахсан хувилбар (анх хүсэхэд нэг удаа шахаж, процессын туршид хадгална).
type gzStatic struct {
	fsys fs.FS
	m    sync.Map // нэр → *gzFile
}

type gzFile struct {
	once sync.Once
	data []byte // nil бол шахах боломжгүй/шаардлагагүй
}

var gzExt = map[string]bool{".js": true, ".css": true, ".json": true, ".svg": true, ".html": true, ".txt": true, ".xml": true, ".map": true}

func (g *gzStatic) get(name string) []byte {
	if !gzExt[path.Ext(name)] {
		return nil
	}
	v, _ := g.m.LoadOrStore(name, &gzFile{})
	f := v.(*gzFile)
	f.once.Do(func() {
		raw, err := fs.ReadFile(g.fsys, name)
		if err != nil || len(raw) < 1024 {
			return
		}
		var b bytes.Buffer
		zw, _ := gzip.NewWriterLevel(&b, gzip.BestCompression)
		_, _ = zw.Write(raw)
		if zw.Close() == nil && b.Len() < len(raw) {
			f.data = b.Bytes()
		}
	})
	return f.data
}

// serve нь хөтөч gzip хүлээн авдаг бол шахсан хувилбарыг өгнө; үгүй бол false.
func (g *gzStatic) serve(w http.ResponseWriter, r *http.Request, name string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if !acceptsGzip(r) {
		return false
	}
	data := g.get(name)
	if data == nil {
		return false
	}
	h := w.Header()
	if ct := mime.TypeByExtension(path.Ext(name)); ct != "" {
		h.Set("Content-Type", ct)
	}
	h.Set("Content-Encoding", "gzip")
	h.Add("Vary", "Accept-Encoding")
	h.Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(data)
	}
	return true
}

func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		enc, q, _ := strings.Cut(strings.TrimSpace(part), ";")
		if strings.EqualFold(strings.TrimSpace(enc), "gzip") {
			if q = strings.TrimSpace(q); strings.HasPrefix(q, "q=") {
				v, err := strconv.ParseFloat(q[2:], 64)
				return err != nil || v > 0
			}
			return true
		}
	}
	return false
}
