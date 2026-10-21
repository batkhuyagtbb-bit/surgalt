package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/store"
)

// buy — суралцагч захиалгыг демо төлбөрөөр төлнө.
func buy(t *testing.T, call2 func(method, path, tok, body string) (int, map[string]any), tok, path string) {
	t.Helper()
	code, o := call2("POST", path, tok, "")
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("захиалга %s: %d %v", path, code, o)
	}
	oid := o["order"].(map[string]any)["id"].(string)
	if code, _ := call2("POST", "/api/orders/"+oid+"/dev-pay", tok, ""); code != 200 {
		t.Fatalf("төлбөр %s: %d", path, code)
	}
}

// Багшийн данс: орлого үлдэгдэлд орно, данс холбоогүй бол татахгүй, үлдэгдлээс илүүг татахгүй, давхар татахгүй;
// хянагч шилжүүлсэн/татгалзсан гэж тэмдэглэнэ (татгалзсан дүн үлдэгдэлд буцна); статистик эх сурвалж, хичээл, бүлгээр задарна.
func TestTeacherPayoutFlow(t *testing.T) {
	srv, _ := newTestServerWith(t, func(c *Config) { c.AdminEmails = []string{"boss@x.mn"} })
	defer srv.Close()
	c2 := func(method, path, tok, body string) (int, map[string]any) { return call(t, srv, method, path, tok, body) }
	tt, _ := register(t, srv, "teach", "teacher")
	other, _ := register(t, srv, "other", "teacher")
	st, _ := register(t, srv, "stud", "student")
	boss, _ := register(t, srv, "boss", "student")

	_, bundle := c2("POST", "/api/courses", tt, `{"title":"Багц","price":10000,"published":true}`)
	_, single := c2("POST", "/api/courses", tt, `{"title":"Хичээлээр","price":0,"published":true}`)
	sid := single["id"].(string)
	_, la := c2("POST", "/api/courses/"+sid+"/lessons", tt, `{"title":"A","price":3000,"section":"Модуль 1"}`)
	buy(t, c2, st, "/api/courses/"+bundle["id"].(string)+"/enroll")
	buy(t, c2, st, "/api/courses/"+sid+"/lessons/"+la["id"].(string)+"/buy")

	if code, _ := c2("GET", "/api/me/payout", st, ""); code != http.StatusForbidden {
		t.Fatalf("суралцагчид данс байхгүй: %d", code)
	}
	_, p := c2("GET", "/api/me/payout", tt, "")
	bal := p["balance"].(map[string]any)
	if bal["earned"].(float64) != 13000 || bal["available"].(float64) != 13000 || p["account"] != nil {
		t.Fatalf("үлдэгдэл: %v", p)
	}
	if _, op := c2("GET", "/api/me/payout", other, ""); op["balance"].(map[string]any)["earned"].(float64) != 0 {
		t.Fatal("өөр багшийн орлого холилдлоо")
	}
	if code, _ := c2("POST", "/api/me/payout/withdraw", tt, `{"amount":0}`); code != http.StatusBadRequest {
		t.Fatalf("данс холбоогүй бол татахгүй: %d", code)
	}
	for _, bad := range []string{
		`{"bank":"nope","account_no":"5000123456","account_name":"Бат Болд"}`,
		`{"bank":"khan","account_no":"12ab","account_name":"Бат Болд"}`,
		`{"bank":"khan","account_no":"5000123456","account_name":"Б"}`,
		`{"bank":"khan","account_no":"5000123456","account_name":"Бат Болд","schedule":"hourly"}`,
	} {
		if code, _ := c2("PUT", "/api/me/payout/account", tt, bad); code != http.StatusBadRequest {
			t.Fatalf("буруу данс 400: %s → %d", bad, code)
		}
	}
	if code, a := c2("PUT", "/api/me/payout/account", tt, `{"bank":"khan","account_no":"5000 123 456","account_name":"  Бат   Болд ","schedule":"monthly"}`); code != 200 || a["account_no"] != "5000123456" || a["account_name"] != "Бат Болд" {
		t.Fatalf("данс хадгалах: %d %v", code, a)
	}
	if code, _ := c2("POST", "/api/me/payout/withdraw", tt, `{"amount":500}`); code != http.StatusBadRequest {
		t.Fatalf("доод дүнгээс бага: %d", code)
	}
	if code, _ := c2("POST", "/api/me/payout/withdraw", tt, `{"amount":20000}`); code != http.StatusBadRequest {
		t.Fatalf("үлдэгдлээс их: %d", code)
	}
	code, w1 := c2("POST", "/api/me/payout/withdraw", tt, `{"amount":5000}`)
	if code != http.StatusCreated || w1["status"] != "pending" || w1["bank"] != "khan" {
		t.Fatalf("татах: %d %v", code, w1)
	}
	code, w2 := c2("POST", "/api/me/payout/withdraw", tt, `{"amount":0}`) // үлдсэн бүгд
	if code != http.StatusCreated || w2["amount"].(float64) != 8000 {
		t.Fatalf("бүх үлдэгдэл: %d %v", code, w2)
	}
	if code, _ := c2("POST", "/api/me/payout/withdraw", tt, `{"amount":0}`); code != http.StatusBadRequest {
		t.Fatalf("үлдэгдэлгүй үед дахин татахгүй: %d", code)
	}

	// Хянагч
	if code, _ := c2("GET", "/api/admin/payouts", tt, ""); code != http.StatusForbidden {
		t.Fatalf("багш хянагч биш: %d", code)
	}
	res, body := rawCall(t, srv, "GET", "/api/admin/payouts?status=pending", boss, nil)
	var rows []map[string]any
	_ = json.Unmarshal(body, &rows)
	if res.StatusCode != 200 || len(rows) != 2 || rows[0]["teacher_username"] != "teach" || rows[0]["bank_name"] != "Хаан банк" {
		t.Fatalf("хянагчийн жагсаалт: %d %s", res.StatusCode, body)
	}
	if code, _ := c2("POST", "/api/admin/payouts/"+w2["id"].(string), boss, `{"status":"rejected"}`); code != http.StatusBadRequest {
		t.Fatal("татгалзахад шалтгаан шаардлагатай")
	}
	if code, _ := c2("POST", "/api/admin/payouts/"+w1["id"].(string), boss, `{"status":"paid","note":"Гүйлгээ #1"}`); code != 200 {
		t.Fatal("шилжүүлсэн")
	}
	if code, _ := c2("POST", "/api/admin/payouts/"+w1["id"].(string), boss, `{"status":"rejected","note":"x"}`); code != http.StatusConflict {
		t.Fatal("шийдвэрлэснийг дахин өөрчлөхгүй")
	}
	if code, _ := c2("POST", "/api/admin/payouts/"+w2["id"].(string), boss, `{"status":"rejected","note":"Данс буруу"}`); code != 200 {
		t.Fatal("татгалзах")
	}
	_, p = c2("GET", "/api/me/payout", tt, "")
	bal = p["balance"].(map[string]any)
	if bal["paid"].(float64) != 5000 || bal["pending"].(float64) != 0 || bal["available"].(float64) != 8000 {
		t.Fatalf("шийдвэрийн дараах үлдэгдэл: %v", bal)
	}
	_, ns := c2("GET", "/api/me/notifications", tt, "")
	if items, _ := ns["items"].([]any); len(items) == 0 {
		t.Fatal("багшид мэдэгдэл очоогүй")
	}

	// Статистик
	_, e := c2("GET", "/api/me/earnings?period=month", tt, "")
	if e["total"].(float64) != 13000 || e["count"].(float64) != 2 || len(e["series"].([]any)) != 12 {
		t.Fatalf("орлогын нийт: %v", e)
	}
	last := e["series"].([]any)[11].(map[string]any)
	if last["amount"].(float64) != 13000 {
		t.Fatalf("энэ сарын баганад орох ёстой: %v", last)
	}
	top := e["top_items"].([]any)[0].(map[string]any)
	if top["label"] != "Багц" || top["amount"].(float64) != 10000 {
		t.Fatalf("хамгийн их орлоготой: %v", top)
	}
	if ls := e["top_lessons"].([]any); len(ls) != 1 || ls[0].(map[string]any)["label"] != "A" {
		t.Fatalf("хичээлээр: %v", ls)
	}
	if ss := e["top_sections"].([]any); len(ss) != 1 || ss[0].(map[string]any)["label"] != "Модуль 1" || ss[0].(map[string]any)["sub"] != "Хичээлээр" {
		t.Fatalf("бүлгээр: %v", ss)
	}
	if ks := e["by_kind"].([]any); len(ks) != 2 {
		t.Fatalf("эх сурвалжаар: %v", ks)
	}
	for _, per := range []string{"day", "week", "quarter"} {
		if code, e := c2("GET", "/api/me/earnings?period="+per, tt, ""); code != 200 || e["total"].(float64) != 13000 {
			t.Fatalf("%s: %d %v", per, code, e)
		}
	}
}

// Автомат татах: «Шууд» бол борлуулалт бүрийн дараа; сар бүр бол хуваарийг тохируулсан сардаа биш, дараагийн сарын
// эхэнд нэг л удаа; доод дүнд хүрээгүй бол үүсгэхгүй.
func TestAutoPayout(t *testing.T) {
	st := store.NewMemory()
	s := New(Config{DevPayments: true, PayoutMin: 1000}, st, auth.NewSigner("test", time.Hour), chat.NewHub(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx := context.Background()
	sell := func(teacher string, amount int64) {
		t.Helper()
		co := &store.Course{TeacherID: teacher, Title: "К", Price: amount, Published: true}
		if err := st.CreateCourse(ctx, co); err != nil {
			t.Fatal(err)
		}
		o, err := st.CreateOrGetPendingOrder(ctx, "buyer-"+co.ID, co)
		if err != nil {
			t.Fatal(err)
		}
		if o, err = st.MarkOrderPaid(ctx, o.ID, amount); err != nil {
			t.Fatal(err)
		}
		s.notifyPaid(ctx, o)
	}
	count := func(teacher string) int {
		ps, _ := st.Payouts(ctx, teacher, "", 100)
		return len(ps)
	}
	for _, u := range []*store.User{{Username: "inst", Email: "i@x.mn", Role: store.RoleTeacher}, {Username: "mon", Email: "m@x.mn", Role: store.RoleTeacher}} {
		if err := st.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	inst, _ := st.UserByUsername(ctx, "inst")
	mon, _ := st.UserByUsername(ctx, "mon")

	now := time.Now()
	_ = st.SetPayoutAccount(ctx, &store.PayoutAccount{TeacherID: inst.ID, Bank: "khan", AccountNo: "123456", AccountName: "И", Schedule: store.PayoutInstant, ScheduleAt: now, UpdatedAt: now})
	sell(inst.ID, 500) // доод дүнгээс бага
	if count(inst.ID) != 0 {
		t.Fatal("доод дүнд хүрээгүй үед татахгүй")
	}
	sell(inst.ID, 2000)
	if ps, _ := st.Payouts(ctx, inst.ID, "", 10); len(ps) != 1 || ps[0].Amount != 2500 || !ps[0].Auto {
		t.Fatalf("шууд татах: %+v", ps)
	}

	// Сар бүр: энэ сард тохируулсан тул энэ сард татахгүй, дараагийн сарын эхэнд нэг удаа.
	_ = st.SetPayoutAccount(ctx, &store.PayoutAccount{TeacherID: mon.ID, Bank: "golomt", AccountNo: "654321", AccountName: "М", Schedule: store.PayoutMonthly, ScheduleAt: now, UpdatedAt: now})
	sell(mon.ID, 5000)
	s.runAutoPayouts(ctx, now)
	if count(mon.ID) != 0 {
		t.Fatal("хуваарийг тохируулсан хугацаанд татахгүй")
	}
	// Өмнөх сард тохируулсан мэт: энэ сарын эхний шалгалтаар нэг удаа, дараагийн борлуулалт дараа сард.
	acct, _ := st.PayoutAccount(ctx, mon.ID)
	acct.ScheduleAt = periodStart(store.PayoutMonthly, now).Add(-time.Hour)
	_ = st.SetPayoutAccount(ctx, acct)
	s.runAutoPayouts(ctx, now)
	sell(mon.ID, 3000)
	s.runAutoPayouts(ctx, now.Add(time.Minute))
	if ps, _ := st.Payouts(ctx, mon.ID, "", 10); len(ps) != 1 || ps[0].Amount != 5000 {
		t.Fatalf("сард нэг л удаа: %+v", ps)
	}
}

func TestPeriodStart(t *testing.T) {
	at := time.Date(2026, 8, 13, 3, 0, 0, 0, mnLoc) // Пүрэв гараг
	for sch, want := range map[string]time.Time{
		store.PayoutDaily:     time.Date(2026, 8, 13, 0, 0, 0, 0, mnLoc),
		store.PayoutWeekly:    time.Date(2026, 8, 10, 0, 0, 0, 0, mnLoc),
		store.PayoutMonthly:   time.Date(2026, 8, 1, 0, 0, 0, 0, mnLoc),
		store.PayoutQuarterly: time.Date(2026, 7, 1, 0, 0, 0, 0, mnLoc),
	} {
		if got := periodStart(sch, at); !got.Equal(want) {
			t.Fatalf("%s: %v, хүлээсэн %v", sch, got, want)
		}
	}
}
