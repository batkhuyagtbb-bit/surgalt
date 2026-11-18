package httpapi

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Маркерын тодруулга: суралцагч бүр зөвхөн өөрийнхөө тодруулгыг уншиж, бичнэ (багшид харагдахгүй).
const (
	highlightsMax    = 1000 // нэг хичээл дээрх тодруулгын дээд тоо
	highlightTextMax = 5000 // нэг тодруулгын текст (тэмдэгт)
	highlightNoteMax = 1000 // нэг тэмдэглэл (тэмдэгт)
	highlightQMax    = 300  // flash card-ийн асуулт (тэмдэгт)
	highlightAMax    = 1000 // flash card-ийн санах хэсэг (тэмдэгт)

	storyNotesMax = 60  // GET /api/me/notes: хичээл бүрийн сүүлийн тэмдэглэл
	storyCardsMax = 200 // GET /api/me/notes: хичээл бүрийн flash card (сүүлийн 60-аас өмнөх нь ч)
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
	if !ownNoteImages(in.Highlights, c.UID) {
		writeErr(w, http.StatusBadRequest, "тэмдэглэлийн зураг буруу")
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
	if !ownNoteImages(in.Highlights, uid) {
		writeErr(w, http.StatusBadRequest, "тэмдэглэлийн зураг буруу")
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
		items := storyNotes(st.Items)
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

// storyNotes — story-д: сүүлийн 60 тэмдэглэл + түүнээс өмнөх flash card-ууд (хичээл бүрт нийт ≤200 карт, шинэ нь
// давуу), анхны дарааллаараа.
func storyNotes(items []store.Highlight) []store.Highlight {
	if len(items) <= storyNotesMax {
		return items
	}
	start := len(items) - storyNotesMax
	cards := 0
	for i := start; i < len(items); i++ {
		if items[i].IsCard() {
			cards++
		}
	}
	var early []store.Highlight
	for i := start - 1; i >= 0 && cards < storyCardsMax; i-- {
		if items[i].IsCard() {
			early = append(early, items[i])
			cards++
		}
	}
	if len(early) == 0 {
		return items[start:]
	}
	slices.Reverse(early)
	return append(early, items[start:]...)
}

var noteImgRe = regexp.MustCompile(`^/note-img/[0-9a-f]{24}/[0-9a-f]{32}\.webp$`)

func validBox(b []float64) bool {
	if len(b) != 4 {
		return false
	}
	for _, v := range b {
		if v < 0 || v > 100 {
			return false
		}
	}
	return b[2] > 0 && b[3] > 0 && b[0]+b[2] <= 100.5 && b[1]+b[3] <= 100.5
}

// ownNoteImages — тэмдэглэлийн зураг зөвхөн тухайн хэрэглэгчийнх байна.
func ownNoteImages(hs []store.Highlight, uid string) bool {
	for _, h := range hs {
		if h.Img != "" && !strings.HasPrefix(h.Img, "/note-img/"+uid+"/") {
			return false
		}
	}
	return true
}

// POST /api/me/note-images — баримтаас хүрээлж авсан хэсгийн зураг (түүхий бие, 6MB хүртэл) → {url}.
func (s *Server) handleNoteImage(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !s.boardLim.Allow("note-img:" + c.UID) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон зураг оруулж байна — түр хүлээнэ үү")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 6<<20)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		writeErr(w, http.StatusRequestEntityTooLarge, "зургийн хэмжээ хэт том байна")
		return
	}
	boardImgSem <- struct{}{}
	name, _, _, err := s.files.SaveNoteImage(c.UID, raw)
	<-boardImgSem
	switch {
	case errors.Is(err, files.ErrImage):
		writeErr(w, http.StatusBadRequest, "зургийг уншиж чадсангүй")
	case errors.Is(err, files.ErrNotesFull):
		writeErr(w, http.StatusRequestEntityTooLarge, err.Error())
	case err != nil:
		s.log.Error("тэмдэглэлийн зураг", "err", err)
		writeErr(w, http.StatusInternalServerError, "зургийг хадгалж чадсангүй — дахин оролдоно уу")
	default:
		writeJSON(w, http.StatusOK, map[string]any{"url": "/note-img/" + c.UID + "/" + name})
	}
}

// GET /note-img/{uid}/{name} — тэмдэглэлийн зураг (таамаглах боломжгүй нэртэй, өөрчлөгддөггүй).
func (s *Server) handleNoteImg(w http.ResponseWriter, r *http.Request) {
	f, fi, err := s.files.OpenNoteImage(r.PathValue("uid"), r.PathValue("name"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "зураг олдсонгүй")
		return
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "private, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "image/webp")
	http.ServeContent(w, r, "", fi.ModTime(), f)
}

func validateHighlights(hs []store.Highlight) string {
	if len(hs) > highlightsMax {
		return "нэг хичээлд 1000-аас олон тодруулга хийх боломжгүй"
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
			case (len(h.Box) > 0 || h.Img != "") && h.Kind != "page":
				return "хүрээг зөвхөн хуудсан дээр зурна"
			case len(h.Box) > 0 && !validBox(h.Box):
				return "хүрээлсэн хэсэг буруу"
			case h.Img != "" && !noteImgRe.MatchString(h.Img):
				return "тэмдэглэлийн зураг буруу"
			case utf8.RuneCountInString(h.Note) > highlightNoteMax:
				return "тэмдэглэл 1000 тэмдэгтээс хэтрэхгүй байх ёстой"
			}
		case h.Page != 0 || h.Sec != 0 || len(h.Box) > 0 || h.Img != "":
			return "тодруулгын мэдээлэл буруу"
		case h.Start < 0 || h.End <= h.Start || h.End-h.Start > highlightTextMax:
			return "тодруулгын байрлал буруу"
		case utf8.RuneCountInString(h.Text) > highlightTextMax:
			return "тодруулсан текст хэт урт байна"
		case utf8.RuneCountInString(h.Note) > highlightNoteMax:
			return "тэмдэглэл 1000 тэмдэгтээс хэтрэхгүй байх ёстой"
		}
		// Flash card: бүх төрөл, өнгөнд; асуулт ба санах хэсэг хоёулаа бөглөгдсөн эсвэл хоёулаа хоосон.
		h.Q, h.A = strings.TrimSpace(h.Q), strings.TrimSpace(h.A)
		switch {
		case (h.Q == "") != (h.A == ""):
			return "flash card-ийн асуулт ба санах хэсгийг хоёуланг нь бөглөнө үү"
		case utf8.RuneCountInString(h.Q) > highlightQMax:
			return "flash card-ийн асуулт 300 тэмдэгтээс хэтрэхгүй байх ёстой"
		case utf8.RuneCountInString(h.A) > highlightAMax:
			return "flash card-ийн санах хэсэг 1000 тэмдэгтээс хэтрэхгүй байх ёстой"
		}
	}
	return ""
}
