package httpapi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"surgalt/internal/store"
)

// ---- Давтах сануулга (мартах муруй, Эббингауз) ----
// Суралцагч хичээлийг анх үзсэнээс 1, 3, 7, 14, 30 хоногийн дараа орой (reviewHour, Улаанбаатар) «хичээлээ давтаарай»
// мэдэгдэл холбоостой очно. Нэг сургалтын хэд хэдэн хичээл нэг өдөр таарвал нэг мэдэгдэлд нэгтгэнэ. Сервер унтарсан
// өдрийнхийг дараагийн өдөр нөхнө; (суралцагч, хичээл, үе шат) бүр нэг л удаа (store.ClaimReviews).

var reviewStages = []int{1, 3, 7, 14, 30}

const (
	reviewHour      = 19 // 19:00-оос хойш — оройн суралцах цаг
	reviewMaxPerDay = 3  // суралцагч бүрт өдөрт хамгийн ихдээ (сургалтаар)
)

// reviewLoop — 10 минут тутам шалгаж, reviewHour өнгөрсөн бол өдөрт нэг удаа сануулна.
func (s *Server) reviewLoop(ctx context.Context) {
	first := time.NewTimer(3 * time.Minute)
	defer first.Stop()
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-first.C:
			s.reviewTick(ctx, time.Now())
		case <-t.C:
			s.reviewTick(ctx, time.Now())
		}
	}
}

func (s *Server) reviewTick(ctx context.Context, now time.Time) {
	local := now.In(mnLoc)
	day := local.Format("2006-01-02")
	s.reviewMu.Lock()
	done := s.reviewDay == day
	s.reviewMu.Unlock()
	if done || local.Hour() < reviewHour {
		return
	}
	n, err := s.sendReviews(ctx, now)
	if err != nil {
		s.log.Warn("давтах сануулга", "err", err)
		return // дараагийн шалгалтаар дахин оролдоно (илгээснийг давтахгүй)
	}
	s.reviewMu.Lock()
	s.reviewDay = day
	s.reviewMu.Unlock()
	if n > 0 {
		s.log.Info("давтах сануулга илгээв", "notifications", n)
	}
}

// sendReviews — өнөөдөр давтах хичээлүүдийг олж, эрхтэйг нь шүүгээд мэдэгдэнэ; илгээсэн мэдэгдлийн тоо.
func (s *Server) sendReviews(ctx context.Context, now time.Time) (int, error) {
	local := now.In(mnLoc)
	today := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, mnLoc)
	var due []store.ReviewItem
	for _, d := range reviewStages {
		// d хоногийн өмнө анх үзсэн; өмнөх өдрийнх (сервер унтарсан бол) нөхөгдөнө — тэмдэглэснийг давтахгүй.
		items, err := s.store.ReviewDue(ctx, today.AddDate(0, 0, -d-1), today.AddDate(0, 0, -d+1))
		if err != nil {
			return 0, err
		}
		for i := range items {
			items[i].Stage = d
		}
		due = append(due, items...)
	}
	if len(due) == 0 {
		return 0, nil
	}
	courses, lessons, err := s.reviewTargets(ctx, due)
	if err != nil {
		return 0, err
	}
	ok := due[:0]
	for _, it := range due {
		c, l := courses[it.CourseID], lessons[it.LessonID]
		if c == nil || l == nil || !c.Published || hiddenFrom(l, c, it.UserID) || teaches(c, it.UserID) {
			continue
		}
		if open, err := s.lessonAccess(ctx, it.UserID, c, l); err != nil || !open {
			continue
		}
		ok = append(ok, it)
	}
	claimed, err := s.store.ClaimReviews(ctx, ok)
	if err != nil {
		return 0, err
	}
	ns := reviewNotices(claimed, courses, lessons)
	for _, n := range ns {
		s.notify(ctx, n)
	}
	return len(ns), nil
}

// reviewTargets — сургалт, хичээлүүдийг нэг дор (устгагдсан нь орохгүй).
func (s *Server) reviewTargets(ctx context.Context, due []store.ReviewItem) (map[string]*store.Course, map[string]*store.Lesson, error) {
	cids, lids, seen := []string{}, []string{}, map[string]bool{}
	for _, it := range due {
		if !seen["c"+it.CourseID] {
			seen["c"+it.CourseID] = true
			cids = append(cids, it.CourseID)
		}
		if !seen["l"+it.LessonID] {
			seen["l"+it.LessonID] = true
			lids = append(lids, it.LessonID)
		}
	}
	cs, err := s.store.CoursesByIDs(ctx, cids)
	if err != nil {
		return nil, nil, err
	}
	ls, err := s.store.LessonsByIDs(ctx, lids)
	if err != nil {
		return nil, nil, err
	}
	courses, lessons := map[string]*store.Course{}, map[string]*store.Lesson{}
	for i := range cs {
		courses[cs[i].ID] = &cs[i]
	}
	for i := range ls {
		lessons[ls[i].ID] = &ls[i]
	}
	return courses, lessons, nil
}

// reviewNotices — суралцагч, сургалт бүрээр нэг мэдэгдэл (өдөрт reviewMaxPerDay хүртэл), холбоос — эхний хичээл.
func reviewNotices(items []store.ReviewItem, courses map[string]*store.Course, lessons map[string]*store.Lesson) []*store.Notification {
	type group struct {
		user, course string
		items        []store.ReviewItem
	}
	byKey := map[string]*group{}
	var groups []*group
	for _, it := range items {
		k := it.UserID + "|" + it.CourseID
		g := byKey[k]
		if g == nil {
			g = &group{user: it.UserID, course: it.CourseID}
			byKey[k] = g
			groups = append(groups, g)
		}
		g.items = append(g.items, it)
	}
	perUser := map[string]int{}
	var out []*store.Notification
	for _, g := range groups {
		if perUser[g.user] >= reviewMaxPerDay {
			continue
		}
		perUser[g.user]++
		// Хамгийн эрт үе шат (шинээр үзсэн) эхэнд — мартахад хамгийн ойр.
		sort.SliceStable(g.items, func(i, j int) bool { return g.items[i].Stage < g.items[j].Stage })
		c, first := courses[g.course], g.items[0]
		n := &store.Notification{UserID: g.user, Type: "review", Count: int64(len(g.items)), Link: "/c/" + g.course + "#l=" + first.LessonID}
		if len(g.items) == 1 {
			title := lessons[first.LessonID].Title
			n.Title = "🧠 Давтах цаг: «" + title + "»"
			n.Body = fmt.Sprintf("«%s» хичээлийг үзээд %s өнгөрлөө — мартахаас өмнө давтаарай. 1-3-7-14-30 хоногийн давталт мэдлэгийг удаан хадгална.",
				title, reviewAgo(first.Stage))
		} else {
			parts := make([]string, 0, 3)
			for i, it := range g.items {
				if i == 3 {
					parts = append(parts, fmt.Sprintf("өөр %d", len(g.items)-3))
					break
				}
				parts = append(parts, fmt.Sprintf("«%s» (%s)", lessons[it.LessonID].Title, reviewAgo(it.Stage)))
			}
			n.Title = fmt.Sprintf("🧠 Давтах цаг: %d хичээл · %s", len(g.items), c.Title)
			n.Body = strings.Join(parts, ", ") + " — үзсэнээс хойш хугацаа өнгөрлөө, мартахаас өмнө давтаарай."
		}
		out = append(out, n)
	}
	return out
}

func reviewAgo(days int) string {
	if days == 30 {
		return "1 сар"
	}
	return fmt.Sprintf("%d хоног", days)
}
