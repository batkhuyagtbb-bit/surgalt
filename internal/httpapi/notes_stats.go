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

// POST /api/me/story-view {"lessons":["…"],"slides":5,"total":7,"review":false}
// Нэг story (нэг хичээл, эсвэл олон хичээлээс холимог «Өнөөдрийн давталт») үзсэнийг бүртгэнэ.
func (s *Server) handleStoryView(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Lessons []string `json:"lessons"`
		Slides  int      `json:"slides"`
		Total   int      `json:"total"`
		Review  bool     `json:"review"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !s.boardLim.Allow("story:" + c.UID) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон хүсэлт")
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
		evs = append(evs, store.ActivityEvent{UserID: c.UID, UserName: name, CourseID: co.ID, LessonID: l.ID, TeacherID: co.TeacherID, Type: storyViewType, Detail: detail, At: time.Now()})
	}
	if len(evs) > 0 {
		if err := s.store.AddActivityEvents(r.Context(), evs); s.storeErr(w, r, err) {
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type noteCounts struct {
	Notes  int            `json:"notes"`
	Kinds  map[string]int `json:"kinds"`  // text | time | page | box
	Colors map[string]int `json:"colors"` // y | g | r | b
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
	}
	type lessonRow struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Position int    `json:"position"`
		Students int    `json:"students"` // тэмдэглэл хөтөлсөн суралцагч
		Views    int    `json:"story_views"`
		noteCounts
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
	for _, e := range views {
		sr := row(e.UserID)
		sr.Views++
		viewers[e.UserID] = true
		if p, t, ok := strings.Cut(strings.Fields(e.Detail + " ")[0], "/"); ok && p == t {
			sr.Done++
			done++
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
		"summary":        map[string]any{"writers": writers, "notes": total.Notes, "kinds": total.Kinds, "colors": total.Colors, "story_views": len(views), "viewers": len(viewers), "story_done": done},
		"students":       students,
		"students_total": studentsTotal,
		"lessons":        order,
	})
}
