package httpapi

import (
	"embed"
	"hash/fnv"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"surgalt/internal/store"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// assetVersion нь статик файлуудын агуулгын хэш. Файл өөрчлөгдөх бүрт автоматаар солигдож,
// "immutable" кэштэй браузер шинэ хувилбарыг татна (гараар дугаар нэмэхээ мартах эрсдэлгүй).
var assetVersion = hashAssets()

func hashAssets() string {
	h := fnv.New64a()
	_ = fs.WalkDir(staticFS, "static", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := staticFS.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(b)
		return nil
	})
	return strconv.FormatUint(h.Sum64(), 36)
}

func money(n int64) string {
	if n == 0 {
		return "Үнэгүй"
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String() + "₮"
}

func initials(name string) string {
	var out []rune
	for _, f := range strings.Fields(name) {
		for _, r := range f {
			out = append(out, unicode.ToUpper(r))
			break
		}
		if len(out) == 2 {
			break
		}
	}
	if len(out) == 0 {
		return "?"
	}
	return string(out)
}

// brandHues нь логоны хар хөхөд зохицсон хөхийн сүүдрүүд (улбар шар нь зөвхөн онцлох өнгө).
// Карт, дүрс бүр эндээс л өнгө авна — сайт бүхэлдээ нэг өнгөний системтэй харагдана.
var brandHues = []int{222, 216, 228, 212, 232, 219, 225, 214}

// hue нь гарчгаас тогтвортой өнгө гаргаж, картын зургийг үүсгэнэ.
func hue(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return brandHues[int(h.Sum32()%uint32(len(brandHues)))]
}

// iconPaths: нэг хэв маягийн шугаман дүрсүүд (24x24, stroke). Emoji-оос ялгаатай нь бүх төхөөрөмжид ижил харагдана.
var iconPaths = map[string]string{
	"pin":     `<path d="M12 21s7-6.2 7-11.5a7 7 0 1 0-14 0C5 14.800 12 21 12 21Z"/><circle cx="12" cy="9.500" r="2.500"/>`,
	"work":    `<rect x="3" y="7" width="18" height="13" rx="2"/><path d="M9 7V5a2 2 0 0 1 2-2h2a2 2 0 0 1 2 2v2M3 13h18"/>`,
	"cal":     `<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M8 3v4M16 3v4M3 10h18"/>`,
	"users":   `<circle cx="9" cy="8" r="3.500"/><path d="M2.500 20c0-3.500 3-5.500 6.500-5.500s6.500 2 6.500 5.500"/><path d="M16 4.600a3.500 3.500 0 0 1 0 6.800M18 14.700c2 .7 3.500 2.400 3.500 5.300"/>`,
	"courses": `<path d="m2 9 10-5 10 5-10 5z"/><path d="M6 11.500V16c0 1.200 2.700 3 6 3s6-1.800 6-3v-4.500"/>`,
	"book":    `<path d="M4 5a2 2 0 0 1 2-2h14v16H6a2 2 0 0 0-2 2z"/><path d="M4 21V5M9 7h7"/>`,
	"play":    `<circle cx="12" cy="12" r="9"/><path d="m10 8.500 5.500 3.500-5.500 3.500z"/>`,
	"live":    `<rect x="3" y="6" width="12" height="12" rx="2"/><path d="m15 10 6-3v10l-6-3z"/>`,
	"chat":    `<path d="M21 12a8 8 0 0 1-11.600 7.100L4 20l1-4.600A8 8 0 1 1 21 12Z"/>`,
	"phone":   `<path d="M5 4h4l2 5-2.500 1.500a11 11 0 0 0 5 5L15 13l5 2v4a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2"/>`,
	"qr":      `<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><path d="M14 14h3v3h-3zM20 14v.01M14 20v.01M17 20h4v-3"/>`,
	"fb":      `<path d="M15 3h-2.500A3.500 3.500 0 0 0 9 6.500V10H6.500v3.500H9V21h3.500v-7.500H15l.600-3.500h-3.100V7.200c0-.700.400-1.100 1.100-1.100H15Z"/>`,
	"msgr":    `<path d="M12 3C7 3 3 6.700 3 11.300c0 2.600 1.300 4.900 3.300 6.400V21l3-1.700c.900.300 1.800.400 2.700.400 5 0 9-3.700 9-8.400S17 3 12 3Z"/><path d="m7.500 13.600 3.100-3.300 2.300 2.300 3.600-2.600-3.100 3.300-2.300-2.300Z"/>`,
	"share":   `<path d="M4 12v7a1 1 0 0 0 1 1h14a1 1 0 0 0 1-1v-7"/><path d="M12 15V3M8 7l4-4 4 4"/>`,
	"edit":    `<path d="M4 20h4L19 9l-4-4L4 16z"/><path d="m13.500 6.500 4 4"/>`,
	"user":    `<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 3.600-6 8-6s8 2 8 6"/>`,
	"layers":  `<path d="m12 3 9 5-9 5-9-5z"/><path d="m3 13 9 5 9-5"/>`,
	"camera":  `<path d="M4 8h3l1.500-2.500h7L17 8h3a1 1 0 0 1 1 1v9a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V9a1 1 0 0 1 1-1Z"/><circle cx="12" cy="13" r="3.500"/>`,
	"search":  `<circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/>`,
	"eye":     `<path d="M2 12s3.600-7 10-7 10 7 10 7-3.600 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>`,
	"check":   `<path d="m5 12.500 4.500 4.500L19 7.500"/>`,
	"plus":    `<path d="M12 5v14M5 12h14"/>`,
	"arrow":   `<path d="M5 12h14"/><path d="m13 6 6 6-6 6"/>`,
	"down":    `<path d="M12 4v12M7 11l5 5 5-5M5 20h14"/>`,
	"link":    `<path d="M10 14a4 4 0 0 0 5.700 0l3-3a4 4 0 0 0-5.700-5.700l-1 1"/><path d="M14 10a4 4 0 0 0-5.700 0l-3 3a4 4 0 0 0 5.700 5.700l1-1"/>`,
	"lock":    `<rect x="4.500" y="10.500" width="15" height="10" rx="2"/><path d="M8 10.500V7a4 4 0 0 1 8 0v3.500"/>`,
	"clock":   `<circle cx="12" cy="12" r="8.500"/><path d="M12 7.500V12l3 2"/>`,
	"card":    `<rect x="2.500" y="5" width="19" height="14" rx="2"/><path d="M2.500 10h19M6.500 15h4"/>`,
	"shield":  `<path d="M12 3 4.500 6v5.500c0 4.600 3.200 8.300 7.500 9.500 4.300-1.200 7.500-4.900 7.500-9.500V6L12 3Z"/><path d="m9 12 2 2 4-4"/>`,
	"chart":   `<path d="M4 20V10M10 20V4M16 20v-7M22 20H2"/>`,
	"money":   `<rect x="3" y="6" width="18" height="12" rx="2"/><circle cx="12" cy="12" r="2.5"/><path d="M7 9v.01M17 15v.01"/>`,
}

// icon нь загварт <svg> оруулна. Нэр нь кодод тогтмол бичигддэг тул HTML аюулгүй.
func icon(name string, size ...int) template.HTML {
	n := 18
	if len(size) > 0 {
		n = size[0]
	}
	sz := strconv.Itoa(n)
	return template.HTML(`<svg data-ico="` + template.HTMLEscapeString(name) + `" viewBox="0 0 24 24" width="` + sz + `" height="` + sz +
		`" fill="none" stroke="currentColor" stroke-width="1.9" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + iconPaths[name] + `</svg>`)
}

// splitWords — «.split» гарчгийг серверт үгээр хуваана (app.js splitText-тэй ижил бүтэц, data-split="1"-тэй хамт):
// JS ажилласны дараа DOM солигдож, хуудас зурагдсаны дараа мөр үсрэхгүй.
func splitWords(s string) template.HTML {
	var b strings.Builder
	for i, w := range strings.Fields(s) {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(`<span class="w"><span style="--i:` + strconv.Itoa(i) + `">` + template.HTMLEscapeString(w) + `</span></span>`)
	}
	return template.HTML(b.String())
}

func parseTemplates() *template.Template {
	funcs := template.FuncMap{
		"money": money, "initials": initials, "hue": hue, "icon": icon, "v": func() string { return assetVersion }, "terms": func() string { return TermsVersion },
		"inc": func(i int) int { return i + 1 },
		// Сургалтын толгойн зураг (handlers_course.go coverOK): төрөл ба эх сурвалж.
		"coverKind": coverKind,
		"coverSrc": func(c string) string {
			switch coverKind(c) {
			case "tpl":
				return "/static/covers/" + c + ".svg"
			case "embed":
				return strings.TrimSpace(strings.TrimPrefix(c, coverEmbedPrefix))
			}
			return c
		},
		// recent: сүүлийн 14 хоногт үүссэн эсэх ("Шинэ" тэмдэг). Хуудас 30 секунд л кэшлэгдэнэ.
		"recent":   func(t time.Time) bool { return time.Since(t) < 14*24*time.Hour },
		"rfc":      func(t time.Time) string { return t.UTC().Format(time.RFC3339) },
		"fmtName":  func(k string) string { return store.LessonFormats[k] },
		"modeName": func(k string) string { return store.LessonModes[k] },
		"first": func(s string) string {
			for _, r := range s {
				return string(unicode.ToUpper(r))
			}
			return ""
		},
		// SEO (seo.go): бүтэцтэй өгөгдөл, гарчиг, тайлбар, баталгаажуулах код.
		"ldProfile": ldProfile, "ldCourse": ldCourse, "ldBook": ldBook, "ldHome": ldHome,
		"metaDesc": metaDesc, "courseDesc": courseDesc, "pageTitle": pageTitle, "abs": absURL,
		"courseOGv": func(pc PublicCourse) string { return courseOGVersion(&pc) },
		"gsv": func() string { return siteVerifyGoogle }, "bsv": func() string { return siteVerifyBing },
		"site": func() string { return siteURL }, "fbapp": func() string { return siteFBApp },
		"sub": func(a, b int) int { return a - b },
		"add": func(a, b int) int { return a + b },
		"dict": func(kv ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(kv); i += 2 {
				m[kv[i].(string)] = kv[i+1]
			}
			return m
		},
		"inlineScript": inlineScript, // CSP-д хэшээр зөвшөөрсөн inline скрипт (csp.go)
		"splitWords":   splitWords,
	}
	return template.Must(template.New("").Funcs(funcs).ParseFS(templateFS, "templates/*.html"))
}

// introName — /static/intro/ доторх дискний файлын зөвшөөрөгдөх нэр (дэд хавтас, «..» орохгүй).
var introName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*.(mp4|webm|jpg)$`)

// introTypes — Windows-ийн бүртгэлээс хамаарахгүй тогтмол MIME (жишээ нь .webm → audio/webm болдог).
var introTypes = map[string]string{".mp4": "video/mp4", ".webm": "video/webm", ".jpg": "image/jpeg"}

// staticHandler — embed static. introDir: танилцуулга видео (HD, том файл) програмд шингээлгүй дискнээс
// (STORAGE_DIR/intro) /static/intro/<нэр>-д өгнө; дискэнд байхгүй бол embed хувилбар руу унана. Range дэмжинэ.
func staticHandler(minify bool, introDir string, log *slog.Logger) http.Handler {
	sub, _ := fs.Sub(staticFS, "static")
	if minify {
		sub = newMinFS(sub, log)
	}
	fsrv := http.StripPrefix("/static/", http.FileServerFS(sub))
	gz := &gzStatic{fsys: sub}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		if name, ok := strings.CutPrefix(r.URL.Path, "/static/intro/"); ok && introDir != "" && introName.MatchString(name) {
			if f, err := os.Open(filepath.Join(introDir, name)); err == nil {
				defer f.Close()
				if st, err := f.Stat(); err == nil && st.Mode().IsRegular() {
					w.Header().Set("Content-Type", introTypes[filepath.Ext(name)])
					http.ServeContent(w, r, name, st.ModTime(), f)
					return
				}
			}
		}
		if gz.serve(w, r, strings.TrimPrefix(r.URL.Path, "/static/")) {
			return
		}
		fsrv.ServeHTTP(w, r)
	})
}
