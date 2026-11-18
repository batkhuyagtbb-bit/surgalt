package httpapi

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"surgalt/internal/store"
)

// ---- Давтах popup (мартах муруй) ----
// Мэдэгдлийн (review.go) «уншсан» төлвөөс үл хамааран суралцагч хичээлээ БОДИТООР давтах хүртэл popup гарна:
// сүүлийн reviewDueWindow-д илгээсэн сануулгаас, илгээснээс хойш тэр хичээлд эргэж ороогүй (сануулгаас хойш идэвхтэй
// суралцах сессгүй — last_at, story үзээгүй, «давтлаа» гэж тэмдэглээгүй) нь. Клиент 10 минутад нэг удаа дуудна; сервер
// хэрэглэгч бүрт reviewDueTTL кэштэй (сануулга илгээх, сесс эхлэх, «давтлаа», story үзэлтэд хүчингүй болно), нэг
// хэрэглэгчийн мөрүүдийг л уншина (review_reminders — user_id-аар, сесс — user_id bloom, үйл явдал — эзэмшигчийн
// teacher_id-аар).

const (
	reviewDueWindow = 7 * 24 * time.Hour
	reviewDueMax    = 5
	reviewDueTTL    = time.Minute
	reviewDoneType  = "review_done" // суралцагч «давтлаа» гэж тэмдэглэсэн (ж: flash card-аар давтсан)
)

// reviewDueChanged — хэрэглэгчийн popup-ийн жагсаалтын кэшийг хүчингүй болгоно.
func (s *Server) reviewDueChanged(uid string) { s.reviewDue.Delete(uid) }

// ReviewDue — popup-ийн нэг мөр.
type ReviewDue struct {
	CourseID    string    `json:"course_id"`
	CourseTitle string    `json:"course_title"`
	LessonID    string    `json:"lesson_id"`
	LessonTitle string    `json:"lesson_title"`
	Stage       int       `json:"stage"`
	ViewedAt    time.Time `json:"viewed_at"`
	SentAt      time.Time `json:"sent_at"`
	URL         string    `json:"url"`
}

// handleReviewsDue: GET /api/me/reviews/due — давтах хугацаа нь болсон хичээлүүд (үе шат өсөхөөр, шинээс; 5 хүртэл).
func (s *Server) handleReviewsDue(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, err := s.reviewDue.GetOrLoad(r.Context(), c.UID, func(ctx context.Context) ([]ReviewDue, error) {
		return s.reviewsDue(ctx, c.UID, time.Now(), "")
	})
	if s.storeErr(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// reviewsDue — sendReviews-тэй ижил шүүлт (нийтлэгдсэн, харагдах хичээл, эрхтэй, багш биш) + илгээснээс хойш давтаагүй.
// only хоосон биш бол зөвхөн тэр хичээлийг шалгана (handleReviewDone).
func (s *Server) reviewsDue(ctx context.Context, uid string, now time.Time, only string) ([]ReviewDue, error) {
	out := []ReviewDue{}
	sent, err := s.store.SentReviews(ctx, uid, now.Add(-reviewDueWindow))
	if err != nil || len(sent) == 0 {
		return out, err
	}
	// Хичээл бүрт сүүлийн сануулга: түүнээс хойш давтсан бол өмнөх үе шатууд нь ч давтагдсан.
	latest := map[string]store.ReviewItem{}
	for _, it := range sent {
		if only != "" && it.LessonID != only {
			continue
		}
		if cur, ok := latest[it.LessonID]; !ok || it.SentAt.After(cur.SentAt) || (it.SentAt.Equal(cur.SentAt) && it.Stage > cur.Stage) {
			latest[it.LessonID] = it
		}
	}
	if len(latest) == 0 {
		return out, nil
	}
	cand := make([]store.ReviewItem, 0, len(latest))
	for _, it := range latest {
		cand = append(cand, it)
	}
	courses, lessons, err := s.reviewTargets(ctx, cand)
	if err != nil {
		return out, err
	}
	since := now
	owners := map[string]bool{} // үйл явдлыг эзэмшигчийн teacher_id-аар (анхдагч түлхүүр) уншина
	kept := cand[:0]
	for _, it := range cand {
		c, l := courses[it.CourseID], lessons[it.LessonID]
		if c == nil || l == nil || l.CourseID != c.ID || !c.Published || hiddenFrom(l, c, uid) || teaches(c, uid) {
			continue
		}
		kept = append(kept, it)
		owners[c.TeacherID] = true
		if it.SentAt.Before(since) {
			since = it.SentAt
		}
	}
	if len(kept) == 0 {
		return out, nil
	}
	// Хичээл бүрт сүүлд эргэж орсон мөч: сессийн сүүлийн идэвх (19:00-ийг дамжин суралцаж байсан ч давтсанд тооцно),
	// story үзэлт, «давтлаа» тэмдэг.
	back := map[string]time.Time{}
	mark := func(lid string, at time.Time) {
		if at.After(back[lid]) {
			back[lid] = at
		}
	}
	ss, err := s.store.Sessions(ctx, store.ActivityFilter{UserID: uid, Since: since}, 2000)
	if err != nil {
		return out, err
	}
	for i := range ss {
		mark(ss[i].LessonID, ss[i].LastAt)
	}
	for tid := range owners {
		for _, typ := range []string{storyViewType, reviewDoneType} {
			evs, err := s.store.ActivityEvents(ctx, store.ActivityFilter{TeacherID: tid, UserID: uid, Type: typ, Since: since}, 1000)
			if err != nil {
				return out, err
			}
			for _, e := range evs {
				mark(e.LessonID, e.At)
			}
		}
	}
	due := kept[:0]
	for _, it := range kept {
		if back[it.LessonID].Before(it.SentAt) {
			due = append(due, it)
		}
	}
	sort.Slice(due, func(i, j int) bool {
		if due[i].Stage != due[j].Stage {
			return due[i].Stage < due[j].Stage
		}
		if !due[i].SentAt.Equal(due[j].SentAt) {
			return due[i].SentAt.After(due[j].SentAt)
		}
		return due[i].LessonID < due[j].LessonID
	})
	// Эрхийг (lessonAccess — хэд хэдэн уншилт) зөвхөн хариунд орох хүртэл шалгана.
	for _, it := range due {
		if len(out) == reviewDueMax {
			break
		}
		c, l := courses[it.CourseID], lessons[it.LessonID]
		open, err := s.lessonAccess(ctx, uid, c, l)
		if err != nil {
			return []ReviewDue{}, err
		}
		if !open {
			continue
		}
		out = append(out, ReviewDue{CourseID: c.ID, CourseTitle: c.Title, LessonID: l.ID, LessonTitle: l.Title, Stage: it.Stage,
			SentAt: it.SentAt, URL: "/c/" + c.ID + "#l=" + l.ID})
	}
	// Анх үзсэн цаг: сургалт бүрт нэг уншилт (байхгүй бол сануулгын үе шатаас).
	prog := map[string]map[string]store.LessonProgress{}
	for i := range out {
		it := &out[i]
		pr, ok := prog[it.CourseID]
		if !ok {
			pr, err = s.store.LessonProgress(ctx, uid, it.CourseID)
			if err != nil {
				return []ReviewDue{}, err
			}
			prog[it.CourseID] = pr
		}
		if p, ok := pr[it.LessonID]; ok && p.ViewedAt.Unix() > 0 {
			it.ViewedAt = p.ViewedAt
		} else {
			it.ViewedAt = it.SentAt.AddDate(0, 0, -it.Stage)
		}
	}
	return out, nil
}

// handleReviewDone: POST /api/me/reviews/{lid}/done — суралцагч хичээлээ давтсан гэж тэмдэглэнэ (review_done үйл явдал);
// дараагийн due-д орохгүй. Хичээл олдохгүй / харагдахгүй / эрхгүй / хаагдсан (lessonBlocked) бол 404. Зөвхөн тэр хичээл
// одоо дуудагчийн due жагсаалтад байвал бичнэ, үгүй бол 204 (юу ч бичихгүй) — үйл явдлыг давтан бичиж lessonBlocks-ийн
// уншдаг цонхноос auto_block-ийг түлхэж гаргахаас сэргийлнэ. Багшид юу ч бичихгүй (204).
func (s *Server) handleReviewDone(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !s.doneLim.Allow(c.UID) { // хэт олон дуудлага: кэшгүй due асуулгыг ажиллуулахгүй, чимээгүй алгасна
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ctx := r.Context()
	ls, err := s.store.LessonsByIDs(ctx, []string{r.PathValue("lid")})
	if s.storeErr(w, r, err) {
		return
	}
	if len(ls) == 0 {
		writeErr(w, http.StatusNotFound, "хичээл олдсонгүй")
		return
	}
	l := &ls[0]
	course, err := s.store.CourseByID(ctx, l.CourseID)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "хичээл олдсонгүй")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	if teaches(course, c.UID) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	open := false
	if course.Published && !hiddenFrom(l, course, c.UID) {
		if open, err = s.lessonAccess(ctx, c.UID, course, l); s.storeErr(w, r, err) {
			return
		}
	}
	if !open || s.lessonBlocked(ctx, c.UID, course, l.ID) != nil {
		writeErr(w, http.StatusNotFound, "хичээл олдсонгүй")
		return
	}
	due, err := s.reviewsDue(ctx, c.UID, time.Now(), l.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if len(due) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ev := store.ActivityEvent{UserID: c.UID, UserName: s.displayName(ctx, c.UID, c.Name), CourseID: course.ID, LessonID: l.ID,
		TeacherID: course.TeacherID, Type: reviewDoneType, At: time.Now()}
	if err := s.store.AddActivityEvents(ctx, []store.ActivityEvent{ev}); s.storeErr(w, r, err) {
		return
	}
	s.reviewDueChanged(c.UID)
	w.WriteHeader(http.StatusNoContent)
}
