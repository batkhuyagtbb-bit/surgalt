package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// ---- Цаг захиалга: багш календарьтаа сул цагаа тэмдэглэнэ → суралцагч профайлаас сонгож захиална (үнэгүй бол
// шууд, төлбөртэй бол QR төлбөрийн дараа) → «Шууд хичээл»-д нэмэгдэж, онлайн бол Google Meet холбоос үүснэ ----

const (
	slotHold       = 20 * time.Minute    // төлбөр хүлээх хугацаанд цагийг өөр хүн авахгүй
	slotLead       = 5 * time.Minute     // эхлэхэд үүнээс бага хугацаа үлдсэн цагийг захиалахгүй
	slotHorizon    = 60 * 24 * time.Hour // зочдод харагдах хугацаа
	maxSlotsPerReq = 60
	maxWeeks       = 53   // давталт: улирал (13), жил (52) долоо хоног
	maxExpanded    = 800  // нэг хүсэлтээр үүсэх цагийн дээд хэмжээ (давталтын дараа)
	maxFutureSlots = 3000 // багшийн нийт цаг
	slotMaxAhead   = 400 * 24 * time.Hour
)

var errSlotTaken = errors.New("slot taken")

var mnWeekdays = [...]string{"Ням", "Даваа", "Мягмар", "Лхагва", "Пүрэв", "Баасан", "Бямба"}

// slotWhen: «10/14 (Мягмар) 15:00» — Улаанбаатарын цагаар.
func slotWhen(t time.Time) string {
	t = t.In(mnLoc)
	return t.Format("01/02") + " (" + mnWeekdays[t.Weekday()] + ") " + t.Format("15:04")
}

// slotPlace: онлайн эсэх, биечлэн бол хаана.
func slotPlace(sl *store.Slot) string {
	if sl.Mode == store.SlotOffline {
		return "Биечлэн · " + sl.Location
	}
	if sl.MeetURL == "" {
		return "Онлайн · холбоосыг багш илгээнэ"
	}
	return "Онлайн · Google Meet"
}

// PublicSlot — зочдод харагдах сул цаг (хэн захиалсан, тэмдэглэл зэрэг хэзээ ч орохгүй).
type PublicSlot struct {
	ID          string    `json:"id"`
	StartsAt    time.Time `json:"starts_at"`
	DurationMin int       `json:"duration_min"`
	Mode        string    `json:"mode"`
	Price       int64     `json:"price"`
	Location    string    `json:"location,omitempty"`
}

func publicSlot(sl *store.Slot) PublicSlot {
	return PublicSlot{ID: sl.ID, StartsAt: sl.StartsAt, DurationMin: sl.DurationMin, Mode: sl.Mode, Price: sl.Price, Location: sl.Location}
}

// freeSlotCount — профайл дээр «Цаг авах» товч гаргах эсэх (кэштэй хуудсанд).
func (s *Server) freeSlotCount(ctx context.Context, teacherID string) int {
	now := time.Now()
	ss, err := s.store.Slots(ctx, teacherID, now.Add(slotLead), now.Add(slotHorizon))
	if err != nil {
		return 0
	}
	n := 0
	for i := range ss {
		if ss[i].Free(now) {
			n++
		}
	}
	return n
}

// handleMySlots: GET /api/me/slots?from=2026-10-12&days=7 — багшийн календарь (бүх төлөвтэй).
func (s *Server) handleMySlots(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	from, err := time.ParseInLocation("2006-01-02", r.URL.Query().Get("from"), mnLoc)
	if err != nil {
		now := time.Now().In(mnLoc)
		from = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, mnLoc)
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 || days > 62 {
		days = 7
	}
	ss, err := s.store.Slots(r.Context(), c.UID, from, from.AddDate(0, 0, days))
	if s.storeErr(w, r, err) {
		return
	}
	type mySlot struct {
		store.Slot
		Held bool `json:"held,omitempty"` // төлбөр хүлээгдэж буй
	}
	now := time.Now()
	out := make([]mySlot, len(ss))
	for i := range ss {
		out[i] = mySlot{Slot: ss[i], Held: ss[i].StudentID == "" && !ss[i].Free(now)}
		if !out[i].Held && ss[i].StudentID == "" {
			out[i].StudentName = "" // хүлээлт дууссан бол хуучин нэрийг харуулахгүй
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAddSlots: POST /api/me/slots {starts: [RFC3339], duration_min, mode, price, location, weeks} — календарь дээр
// дарж тэмдэглэсэн сул цагууд. weeks > 1 бол долоо хоног бүр давтана (улирал 13, жил 52): цаг бүр өөрийн цувралтай
// (series_id) — давталтыг дараа нь хамт хасна; давтахад өөр цагтай давхцах долоо хоногийг алгасна. Давталтгүй үед
// давхцвал хадгалахгүй (409).
func (s *Server) handleAddSlots(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	var in struct {
		Starts      []time.Time `json:"starts"`
		DurationMin int         `json:"duration_min"`
		Mode        string      `json:"mode"`
		Price       int64       `json:"price"`
		Location    string      `json:"location"`
		Weeks       int         `json:"weeks"` // 1 (давтахгүй) … 53
	}
	if !decode(w, r, &in) {
		return
	}
	in.Location = strings.Join(strings.Fields(in.Location), " ")
	in.Weeks = max(in.Weeks, 1)
	now := time.Now()
	switch {
	case in.Weeks > maxWeeks:
		writeErr(w, http.StatusBadRequest, "давталт 53 долоо хоногоос ихгүй")
		return
	case len(in.Starts)*in.Weeks > maxExpanded:
		writeErr(w, http.StatusBadRequest, "хэт олон цаг — сонголт эсвэл давталтаа багасгана уу")
		return
	case len(in.Starts) == 0 || len(in.Starts) > maxSlotsPerReq:
		writeErr(w, http.StatusBadRequest, "1-60 цаг сонгоно уу")
		return
	case in.DurationMin < 15 || in.DurationMin > 240:
		writeErr(w, http.StatusBadRequest, "үргэлжлэх хугацаа 15-240 минут")
		return
	case in.Mode != store.SlotOnline && in.Mode != store.SlotOffline:
		writeErr(w, http.StatusBadRequest, "онлайн эсвэл биечлэн")
		return
	case in.Price < 0 || in.Price > maxPrice:
		writeErr(w, http.StatusBadRequest, "үнэ 0-100,000,000₮")
		return
	case utf8.RuneCountInString(in.Location) > 120:
		writeErr(w, http.StatusBadRequest, "байршил 120 тэмдэгтээс ихгүй")
		return
	}
	if in.Mode == store.SlotOffline && in.Location == "" {
		if u, err := s.store.UserByID(r.Context(), c.UID); err == nil {
			in.Location = u.Location
		}
		if in.Location == "" {
			writeErr(w, http.StatusBadRequest, "биечлэн уулзах газраа бичнэ үү")
			return
		}
	}
	if in.Mode == store.SlotOnline {
		in.Location = ""
	}
	existing, err := s.store.Slots(r.Context(), c.UID, now.Add(-24*time.Hour), now.Add(slotMaxAhead+60*24*time.Hour))
	if s.storeErr(w, r, err) {
		return
	}
	if len(existing)+len(in.Starts)*in.Weeks > maxFutureSlots {
		writeErr(w, http.StatusBadRequest, "хэт олон сул цаг — хуучнаа цэвэрлэнэ үү")
		return
	}
	dur := time.Duration(in.DurationMin) * time.Minute
	overlaps := func(a time.Time, b *store.Slot) bool { return a.Before(b.End()) && b.StartsAt.Before(a.Add(dur)) }
	busy := func(st time.Time, add []*store.Slot) bool {
		for i := range existing {
			if overlaps(st, &existing[i]) {
				return true
			}
		}
		for _, o := range add {
			if overlaps(st, o) {
				return true
			}
		}
		return false
	}
	var add []*store.Slot
	for _, st0 := range in.Starts {
		st0 = st0.Truncate(time.Minute)
		if st0.Before(now.Add(slotLead)) || st0.After(now.Add(slotMaxAhead)) {
			writeErr(w, http.StatusBadRequest, "өнгөрсөн эсвэл хэт хол цаг")
			return
		}
		series := ""
		if in.Weeks > 1 {
			series = randHex(8)
		}
		for wk := 0; wk < in.Weeks; wk++ {
			st := st0.AddDate(0, 0, 7*wk) // Улаанбаатарт зуны цаг байхгүй — цагийн зөрүү гарахгүй
			if busy(st, add) {
				if in.Weeks == 1 {
					writeErr(w, http.StatusConflict, "Энэ цаг өөр тэмдэглэсэн цагтай давхцаж байна")
					return
				}
				continue // давталтад: тэр долоо хоногийг алгасна
			}
			add = append(add, &store.Slot{TeacherID: c.UID, StartsAt: st, DurationMin: in.DurationMin, Mode: in.Mode, Price: in.Price,
				Location: in.Location, SeriesID: series})
		}
	}
	if len(add) == 0 {
		writeErr(w, http.StatusConflict, "Эдгээр цаг аль хэдийн тэмдэглэгдсэн байна")
		return
	}
	if err := s.store.AddSlots(r.Context(), add); s.storeErr(w, r, err) {
		return
	}
	s.profiles.Delete(c.Name) // профайл дээрх «Цаг авах» товч шууд гарна
	writeJSON(w, http.StatusCreated, add)
}

// handleDeleteSlot: DELETE /api/me/slots/{id}[?series=1] — сул цагийг хасна; захиалсан бол захиалгыг цуцалж (Шууд
// хичээлээс хасна) суралцагчид мэдэгдэнэ. series=1 бол энэ цагаас хойших давталтын СУЛ цагуудыг хамт хасна
// (захиалсан, төлбөр хүлээгдэж буй цаг хэвээр үлдэнэ).
func (s *Server) handleDeleteSlot(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	sl, err := s.store.SlotByID(r.Context(), r.PathValue("id"))
	if err != nil || sl.TeacherID != c.UID {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	if r.URL.Query().Get("series") == "1" && sl.SeriesID != "" && sl.StudentID == "" {
		ss, err := s.store.Slots(r.Context(), c.UID, sl.StartsAt, sl.StartsAt.Add(slotMaxAhead+60*24*time.Hour))
		if s.storeErr(w, r, err) {
			return
		}
		var ids []string
		for i := range ss {
			if ss[i].SeriesID == sl.SeriesID {
				ids = append(ids, ss[i].ID)
			}
		}
		n, err := s.store.DeleteFreeSlots(r.Context(), ids, time.Now())
		if s.storeErr(w, r, err) {
			return
		}
		s.profiles.Delete(c.Name)
		writeJSON(w, http.StatusOK, map[string]int{"deleted": n})
		return
	}
	if err := s.store.DeleteSlot(r.Context(), sl.ID); s.storeErr(w, r, err) {
		return
	}
	if sl.StudentID != "" {
		if sl.MeetingID != "" {
			if err := s.store.DeleteMeeting(r.Context(), sl.MeetingID); err != nil {
				s.log.Warn("цуцалсан уулзалтыг хасах", "err", err)
			}
		}
		body := "Өөр цаг сонгож дахин захиална уу."
		if sl.Price > 0 {
			body = "Төлсөн мөнгийг багш буцаан олгоно — чатаар холбогдоорой."
		}
		s.notify(r.Context(), &store.Notification{UserID: sl.StudentID, Type: NotifBooking, Count: 1,
			Title: "❌ " + s.displayName(r.Context(), c.UID, c.Name) + " уулзалтыг цуцаллаа · " + slotWhen(sl.StartsAt), Body: body, Link: "/t/" + c.Name})
	}
	s.profiles.Delete(c.Name)
	w.WriteHeader(http.StatusNoContent)
}

// handleTeacherSlots: GET /api/teachers/{username}/slots — захиалж болох сул цагууд; нэвтэрсэн бол энэ багштай
// өөрийн захиалгууд (mine).
func (s *Server) handleTeacherSlots(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.UserByUsername(r.Context(), r.PathValue("username"))
	if s.storeErr(w, r, err) {
		return
	}
	if t.Role != store.RoleTeacher {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	now := time.Now()
	ss, err := s.store.Slots(r.Context(), t.ID, now.Add(slotLead), now.Add(slotHorizon))
	if s.storeErr(w, r, err) {
		return
	}
	uid := ""
	if p, ok := s.principal(r); ok && !p.IsGuest() {
		uid = p.UID
	}
	free, mine := []PublicSlot{}, []store.Slot{}
	for i := range ss {
		sl := &ss[i]
		switch {
		case uid != "" && sl.StudentID == uid:
			sl.HoldBy, sl.HoldUntil = "", nil
			mine = append(mine, *sl)
		case sl.Free(now) || (uid != "" && sl.HoldBy == uid): // өөрөө барьсан (төлбөрөө дуусгаагүй) цаг өөрт нь харагдана
			free = append(free, publicSlot(sl))
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, map[string]any{"slots": free, "mine": mine})
}

// handleBookSlot: POST /api/slots/{id}/book {note} — үнэгүй бол шууд баталгаажна; төлбөртэй бол цагийг 20 минут
// барьж QR төлбөрийн мэдээлэл өгнө (төлбөр орохоор notifyPaid → slotPaid баталгаажуулна).
func (s *Server) handleBookSlot(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Note string `json:"note"`
	}
	_ = json.NewDecoder(io.LimitReader(r.Body, 8<<10)).Decode(&in) // бие заавал биш
	in.Note = strings.TrimSpace(in.Note)
	if utf8.RuneCountInString(in.Note) > 500 {
		writeErr(w, http.StatusBadRequest, "тэмдэглэл 500 тэмдэгтээс ихгүй")
		return
	}
	ctx, now := r.Context(), time.Now()
	sl, err := s.store.SlotByID(ctx, r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	switch {
	case sl.TeacherID == c.UID:
		writeErr(w, http.StatusBadRequest, "өөрийн цагийг захиалах боломжгүй")
		return
	case sl.StudentID == c.UID:
		writeJSON(w, http.StatusOK, map[string]any{"booked": true, "slot": sl})
		return
	case sl.StartsAt.Before(now.Add(slotLead)):
		writeErr(w, http.StatusGone, "энэ цаг өнгөрсөн байна")
		return
	}
	name := s.displayName(ctx, c.UID, c.Name)
	fresh := false
	sl, err = s.store.UpdateSlot(ctx, sl.ID, func(x *store.Slot) error {
		switch {
		case x.StudentID == c.UID:
			return nil // зэрэг ирсэн давхар хүсэлт
		case !x.Free(now) && x.HoldBy != c.UID:
			return errSlotTaken
		}
		x.StudentName, x.Note = name, in.Note
		if x.Price == 0 {
			x.StudentID, x.HoldBy, x.HoldUntil, fresh = c.UID, "", nil, true
		} else {
			until := now.Add(slotHold)
			x.HoldBy, x.HoldUntil = c.UID, &until
		}
		return nil
	})
	if errors.Is(err, errSlotTaken) {
		writeErr(w, http.StatusConflict, "Энэ цагийг өөр хүн авчихлаа — өөр цаг сонгоно уу")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	if sl.StudentID == c.UID {
		if fresh {
			s.slotConfirmed(ctx, sl)
		}
		writeJSON(w, http.StatusOK, map[string]any{"booked": true, "slot": sl})
		return
	}
	teacher := s.displayName(ctx, sl.TeacherID, "Багш")
	o, err := s.store.CreateOrGetPendingSlotOrder(ctx, c.UID, sl, "Уулзах цаг: "+teacher+" · "+slotWhen(sl.StartsAt))
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"booked": false, "order": o, "payment": s.paymentInfo(r, o)})
}

// slotPaid: төлбөртэй цагийн төлбөр орсон — захиалгыг баталгаажуулна (notifyPaid-аас нэг л удаа).
func (s *Server) slotPaid(ctx context.Context, o *store.Order) {
	name, fresh := s.displayName(ctx, o.UserID, "Суралцагч"), false
	sl, err := s.store.UpdateSlot(ctx, o.SlotID, func(x *store.Slot) error {
		switch {
		case x.StudentID == o.UserID:
			return nil
		case x.StudentID != "" || (!x.Free(time.Now()) && x.HoldBy != o.UserID):
			return errSlotTaken
		}
		x.StudentID, x.StudentName, x.HoldBy, x.HoldUntil, fresh = o.UserID, name, "", nil, true
		return nil
	})
	if err != nil { // төлбөр орсон ч цаг цуцлагдсан/өөр хүнд очсон (маш ховор) — багш мөнгийг буцаана
		s.log.Warn("төлсөн цаг баталгаажсангүй", "order", o.ID, "err", err)
		s.notify(ctx,
			&store.Notification{UserID: o.TeacherID, Type: NotifBooking, Count: 1, Title: "⚠️ " + name + " төлбөр төлсөн боловч цаг баталгаажсангүй",
				Body: o.Title + " — мөнгийг буцаах эсвэл өөр цаг санал болгоно уу.", Link: "/me#live"},
			&store.Notification{UserID: o.UserID, Type: NotifBooking, Count: 1, Title: "⚠️ Таны төлсөн цаг өөр хүнд очсон байна",
				Body: o.Title + " — багш тантай холбогдож буцаан олгоно.", Link: "/"})
		return
	}
	if fresh {
		s.slotConfirmed(ctx, sl)
	}
}

// slotConfirmed: баталгаажсан захиалгыг «Шууд хичээл»-д нэмнэ (онлайн бол багшийн Google Meet-ээр холбоос), хоёр
// талд мэдэгдэнэ. Meet холбоогүй бол уулзалт холбоосгүй үүсч, багшид сануулна.
func (s *Server) slotConfirmed(ctx context.Context, sl *store.Slot) {
	title := "Цаг захиалга: " + sl.StudentName
	if sl.Mode == store.SlotOffline {
		title += " · биечлэн"
	}
	var m *store.Meeting
	if sl.Mode == store.SlotOnline && s.Meet != nil {
		mt, err := s.createMeet(ctx, sl.TeacherID, "", title, sl.StartsAt, sl.DurationMin, 0, false)
		if err != nil {
			s.log.Warn("цаг захиалгын Meet", "err", err)
		} else {
			m = mt
		}
	}
	if m == nil {
		m = &store.Meeting{TeacherID: sl.TeacherID, Title: title, StartsAt: sl.StartsAt, DurationMin: sl.DurationMin}
		if err := s.store.CreateMeeting(ctx, m); err != nil {
			s.log.Warn("цаг захиалгын уулзалт", "err", err)
			m = nil
		}
	}
	if m != nil {
		if up, err := s.store.UpdateSlot(ctx, sl.ID, func(x *store.Slot) error { x.MeetingID, x.MeetURL = m.ID, m.MeetURL; return nil }); err == nil {
			*sl = *up
		}
	}
	teacher := "Багш"
	if u, err := s.store.UserByID(ctx, sl.TeacherID); err == nil {
		teacher = u.DisplayName
		s.profiles.Delete(u.Username) // сул цагийн тоо өөрчлөгдөнө
	}
	when, place := slotWhen(sl.StartsAt), slotPlace(sl)
	tb := place
	if sl.Note != "" {
		tb += " — «" + short(sl.Note) + "»"
	}
	if sl.Mode == store.SlotOnline && sl.MeetURL == "" && s.Meet != nil {
		tb += " · Google Meet холбоогүй тул холбоосыг чатаар илгээнэ үү"
	}
	sb := place
	if sl.MeetURL != "" {
		sb += ": " + sl.MeetURL
	}
	s.notify(ctx,
		&store.Notification{UserID: sl.TeacherID, Type: NotifBooking, Count: 1, Title: "📅 " + sl.StudentName + " цаг захиаллаа · " + when, Body: tb, Link: "/me#live"},
		&store.Notification{UserID: sl.StudentID, Type: NotifBooking, Count: 1, Title: "✅ Цаг баталгаажлаа · " + teacher + " · " + when, Body: sb, Link: "/#my-live"})
}
