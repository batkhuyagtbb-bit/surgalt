package httpapi

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---- Content-Security-Policy: XSS гарсан ч халдагчийн скрипт ажиллахгүй (нэвтрэх токен localStorage-д) ----
// Скрипт зөвхөн манай серверээс, доорх CDN-ийн тодорхой сангуудаас, эсвэл SHA-256 хэшээр зөвшөөрсөн inline скрипт.
// Inline event handler (onclick=…), javascript: холбоос, eval ажиллахгүй. Зөрчлийг /api/csp-report логт бичнэ.
// Багшийн HTML embed — тусдаа /embed-frame (sandbox, өөрийн CSP) дотор; эх хуудасны CSP-д хамаарахгүй.
// CSP_MODE=report — зөвхөн мэдээлнэ (хаахгүй), off — унтраана.

// inlineScripts — загварт {{inlineScript "нэр"}}-ээр орно; хэш нь яг энэ текстээс тооцогдох тул хэзээ ч зөрөхгүй.
// Загварын утга (хуудасны нэр г.м)-ыг энд биш, HTML атрибутаас уншина.
var inlineScripts = map[string]string{
	// Нэвтэрсэн хэрэглэгчид: цэс, чатын самбар JS-ээр солигдох тул угсрагдаж дуустал нууна (app.js зөөлөн гаргана),
	// самбарын зайг хуудас зурагдахаас өмнө бэлдэнэ. Зочинд серверийн HTML бэлэн — нуухгүй, шууд харагдана.
	"boot": `try{if(localStorage.getItem("sg_token")){var h=document.documentElement;h.classList.add("authed","booting");if(localStorage.getItem("sg_user")&&["home","profile","course","me"].indexOf(h.getAttribute("data-pg"))>=0){h.classList.add("has-rail","rail-boot");if(localStorage.getItem("sg_rail_collapsed")==="1")h.classList.add("rail-collapsed")}}}catch(e){}`,
	// Утсан дээрх туршилтын төлбөр (pay.html): хаяг нь товчны data-url-д.
	"pay": `(function(){var b=document.getElementById("ppPay");if(!b)return;b.onclick=async function(){b.disabled=true;b.textContent="Төлж байна…";try{var r=await fetch(b.dataset.url,{method:"POST"});if(!r.ok)throw new Error();document.getElementById("ppState").innerHTML='<p class="pp-ok"><b>✓</b> Төлбөр амжилттай төлөгдлөө.<br>Компьютер дээрх хуудас өөрөө нээгдэнэ.</p>'}catch(e){b.disabled=false;b.textContent="Дахин оролдох"}}})();`,
}

func inlineScript(name string) template.HTML {
	return template.HTML("<script>" + inlineScripts[name] + "</script>")
}

// cspScripts — гадны скриптүүд (app.js, intro2.html-ийн хаягтай яг таарна; csp_test.go шалгана). Замаар хязгаарласан тул
// CDN дээрх бусад сан (CSP-г тойрох «gadget») ачаалагдахгүй.
var cspScripts = []string{
	"https://cdnjs.cloudflare.com/ajax/libs/pdf.js/4.10.38/",
	"https://cdnjs.cloudflare.com/ajax/libs/xlsx/0.18.5/",
	"https://cdnjs.cloudflare.com/ajax/libs/jszip/3.10.1/",
	"https://cdnjs.cloudflare.com/ajax/libs/html2canvas/1.4.1/",
	"https://cdnjs.cloudflare.com/ajax/libs/gsap/3.12.5/",
	"https://cdn.jsdelivr.net/npm/docx-preview@0.3.5/",
	"https://cdn.jsdelivr.net/npm/pptx-preview@1.0.7/",
	"https://cdn.jsdelivr.net/npm/hot-formula-parser@4.0.0/",
	"https://cdn.jsdelivr.net/npm/three@0.170.0/",
	"https://www.youtube.com/iframe_api",
	"https://www.youtube.com/s/player/",
}

var inlineTagRe = regexp.MustCompile(`(?s)<script>(.*?)</script>`)

func scriptHash(js string) string {
	sum := sha256.Sum256([]byte(js))
	return "'sha256-" + base64.StdEncoding.EncodeToString(sum[:]) + "'"
}

// cspHashes — inlineScripts болон template-гүй static/intro2.html-ийн inline скриптүүдийн хэш.
func cspHashes() []string {
	var out []string
	for _, js := range inlineScripts {
		out = append(out, scriptHash(js))
	}
	if b, err := staticFS.ReadFile("static/intro2.html"); err == nil {
		for _, m := range inlineTagRe.FindAllSubmatch(b, -1) {
			out = append(out, scriptHash(string(m[1])))
		}
	}
	return out
}

func buildCSP(frameAncestors string) string {
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' " + strings.Join(cspHashes(), " ") + " 'wasm-unsafe-eval' " + strings.Join(cspScripts, " "),
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"img-src 'self' data: blob: https:",
		"media-src 'self' data: blob: https:",
		"font-src 'self' data: blob: https://fonts.gstatic.com",
		"connect-src 'self' https: wss: blob: data:",
		"frame-src 'self' https: blob:",
		"child-src 'self' https: blob:",
		"worker-src 'self' blob:",
		"manifest-src 'self'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self' https:",
		"frame-ancestors " + frameAncestors,
		"report-uri /api/csp-report",
	}, "; ")
}

var (
	cspPage  = buildCSP("'self'")
	cspAdmin = buildCSP("'none'")
	// Багшийн HTML embed: sandbox (allow-same-origin-гүй — сайтын токен, өгөгдөлд хүрэхгүй), дотор нь юу ч ажиллана.
	// Шууд нээсэн ч sandbox хэвээр (CSP sandbox), зөвхөн манай хуудас iframe-д оруулна.
	cspEmbed = "sandbox allow-scripts allow-forms allow-presentation; default-src * data: blob: 'unsafe-inline' 'unsafe-eval'; frame-ancestors 'self'"
)

// cspHeader — CSP_MODE-оос хамаарсан толгойн нэр; "" бол CSP тавихгүй.
func (s *Server) cspHeader() string {
	switch s.cfg.CSPMode {
	case "off":
		return ""
	case "report":
		return "Content-Security-Policy-Report-Only"
	}
	return "Content-Security-Policy"
}

// setCSP — хуудас өөрийн бодлогыг тавина (cspMW-ийн анхдагчийг орлоно).
func (s *Server) setCSP(w http.ResponseWriter, policy string) {
	if h := s.cspHeader(); h != "" {
		w.Header().Set(h, policy)
	}
}

// cspMW — HTML хариунд анхдагч бодлого (API, static, WebSocket-д хамаарахгүй; PDF, видео зэрэг файлыг хөндөхгүй).
func (s *Server) cspMW(next http.Handler) http.Handler {
	hdr := s.cspHeader()
	if hdr == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/static/") || r.Header.Get("Upgrade") != "" {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(&cspWriter{ResponseWriter: w, hdr: hdr}, r)
	})
}

type cspWriter struct {
	http.ResponseWriter
	hdr  string
	done bool
}

func (c *cspWriter) set(body []byte) {
	if c.done {
		return
	}
	c.done = true
	h := c.ResponseWriter.Header()
	if h.Get("Content-Security-Policy") != "" || h.Get("Content-Security-Policy-Report-Only") != "" {
		return
	}
	ct := h.Get("Content-Type")
	if ct == "" && h.Get("Content-Encoding") == "" && body != nil {
		ct = http.DetectContentType(body)
	}
	if strings.HasPrefix(ct, "text/html") {
		h.Set(c.hdr, cspPage)
	}
}

func (c *cspWriter) WriteHeader(code int) { c.set(nil); c.ResponseWriter.WriteHeader(code) }
func (c *cspWriter) Write(b []byte) (int, error) {
	c.set(b)
	return c.ResponseWriter.Write(b)
}
func (c *cspWriter) Unwrap() http.ResponseWriter { return c.ResponseWriter }

// embedFrameHTML — эх хуудас postMessage-ээр embed-ийн HTML-ийг илгээнэ ({sgEmbed}), энэ хуудас түүгээр өөрийгөө солино.
// document.open() сонсогчийг арилгах тул нэг л удаа.
const embedFrameHTML = `<!doctype html><html><head><meta charset="utf-8"><meta name="robots" content="noindex"><script>addEventListener("message",function(e){var d=e.data;if(e.source!==parent||!d||typeof d.sgEmbed!=="string")return;document.open();document.write(d.sgEmbed);document.close()});parent.postMessage({sgEmbedReady:1},"*")</script></head><body></body></html>`

// pageEmbedFrame: GET /embed-frame — багшийн HTML embed-ийн sandbox хуудас (агуулга эх хуудаснаас ирнэ, энд өгөгдөл алга).
func (s *Server) pageEmbedFrame(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "public, max-age=86400")
	h.Set("X-Robots-Tag", "noindex")
	h.Set("Content-Security-Policy", cspEmbed) // report горимд ч sandbox заавал
	_, _ = io.WriteString(w, embedFrameHTML)
}

// cspLog — нэг төрлийн зөрчлийг 10 минутад нэг л удаа логт (хөтчийн өргөтгөл, бот олноор илгээж болно).
var cspLog = struct {
	sync.Mutex
	seen map[string]time.Time
}{seen: map[string]time.Time{}}

// handleCSPReport: POST /api/csp-report — хөтөч CSP зөрчлийг илгээнэ (application/csp-report эсвэл reports+json).
func (s *Server) handleCSPReport(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 16<<10))
	w.WriteHeader(http.StatusNoContent)
	var reps []map[string]any
	var one struct {
		Report map[string]any `json:"csp-report"`
	}
	if json.Unmarshal(body, &one) == nil && one.Report != nil {
		reps = append(reps, one.Report)
	} else {
		var list []struct {
			Body map[string]any `json:"body"`
		}
		if json.Unmarshal(body, &list) == nil {
			for _, x := range list {
				if x.Body != nil {
					reps = append(reps, x.Body)
				}
			}
		}
	}
	str := func(m map[string]any, keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k].(string); ok && v != "" {
				return v
			}
			if v, ok := m[k].(float64); ok {
				return strconv.FormatFloat(v, 'f', -1, 64)
			}
		}
		return ""
	}
	for _, rep := range reps {
		dir := str(rep, "effective-directive", "effectiveDirective", "violated-directive")
		blocked := str(rep, "blocked-uri", "blockedURL")
		doc := str(rep, "document-uri", "documentURL")
		if i := strings.IndexAny(doc, "?#"); i >= 0 {
			doc = doc[:i]
		}
		if src := str(rep, "source-file", "sourceFile"); strings.Contains(blocked+" "+src, "-extension://") {
			continue // хөтчийн өргөтгөл (манай кодын асуудал биш)
		}
		key := dir + "|" + blocked + "|" + doc
		now := time.Now()
		cspLog.Lock()
		last, dup := cspLog.seen[key]
		if !dup || now.Sub(last) > 10*time.Minute {
			cspLog.seen[key] = now
		}
		if len(cspLog.seen) > 2000 {
			cspLog.seen = map[string]time.Time{}
		}
		cspLog.Unlock()
		if dup && now.Sub(last) <= 10*time.Minute {
			continue
		}
		s.log.Warn("CSP зөрчил", "directive", dir, "blocked", blocked, "page", doc,
			"source", str(rep, "source-file", "sourceFile"), "line", str(rep, "line-number", "lineNumber"),
			"sample", str(rep, "script-sample", "sample"), "disposition", str(rep, "disposition"))
	}
}
