package httpapi

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"surgalt/internal/store"
)

// Маркерын тодруулга: суралцагч бүр зөвхөн өөрийнхөө тодруулгыг уншиж, бичнэ (багшид харагдахгүй).
const (
	highlightsMax    = 1000 // нэг хичээл дээрх тодруулгын дээд тоо
	highlightTextMax = 5000 // нэг тодруулгын текст (тэмдэгт)
	highlightNoteMax = 1000 // нэг тэмдэглэл (тэмдэгт)
)

func (s *Server) handleGetHighlights(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	_, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	hs, err := s.store.Highlights(r.Context(), c.UID, l.ID)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"highlights": hs})
}

func (s *Server) handleSetHighlights(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	_, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	var in struct {
		Highlights []store.Highlight `json:"highlights"`
	}
	if !decode(w, r, &in) {
		return
	}
	if msg := validateHighlights(in.Highlights); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.SetHighlights(r.Context(), c.UID, l.ID, in.Highlights); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"highlights": in.Highlights})
}

func validateHighlights(hs []store.Highlight) string {
	if len(hs) > highlightsMax {
		return "нэг хичээл дээр 1000-аас олон тодруулга хийхгүй"
	}
	for i := range hs {
		h := &hs[i]
		h.Note = strings.TrimSpace(h.Note)
		switch {
		case h.ID == "" || len(h.ID) > 32 || h.Block == "" || len(h.Block) > 64:
			return "тодруулгын мэдээлэл буруу"
		case !store.HighlightColors[h.Color]:
			return "маркерын өнгө буруу"
		case h.Start < 0 || h.End <= h.Start || h.End-h.Start > highlightTextMax:
			return "тодруулгын байрлал буруу"
		case utf8.RuneCountInString(h.Text) > highlightTextMax:
			return "тодруулсан текст хэт урт"
		case utf8.RuneCountInString(h.Note) > highlightNoteMax:
			return "тэмдэглэл 1000 тэмдэгтээс хэтрэхгүй"
		}
	}
	return ""
}
