package httpapi

import (
	"net/http"
	"strings"
	"time"
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

// Номын тэмдэглэл: уншиж чадах хүн бүр (нэвтэрсэн) өөрийн тэмдэглэлээ хуудас бүрт хөтөлнө.
// Хадгалалт нь хичээлийнхтэй ижил (түлхүүр "book:<номын ID>").
func (s *Server) bookNotesOpen(w http.ResponseWriter, r *http.Request) (uid, key string, ok bool) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return "", "", false
	}
	b, _, _, ok := s.publishedBook(w, r)
	if !ok {
		return "", "", false
	}
	return c.UID, "book:" + b.ID, true
}

func (s *Server) handleGetBookNotes(w http.ResponseWriter, r *http.Request) {
	uid, key, ok := s.bookNotesOpen(w, r)
	if !ok {
		return
	}
	hs, err := s.store.Highlights(r.Context(), uid, key)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"highlights": hs})
}

func (s *Server) handleSetBookNotes(w http.ResponseWriter, r *http.Request) {
	uid, key, ok := s.bookNotesOpen(w, r)
	if !ok {
		return
	}
	var in struct {
		Highlights []store.Highlight `json:"highlights"`
	}
	if !decode(w, r, &in) {
		return
	}
	for _, h := range in.Highlights {
		if h.Kind != "page" {
			writeErr(w, http.StatusBadRequest, "номд зөвхөн хуудасны тэмдэглэл хөтөлнө")
			return
		}
	}
	if msg := validateHighlights(in.Highlights); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.SetHighlights(r.Context(), uid, key, in.Highlights); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"highlights": in.Highlights})
}

// GET /api/me/notes — суралцагчийн бүх тэмдэглэл хичээл (ном) бүрээр: профайлын «story».
// Зөвхөн өөрт нь харагдана.
func (s *Server) handleMyNotes(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	sets, err := s.store.UserHighlights(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	if len(sets) > 40 {
		sets = sets[:40]
	}
	var lids []string
	for _, st := range sets {
		if !strings.HasPrefix(st.LessonID, "book:") {
			lids = append(lids, st.LessonID)
		}
	}
	lessons, err := s.store.LessonsByIDs(r.Context(), lids)
	if s.storeErr(w, r, err) {
		return
	}
	byID := map[string]store.Lesson{}
	for _, l := range lessons {
		byID[l.ID] = l
	}
	courses := map[string]*store.Course{}
	type story struct {
		ID      string            `json:"id"`
		Kind    string            `json:"kind"` // lesson | book
		Title   string            `json:"title"`
		Sub     string            `json:"sub"`
		URL     string            `json:"url"`
		Updated time.Time         `json:"updated_at"`
		Notes   []store.Highlight `json:"notes"`
	}
	out := []story{}
	for _, st := range sets {
		items := st.Items
		if len(items) > 60 {
			items = items[len(items)-60:]
		}
		if bid, ok := strings.CutPrefix(st.LessonID, "book:"); ok {
			b, err := s.store.BookByID(r.Context(), bid)
			if err != nil {
				continue
			}
			kind := "Ном"
			if b.Kind == "article" {
				kind = "Өгүүлэл"
			}
			out = append(out, story{ID: st.LessonID, Kind: "book", Title: b.Title, Sub: kind, URL: "/b/" + b.ID, Updated: st.UpdatedAt, Notes: items})
			continue
		}
		l, ok := byID[st.LessonID]
		if !ok {
			continue
		}
		co := courses[l.CourseID]
		if co == nil {
			if co, err = s.store.CourseByID(r.Context(), l.CourseID); err != nil {
				continue
			}
			courses[l.CourseID] = co
		}
		out = append(out, story{ID: l.ID, Kind: "lesson", Title: l.Title, Sub: co.Title, URL: "/c/" + co.ID + "#l=" + l.ID, Updated: st.UpdatedAt, Notes: items})
	}
	writeJSON(w, http.StatusOK, map[string]any{"stories": out})
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
		case h.Kind != "": // видеоны мөч, баримт/номын хуудас дээрх тэмдэглэл
			switch {
			case h.Kind != "time" && h.Kind != "page":
				return "тэмдэглэлийн төрөл буруу"
			case h.Kind == "time" && (h.Sec < 0 || h.Sec > 86400 || h.Page != 0):
				return "видеоны хугацаа буруу"
			case h.Kind == "page" && (h.Page < 1 || h.Page > 100000 || h.Sec != 0):
				return "хуудасны дугаар буруу"
			case h.Start != 0 || h.End != 0 || utf8.RuneCountInString(h.Text) > 200:
				return "тэмдэглэлийн мэдээлэл буруу"
			case utf8.RuneCountInString(h.Note) > highlightNoteMax:
				return "тэмдэглэл 1000 тэмдэгтээс хэтрэхгүй"
			}
		case h.Page != 0 || h.Sec != 0:
			return "тодруулгын мэдээлэл буруу"
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
