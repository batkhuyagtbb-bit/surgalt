package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// Мэдэгдлийн төрлүүд
const (
	NotifProfileView = "profile_view"
	NotifCourseView  = "course_view"
	NotifLessonView  = "lesson_view"
	NotifEnroll      = "enroll"
	NotifPurchase    = "purchase"
	NotifMessage     = "message"
	NotifStorage     = "storage"
)

type viewKey struct{ teacher, kind, id string }

type viewAgg struct {
	count       int64
	title, link string
}

// notifier нь үзэлтүүдийг санах ойд нэгтгэж, flushEvery тутамд нэг мэдэгдэл болгоно.
// 10000 rps-д үзэлт бүрийг DB-д бичвэл сан хэт ачаалагдана — нэгтгэснээр
// 30 секундэд багш бүрт хамгийн ихдээ хэдхэн бичилт болно.
type notifier struct {
	mu      sync.Mutex
	views   map[viewKey]*viewAgg
	seen    map[string]struct{}  // ip|kind|id — нэг цонхонд нэг л удаа тоолно
	msgLast map[string]time.Time // яриа -> сүүлд мэдэгдсэн
}

const (
	flushEvery   = 30 * time.Second
	maxSeen      = 200_000
	msgThrottle  = 2 * time.Minute
	notifBodyLen = 140
)

func newNotifier() *notifier {
	return &notifier{views: map[viewKey]*viewAgg{}, seen: map[string]struct{}{}, msgLast: map[string]time.Time{}}
}

// trackView нь маш хурдан (зөвхөн map-д нэмнэ) тул халуун зам дээр аюулгүй.
func (s *Server) trackView(r *http.Request, teacherID, kind, id, title, link string) {
	n := s.notif
	dedupe := s.clientIP(r) + "|" + kind + "|" + id
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.seen) < maxSeen {
		if _, ok := n.seen[dedupe]; ok {
			return
		}
		n.seen[dedupe] = struct{}{}
	}
	k := viewKey{teacherID, kind, id}
	a := n.views[k]
	if a == nil {
		a = &viewAgg{title: title, link: link}
		n.views[k] = a
	}
	a.count++
}

func (s *Server) notifyLoop(ctx context.Context) {
	t := time.NewTicker(flushEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			s.flushViews(context.Background()) // зогсохдоо алдахгүй
			return
		case <-t.C:
			s.flushViews(ctx)
		}
	}
}

func (s *Server) flushViews(ctx context.Context) {
	n := s.notif
	n.mu.Lock()
	views := n.views
	n.views, n.seen = map[viewKey]*viewAgg{}, map[string]struct{}{}
	for k, t := range n.msgLast {
		if time.Since(t) > msgThrottle {
			delete(n.msgLast, k)
		}
	}
	n.mu.Unlock()
	if len(views) == 0 {
		return
	}
	profile, course := map[string]int64{}, map[string]int64{}
	var out []*store.Notification
	for k, a := range views {
		nt := &store.Notification{UserID: k.teacher, Type: k.kind, Link: a.link, Count: a.count}
		switch k.kind {
		case NotifProfileView:
			profile[k.teacher] += a.count
			nt.Title = fmt.Sprintf("🔥 %d хүн таны профайлыг үзлээ", a.count)
		case NotifCourseView:
			course[k.id] += a.count
			nt.Title = fmt.Sprintf("👀 %d хүн сургалтыг тань үзлээ", a.count)
			nt.Body = a.title
		case NotifLessonView:
			nt.Title = fmt.Sprintf("▶ %d хүн үнэгүй хичээлийг үзлээ", a.count)
			nt.Body = a.title
		}
		out = append(out, nt)
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := s.store.IncViews(cctx, profile, course); err != nil {
		s.log.Warn("views flush", "err", err)
	}
	s.saveNotifs(cctx, out...)
}

// notify нь шууд мэдэгдэл (худалдан авалт, элсэлт, мессеж) илгээнэ.
func (s *Server) notify(ctx context.Context, ns ...*store.Notification) {
	cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	s.saveNotifs(cctx, ns...)
}

func (s *Server) saveNotifs(ctx context.Context, ns ...*store.Notification) {
	if len(ns) == 0 {
		return
	}
	if err := s.store.AddNotifications(ctx, ns); err != nil {
		s.log.Warn("notifications", "err", err)
		return
	}
	if s.PublishNotif != nil {
		for _, n := range ns {
			s.PublishNotif(*n)
		}
	}
}

func short(s string) string {
	if utf8.RuneCountInString(s) <= notifBodyLen {
		return s
	}
	return string([]rune(s)[:notifBodyLen]) + "…"
}

// notifyMessage: зочны мессежийг багшид мэдэгдэнэ (яриа тутамд 2 минутад нэг удаа).
func (s *Server) notifyMessage(ctx context.Context, conv *store.Conversation, m *store.Message) {
	if m.Sender != store.SenderVisitor {
		return
	}
	n := s.notif
	n.mu.Lock()
	if t, ok := n.msgLast[conv.ID]; ok && time.Since(t) < msgThrottle {
		n.mu.Unlock()
		return
	}
	n.msgLast[conv.ID] = time.Now()
	n.mu.Unlock()
	title := "💬 " + conv.VisitorName + " танд бичлээ"
	if conv.Kind == store.ConvGroup {
		who := m.SenderName
		if who == "" {
			who = "Суралцагч"
		}
		title = "💬 " + who + " · " + conv.VisitorName + " бүлэгт бичлээ"
	}
	s.notify(ctx, &store.Notification{UserID: conv.TeacherID, Type: NotifMessage, Count: 1,
		Title: title, Body: short(m.Body), Link: "/me#chat=" + conv.ID})
}

// notifyPaid нь төлбөр баталгаажсаны дараа багш болон худалдан авагчид мэдэгдэнэ.
func (s *Server) notifyPaid(ctx context.Context, o *store.Order) {
	if o.Kind == store.OrderKindStorage {
		s.notify(ctx, &store.Notification{UserID: o.UserID, Type: NotifStorage, Count: 1,
			Title: "💾 Файлын сангийн багтаамж нэмэгдлээ", Body: fmt.Sprintf("%s · %d сар", sizeLabel(o.StorageMB), o.Months), Link: "/me#files"})
		return
	}
	if o.Kind == store.OrderKindBook {
		buyer := "Уншигч"
		if u, err := s.store.UserByID(ctx, o.UserID); err == nil {
			buyer = u.DisplayName
		}
		cctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		_ = s.store.AddBookEvent(cctx, store.BookEvent{BookID: o.BookID, TeacherID: o.TeacherID, UserID: o.UserID, UserName: buyer, Type: "purchase", Detail: money(o.Amount)})
		s.notify(ctx,
			&store.Notification{UserID: o.TeacherID, Type: NotifPurchase, Count: 1, Title: fmt.Sprintf("📚 Ном зарагдлаа · %s", money(o.Amount)), Body: buyer + " — " + o.Title, Link: "/me#books"},
			&store.Notification{UserID: o.UserID, Type: NotifPurchase, Count: 1, Title: "✅ Номыг худалдаж авлаа — бүтнээр нь уншина уу", Body: o.Title, Link: "/b/" + o.BookID})
		return
	}
	title := o.Title
	if title == "" {
		if c, err := s.store.CourseByID(ctx, o.CourseID); err == nil {
			title = c.Title
		}
	}
	buyer := "Суралцагч"
	if u, err := s.store.UserByID(ctx, o.UserID); err == nil {
		buyer = u.DisplayName
	}
	s.notify(ctx,
		&store.Notification{UserID: o.TeacherID, Type: NotifPurchase, Count: 1,
			Title: fmt.Sprintf("💰 Шинэ борлуулалт · %s", money(o.Amount)), Body: buyer + " — " + title, Link: "/me#overview"},
		&store.Notification{UserID: o.UserID, Type: NotifPurchase, Count: 1,
			Title: "✅ Худалдан авалт амжилттай — нээгдлээ", Body: title, Link: "/c/" + o.CourseID},
	)
}

func (s *Server) handleNotifications(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	items, unread, err := s.store.Notifications(r.Context(), c.UID, 50)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "unread": unread})
}

func (s *Server) handleNotificationsRead(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if err := s.store.MarkNotificationsRead(r.Context(), c.UID); s.storeErr(w, r, err) {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
