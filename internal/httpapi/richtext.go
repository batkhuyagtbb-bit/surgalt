package httpapi

import (
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	"surgalt/internal/files"
)

// Текстийн хэсгийн Word шиг агуулга (Block.HTML): багшийн засварлагчаас ирсэн HTML-ийг зөвшөөрсөн шошго,
// шинж, загвараар (style) цэвэрлэнэ. Скрипт, iframe, event шинж, javascript: холбоос бүгд хасагдана.
// Зураг: багшийн өөрийн сангийн файл (/files/…, гарын үсгийг хасаж хадгалж, үзүүлэхдээ дахин гарын үсэг зурна) эсвэл https холбоос.

const (
	maxRichBytes  = 300_000 // нэг текстийн хэсгийн HTML
	maxRichImages = 40
)

// richTags — зөвшөөрөгдсөн шошго → хадгалах нэр (h1 → h2, font → span г.м).
var richTags = map[string]string{
	"p": "p", "div": "div", "br": "br", "b": "b", "strong": "b", "i": "i", "em": "i", "u": "u", "ins": "u",
	"s": "s", "strike": "s", "del": "s", "sub": "sub", "sup": "sup", "mark": "mark", "span": "span", "font": "span",
	"h1": "h2", "h2": "h2", "h3": "h3", "h4": "h4", "h5": "h4", "h6": "h4",
	"ul": "ul", "ol": "ol", "li": "li", "blockquote": "blockquote", "pre": "pre", "code": "code",
	"a": "a", "img": "img", "hr": "hr", "figure": "figure", "figcaption": "figcaption",
	"table": "table", "thead": "thead", "tbody": "tbody", "tfoot": "tbody", "tr": "tr", "th": "th", "td": "td", "caption": "caption",
}

// richDrop — агуулгатай нь бүхэлд нь хасагдах шошго.
var richDrop = map[string]bool{"script": true, "style": true, "iframe": true, "object": true, "embed": true, "noscript": true,
	"template": true, "svg": true, "math": true, "head": true, "title": true, "textarea": true, "select": true, "button": true,
	"form": true, "audio": true, "video": true, "canvas": true, "frame": true, "frameset": true, "applet": true, "xml": true}

var richVoid = map[string]bool{"br": true, "img": true, "hr": true}

// richBreak — цэвэр текст гаргахад мөр таслах шошго.
var richBreak = map[string]bool{"p": true, "div": true, "br": true, "li": true, "h2": true, "h3": true, "h4": true, "tr": true,
	"blockquote": true, "pre": true, "table": true, "ul": true, "ol": true, "hr": true, "figure": true, "figcaption": true, "caption": true}

// richClasses — засварлагчийн зургийн байрлал (зүүн, баруун, голд).
var richClasses = map[string]bool{"rt-left": true, "rt-right": true, "rt-center": true}

var (
	cssColorRe = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|rgba?\(\s*\d{1,3}%?\s*,\s*\d{1,3}%?\s*,\s*\d{1,3}%?\s*(,\s*(0|1|0?\.\d{1,3})\s*)?\)|[a-zA-Z]{3,20})$`)
	cssSizeRe  = regexp.MustCompile(`^(\d{1,3}(\.\d{1,2})?(px|pt|em|rem|%)|xx-small|x-small|small|medium|large|x-large|xx-large|xxx-large|smaller|larger)$`)
	cssLenRe   = regexp.MustCompile(`^\d{1,4}(\.\d{1,2})?(px|em|rem|%)$`)
	cssDecoRe  = regexp.MustCompile(`^((underline|line-through|overline|none)\s*){1,3}$`)
	richSrcRe  = regexp.MustCompile(`src="(/files/[^"?]+)"`)
)

var cssEnum = map[string]map[string]bool{
	"text-align":     {"left": true, "center": true, "right": true, "justify": true, "start": true, "end": true},
	"font-weight":    {"bold": true, "bolder": true, "normal": true, "lighter": true, "100": true, "200": true, "300": true, "400": true, "500": true, "600": true, "700": true, "800": true, "900": true},
	"font-style":     {"italic": true, "normal": true, "oblique": true},
	"vertical-align": {"super": true, "sub": true, "baseline": true, "top": true, "middle": true, "bottom": true},
}

// cleanStyle — зөвшөөрсөн CSS шинжүүд (өнгө, хэмжээ, зэрэгцүүлэлт, догол, зургийн өргөн).
func cleanStyle(tag, style string) string {
	var out []string
	seen := map[string]bool{}
	for _, decl := range strings.Split(style, ";") {
		k, v, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "!important"))
		lv := strings.ToLower(v)
		if v == "" || len(v) > 60 || strings.ContainsAny(v, `\<>"`) || strings.Contains(lv, "url") || strings.Contains(lv, "expression") || strings.Contains(v, "/*") {
			continue
		}
		good := false
		switch k {
		case "background":
			k, good = "background-color", cssColorRe.MatchString(v)
		case "color", "background-color":
			good = cssColorRe.MatchString(v)
		case "font-size":
			good = cssSizeRe.MatchString(lv)
		case "text-decoration", "text-decoration-line":
			k, good = "text-decoration", cssDecoRe.MatchString(lv)
		case "margin-left", "padding-left", "text-indent":
			good = cssLenRe.MatchString(lv)
		case "width", "max-width":
			good = (tag == "img" || tag == "table" || tag == "td" || tag == "th") && cssLenRe.MatchString(lv)
		default:
			good = cssEnum[k][lv]
		}
		if good && !seen[k] {
			seen[k] = true
			out = append(out, k+":"+v)
		}
	}
	return strings.Join(out, ";")
}

// richURL — холбоос: зөвхөн http(s), mailto.
func richURL(u string) bool {
	l := strings.ToLower(u)
	return len(u) <= maxBlockURL && (strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "mailto:")) && !strings.ContainsAny(u, " \t\n<>\"")
}

type richEl struct{ src, out string } // out == "" — хасагдсан (хүүхдүүд нь үлдэнэ)

// sanitizeRich — HTML-ийг цэвэрлэж, цэвэр текст (хайлт, тойм, AI-д) болон зургийн тоог буцаана.
func sanitizeRich(src, teacherID string) (out, text string, imgs int) {
	var b, t strings.Builder
	var stack []richEl
	skip, skipDepth := "", 0
	z := html.NewTokenizer(strings.NewReader(src))
	brk := func() { t.WriteByte('\n') }
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		tok := z.Token()
		name := strings.ToLower(tok.Data)
		if skip != "" { // хасагдах шошгын доторх бүх зүйл
			switch {
			case tt == html.StartTagToken && name == skip:
				skipDepth++
			case tt == html.EndTagToken && name == skip:
				if skipDepth--; skipDepth == 0 {
					skip = ""
				}
			}
			continue
		}
		switch tt {
		case html.TextToken:
			b.WriteString(html.EscapeString(tok.Data))
			t.WriteString(tok.Data)
		case html.StartTagToken, html.SelfClosingTagToken:
			if richDrop[name] {
				if tt == html.StartTagToken {
					skip, skipDepth = name, 1
				}
				continue
			}
			tag := richTags[name]
			attrs := richAttrs(tag, tok.Attr, teacherID)
			if tag == "img" {
				if attrs == "" || imgs >= maxRichImages {
					continue
				}
				imgs++
			}
			if tag == "a" && !strings.Contains(attrs, "href=") {
				tag = ""
			}
			if richBreak[tag] {
				brk()
			} else if tag == "td" || tag == "th" {
				t.WriteByte(' ')
			}
			if tag != "" {
				b.WriteString("<" + tag + attrs + ">")
			}
			if tag == "" || !richVoid[tag] {
				if tt == html.StartTagToken && !richVoid[name] {
					stack = append(stack, richEl{name, tag})
				} else if tag != "" && !richVoid[tag] {
					b.WriteString("</" + tag + ">")
				}
			}
		case html.EndTagToken:
			i := len(stack) - 1
			for i >= 0 && stack[i].src != name {
				i--
			}
			if i < 0 {
				continue
			}
			for j := len(stack) - 1; j >= i; j-- {
				if o := stack[j].out; o != "" {
					b.WriteString("</" + o + ">")
					if richBreak[o] {
						brk()
					}
				}
			}
			stack = stack[:i]
		}
	}
	for j := len(stack) - 1; j >= 0; j-- {
		if o := stack[j].out; o != "" {
			b.WriteString("</" + o + ">")
		}
	}
	return b.String(), plainRich(t.String()), imgs
}

// richAttrs — шошго бүрийн зөвшөөрсөн шинжүүд (" k=\"v\"…"). img-д зөв src байхгүй бол хоосон.
func richAttrs(tag string, as []html.Attribute, teacherID string) string {
	if tag == "" {
		return ""
	}
	var b strings.Builder
	put := func(k, v string) { b.WriteString(" " + k + `="` + html.EscapeString(v) + `"`) }
	src := ""
	for _, a := range as {
		k, v := strings.ToLower(a.Key), strings.TrimSpace(a.Val)
		switch {
		case k == "style":
			if st := cleanStyle(tag, v); st != "" {
				put("style", st)
			}
		case k == "class":
			var cs []string
			for _, c := range strings.Fields(v) {
				if richClasses[c] {
					cs = append(cs, c)
				}
			}
			if len(cs) > 0 {
				put("class", strings.Join(cs, " "))
			}
		case k == "href" && tag == "a" && richURL(v):
			put("href", v)
			put("target", "_blank")
			put("rel", "noopener nofollow")
		case k == "src" && tag == "img":
			if p := stripSig(v); strings.HasPrefix(p, "/files/") && len(p) <= maxBlockURL && files.OwnedBy(p, teacherID) && !strings.Contains(p, "..") && !strings.ContainsAny(p, "\\\"<> ") {
				src = p
			} else if strings.HasPrefix(strings.ToLower(v), "https://") && richURL(v) {
				src = v
			}
		case k == "alt" && tag == "img":
			put("alt", truncRunes(v, 200))
		case (k == "colspan" || k == "rowspan") && (tag == "td" || tag == "th"):
			if n, err := strconv.Atoi(v); err == nil && n > 1 && n <= 20 {
				put(k, strconv.Itoa(n))
			}
		case k == "start" && tag == "ol":
			if n, err := strconv.Atoi(v); err == nil && n > 1 && n <= 10000 {
				put(k, strconv.Itoa(n))
			}
		case k == "color" && tag == "span" && cssColorRe.MatchString(v): // <font color>
			put("style", "color:"+v)
		}
	}
	if tag == "img" {
		if src == "" {
			return ""
		}
		return ` src="` + html.EscapeString(src) + `"` + b.String()
	}
	return b.String()
}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// plainRich — мөр бүрийг цэгцэлж, илүүдэл хоосон мөрийг хасна.
func plainRich(s string) string {
	s = strings.ReplaceAll(s, " ", " ")
	var lines []string
	blank := false
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.Join(strings.Fields(ln), " ")
		if ln == "" {
			if !blank && len(lines) > 0 {
				lines = append(lines, "")
			}
			blank = true
			continue
		}
		blank = false
		lines = append(lines, ln)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// richMedia — HTML доторх өөрийн сангийн зургийг гарын үсэгтэй холбоос болгоно.
func (s *Server) richMedia(h string) string {
	if !strings.Contains(h, `src="/files/`) {
		return h
	}
	return richSrcRe.ReplaceAllStringFunc(h, func(m string) string {
		p := html.UnescapeString(richSrcRe.FindStringSubmatch(m)[1])
		return `src="` + html.EscapeString(s.media(p)) + `"`
	})
}

// richFileRefs — HTML доторх өөрийн сангийн зургууд (файл устгахад ашиглагдаж байгааг тооцно).
func richFileRefs(h string) []string {
	var out []string
	for _, m := range richSrcRe.FindAllStringSubmatch(h, -1) {
		out = append(out, html.UnescapeString(m[1]))
	}
	return out
}
