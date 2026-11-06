package httpapi

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/store"
)

func newPlanServer(t *testing.T) (*httptest.Server, *Server, store.Store) {
	t.Helper()
	st := newTestStore(t)
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{TrustProxy: true, DevPayments: true, StorageFreeMB: 10, ServiceFeePct: 1, AdminEmails: []string{"boss@x.mn"}},
		st, auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv, s, st
}

// Үнэгүй 2 сар (гүйлгээний шимтгэл 2%) → дуусахад студи, шинэ борлуулалт хаагдана (өмнө элссэн суралцагч, орлого татах нь
// хэвээр) → «Данс»-аас бүтэн жил авахад нээгдэнэ, хувь авахгүй (2%); хуучин багц худалдаалахгүй; хянагч багц олгоно.
func TestTeacherPlans(t *testing.T) {
	defer func(v, w time.Time) { store.PlanTrialFrom, store.PlanTrialShortFrom = v, w }(store.PlanTrialFrom, store.PlanTrialShortFrom)
	store.PlanTrialFrom, store.PlanTrialShortFrom = time.Now().AddDate(-1, 0, 0), time.Now().AddDate(-1, 0, 0)
	store.SetTeacherCommission(21) // TEACHER_COMMISSION_PERCENT — зөвхөн хуучин захиалгад
	defer store.SetTeacherCommission(0)
	srv, s, st := newPlanServer(t)
	ctx := context.Background()
	tt, tid := register(t, srv, "teach", "teacher")

	_, p := call(t, srv, "GET", "/api/me/plan", tt, "")
	if p["plan"] != store.PlanTrial || p["active"] != true || p["commission_pct"] != 2.0 || p["trial_months"] != 2.0 || p["days_left"].(float64) < 58 || p["days_left"].(float64) > 63 {
		t.Fatalf("шинэ багш үнэгүй 2 сар: %v", p)
	}
	if code, c := call(t, srv, "GET", "/api/plans", "", ""); code != 200 || len(c["plans"].([]any)) != 3 {
		t.Fatalf("үнийн жагсаалт: %d %v", code, c)
	}
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Курс","price":10000,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","content":"x"}`)
	lid := l["id"].(string)

	// Үнэгүй хугацаанд зарсан: 2%
	s1, _ := register(t, srv, "stud1", "student")
	_, d := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")
	oid := d["order"].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/orders/"+oid+"/dev-pay", s1, ""); code != 200 {
		t.Fatalf("төлөх: %d", code)
	}

	// Үнэгүй хугацааг дуусгана (2 хоногийн өмнө дууссан).
	if _, err := st.UpdateUser(ctx, tid, func(u *store.User) error { u.CreatedAt = time.Now().AddDate(0, -store.PlanTrialMonths, -2); return nil }); err != nil {
		t.Fatal(err)
	}
	s.planEnds.Delete(tid)
	code, e := call(t, srv, "POST", "/api/courses", tt, `{"title":"Шинэ","price":0}`)
	if code != http.StatusPaymentRequired || e["code"] != "plan_expired" {
		t.Fatalf("эрх дууссан багш сургалт үүсгэхгүй: %d %v", code, e)
	}
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Засвар","price":10000,"published":true}`); code != http.StatusPaymentRequired {
		t.Fatalf("эрх дууссан багш сургалт засахгүй: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/me/payout", tt, ""); code != 200 {
		t.Fatalf("орлого татах хэсэг нээлттэй байх: %d", code)
	}
	if _, p := call(t, srv, "GET", "/api/me/plan", tt, ""); p["active"] != false || p["plan"] != "" {
		t.Fatalf("дууссан төлөв: %v", p)
	}
	s2, _ := register(t, srv, "stud2", "student")
	if code, e := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s2, ""); code != http.StatusForbidden || !strings.Contains(e["error"].(string), "түр хаагдсан") {
		t.Fatalf("эрх дууссан багшийн сургалтад шинээр элсэхгүй: %d %v", code, e)
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, s1, ""); code != 200 {
		t.Fatalf("өмнө төлсөн суралцагчийн эрх хэвээр: %d", code)
	}

	// Сануулга: дууссан тухай нэг л удаа.
	s.planNotices(ctx, time.Now())
	s.planNotices(ctx, time.Now())
	_, ns := callList(t, srv, "GET", "/api/me/notifications", tt)
	if n := countTitle(ns, "дууслаа"); n != 1 {
		t.Fatalf("дууссан сануулга нэг удаа: %d %v", n, ns)
	}

	// Бүтэн жил авах: QPay (демо) → эрх нээгдэнэ, мэдэгдэл. Байхгүй, хуучин багц худалдаалахгүй.
	for _, id := range []string{"gold", "pro"} {
		if code, _ := call(t, srv, "POST", "/api/me/plan/purchase", tt, `{"plan":"`+id+`"}`); code != 400 {
			t.Fatalf("%s багц: %d", id, code)
		}
	}
	code, d = call(t, srv, "POST", "/api/me/plan/purchase", tt, `{"plan":"year"}`)
	if pm := d["payment"].(map[string]any); code != 201 || d["order"].(map[string]any)["amount"] != 159000.0 || pm["provider"] != "demo" || pm["amount"] != 159000.0 || pm["fee"] != 0.0 {
		t.Fatalf("багц худалдах: %d %v", code, d)
	}
	poid := d["order"].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/orders/"+poid+"/dev-pay", tt, ""); code != 200 {
		t.Fatalf("багц төлөх: %d", code)
	}
	_, p = call(t, srv, "GET", "/api/me/plan", tt, "")
	if p["plan"] != "year" || p["active"] != true || p["commission_pct"] != 2.0 || p["days_left"].(float64) < 364 {
		t.Fatalf("бүтэн жил: %v", p)
	}
	// Бүтэн жилд 1GB файлын сан (үнэгүй 10MB-ийг орлоно); нэр, мэдэгдэлд «Бүтэн жил».
	if _, sq := call(t, srv, "GET", "/api/me/storage", tt, ""); p["name"] != "Бүтэн жил" || sq["quota"] != float64(1024<<20) || sq["base_plan"] != "Бүтэн жил" || sq["base_label"] != "1GB" {
		t.Fatalf("багцын файлын сан: %v %v", p["name"], sq)
	}
	if _, ns := callList(t, srv, "GET", "/api/me/notifications", tt); countTitle(ns, "сунгагдлаа") != 1 || !strings.Contains(fmt.Sprint(ns), "Бүтэн жил") {
		t.Fatalf("мэдэгдэлд багцын нэр: %v", ns)
	}
	if code, _ := call(t, srv, "POST", "/api/courses", tt, `{"title":"Шинэ","price":0}`); code != 201 {
		t.Fatalf("сунгасны дараа шууд нээгдэнэ: %d", code)
	}
	if _, ns := callList(t, srv, "GET", "/api/me/notifications", tt); countTitle(ns, "сунгагдлаа") != 1 {
		t.Fatalf("сунгасан мэдэгдэл: %v", ns)
	}

	// Бүтэн жилийн үед зарсан: 2%. Үлдэгдэл: 10000×2% + 10000×2% = 400.
	_, d = call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s2, "")
	oid2 := d["order"].(map[string]any)["id"].(string)
	call(t, srv, "POST", "/api/orders/"+oid2+"/dev-pay", s2, "")
	_, pay := call(t, srv, "GET", "/api/me/payout", tt, "")
	if b := pay["balance"].(map[string]any); b["earned"] != 20000.0 || b["commission"] != 400.0 || b["net"] != 19600.0 {
		t.Fatalf("шимтгэл борлуулалт бүрээр: %v", b)
	}
	// Багцын төлбөр багшийн орлого, худалдан авалтын түүхэнд орохгүй.
	if _, sales := call(t, srv, "GET", "/api/me/sales?limit=10", tt, ""); sales["total_amount"] != 20000.0 {
		t.Fatalf("борлуулалт: %v", sales)
	}

	// Хянагч багц олгоно; багш биш хүнд олгохгүй.
	boss, _ := register(t, srv, "boss", "student")
	_, sid := register(t, srv, "stud3", "student")
	if code, _ := call(t, srv, "POST", "/api/admin/teachers/"+sid+"/plan", boss, `{"plan":"half"}`); code != http.StatusConflict {
		t.Fatalf("суралцагчид олгохгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/admin/teachers/"+tid+"/plan", tt, `{"plan":"half"}`); code != http.StatusForbidden {
		t.Fatalf("зөвхөн хянагч: %d", code)
	}
	before := p["until"].(string)
	code, g := call(t, srv, "POST", "/api/admin/teachers/"+tid+"/plan", boss, `{"plan":"half"}`)
	if code != 200 || g["plan"] != "year" || g["until"].(string) <= before || len(g["upcoming"].([]any)) != 1 {
		t.Fatalf("олгосон хагас жил бүтэн жилийн араас: %d %v", code, g)
	}
}

func countTitle(items []any, sub string) int {
	n := 0
	for _, x := range items {
		if strings.Contains(x.(map[string]any)["title"].(string), sub) {
			n++
		}
	}
	return n
}

// Сануулга: 7 хоног, 1 хоног, дууссан, дараа нь долоо хоног бүр төлбөр нэхэх (4 хүртэл) — тус бүр нэг удаа.
func TestPlanNotice(t *testing.T) {
	now := time.Now()
	u := &store.User{Role: store.RoleTeacher, CreatedAt: now.AddDate(0, -store.PlanTrialMonths, 5)} // 5 хоногийн дараа дуусна
	defer func(v, w time.Time) { store.PlanTrialFrom, store.PlanTrialShortFrom = v, w }(store.PlanTrialFrom, store.PlanTrialShortFrom)
	store.PlanTrialFrom, store.PlanTrialShortFrom = now.AddDate(-1, 0, 0), now.AddDate(-1, 0, 0)
	n, key := planNotice(u, now)
	if n == nil || !strings.HasSuffix(key, ":7") || !strings.Contains(n.Title, "Үнэгүй хугацаа") {
		t.Fatalf("7 хоногийн сануулга: %v %q", n, key)
	}
	u.Plan.Notice = key
	if n, _ := planNotice(u, now); n != nil {
		t.Fatal("давтахгүй")
	}
	if n, key := planNotice(u, u.PlanUntil().Add(-time.Hour)); n == nil || !strings.HasSuffix(key, ":1") {
		t.Fatalf("1 хоног: %v %q", n, key)
	}
	if n, key := planNotice(u, u.PlanUntil().Add(time.Hour)); n == nil || !strings.HasSuffix(key, ":0") {
		t.Fatalf("дууссан: %v %q", n, key)
	}
	if n, _ := planNotice(u, now.AddDate(0, -2, 0)); n != nil {
		t.Fatal("эрт сануулахгүй")
	}
	end := u.PlanUntil()
	if n, key := planNotice(u, end.Add(8*24*time.Hour)); n == nil || !strings.HasSuffix(key, ":w1") || !strings.Contains(n.Title, "төлнө үү") {
		t.Fatalf("1 долоо хоногийн дараа төлбөр нэхэх: %v %q", n, key)
	}
	if _, key := planNotice(u, end.Add(29*24*time.Hour)); !strings.HasSuffix(key, ":w4") {
		t.Fatalf("4 дэх долоо хоног: %q", key)
	}
	if n, _ := planNotice(u, end.Add(35*24*time.Hour)); n != nil {
		t.Fatal("30 хоногоос хойш сануулахгүй")
	}
}
