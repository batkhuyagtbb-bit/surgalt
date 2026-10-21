package store

import "context"

// Highlight — суралцагч хичээлийн текстийг маркераар тодруулсан хэсэг. Зөвхөн тухайн суралцагчид
// харагдана (багш, бусад суралцагч харахгүй).
type Highlight struct {
	ID    string `json:"id"`
	Block string `json:"b"` // текст блокийн ID (блокгүй хуучин хичээлд "content")
	Start int    `json:"s"` // блокийн текст доторх эхлэл (тэмдэгтээр)
	End   int    `json:"e"`
	Color string `json:"c"` // HighlightColors
	Text  string `json:"t"` // тодруулсан текст: багш агуулгаа засвал байрлалыг дахин олоход
	Note  string `json:"n,omitempty"`
}

// HighlightColors — маркерын өнгө: y чухал, g ойлгосон, r ойлгоогүй, b нэр томьёо.
var HighlightColors = map[string]bool{"y": true, "g": true, "r": true, "b": true}

type HighlightStore interface {
	// Highlights — суралцагчийн тухайн хичээл дээрх тодруулгууд (байхгүй бол хоосон).
	Highlights(ctx context.Context, userID, lessonID string) ([]Highlight, error)
	// SetHighlights — тухайн хичээлийн тодруулгуудыг бүхэлд нь солино.
	SetHighlights(ctx context.Context, userID, lessonID string, hs []Highlight) error
}
