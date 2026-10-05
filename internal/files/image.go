package files

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"

	"github.com/gen2brain/webp"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// MaxImageSide — оруулсан зургийн урт талыг энэ хэмжээ хүртэл багасгана.
const MaxImageSide = 1000

// maxPixels — "decompression bomb"-оос хамгаална (жишээ нь 100000x100000 PNG).
const maxPixels = 60_000_000

// webpQuality — вэбд хамгийн сайн хэмжээ/чанарын харьцаа.
const webpQuality = 80

var ErrImage = errors.New("зургийг уншиж чадсангүй")

func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	}
	return false // gif: хөдөлгөөнийг хадгалахын тулд хөндөхгүй
}

// processImage нь зургийг вэбд хамгийн хурдан ачаалагдах WebP болгоно:
// EXIF эргэлтийг засч, урт талыг MaxImageSide хүртэл багасгаж, мета өгөгдлийг (GPS г.м) арилгана.
func processImage(path string) (w, h int, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return 0, 0, ErrImage
	}
	if cfg.Width*cfg.Height > maxPixels {
		return 0, 0, fmt.Errorf("%w: %dx%d хэт том", ErrImage, cfg.Width, cfg.Height)
	}
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return 0, 0, ErrImage
	}
	src = applyOrientation(src, jpegOrientation(raw))

	b := src.Bounds()
	w, h = b.Dx(), b.Dy()
	if m := max(w, h); m > MaxImageSide {
		w, h = max(1, w*MaxImageSide/m), max(1, h*MaxImageSide/m)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)

	tmp := path + ".webp-tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return 0, 0, err
	}
	err = webp.Encode(out, dst, webp.Options{Quality: webpQuality, Method: 4})
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return 0, 0, err
	}
	return w, h, os.Rename(tmp, path)
}

// jpegOrientation нь JPEG-ийн EXIF Orientation (1-8) утгыг уншина; олдохгүй бол 1.
func jpegOrientation(b []byte) int {
	if len(b) < 4 || b[0] != 0xFF || b[1] != 0xD8 {
		return 1
	}
	r := bytes.NewReader(b[2:])
	for {
		var marker [2]byte
		if _, err := io.ReadFull(r, marker[:]); err != nil || marker[0] != 0xFF {
			return 1
		}
		var size uint16
		if binary.Read(r, binary.BigEndian, &size) != nil || size < 2 {
			return 1
		}
		seg := make([]byte, size-2)
		if _, err := io.ReadFull(r, seg); err != nil {
			return 1
		}
		if marker[1] == 0xDA { // scan эхэлсэн — EXIF байхгүй
			return 1
		}
		if marker[1] != 0xE1 || len(seg) < 14 || string(seg[:6]) != "Exif\x00\x00" {
			continue
		}
		t := seg[6:]
		var bo binary.ByteOrder
		switch string(t[:2]) {
		case "II":
			bo = binary.LittleEndian
		case "MM":
			bo = binary.BigEndian
		default:
			return 1
		}
		off := int(bo.Uint32(t[4:8]))
		if off+2 > len(t) {
			return 1
		}
		n := int(bo.Uint16(t[off:]))
		for i := 0; i < n; i++ {
			e := off + 2 + i*12
			if e+12 > len(t) {
				return 1
			}
			if bo.Uint16(t[e:]) == 0x0112 {
				if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
					return v
				}
				return 1
			}
		}
		return 1
	}
}

// applyOrientation нь EXIF-ийн дагуу зургийг эргүүлж/тусгана.
func applyOrientation(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2:
				nx, ny = w-1-x, y
			case 3:
				nx, ny = w-1-x, h-1-y
			case 4:
				nx, ny = x, h-1-y
			case 5:
				nx, ny = y, x
			case 6:
				nx, ny = h-1-y, x
			case 7:
				nx, ny = h-1-y, w-1-x
			case 8:
				nx, ny = y, w-1-x
			}
			dst.Set(nx, ny, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
