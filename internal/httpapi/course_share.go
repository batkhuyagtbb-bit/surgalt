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
	"net/http"
	"strconv"
	"strings"

	"github.com/skip2/go-qrcode"
)

// Сургалтыг хуваалцах: сургалт бүрийн QR (уншуулбал сургалт руу орж, нэвтрээгүй бол нэвтрээд буцаж ирнэ)
// ба Facebook зэрэгт хуваалцахад гарах зураг (og.jpg). Зөвхөн нийтэлсэн сургалтад.

// courseJoinURL — QR-д орох холбоос: «#join» нь хуудсанд нэвтрүүлж, үнэгүй бол элсүүлж, нууц үгтэй бол нууц үг асууна.
func courseJoinURL(base, id string) string { return base + "/c/" + id + "#join" }

// handleCourseQR: GET /c/{id}/qr.png?size=256..1024&download=1
func (s *Server) handleCourseQR(w http.ResponseWriter, r *http.Request) {
	c, err := s.publicCourse(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	id := c.Data.Course.ID
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	size = min(max(size, 256), 1024)
	url := courseJoinURL(s.baseURL(r), id)
	png, err := s.qrs.GetOrLoad(r.Context(), url+"|"+strconv.Itoa(size), func(context.Context) ([]byte, error) {
		return qrcode.Encode(url, qrcode.High, size)
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "QR үүсгэж чадсангүй")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if r.URL.Query().Get("download") == "1" {
		w.Header().Set("Content-Disposition", `attachment; filename="surgalt-`+id+`-qr.png"`)
	}
	_, _ = w.Write(png)
}

// courseOGLayout — зургийн загварын хувилбар: загвар өөрчлөгдвөл ахиулна (og:image холбоос солигдож Facebook шинээр татна).
const courseOGLayout = 2

// courseOGVersion — хуваалцах зургийг өөрчлөгдөх мэдээллээр нь кэшлэнэ (нэр, толгойн зураг, үнэ, хичээлийн тоо, багш).
func courseOGVersion(pc *PublicCourse) string {
	c, t := pc.Course, pc.Teacher
	sum := sha256.Sum256([]byte(fmt.Sprint(courseOGLayout, "|", c.Title, "|", c.Cover, "|", c.Price, "|", c.Locked, "|", c.LessonCount, c.FreeLessonCount, "|", t.DisplayName, "|", t.AvatarURL, "|", pc.Org)))
	return hex.EncodeToString(sum[:5])
}

// handleCourseOG: GET /c/{id}/og.jpg — сургалтыг хуваалцах зураг (1200×630, кэштэй).
func (s *Server) handleCourseOG(w http.ResponseWriter, r *http.Request) {
	c, err := s.publicCourse(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	pc := c.Data
	img, err := s.ogs.GetOrLoad(r.Context(), "c:"+pc.Course.ID+"|"+courseOGVersion(&pc), func(ctx context.Context) ([]byte, error) {
		return s.renderCourseOG(ctx, &pc)
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

func (s *Server) renderCourseOG(ctx context.Context, pc *PublicCourse) ([]byte, error) {
	c, t := pc.Course, pc.Teacher
	bold, reg := ogFonts()
	dst := image.NewRGBA(image.Rect(0, 0, ogW, ogH))

	// Дэвсгэр: толгойн зураг (зураг бол) бараан бүрхүүлтэй, эс бөгөөс брэндийн градиент.
	bg := ogBrandBG()
	var cover image.Image
	if coverKind(c.Cover) == "img" {
		cover = s.ogPicture(ctx, c.Cover)
	}
	if cover != nil {
		draw.Draw(dst, dst.Bounds(), ogCover(cover, ogW, ogH, 0.5), image.Point{}, draw.Src)
		draw.DrawMask(dst, dst.Bounds(), bg, image.Point{}, image.NewUniform(color.Alpha{200}), image.Point{}, draw.Over)
	} else {
		draw.Draw(dst, dst.Bounds(), bg, image.Point{}, draw.Src)
	}
	ogDots(dst)

	// Баруун тал: том QR (утасны камераар дэлгэцээс ч шууд уншигдахаар), доор нь «Уншуулаад нэвтэр».
	const qBox, qx, qy = 404, ogW - 72 - 404, 84
	if q, err := qrcode.New(courseJoinURL(strings.TrimRight(s.cfg.PublicURL, "/"), c.ID), qrcode.Medium); err == nil {
		q.DisableBorder = true
		ogFill(dst, image.Rect(qx, qy, qx+qBox, qy+qBox), 28, ogWhite)
		const qs = qBox - 44 // цагаан хүрээ (quiet zone) 22px
		qi := q.Image(qs)
		draw.Draw(dst, image.Rect(qx+22, qy+22, qx+22+qs, qy+22+qs), qi, qi.Bounds().Min, draw.Over)
		qt := newOGText(bold, 28)
		hint := "Уншуулаад нэвтэр"
		qt.draw(dst, ogWhite, qx+qBox/2-qt.width(hint)/2, qy+qBox+50, hint)
	}

	// Зүүн тал: шошго (СУРГАЛТ · НУУЦ ҮГТЭЙ / ҮНЭГҮЙ / үнэ), нэр, багш, хичээлийн тоо, брэнд.
	const tx, maxW = 72, qx - 72 - 48
	lt := newOGText(bold, 22)
	label := "СУРГАЛТ"
	switch paid := c.LessonCount - c.FreeLessonCount; {
	case c.Locked:
		label += "  ·  НУУЦ ҮГТЭЙ"
	case c.Price > 0:
		label += "  ·  " + money(c.Price)
	case paid > 0:
		label += "  ·  ХИЧЭЭЛЭЭР"
	default:
		label += "  ·  ҮНЭГҮЙ"
	}
	label = lt.fit(label, maxW)
	ogFill(dst, image.Rect(tx, 72, tx+lt.width(label)+36, 112), 20, ogOrange)
	lt.draw(dst, ogNavy, tx+18, 100, label)

	// Сургалтын нэр (4 мөр хүртэл; урт нэрт жижиг үсэг).
	size := 52.0
	nt := newOGText(bold, size)
	lines := nt.wrap(nt.clean(c.Title), maxW, 4)
	if len(lines) <= 2 { // богино нэр — томоор
		size = 60
		nt = newOGText(bold, size)
		lines = nt.wrap(nt.clean(c.Title), maxW, 3)
	}
	step := int(size * 1.18)
	y := 178
	for _, ln := range lines {
		nt.draw(dst, ogWhite, tx, y, ln)
		y += step
	}

	// Багш: жижиг дугуй аватар + нэр.
	const ad = 60
	ay := min(max(y-24, 280), 400)
	ogFill(dst, image.Rect(tx-4, ay-4, tx+ad+4, ay+ad+4), ad, ogWhite)
	circle := image.NewAlpha(image.Rect(0, 0, ad, ad))
	ogRoundRect(circle, circle.Bounds(), ad/2, image.Opaque)
	if av := s.ogPicture(ctx, t.AvatarURL); av != nil {
		draw.DrawMask(dst, image.Rect(tx, ay, tx+ad, ay+ad), ogCover(av, ad, ad, 0.5), image.Point{}, circle, image.Point{}, draw.Over)
	} else {
		draw.DrawMask(dst, image.Rect(tx, ay, tx+ad, ay+ad), image.NewUniform(hsl(float64(hue(t.Username)), 0.55, 0.42)), image.Point{}, circle, image.Point{}, draw.Over)
		it := newOGText(bold, 26)
		ini := it.clean(ogInitials(t.DisplayName))
		it.draw(dst, ogWhite, tx+(ad-it.width(ini))/2, ay+ad/2+9, ini)
	}
	who := "Багш"
	if pc.Org {
		who = "Байгууллага"
	}
	tt := newOGText(bold, 28)
	st := newOGText(reg, 21)
	tt.draw(dst, ogWhite, tx+ad+18, ay+28, tt.fit(tt.clean(t.DisplayName), maxW-ad-18))
	st.draw(dst, ogSoft, tx+ad+18, ay+54, who)

	// Хичээлийн тоо.
	if c.LessonCount > 0 {
		ct := newOGText(bold, 24)
		chip := fmt.Sprintf("%d хичээл", c.LessonCount)
		if top := ay + ad + 20; top+46 <= 520 {
			w := ct.width(chip) + 36
			ogFill(dst, image.Rect(tx, top, tx+w, top+46), 23, color.NRGBA{255, 255, 255, 40})
			ct.draw(dst, ogWhite, tx+18, top+32, chip)
		}
	}

	// Брэнд, холбоос (зүүн доод).
	bt := newOGText(bold, 42)
	w := bt.draw(dst, ogWhite, 72, 566, "surgalt")
	bt.draw(dst, ogOrange, 72+w, 566, ".mn")
	ut := newOGText(reg, 22)
	link := strings.TrimPrefix(strings.TrimPrefix(s.cfg.PublicURL, "https://"), "http://")
	if link == "" {
		link = "surgalt.mn"
	}
	ut.draw(dst, color.RGBA{214, 222, 245, 255}, 72, 600, ut.fit(link+"/c/"+c.ID, maxW))

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
