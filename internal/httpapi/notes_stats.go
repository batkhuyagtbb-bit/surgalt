package httpapi

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"surgalt/internal/store"
)

// Тэмдэглэлийн статистик: суралцагч нүүр хуудсандаа тэмдэглэлийн story-гоо үзэхэд бүртгэж, багш өөрийн
// сургалтын суралцагчид хэдэн тэмдэглэл (төрөл, өнгөөр) хөтөлж, story-гоо хэдэн удаа үзсэнийг харна.
// Тэмдэглэлийн агуулга (текст, зураг) багшид харагдахгүй — зөвхөн тоо.

const storyViewType = "story_view"

var lessonIDRe = regexp.MustCompile(`^[0-9a-f]{24}$`)

// cardsMax — нэг story-д нэг хичээлийн давтсан flash card-ийн дээд тоо.
const cardsMax = 500

// storyCards — нэг хичээлийн flash card давталт: seen — эргүүлж үзсэн, known — «Санасан».
type storyCards struct {
	Seen  int `json:"seen"`
	Known int `json:"known"`
}

// POST /api/me/story-view {"lessons":["…"],"slides":5,"total":7,"review":false,"cards":{"<lessonID>":{"seen":3,"known":2}}}
// Нэг story (нэг хичээл, эсвэл олон хичээлээс холимог «Өнөөдрийн давталт») үзсэнийг бүртгэнэ. Event.Detail нь
// "<slides>/<total>[ давталт][ карт <known>/<seen>]" (карт нь хичээл бүрт өөрийнх).
func (s *Server) handleStoryView(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Lessons []string              `json:"lessons"`
		Slides  int                   `json:"slides"`
		Total   int                   `json:"total"`
		Review  bool                  `json:"review"`
		Cards   map[string]storyCards `json:"cards"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !s.boardLim.Allow("story:" + c.UID) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон хүсэлт ирлээ — түр хүлээгээд дахин оролдоно уу")
		return
	}
	ids := []string{}
	for _, id := range in.Lessons {
		if lessonIDRe.MatchString(id) && !slices.Contains(ids, id) && len(ids) < 20 {
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 || in.Slides < 1 || in.Total < 1 || in.Slides > 500 || in.Total > 500 {
		writeErr(w, http.StatusBadRequest, "мэдээлэл буруу")
		return
	}
	// Зөвхөн өөрийн тэмдэглэлтэй хичээл (story нь тэмдэглэлээс бүтдэг): дурын хичээлд үзэлт бичиж, багшийн
	// статистикт мөр болж орохгүй.
	sets, err := s.store.UserHighlights(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	noted := make(map[string]bool, len(sets))
	for _, st := range sets {
		noted[st.LessonID] = true
	}
	ids = slices.DeleteFunc(ids, func(id string) bool { return !noted[id] })
	if len(ids) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	for _, id := range ids { // бусад түлхүүрийг алгасна
		if k, ok := in.Cards[id]; ok && (k.Known < 0 || k.Known > k.Seen || k.Seen > cardsMax) {
			writeErr(w, http.StatusBadRequest, "мэдээлэл буруу")
			return
		}
	}
	lessons, err := s.store.LessonsByIDs(r.Context(), ids)
	if s.storeErr(w, r, err) {
		return
	}
	detail := strconv.Itoa(min(in.Slides, in.Total)) + "/" + strconv.Itoa(in.Total)
	if in.Review {
		detail += " давталт"
	}
	name := s.displayName(r.Context(), c.UID, c.Name)
	evs := make([]store.ActivityEvent, 0, len(lessons))
	for _, l := range lessons {
		co, err := s.store.CourseByID(r.Context(), l.CourseID)
		if err != nil {
			continue
		}
		d := detail
		if k, ok := in.Cards[l.ID]; ok && k.Seen > 0 {
			d += " карт " + strconv.Itoa(k.Known) + "/" + strconv.Itoa(k.Seen)
		}
		evs = append(evs, store.ActivityEvent{UserID: c.UID, UserName: name, CourseID: co.ID, LessonID: l.ID, TeacherID: co.TeacherID, Type: storyViewType, Detail: d, At: time.Now()})
	}
	if len(evs) > 0 {
		if err := s.store.AddActivityEvents(r.Context(), evs); s.storeErr(w, r, err) {
			return
		}
		s.reviewDueChanged(c.UID) // давтах popup: story-гоор давтлаа
	}
	w.WriteHeader(http.StatusNoContent)
}

type noteCounts struct {
	Notes  int            `json:"notes"`
	Kinds  map[string]int `json:"kinds"`  // text | time | page | box
	Colors map[string]int `json:"colors"` // y | g | r | b
	Cards  int            `json:"cards"`  // flash card (асуулт ба санах хэсэгтэй) тэмдэглэл
}

// cardReview — story үзэлтийн давталтын тоо: seen — эргүүлж үзсэн карт, known — «Санасан».
type cardReview struct {
	Seen  int `json:"cards_seen"`
	Known int `json:"cards_known"`
}

// storyDetailCards — "<slides>/<total>[ давталт][ карт <known>/<seen>]"-ээс картын тоо (байхгүй бол 0, 0).
func storyDetailCards(detail string) (known, seen int) {
	f := strings.Fields(detail)
	for i := 0; i+1 < len(f); i++ {
		if f[i] != "карт" {
			continue
		}
		k, sn, ok := strings.Cut(f[i+1], "/")
		if !ok {
			return 0, 0
		}
		kn, err1 := strconv.Atoi(k)
		se, err2 := strconv.Atoi(sn)
		if err1 != nil || err2 != nil || kn < 0 || se < 0 || kn > se || se > cardsMax {
			return 0, 0
		}
		return kn, se
	}
	return 0, 0
}

func newNoteCounts() noteCounts {
	return noteCounts{Kinds: map[string]int{"text": 0, "time": 0, "page": 0, "box": 0}, Colors: map[string]int{"y": 0, "g": 0, "r": 0, "b": 0}}
}

func (n *noteCounts) add(h store.Highlight) {
	n.Notes++
	switch {
	case h.Kind == "page" && len(h.Box) > 0:
		n.Kinds["box"]++
	case h.Kind != "":
		n.Kinds[h.Kind]++
	default:
		n.Kinds["text"]++
	}
	n.Colors[h.Color]++
	if h.IsCard() {
		n.Cards++
	}
}

// GET /api/me/courses/{id}/notes-stats — багш: суралцагч бүр, хичээл бүрийн тэмдэглэл ба story үзэлт.
func (s *Server) handleNotesStats(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	ctx := r.Context()
	lessons, err := s.store.LessonsByCourse(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	lids := make([]string, len(lessons))
	for i, l := range lessons {
		lids[i] = l.ID
	}
	sets, err := s.store.LessonHighlights(ctx, lids)
	if s.storeErr(w, r, err) {
		return
	}
	views, err := s.store.ActivityEvents(ctx, store.ActivityFilter{CourseID: course.ID, Type: storyViewType}, 50000)
	if s.storeErr(w, r, err) {
		return
	}
	type studentRow struct {
		UserID    string     `json:"user_id"`
		Name      string     `json:"name"`
		Username  string     `json:"username,omitempty"`
		Lessons   int        `json:"lessons"` // тэмдэглэлтэй хичээл
		Views     int        `json:"story_views"`
		Done      int        `json:"story_done"` // эцэс хүртэл үзсэн
		LastNote  *time.Time `json:"last_note,omitempty"`
		LastView  *time.Time `json:"last_view,omitempty"`
		lessonSet map[string]bool
		noteCounts
		cardReview
	}
	type lessonRow struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Position int    `json:"position"`
		Students int    `json:"students"` // тэмдэглэл хөтөлсөн суралцагч
		Views    int    `json:"story_views"`
		noteCounts
		cardReview
	}
	rows := map[string]*studentRow{}
	row := func(uid string) *studentRow {
		if x := rows[uid]; x != nil {
			return x
		}
		x := &studentRow{UserID: uid, lessonSet: map[string]bool{}, noteCounts: newNoteCounts()}
		rows[uid] = x
		return x
	}
	lrows := map[string]*lessonRow{}
	order := []*lessonRow{}
	for _, l := range lessons {
		lr := &lessonRow{ID: l.ID, Title: l.Title, Position: l.Position, noteCounts: newNoteCounts()}
		lrows[l.ID] = lr
		order = append(order, lr)
	}
	total := newNoteCounts()
	for _, st := range sets {
		sr, lr := row(st.UserID), lrows[st.LessonID]
		if lr == nil {
			continue
		}
		sr.lessonSet[st.LessonID] = true
		lr.Students++
		if t := st.UpdatedAt; !t.IsZero() && (sr.LastNote == nil || t.After(*sr.LastNote)) {
			sr.LastNote = &t
		}
		for _, h := range st.Items {
			sr.add(h)
			lr.add(h)
			total.add(h)
		}
	}
	viewers, done := map[string]bool{}, 0
	var cardsAll cardReview
	reviewers := map[string]bool{} // flash card давтсан суралцагч
	for _, e := range views {
		sr := row(e.UserID)
		sr.Views++
		viewers[e.UserID] = true
		if p, t, ok := strings.Cut(strings.Fields(e.Detail + " ")[0], "/"); ok && p == t {
			sr.Done++
			done++
		}
		if known, seen := storyDetailCards(e.Detail); seen > 0 {
			sr.cardReview.Seen += seen
			sr.cardReview.Known += known
			cardsAll.Seen += seen
			cardsAll.Known += known
			reviewers[e.UserID] = true
			if lr := lrows[e.LessonID]; lr != nil {
				lr.cardReview.Seen += seen
				lr.cardReview.Known += known
			}
		}
		if t := e.At; sr.LastView == nil || t.After(*sr.LastView) {
			sr.LastView = &t
		}
		if sr.Name == "" {
			sr.Name = e.UserName
		}
		if lr := lrows[e.LessonID]; lr != nil {
			lr.Views++
		}
	}
	ids := make([]string, 0, len(rows))
	for uid, x := range rows {
		x.Lessons = len(x.lessonSet)
		ids = append(ids, uid)
	}
	if us, err := s.store.UsersByIDs(ctx, ids); err == nil {
		for _, u := range us {
			if x := rows[u.ID]; x != nil {
				x.Name, x.Username = u.DisplayName, u.Username
			}
		}
	}
	students := make([]*studentRow, 0, len(rows))
	for _, x := range rows {
		if x.Name == "" {
			x.Name = "Суралцагч"
		}
		students = append(students, x)
	}
	slices.SortFunc(students, func(a, b *studentRow) int {
		if a.Notes != b.Notes {
			return b.Notes - a.Notes
		}
		return b.Views - a.Views
	})
	writers := 0
	for _, x := range students {
		if x.Notes > 0 {
			writers++
		}
	}
	studentsTotal := len(students)
	if len(students) > 300 { // олон мянган суралцагчтай сургалтад хүснэгтэд эхний 300 (их тэмдэглэлтэй нь); нэгтгэл бүгдээр
		students = students[:300]
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": map[string]any{"writers": writers, "notes": total.Notes, "kinds": total.Kinds, "colors": total.Colors, "story_views": len(views), "viewers": len(viewers), "story_done": done,
			"cards": total.Cards, "cards_seen": cardsAll.Seen, "cards_known": cardsAll.Known, "card_reviewers": len(reviewers)},
		"students":       students,
		"students_total": studentsTotal,
		"lessons":        order,
	})
}
