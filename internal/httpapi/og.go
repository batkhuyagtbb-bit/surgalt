package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/skip2/go-qrcode"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
	_ "golang.org/x/image/webp"

	"surgalt/internal/files"
)

// Нийгмийн сүлжээнд хуваалцах зураг (Open Graph, 1200×630): багшийн профайлыг Facebook, Messenger дээр
// загварлаг харагдуулна — нүүр зураг (бараан), дугуй аватар, нэр, мэргэжил, тоо, брэнд, QR.
// Фонт: Windows-ийн Segoe UI (монгол Ө, Ү-тэй; OG_FONT_DIR-ээр өөрчилнө), олдохгүй бол Go фонт.

const ogW, ogH = 1200, 630

var (
	ogNavy   = color.RGBA{31, 60, 143, 255}
	ogIndigo = color.RGBA{59, 47, 160, 255}
	ogOrange = color.RGBA{234, 160, 46, 255}
	ogWhite  = color.RGBA{255, 255, 255, 255}
	ogSoft   = color.RGBA{226, 232, 248, 255}
	ogGlow   = color.RGBA{132, 150, 255, 255} // баруун дээд талын цайвар туяа
)

var ogFont struct {
	once      sync.Once
	bold, reg *opentype.Font
}

func ogFonts() (bold, reg *opentype.Font) {
	ogFont.once.Do(func() {
		dir := os.Getenv("OG_FONT_DIR")
		if dir == "" {
			dir = filepath.Join(os.Getenv("WINDIR"), "Fonts")
		}
		load := func(fallback []byte, names ...string) *opentype.Font {
			for _, n := range names {
				if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
					if f, err := opentype.Parse(b); err == nil {
						return f
					}
				}
			}
			f, _ := opentype.Parse(fallback)
			return f
		}
		ogFont.bold = load(gobold.TTF, "segoeuib.ttf", "arialbd.ttf", "DejaVuSans-Bold.ttf", "NotoSans-Bold.ttf")
		ogFont.reg = load(goregular.TTF, "segoeui.ttf", "arial.ttf", "DejaVuSans.ttf", "NotoSans-Regular.ttf")
	})
	return ogFont.bold, ogFont.reg
}

type ogText struct {
	f    *opentype.Font
	face font.Face
}

func newOGText(f *opentype.Font, size float64) ogText {
	face, _ := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	return ogText{f, face}
}

// clean — фонтод дүрсгүй тэмдэгт (emoji г.м) хасна.
func (t ogText) clean(s string) string {
	var buf sfnt.Buffer
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return ' '
		}
		if i, err := t.f.GlyphIndex(&buf, r); err != nil || i == 0 || unicode.IsControl(r) {
			return -1
		}
		return r
	}, s))
}

func (t ogText) width(s string) int { return font.MeasureString(t.face, s).Ceil() }

// fit — maxW-д багтахгүй бол «…»-ээр таслана.
func (t ogText) fit(s string, maxW int) string {
	if t.width(s) <= maxW {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && t.width(string(r)+"…") > maxW {
		r = r[:len(r)-1]
	}
	return strings.TrimSpace(string(r)) + "…"
}

// wrap — үгээр мөр болгоно (дээд тал нь lines мөр, сүүлийнх нь таслагдана).
func (t ogText) wrap(s string, maxW, lines int) []string {
	words := strings.Fields(s)
	var out []string
	cur := ""
	for i, w := range words {
		next := strings.TrimSpace(cur + " " + w)
		if cur == "" || t.width(next) <= maxW {
			cur = next
			continue
		}
		out = append(out, cur)
		if len(out) == lines-1 {
			cur = strings.Join(words[i:], " ")
			break
		}
		cur = w
	}
	if cur != "" {
		out = append(out, cur)
	}
	for i := range out {
		out[i] = t.fit(out[i], maxW)
	}
	return out
}

func (t ogText) draw(dst draw.Image, c color.Color, x, y int, s string) int {
	d := &font.Drawer{Dst: dst, Src: image.NewUniform(c), Face: t.face, Dot: fixed.P(x, y)}
	d.DrawString(s)
	return d.Dot.X.Round() - x
}

// ogRoundRect — булан нь бөөрөнхий тэгш өнцөгт (антиалиас).
func ogRoundRect(dst draw.Image, r image.Rectangle, rad float32, src image.Image) {
	w, h := float32(r.Dx()), float32(r.Dy())
	rad = min(rad, w/2, h/2)
	k := rad * 0.5523
	z := vector.NewRasterizer(r.Dx(), r.Dy())
	z.MoveTo(rad, 0)
	z.LineTo(w-rad, 0)
	z.CubeTo(w-rad+k, 0, w, rad-k, w, rad)
	z.LineTo(w, h-rad)
	z.CubeTo(w, h-rad+k, w-rad+k, h, w-rad, h)
	z.LineTo(rad, h)
	z.CubeTo(rad-k, h, 0, h-rad+k, 0, h-rad)
	z.LineTo(0, rad)
	z.CubeTo(0, rad-k, rad-k, 0, rad, 0)
	z.ClosePath()
	z.Draw(dst, r, src, r.Min)
}

func ogFill(dst draw.Image, r image.Rectangle, rad float32, c color.Color) {
	ogRoundRect(dst, r, rad, image.NewUniform(c))
}

// ogCover — зургийг тэгш өнцөгтийг дүүргэтэл томруулж, голоос нь тайрна.
func ogCover(src image.Image, w, h int) *image.RGBA {
	sb := src.Bounds()
	scale := math.Max(float64(w)/float64(sb.Dx()), float64(h)/float64(sb.Dy()))
	cw, ch := int(float64(w)/scale), int(float64(h)/scale)
	min0 := image.Pt(sb.Min.X+(sb.Dx()-cw)/2, sb.Min.Y+(sb.Dy()-ch)/2)
	crop := image.Rectangle{Min: min0, Max: min0.Add(image.Pt(cw, ch))}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(out, out.Bounds(), src, crop, draw.Src, nil)
	return out
}

// ogImageHosts — гадаад аватар (Google, Facebook-ээр нэвтэрсэн) зөвхөн эдгээр хостоос.
var ogImageHosts = []string{".googleusercontent.com", ".fbcdn.net", ".fbsbx.com"}

// ogPicture — өөрийн нээлттэй сангийн зураг эсвэл зөвшөөрсөн хостын https зураг.
func (s *Server) ogPicture(ctx context.Context, u string) image.Image {
	u = stripSig(strings.TrimSpace(u))
	var rd io.Reader
	switch {
	case strings.HasPrefix(u, "/files/"):
		teacher, vis, name, ok := files.SplitPath(u)
		if !ok || vis != files.Public {
			return nil
		}
		f, _, err := s.files.Open(teacher, vis, name)
		if err != nil {
			return nil
		}
		defer f.Close()
		rd = f
	case strings.HasPrefix(u, "https://"):
		pu, err := url.Parse(u)
		if err != nil {
			return nil
		}
		allowed := false
		for _, h := range ogImageHosts {
			allowed = allowed || strings.HasSuffix("."+pu.Hostname(), h)
		}
		if !allowed {
			return nil
		}
		cctx, cancel := context.WithTimeout(ctx, 4*time.Second)
		defer cancel()
		req, _ := http.NewRequestWithContext(cctx, http.MethodGet, u, nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil
		}
		rd = io.LimitReader(res.Body, 6<<20)
	default:
		return nil
	}
	img, _, err := image.Decode(rd)
	if err != nil {
		return nil
	}
	return img
}

func ogInitials(name string) string {
	var b []rune
	for _, w := range strings.Fields(name) {
		if r := []rune(w); len(r) > 0 && len(b) < 2 {
			b = append(b, unicode.ToUpper(r[0]))
		}
	}
	if len(b) == 0 {
		return "S"
	}
	return string(b)
}

// renderProfileOG — профайлын хуваалцах зураг (JPEG).
func (s *Server) renderProfileOG(ctx context.Context, p *PublicProfile) ([]byte, error) {
	t := p.Teacher
	bold, reg := ogFonts()
	dst := image.NewRGBA(image.Rect(0, 0, ogW, ogH))

	// Дэвсгэр: брэндийн градиент (диагональ) + баруун дээд талд цайвар туяа; нүүр зурагтай бол түүн дээр бараан бүрхүүл.
	bg := image.NewRGBA(dst.Bounds())
	for y := 0; y < ogH; y++ {
		for x := 0; x < ogW; x++ {
			k := (float64(x)/ogW*0.6 + float64(y)/ogH*0.4)
			c := color.RGBA{uint8(float64(ogNavy.R)*(1-k) + float64(ogIndigo.R)*k), uint8(float64(ogNavy.G)*(1-k) + float64(ogIndigo.G)*k), uint8(float64(ogNavy.B)*(1-k) + float64(ogIndigo.B)*k), 255}
			if d := math.Hypot(float64(x-1080), float64(y-40)) / 620; d < 1 {
				a := (1 - d) * (1 - d) * 0.38
				c.R = uint8(float64(c.R)*(1-a) + float64(ogGlow.R)*a)
				c.G = uint8(float64(c.G)*(1-a) + float64(ogGlow.G)*a)
				c.B = uint8(float64(c.B)*(1-a) + float64(ogGlow.B)*a)
			}
			bg.SetRGBA(x, y, c)
		}
	}
	if cover := s.ogPicture(ctx, t.CoverURL); cover != nil {
		draw.Draw(dst, dst.Bounds(), ogCover(cover, ogW, ogH), image.Point{}, draw.Src)
		draw.DrawMask(dst, dst.Bounds(), bg, image.Point{}, image.NewUniform(color.Alpha{196}), image.Point{}, draw.Over)
	} else {
		draw.Draw(dst, dst.Bounds(), bg, image.Point{}, draw.Src)
	}
	dot := image.NewUniform(color.NRGBA{255, 255, 255, 26})
	for y := 18; y < ogH; y += 30 {
		for x := 18; x < ogW; x += 30 {
			ogRoundRect(dst, image.Rect(x, y, x+3, y+3), 1.5, dot)
		}
	}
	ogFill(dst, image.Rect(0, ogH-10, ogW, ogH), 0, ogOrange)

	// Аватар: цагаан хүрээтэй дугуй.
	const ax, ay, ad = 72, 104, 220
	ogFill(dst, image.Rect(ax-8, ay-8, ax+ad+8, ay+ad+8), ad, ogWhite)
	circle := image.NewAlpha(image.Rect(0, 0, ad, ad))
	ogRoundRect(circle, circle.Bounds(), ad/2, image.Opaque)
	if av := s.ogPicture(ctx, t.AvatarURL); av != nil {
		draw.DrawMask(dst, image.Rect(ax, ay, ax+ad, ay+ad), ogCover(av, ad, ad), image.Point{}, circle, image.Point{}, draw.Over)
	} else {
		h := float64(hue(t.Username))
		c := hsl(h, 0.55, 0.42)
		draw.DrawMask(dst, image.Rect(ax, ay, ax+ad, ay+ad), image.NewUniform(c), image.Point{}, circle, image.Point{}, draw.Over)
		it := newOGText(bold, 92)
		ini := it.clean(ogInitials(t.DisplayName))
		it.draw(dst, ogWhite, ax+(ad-it.width(ini))/2, ay+ad/2+33, ini)
	}

	// Текст: «БАГШ» шошго, нэр, мэргэжил.
	const tx, maxW = 340, ogW - 340 - 72
	lt := newOGText(bold, 22)
	label := "БАГШ"
	if loc := lt.clean(t.Location); loc != "" {
		label += "  ·  " + strings.ToUpper(loc)
	}
	label = lt.fit(label, maxW-40)
	ogFill(dst, image.Rect(tx, 100, tx+lt.width(label)+36, 140), 20, ogOrange)
	lt.draw(dst, ogNavy, tx+18, 128, label)
	nt := newOGText(bold, 64)
	y := 214
	for _, ln := range nt.wrap(nt.clean(t.DisplayName), maxW, 2) {
		nt.draw(dst, ogWhite, tx, y, ln)
		y += 72
	}
	ht := newOGText(reg, 31)
	head := t.Headline
	if head == "" {
		head = strings.Join(t.Subjects, " · ")
	}
	y += 2
	for _, ln := range ht.wrap(ht.clean(head), maxW, 2) {
		ht.draw(dst, ogSoft, tx, y, ln)
		y += 42
	}

	// Тоо: сургалт, суралцагч, ном (QR-ын зүүн талд багтах хэмжээгээр).
	ct := newOGText(bold, 25)
	var chips []string
	for _, c := range []struct {
		n    int64
		unit string
	}{{int64(t.CourseCount), "сургалт"}, {t.StudentCount, "суралцагч"}, {int64(len(p.Books)), "ном"}, {int64(t.LessonCount), "хичээл"}} {
		if c.n > 0 {
			chips = append(chips, fmt.Sprintf("%d %s", c.n, c.unit))
		}
	}
	if len(chips) == 0 {
		chips = []string{"Шинэ багш", "surgalt.mn"}
	}
	cx, chipTop := tx, 430
	for _, c := range chips {
		w := ct.width(c) + 40
		if cx+w > 940 {
			break
		}
		ogFill(dst, image.Rect(cx, chipTop, cx+w, chipTop+50), 25, color.NRGBA{255, 255, 255, 40})
		ct.draw(dst, ogWhite, cx+20, chipTop+34, c)
		cx += w + 12
	}

	// Брэнд, холбоос (зүүн доод), QR (баруун доод).
	bt := newOGText(bold, 42)
	w := bt.draw(dst, ogWhite, 72, 566, "surgalt")
	bt.draw(dst, ogOrange, 72+w, 566, ".mn")
	ut := newOGText(reg, 24)
	link := strings.TrimPrefix(strings.TrimPrefix(s.cfg.PublicURL, "https://"), "http://")
	if link == "" {
		link = "surgalt.mn"
	}
	ut.draw(dst, color.RGBA{214, 222, 245, 255}, 72, 600, ut.fit(link+"/t/"+t.Username, 820))
	if q, err := qrcode.New(s.cfg.PublicURL+"/t/"+t.Username, qrcode.Medium); err == nil {
		q.DisableBorder = true
		ogFill(dst, image.Rect(968, 376, 1128, 536), 18, ogWhite)
		qi := q.Image(132)
		draw.Draw(dst, image.Rect(982, 390, 982+132, 390+132), qi, qi.Bounds().Min, draw.Over)
		qt := newOGText(bold, 19)
		hint := "Уншуулаад нэвтэр"
		qt.draw(dst, ogWhite, 1048-qt.width(hint)/2, 570, hint)
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// hsl → RGBA (аватаргүй үед нэрийн өнгөөр дугуй).
func hsl(h, s, l float64) color.RGBA {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g = c, x
	case h < 120:
		r, g = x, c
	case h < 180:
		g, b = c, x
	case h < 240:
		g, b = x, c
	case h < 300:
		r, b = x, c
	default:
		r, b = c, x
	}
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}

// profileShareDesc — хуваалцахад харагдах тайлбар: мэргэжил, тоо, товч танилцуулга.
func profileShareDesc(t PublicTeacher, books int) string {
	var parts, nums []string
	if t.Headline != "" {
		parts = append(parts, t.Headline)
	}
	for _, c := range []struct {
		n    int64
		unit string
	}{{int64(t.CourseCount), "сургалт"}, {t.StudentCount, "суралцагч"}, {int64(books), "ном"}} {
		if c.n > 0 {
			nums = append(nums, fmt.Sprintf("%d %s", c.n, c.unit))
		}
	}
	if len(nums) > 0 {
		parts = append(parts, strings.Join(nums, " · "))
	}
	if bio := strings.Join(strings.Fields(t.Bio), " "); bio != "" {
		parts = append(parts, bio)
	}
	d := strings.Join(parts, " — ")
	if d == "" {
		d = t.DisplayName + " — surgalt.mn дээрх багшийн профайл"
	}
	if r := []rune(d); len(r) > 200 {
		d = string(r[:197]) + "…"
	}
	return d
}

// ogVersion — зураг өөрчлөгдөхөд холбоос солигдож, Facebook шинээр татна.
func ogVersion(t PublicTeacher, books int) string {
	sum := sha256.Sum256([]byte(fmt.Sprint(t.DisplayName, "|", t.Headline, "|", t.AvatarURL, "|", t.CoverURL, "|", t.Location, "|", t.Subjects, "|", t.CourseCount, t.StudentCount, t.LessonCount, books)))
	return hex.EncodeToString(sum[:5])
}

// handleProfileOG: GET /t/{username}/og.jpg — профайлыг хуваалцах зураг (кэштэй).
func (s *Server) handleProfileOG(w http.ResponseWriter, r *http.Request) {
	p, err := s.publicProfile(r.Context(), r.PathValue("username"))
	if s.storeErr(w, r, err) {
		return
	}
	img, err := s.ogs.GetOrLoad(r.Context(), p.Data.Teacher.Username+"|"+ogVersion(p.Data.Teacher, len(p.Data.Books)), func(ctx context.Context) ([]byte, error) {
		return s.renderProfileOG(ctx, &p.Data)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "зураг үүсгэж чадсангүй")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "image/jpeg")
	h.Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(img)
}
