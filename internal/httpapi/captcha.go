package httpapi

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	mrand "math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/image/math/f64"

	xdraw "golang.org/x/image/draw"
)

// ---- Хялбар тоон captcha (нууц үг сэргээх г.м.) ----
// Сервер 4 оронтой тоог зурагт (бага зэрэг хазайлгаж, зураас, цэгтэй) зурж, хариуг санах ойд 10 минут хадгална.
// Нэг код зөвхөн нэг удаа шалгагдана (зөв, буруу аль нь ч устна) — таахыг давтах боломжгүй.

const (
	captchaLen    = 4
	captchaTTL    = 10 * time.Minute
	captchaMax    = 20000 // санах ойд хадгалах дээд тоо
	captchaDigits = "23456789"
)

type captchaEntry struct {
	code string
	exp  time.Time
}

type captchaStore struct {
	mu sync.Mutex
	m  map[string]captchaEntry
}

var captchas = &captchaStore{m: map[string]captchaEntry{}}

func (c *captchaStore) issue(now time.Time) (id, code string) {
	b := make([]byte, captchaLen)
	_, _ = rand.Read(b)
	d := make([]byte, captchaLen)
	for i := range b {
		d[i] = captchaDigits[int(b[i])%len(captchaDigits)]
	}
	id, code = randHex(12), string(d)
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= captchaMax/2 {
		for k, e := range c.m {
			if now.After(e.exp) || len(c.m) >= captchaMax {
				delete(c.m, k)
			}
		}
	}
	c.m[id] = captchaEntry{code, now.Add(captchaTTL)}
	return id, code
}

// check — нэг удаа: кодыг устгаад зөв эсэхийг буцаана.
func (c *captchaStore) check(id, answer string, now time.Time) bool {
	c.mu.Lock()
	e, ok := c.m[id]
	delete(c.m, id)
	c.mu.Unlock()
	answer = strings.TrimSpace(answer)
	return ok && now.Before(e.exp) && len(answer) == len(e.code) && subtle.ConstantTimeCompare([]byte(answer), []byte(e.code)) == 1
}

// captchaPNG — тоог хүнд уншигдахуйц, машинд арай хэцүү зураг болгоно.
func captchaPNG(code string) ([]byte, error) {
	const step, pad, h = 38, 14, 56
	w := pad*2 + step*len(code)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{240, 244, 252, 255}), image.Point{}, draw.Src)
	for i := 0; i < w*h/18; i++ {
		v := uint8(150 + mrand.IntN(80))
		img.Set(mrand.IntN(w), mrand.IntN(h), color.RGBA{v, v, uint8(min(255, int(v)+30)), 255})
	}
	inks := []color.RGBA{{31, 60, 143, 255}, {59, 47, 160, 255}, {20, 110, 120, 255}, {150, 70, 20, 255}}
	bold, _ := ogFonts()
	for i, r := range code {
		t := newOGText(bold, 34+mrand.Float64()*6)
		s := string(r)
		tile := image.NewRGBA(image.Rect(0, 0, step+10, h))
		t.draw(tile, inks[mrand.IntN(len(inks))], (tile.Bounds().Dx()-t.width(s))/2, 40+mrand.IntN(7)-3, s)
		a := (mrand.Float64() - 0.5) * 0.5 // ±14°
		cx, cy := float64(tile.Bounds().Dx())/2, float64(h)/2
		ox := float64(pad+i*step) - 5
		cos, sin := math.Cos(a), math.Sin(a)
		m := f64.Aff3{cos, -sin, ox + cx - cos*cx + sin*cy, sin, cos, cy - sin*cx - cos*cy}
		xdraw.BiLinear.Transform(img, m, tile, tile.Bounds(), draw.Over, nil)
	}
	for k := 0; k < 2; k++ {
		amp, per, ph, y0 := 4+mrand.Float64()*6, 30+mrand.Float64()*40, mrand.Float64()*6, 18+mrand.Float64()*20
		ink := color.RGBA{uint8(90 + mrand.IntN(60)), uint8(100 + mrand.IntN(60)), 170, 255}
		for x := 0; x < w; x++ {
			y := int(y0 + amp*math.Sin(float64(x)/per*2*math.Pi+ph))
			img.Set(x, y, ink)
			img.Set(x, y+1, ink)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// handleCaptcha: GET /api/auth/captcha → {id, image (data:image/png)}.
func (s *Server) handleCaptcha(w http.ResponseWriter, r *http.Request) {
	if !s.capLim.Allow(s.clientIP(r)) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон оролдлого")
		return
	}
	id, code := captchas.issue(time.Now())
	b, err := captchaPNG(code)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "дотоод алдаа")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "image": "data:image/png;base64," + base64.StdEncoding.EncodeToString(b)})
}
