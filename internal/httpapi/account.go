package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"surgalt/internal/store"
)

// ---- Хууль, итгэл: өгөгдлөө татаж авах, бүртгэлээ устгах (Хувь хүний мэдээлэл хамгаалах тухай хууль) ----

// liveUser — токены эзэн бүртгэлээ устгаагүй эсэх (5 минут кэшлэнэ; устгахад шууд хасна). Түр зуурын алдаанд
// нэвтрэлтийг хаахгүй.
func (s *Server) liveUser(ctx context.Context, uid string) bool {
	_, err := s.alive.GetOrLoad(ctx, uid, func(ctx context.Context) (bool, error) {
		_, err := s.store.UserByID(ctx, uid)
		return err == nil, err
	})
	return !errors.Is(err, store.ErrNotFound)
}

const (
	exportMax     = 5000 // хүснэгт бүрээс авах дээд мөр
	exportConvMax = 2000 // нэг ярианаас авах дээд мессеж
)

// exportConv — татаж авах файл дахь нэг яриа.
type exportConv struct {
	ID       string          `json:"id"`
	Kind     string          `json:"kind"` // "" (багштай) | dm | group | team
	Title    string          `json:"title"`
	Messages []store.Message `json:"messages"`
}

// allMessages — ярианы мессежийг хуучнаас шинэ рүү (дээд тал нь max) хуудаслаж уншина.
func (s *Server) allMessages(ctx context.Context, convID string, max int) ([]store.Message, error) {
	var out []store.Message
	before := ""
	for len(out) < max {
		page, err := s.store.Messages(ctx, convID, before, 200)
		if err != nil {
			return nil, err
		}
		out = append(page, out...)
		if len(page) < 200 {
			break
		}
		before = page[0].ID
	}
	return out, nil
}

// handleExport: GET /api/me/export — өөрийн хувийн мэдээллийг JSON файлаар татаж авна: профайл, зөвшөөрөл,
// захиалга, элсэлт, суралцахуй, хичээлийн хяналтын бүртгэл, цаг захиалга, мэдэгдэл, чат. Бүлэг чатаас зөвхөн
// өөрийн бичсэн мессеж орно (бусдын мэдээлэл тул).
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	u, err := s.store.UserByID(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	me := s.me(u)
	out := map[string]any{
		"exported_at": time.Now(),
		"about":       "surgalt.mn дээрх таны хувийн мэдээлэл (Хувь хүний мэдээлэл хамгаалах тухай хууль). Нууцлалын бодлого: /privacy",
		"profile":     me,
	}
	add := func(name string, v any, err error) {
		if err != nil {
			s.log.Warn("export", "part", name, "err", err)
			out[name] = nil
			return
		}
		out[name] = v
	}
	f := store.ActivityFilter{UserID: c.UID}
	orders, err := s.store.UserOrders(ctx, c.UID, exportMax)
	add("orders", orders, err)
	courses, err := s.store.EnrolledCourses(ctx, c.UID)
	type brief struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	enrolled := make([]brief, len(courses))
	for i, x := range courses {
		enrolled[i] = brief{x.ID, x.Title}
	}
	add("enrolled_courses", enrolled, err)
	attempts, err := s.store.ExamAttempts(ctx, f)
	add("exam_attempts", attempts, err)
	sessions, err := s.store.Sessions(ctx, f, exportMax)
	add("study_sessions", sessions, err)
	events, err := s.store.ActivityEvents(ctx, f, exportMax)
	add("monitoring_events", events, err)
	quiz, err := s.store.QuizLogs(ctx, f, exportMax)
	add("quiz_answers", quiz, err)
	refl, err := s.store.Reflections(ctx, f, exportMax)
	add("reflections", refl, err)
	watches, err := s.store.VideoWatches(ctx, f)
	add("video_watches", watches, err)
	bookings, err := s.store.StudentSlots(ctx, c.UID, time.Time{})
	add("bookings", bookings, err)
	notifs, _, err := s.store.Notifications(ctx, c.UID, 500)
	add("notifications", notifs, err)
	if u.Role == store.RoleTeacher {
		own, err := s.store.CoursesByTeacher(ctx, c.UID, false)
		mine := make([]brief, len(own))
		for i, x := range own {
			mine[i] = brief{x.ID, x.Title}
		}
		add("my_courses", mine, err)
		slots, err := s.store.Slots(ctx, c.UID, time.Time{}, time.Now().AddDate(2, 0, 0))
		add("my_slots", slots, err)
	}

	// Чат: багштай ба ангийн найзын (1:1) яриа бүтнээрээ, бүлгээс зөвхөн өөрийн мессеж.
	var convs []store.Conversation
	for _, load := range []func(context.Context, string, int) ([]store.Conversation, error){
		s.store.UserConversations, s.store.UserDMConversations, s.store.UserTeamConversations, s.store.UserGroupConversations,
	} {
		cs, err := load(ctx, c.UID, 500)
		if err != nil {
			s.log.Warn("export chats", "err", err)
			continue
		}
		convs = append(convs, cs...)
	}
	if u.Role == store.RoleTeacher {
		if cs, err := s.store.TeacherConversations(ctx, c.UID, 500); err == nil {
			convs = append(convs, cs...)
		}
	}
	seen, chats := map[string]bool{}, []exportConv{}
	for _, cv := range convs {
		if seen[cv.ID] {
			continue
		}
		seen[cv.ID] = true
		msgs, err := s.allMessages(ctx, cv.ID, exportConvMax)
		if err != nil {
			s.log.Warn("export messages", "err", err, "conv", cv.ID)
			continue
		}
		if !oneToOne(&cv) {
			own := msgs[:0]
			for _, m := range msgs {
				if m.SenderID == c.UID {
					own = append(own, m)
				}
			}
			msgs = own
		}
		for i := range msgs {
			msgs[i].Attachment, msgs[i].AttachmentURL = "", "" // хугацаатай холбоос татсан файлд хэрэггүй
		}
		chats = append(chats, exportConv{ID: cv.ID, Kind: cv.Kind, Title: cv.VisitorName, Messages: msgs})
	}
	out["chats"] = chats

	name := u.Username
	if name == "" {
		name = "me"
	}
	w.Header().Set("Content-Disposition", `attachment; filename="surgalt-`+name+`-`+time.Now().In(mnLoc).Format("2006-01-02")+`.json"`)
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, out)
}

// deleteConfirm — бүртгэл устгахыг баталгаажуулах үг.
const deleteConfirm = "УСТГАХ"

var errSkip = errors.New("skip")

// handleDeleteAccount: POST /api/me/delete {confirm:"УСТГАХ"} — бүртгэл ба хувийн мэдээллийг устгана
// (store.DeleteUserData). Сургалт, номтой эсвэл захиалгатай цагтай багшийг эхлээд тэдгээрийг шийдвэрлүүлнэ.
func (s *Server) handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Confirm string `json:"confirm"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.ToUpper(strings.TrimSpace(in.Confirm)) != deleteConfirm {
		writeErr(w, http.StatusBadRequest, "баталгаажуулахын тулд «"+deleteConfirm+"» гэж бичнэ үү")
		return
	}
	ctx, now := r.Context(), time.Now()
	u, err := s.store.UserByID(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	if u.Role == store.RoleTeacher {
		courses, err := s.store.CoursesByTeacher(ctx, u.ID, false)
		if s.storeErr(w, r, err) {
			return
		}
		books, err := s.store.BooksByTeacher(ctx, u.ID, false)
		if s.storeErr(w, r, err) {
			return
		}
		if len(courses) > 0 || len(books) > 0 {
			writeErr(w, http.StatusConflict, "Сургалт, номтой багшийн бүртгэлийг шууд устгахгүй — суралцагчдын төлбөр, хандалт хамааралтай. Эхлээд сургалт, номоо устгах эсвэл бидэнтэй холбогдоно уу.")
			return
		}
		slots, err := s.store.Slots(ctx, u.ID, now, now.AddDate(2, 0, 0))
		if s.storeErr(w, r, err) {
			return
		}
		var free []string
		for _, sl := range slots {
			if !sl.Free(now) {
				writeErr(w, http.StatusConflict, "Захиалгатай цагаа эхлээд цуцлана уу (Хуваарь → Захиалгууд).")
				return
			}
			free = append(free, sl.ID)
		}
		if _, err := s.store.DeleteFreeSlots(ctx, free, now); s.storeErr(w, r, err) {
			return
		}
	}
	s.releaseBookings(ctx, u)
	if err := s.store.DeleteUserData(ctx, u.ID); s.storeErr(w, r, err) {
		return
	}
	if err := s.files.DeleteOwner(u.ID); err != nil {
		s.log.Warn("бүртгэл устгах: файл", "err", err, "user", u.ID)
	}
	s.alive.Delete(u.ID)
	s.profiles.Delete(u.Username)
	s.log.Info("бүртгэл устгагдлаа", "user", u.ID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// releaseBookings — бүртгэлээ устгасан суралцагчийн удахгүй болох цагийг чөлөөлж, багшид нь мэдэгдэнэ.
func (s *Server) releaseBookings(ctx context.Context, u *store.User) {
	slots, err := s.store.StudentSlots(ctx, u.ID, time.Now())
	if err != nil {
		s.log.Warn("бүртгэл устгах: захиалга", "err", err)
		return
	}
	byTeacher, order := map[string][]string{}, []string{}
	for _, sl := range slots {
		meeting := sl.MeetingID
		if _, err := s.store.UpdateSlot(ctx, sl.ID, func(x *store.Slot) error {
			if x.StudentID != u.ID {
				return errSkip
			}
			x.StudentID, x.StudentName, x.Note, x.HoldBy, x.HoldUntil, x.MeetingID, x.MeetURL = "", "", "", "", nil, "", ""
			return nil
		}); err != nil {
			continue
		}
		if meeting != "" {
			_ = s.store.DeleteMeeting(ctx, meeting)
		}
		when := slotWhen(sl.StartsAt)
		if sl.Price > 0 {
			when += " (төлбөртэй)"
		}
		if _, ok := byTeacher[sl.TeacherID]; !ok {
			order = append(order, sl.TeacherID)
		}
		byTeacher[sl.TeacherID] = append(byTeacher[sl.TeacherID], when)
	}
	var ns []*store.Notification
	for _, tid := range order {
		ns = append(ns, &store.Notification{UserID: tid, Type: NotifBooking, Link: "/me#live",
			Title: "📅 Суралцагч бүртгэлээ устгасан — цаг чөлөөлөгдлөө", Body: strings.Join(byTeacher[tid], ", ")})
	}
	if len(ns) > 0 {
		s.notify(ctx, ns...)
	}
}
