package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// seedBooks нь демо багшид нэг төлбөртэй ном, нэг үнэгүй өгүүлэл үүсгэнэ (хуудсуудыг зурж).
func seedBooks(ctx context.Context, st store.Store, fs *files.Store, teacherID string) error {
	books := []struct {
		b     store.Book
		pages int
		hue   color.RGBA
	}{
		{store.Book{Kind: "book", Title: "ЭЕШ Математик — 100 бодлогын эмхэтгэл", Author: "Сарангэрэл Батболд", Price: 15000, PreviewN: 3,
			Description: "ЭЕШ-д хамгийн их гардаг 100 бодлогыг бодолттой нь. Алгебр, геометр, магадлал, тригонометр.\nХуудас бүр тайлбартай, алхам алхмаар."}, 14, color.RGBA{31, 60, 143, 255}},
		{store.Book{Kind: "article", Title: "Логарифмыг 10 минутад ойлгох", Author: "Сарангэрэл Батболд", Price: 0, PreviewN: 3,
			Description: "Логарифмын үндсэн санааг энгийн жишээгээр тайлбарласан богино өгүүлэл."}, 5, color.RGBA{234, 160, 46, 255}},
	}
	for _, x := range books {
		b := x.b
		b.TeacherID = teacherID
		if err := st.CreateBook(ctx, &b); err != nil {
			return err
		}
		for p := 1; p <= x.pages; p++ {
			if err := fs.SaveBookPage(teacherID, b.ID, p, bytes.NewReader(demoPage(p, x.pages, x.hue))); err != nil {
				return err
			}
		}
		if err := st.SetBookPages(ctx, b.ID, teacherID, x.pages); err != nil {
			return err
		}
		b.Published = true
		if err := st.UpdateBook(ctx, &b); err != nil {
			return err
		}
	}
	return nil
}

// demoPage — демо хуудас: өнгөт толгой, том хуудасны дугаар, текстийн мөр мэт зураас.
func demoPage(p, total int, hue color.RGBA) []byte {
	const w, h = 1000, 1414
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{255, 255, 255, 255}), image.Point{}, draw.Src)
	if p == 1 { // нүүр
		draw.Draw(img, img.Bounds(), image.NewUniform(hue), image.Point{}, draw.Src)
		bigText(img, "SURGALT.MN", 120, 260, 9, color.RGBA{255, 255, 255, 255})
		bigText(img, "DEMO BOOK", 120, 520, 12, color.RGBA{255, 255, 255, 230})
		return enc(img)
	}
	draw.Draw(img, image.Rect(0, 0, w, 120), image.NewUniform(hue), image.Point{}, draw.Src)
	bigText(img, fmt.Sprintf("PAGE %d / %d", p, total), 70, 30, 5, color.RGBA{255, 255, 255, 255})
	gray := image.NewUniform(color.RGBA{214, 219, 230, 255})
	for i, y := 0, 200; y < h-140; i, y = i+1, y+52 {
		lw := w - 140 - (i*137)%260
		if i%7 == 6 {
			continue
		}
		draw.Draw(img, image.Rect(70, y, 70+lw, y+16), gray, image.Point{}, draw.Src)
	}
	bigText(img, fmt.Sprint(p), w/2-20, h-90, 4, color.RGBA{120, 128, 150, 255})
	return enc(img)
}

func bigText(dst *image.RGBA, s string, x, y, scale int, c color.RGBA) {
	face := basicfont.Face7x13
	tw := font.MeasureString(face, s).Ceil() + 2
	small := image.NewRGBA(image.Rect(0, 0, tw, 16))
	(&font.Drawer{Dst: small, Src: image.NewUniform(c), Face: face, Dot: fixed.P(1, 12)}).DrawString(s)
	r := image.Rect(x, y, x+tw*scale, y+16*scale)
	xdraw.NearestNeighbor.Scale(dst, r, small, small.Bounds(), draw.Over, nil)
}

func enc(img image.Image) []byte {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}
