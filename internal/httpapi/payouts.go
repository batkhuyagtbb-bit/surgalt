package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"surgalt/internal/store"
)

// ---- Багшийн «Данс»: банкны данс холбох, орлогоо татах (гараар эсвэл хуваариар), орлогын статистик ----
// Худалдан авагчийн төлбөр платформын QPay merchant дансанд ордог. Татах хүсэлтийг хянагч (/admin) банкаар
// шилжүүлээд «шилжүүлсэн» гэж тэмдэглэнэ — тэр үед багшид мэдэгдэнэ.

// payOption — сонголтын жагсаалтын нэг мөр (түлхүүр → харагдах нэр).
type payOption struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// payoutBanks — Монголын арилжааны банкууд.
var payoutBanks = []payOption{
	{"khan", "Хаан банк"}, {"golomt", "Голомт банк"}, {"tdb", "Худалдаа хөгжлийн банк"}, {"xac", "Хас банк"},
	{"state", "Төрийн банк"}, {"capitron", "Капитрон банк"}, {"bogd", "Богд банк"}, {"nibank", "Үндэсний хөрөнгө оруулалтын банк"},
	{"arig", "Ариг банк"}, {"transport", "Тээвэр хөгжлийн банк"}, {"ckb", "Чингис хаан банк"}, {"mbank", "М банк"},
}

func bankName(key string) string {
	for _, b := range payoutBanks {
		if b.Key == key {
			return b.Name
		}
	}
	return key
}

// payoutSchedules — татах хуваарь (дарааллаараа UI-д харагдана).
var payoutSchedules = []payOption{
	{store.PayoutManual, "Гараар"}, {store.PayoutInstant, "Шууд"}, {store.PayoutDaily, "Өдөр бүр"},
	{store.PayoutWeekly, "7 хоног бүр"}, {store.PayoutMonthly, "Сар бүр"}, {store.PayoutQuarterly, "Улирал бүр"},
}

func validSchedule(s string) bool {
	for _, x := range payoutSchedules {
		if x.Key == s {
			return true
		}
	}
	return false
}

// acctNoRe — дансны дугаар (цифр) эсвэл IBAN (MN + цифр); зай, зураасыг хасаж шалгана.
var acctNoRe = regexp.MustCompile(`^(MN[0-9]{18}|[0-9]{6,20})$`)

func (s *Server) payoutMin() int64 {
	if s.cfg.PayoutMin > 0 {
		return s.cfg.PayoutMin
	}
	return 1000
}

// handleMyPayout: GET /api/me/payout — данс, үлдэгдэл, сүүлийн татах хүсэлтүүд.
func (s *Server) handleMyPayout(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	var acct *store.PayoutAccount
	if a, err := s.store.PayoutAccount(ctx, c.UID); err == nil {
		acct = a
	} else if !errors.Is(err, store.ErrNotFound) && s.storeErr(w, r, err) {
		return
	}
	bal, err := s.store.TeacherBalance(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	ps, err := s.store.Payouts(ctx, c.UID, "", 50)
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": acct, "balance": bal, "payouts": ps, "min": s.payoutMin(),
		"banks": payoutBanks, "schedules": payoutSchedules})
}

// handleSetPayoutAccount: PUT /api/me/payout/account {bank, account_no, account_name, schedule}
func (s *Server) handleSetPayoutAccount(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	var in struct {
		Bank        string `json:"bank"`
		AccountNo   string `json:"account_no"`
		AccountName string `json:"account_name"`
		Schedule    string `json:"schedule"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.AccountNo = strings.ToUpper(strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(in.AccountNo)))
	in.AccountName = strings.Join(strings.Fields(in.AccountName), " ")
	if in.Schedule == "" {
		in.Schedule = store.PayoutManual
	}
	switch {
	case bankName(in.Bank) == in.Bank:
		writeErr(w, http.StatusBadRequest, "банкаа сонгоно уу")
		return
	case !acctNoRe.MatchString(in.AccountNo):
		writeErr(w, http.StatusBadRequest, "дансны дугаар буруу (6–20 цифр эсвэл MN-ээр эхэлсэн IBAN)")
		return
	case utf8.RuneCountInString(in.AccountName) < 2 || utf8.RuneCountInString(in.AccountName) > 100:
		writeErr(w, http.StatusBadRequest, "данс эзэмшигчийн нэрээ бичнэ үү")
		return
	case !validSchedule(in.Schedule):
		writeErr(w, http.StatusBadRequest, "татах хуваарь буруу")
		return
	}
	ctx, now := r.Context(), time.Now()
	a := &store.PayoutAccount{TeacherID: c.UID, Bank: in.Bank, AccountNo: in.AccountNo, AccountName: in.AccountName,
		Schedule: in.Schedule, ScheduleAt: now, UpdatedAt: now}
	if old, err := s.store.PayoutAccount(ctx, c.UID); err == nil && old.Schedule == in.Schedule {
		a.ScheduleAt = old.ScheduleAt // зөвхөн данс сольсон бол хуваарийн тоолол хэвээр
	}
	if err := s.store.SetPayoutAccount(ctx, a); s.storeErr(w, r, err) {
		return
	}
	if a.Schedule == store.PayoutInstant {
		s.autoPayout(ctx, a, now)
	}
	writeJSON(w, http.StatusOK, a)
}

// handleWithdraw: POST /api/me/payout/withdraw {amount} — одоо татах (amount 0 бол бүх үлдэгдэл).
func (s *Server) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	var in struct {
		Amount int64 `json:"amount"`
	}
	if !decode(w, r, &in) {
		return
	}
	ctx := r.Context()
	bal, err := s.store.TeacherBalance(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	want := in.Amount
	if want <= 0 {
		want = bal.Available
	}
	switch {
	case want < s.payoutMin():
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("хамгийн багадаа %s татна", money(s.payoutMin())))
		return
	case want > bal.Available:
		writeErr(w, http.StatusBadRequest, "үлдэгдэл хүрэлцэхгүй байна")
		return
	}
	p, err := s.store.RequestPayout(ctx, c.UID, want, false)
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, http.StatusBadRequest, "эхлээд дансаа холбоно уу")
		return
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, "үлдэгдэл хүрэлцэхгүй байна")
		return
	case s.storeErr(w, r, err):
		return
	}
	s.payoutRequested(ctx, p)
	writeJSON(w, http.StatusCreated, p)
}

// payoutRequested — шинэ татах хүсэлтийг хянагчдад мэдэгдэнэ.
func (s *Server) payoutRequested(ctx context.Context, p *store.Payout) {
	who := s.displayName(ctx, p.TeacherID, "Багш")
	how := "гараар"
	if p.Auto {
		how = "хуваариар"
	}
	s.notifyAdmins(ctx, &store.Notification{Type: NotifPayout, Link: "/admin#payouts",
		Title: "🏦 Татах хүсэлт · " + money(p.Amount), Body: fmt.Sprintf("%s (%s) — %s %s", who, how, bankName(p.Bank), p.AccountNo)})
}

// ---- автомат татах ----

// periodStart — хуваарийн одоогийн хугацааны эхлэл (Улаанбаатарын цагаар): өдөр, 7 хоног (Даваа), сар, улирал.
func periodStart(schedule string, now time.Time) time.Time {
	t := now.In(mnLoc)
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, mnLoc)
	switch schedule {
	case store.PayoutDaily:
		return day
	case store.PayoutWeekly:
		return day.AddDate(0, 0, -((int(t.Weekday()) + 6) % 7))
	case store.PayoutMonthly:
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, mnLoc)
	case store.PayoutQuarterly:
		return time.Date(t.Year(), (t.Month()-1)/3*3+1, 1, 0, 0, 0, 0, mnLoc)
	}
	return time.Time{} // instant: хугацааны хязгааргүй
}

// autoPayout — хуваарь нь болсон бол (тухайн хугацаанд автомат хүсэлт үүсээгүй, хуваарийг энэ хугацаанаас
// өмнө тохируулсан) бүх үлдэгдлийг татах хүсэлт болгоно. Үлдэгдэл доод хэмжээнд хүрээгүй бол дараа нь дахин шалгана.
func (s *Server) autoPayout(ctx context.Context, a *store.PayoutAccount, now time.Time) {
	if a.Schedule == store.PayoutManual || a.Schedule == "" {
		return
	}
	if a.Schedule != store.PayoutInstant {
		start := periodStart(a.Schedule, now)
		if a.ScheduleAt.After(start) {
			return // хуваарийг энэ хугацааны дунд тохируулсан: дараагийн хугацаанаас эхэлнэ
		}
		last, err := s.store.Payouts(ctx, a.TeacherID, "", 20)
		if err != nil {
			s.log.Warn("автомат татах", "err", err, "teacher", a.TeacherID)
			return
		}
		for _, p := range last {
			if p.Auto && p.Status != store.PayoutRejected && !p.CreatedAt.Before(start) {
				return // энэ хугацаанд аль хэдийн татсан
			}
		}
	}
	bal, err := s.store.TeacherBalance(ctx, a.TeacherID)
	if err != nil || bal.Available < s.payoutMin() {
		return
	}
	p, err := s.store.RequestPayout(ctx, a.TeacherID, bal.Available, true)
	if err != nil {
		if !errors.Is(err, store.ErrConflict) {
			s.log.Warn("автомат татах", "err", err, "teacher", a.TeacherID)
		}
		return
	}
	s.payoutRequested(ctx, p)
	s.notify(ctx, &store.Notification{UserID: a.TeacherID, Type: NotifPayout, Link: "/me#wallet",
		Title: "🏦 Орлогоо татах хүсэлт илгээгдлээ · " + money(p.Amount), Body: bankName(p.Bank) + " " + p.AccountNo + " руу удахгүй шилжинэ"})
}

// instantPayout — «Шууд» хуваарьтай багшид борлуулалт бүрийн дараа татах хүсэлт үүсгэнэ.
func (s *Server) instantPayout(ctx context.Context, teacherID string) {
	if teacherID == "" {
		return
	}
	a, err := s.store.PayoutAccount(ctx, teacherID)
	if err != nil || a.Schedule != store.PayoutInstant {
		return
	}
	s.autoPayout(ctx, a, time.Now())
}

// payoutLoop — хуваарьтай бүх дансыг үе үе шалгана (өдөр, 7 хоног, сар, улирал эхлэхэд).
func (s *Server) payoutLoop(ctx context.Context) {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.runAutoPayouts(ctx, time.Now())
		}
	}
}

func (s *Server) runAutoPayouts(ctx context.Context, now time.Time) {
	as, err := s.store.PayoutAccounts(ctx)
	if err != nil {
		s.log.Warn("автомат татах: данс", "err", err)
		return
	}
	for i := range as {
		s.autoPayout(ctx, &as[i], now)
	}
}

// ---- орлогын статистик ----

var earningKinds = map[string]string{
	store.OrderKindCourse: "Сургалтын багц", store.OrderKindLesson: "Дан хичээл", store.OrderKindLate: "Шалгалт, даалгаврын төлбөр",
	store.OrderKindFee: "Шалгалт, даалгаврын төлбөр", store.OrderKindMeeting: "Шууд хичээл", store.OrderKindSlot: "Цаг захиалга",
	store.OrderKindBook: "Ном",
}

type earnRow struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Sub    string `json:"sub,omitempty"`
	Amount int64  `json:"amount"`
	Count  int64  `json:"count"`
}

type earnAgg struct {
	rows  map[string]*earnRow
	order []string
}

func (a *earnAgg) add(key, label, sub string, amount int64) {
	if a.rows == nil {
		a.rows = map[string]*earnRow{}
	}
	r, ok := a.rows[key]
	if !ok {
		r = &earnRow{Key: key, Label: label, Sub: sub}
		a.rows[key] = r
		a.order = append(a.order, key)
	}
	r.Amount += amount
	r.Count++
}

// top — дүнгээр буурах дарааллаар (тэнцвэл тоогоор), limit хүртэл.
func (a *earnAgg) top(limit int) []earnRow {
	out := make([]earnRow, 0, len(a.order))
	for _, k := range a.order {
		out = append(out, *a.rows[k])
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Amount != out[j].Amount {
			return out[i].Amount > out[j].Amount
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

// earnBuckets — хугацааны цонх: day = сүүлийн 30 хоног, week = 12 долоо хоног, month = 12 сар, quarter = 8 улирал.
func earnBuckets(period string, now time.Time) []time.Time {
	var n int
	var step func(time.Time, int) time.Time
	start := periodStart(map[string]string{"day": store.PayoutDaily, "week": store.PayoutWeekly, "month": store.PayoutMonthly, "quarter": store.PayoutQuarterly}[period], now)
	switch period {
	case "week":
		n, step = 12, func(t time.Time, k int) time.Time { return t.AddDate(0, 0, 7*k) }
	case "month":
		n, step = 12, func(t time.Time, k int) time.Time { return t.AddDate(0, k, 0) }
	case "quarter":
		n, step = 8, func(t time.Time, k int) time.Time { return t.AddDate(0, 3*k, 0) }
	default:
		n, step = 30, func(t time.Time, k int) time.Time { return t.AddDate(0, 0, k) }
	}
	out := make([]time.Time, n+1) // сүүлийнх нь одоогийн хугацааны төгсгөл
	for i := range out {
		out[i] = step(start, i-n+1)
	}
	return out
}

func bucketLabel(period string, t time.Time) string {
	switch period {
	case "month":
		return fmt.Sprintf("%d/%02d", t.Year(), int(t.Month()))
	case "quarter":
		return fmt.Sprintf("%d-%d улирал", t.Year(), (int(t.Month())-1)/3+1)
	}
	return fmt.Sprintf("%02d/%02d", int(t.Month()), t.Day())
}

// handleMyEarnings: GET /api/me/earnings?period=day|week|month|quarter — орлогын график ба задаргаа
// (эх сурвалж, хамгийн их орлоготой сургалт/ном, хичээл, бүлэг) сонгосон цонхонд.
func (s *Server) handleMyEarnings(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	period := r.URL.Query().Get("period")
	switch period {
	case "day", "week", "month", "quarter":
	default:
		period = "day"
	}
	ctx, now := r.Context(), time.Now()
	orders, err := s.store.TeacherPaidOrders(ctx, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	edges := earnBuckets(period, now)
	from := edges[0]
	type point struct {
		Label  string    `json:"label"`
		From   time.Time `json:"from"`
		Amount int64     `json:"amount"`
		Count  int64     `json:"count"`
	}
	series := make([]point, len(edges)-1)
	for i := range series {
		series[i] = point{Label: bucketLabel(period, edges[i]), From: edges[i]}
	}
	// Цонхонд орсон захиалгууд ба тэдгээрийн сургалт, хичээлийн нэр, бүлэг.
	var in []store.Order
	courseIDs := map[string]bool{}
	for _, o := range orders {
		if o.PaidAt == nil || o.PaidAt.Before(from) {
			continue
		}
		in = append(in, o)
		if o.CourseID != "" {
			courseIDs[o.CourseID] = true
		}
		i := sort.Search(len(edges), func(k int) bool { return edges[k].After(*o.PaidAt) }) - 1
		if i >= 0 && i < len(series) {
			series[i].Amount += o.Amount
			series[i].Count++
		}
	}
	ids := make([]string, 0, len(courseIDs))
	for id := range courseIDs {
		ids = append(ids, id)
	}
	courseTitle, lessons := map[string]string{}, map[string]store.Lesson{}
	if len(ids) > 0 {
		if cs, err := s.store.CoursesByIDs(ctx, ids); err == nil {
			for _, co := range cs {
				courseTitle[co.ID] = co.Title
			}
		}
		if ls, err := s.store.LessonOutlines(ctx, ids); err == nil {
			for _, l := range ls {
				lessons[l.ID] = l
			}
		}
	}
	var kinds, items, lessonAgg, sections earnAgg
	var total, count int64
	for _, o := range in {
		total += o.Amount
		count++
		label := earningKinds[o.Kind]
		if label == "" {
			label = o.Kind
		}
		kinds.add(label, label, "", o.Amount)
		switch {
		case o.CourseID != "":
			t := courseTitle[o.CourseID]
			if t == "" {
				t = o.Title
			}
			items.add("course:"+o.CourseID, t, "Сургалт", o.Amount)
		case o.BookID != "":
			items.add("book:"+o.BookID, o.Title, "Ном", o.Amount)
		case o.MeetingID != "":
			items.add("meeting:"+o.MeetingID, o.Title, "Шууд хичээл", o.Amount)
		default:
			items.add("slot", "Цаг захиалга", "Уулзалт", o.Amount)
		}
		if o.LessonID == "" {
			continue // сургалтын багцын орлого тодорхой хичээл, бүлэгт хуваагдахгүй
		}
		l, known := lessons[o.LessonID]
		title := l.Title
		if !known {
			title = strings.TrimPrefix(o.Title, courseTitle[o.CourseID]+" — ")
		}
		lessonAgg.add(o.LessonID, title, courseTitle[o.CourseID], o.Amount)
		if l.Section != "" {
			sections.add(o.CourseID+"|"+l.Section, l.Section, courseTitle[o.CourseID], o.Amount)
		}
	}
	// Өмнөх ижил урттай цонхтой харьцуулалт (өсөлт %).
	span := now.Sub(from)
	var prev int64
	for _, o := range orders {
		if o.PaidAt != nil && !o.PaidAt.Before(from.Add(-span)) && o.PaidAt.Before(now.Add(-span)) {
			prev += o.Amount
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"period": period, "from": from, "total": total, "count": count, "prev_total": prev, "series": series,
		"by_kind": kinds.top(10), "top_items": items.top(10), "top_lessons": lessonAgg.top(10), "top_sections": sections.top(10),
	})
}

// ---- хянагч ----

// handleAdminPayouts: GET /api/admin/payouts?status=pending|paid|rejected|all
func (s *Server) handleAdminPayouts(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "":
		status = store.PayoutPending
	case "all":
		status = ""
	case store.PayoutPending, store.PayoutPaid, store.PayoutRejected:
	default:
		writeErr(w, http.StatusBadRequest, "төлөв буруу")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	ctx := r.Context()
	ps, err := s.store.Payouts(ctx, "", status, limit)
	if s.storeErr(w, r, err) {
		return
	}
	ids := make([]string, len(ps))
	for i := range ps {
		ids[i] = ps[i].TeacherID
	}
	users, err := s.store.UsersByIDs(ctx, ids)
	if s.storeErr(w, r, err) {
		return
	}
	byID := map[string]store.User{}
	for _, u := range users {
		byID[u.ID] = u
	}
	type row struct {
		store.Payout
		BankName        string `json:"bank_name"`
		TeacherName     string `json:"teacher_name"`
		TeacherUsername string `json:"teacher_username"`
		TeacherEmail    string `json:"teacher_email"`
	}
	out := make([]row, len(ps))
	for i, p := range ps {
		u := byID[p.TeacherID]
		out[i] = row{Payout: p, BankName: bankName(p.Bank), TeacherName: u.DisplayName, TeacherUsername: u.Username, TeacherEmail: u.Email}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAdminPayout: POST /api/admin/payouts/{id} {status: paid|rejected, note} — шилжүүлснийг (эсвэл татгалзсаныг)
// тэмдэглэж, багшид мэдэгдэнэ. Зөвхөн хүлээгдэж буй хүсэлтийг өөрчилнө.
func (s *Server) handleAdminPayout(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requireAdmin(w, r)
	if !ok {
		return
	}
	var in struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Note = strings.TrimSpace(in.Note)
	switch {
	case in.Status != store.PayoutPaid && in.Status != store.PayoutRejected:
		writeErr(w, http.StatusBadRequest, "төлөв буруу")
		return
	case utf8.RuneCountInString(in.Note) > 300:
		writeErr(w, http.StatusBadRequest, "тайлбар 300 тэмдэгтээс ихгүй")
		return
	case in.Status == store.PayoutRejected && in.Note == "":
		writeErr(w, http.StatusBadRequest, "татгалзсан шалтгаанаа бичнэ үү")
		return
	}
	ctx, now := r.Context(), time.Now()
	errDone := errors.New("done")
	p, err := s.store.UpdatePayout(ctx, r.PathValue("id"), func(p *store.Payout) error {
		if p.Status != store.PayoutPending {
			return errDone
		}
		p.Status, p.Note, p.ProcessedAt, p.ProcessedBy = in.Status, in.Note, &now, admin.DisplayName
		return nil
	})
	if errors.Is(err, errDone) {
		writeErr(w, http.StatusConflict, "энэ хүсэлтийг аль хэдийн шийдвэрлэсэн")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	n := &store.Notification{UserID: p.TeacherID, Type: NotifPayout, Link: "/me#wallet"}
	if p.Status == store.PayoutPaid {
		n.Title = "✅ Орлого данс руу шилжлээ · " + money(p.Amount)
		n.Body = bankName(p.Bank) + " " + p.AccountNo
	} else {
		n.Title = "⚠️ Татах хүсэлт татгалзагдлаа · " + money(p.Amount)
		n.Body = in.Note + " — дүн үлдэгдэлд буцлаа"
	}
	s.notify(ctx, n)
	writeJSON(w, http.StatusOK, p)
}
