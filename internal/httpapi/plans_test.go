package httpapi

import (
	"context"
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

// Үнэгүй 3 сар (21%) → дуусахад студи, шинэ борлуулалт хаагдана (өмнө элссэн суралцагч, орлого татах нь хэвээр)
// → «Данс»-аас Pro авахад нээгдэж, шинэ борлуулалт 8%-иар тооцогдоно; хянагч багц олгоно.
func TestTeacherPlans(t *testing.T) {
	defer func(v time.Time) { store.PlanTrialFrom = v }(store.PlanTrialFrom)
	store.PlanTrialFrom = time.Now().AddDate(-1, 0, 0)
	store.SetTeacherCommission(21) // TEACHER_COMMISSION_PERCENT — үнэгүй хугацааны шимтгэл
	defer store.SetTeacherCommission(0)
	srv, s, st := newPlanServer(t)
	ctx := context.Background()
	tt, tid := register(t, srv, "teach", "teacher")

	_, p := call(t, srv, "GET", "/api/me/plan", tt, "")
	if p["plan"] != store.PlanTrial || p["active"] != true || p["commission_pct"] != 21.0 || p["days_left"].(float64) < 89 {
		t.Fatalf("шинэ багш үнэгүй 3 сар: %v", p)
	}
	if code, c := call(t, srv, "GET", "/api/plans", "", ""); code != 200 || len(c["plans"].([]any)) != 4 {
		t.Fatalf("үнийн жагсаалт: %d %v", code, c)
	}
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Курс","price":10000,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","content":"x"}`)
	lid := l["id"].(string)

	// Үнэгүй хугацаанд зарсан: 21%
	s1, _ := register(t, srv, "stud1", "student")
	_, d := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")
	oid := d["order"].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/orders/"+oid+"/dev-pay", s1, ""); code != 200 {
		t.Fatalf("төлөх: %d", code)
	}

	// Үнэгүй хугацааг дуусгана (2 хоногийн өмнө дууссан).
	if _, err := st.UpdateUser(ctx, tid, func(u *store.User) error { u.CreatedAt = time.Now().AddDate(0, -3, -2); return nil }); err != nil {
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

	// Pro авах: QPay (демо) → эрх нээгдэнэ, мэдэгдэл.
	if code, _ := call(t, srv, "POST", "/api/me/plan/purchase", tt, `{"plan":"gold"}`); code != 400 {
		t.Fatalf("байхгүй багц: %d", code)
	}
	code, d = call(t, srv, "POST", "/api/me/plan/purchase", tt, `{"plan":"pro"}`)
	if pm := d["payment"].(map[string]any); code != 201 || d["order"].(map[string]any)["amount"] != 500000.0 || pm["provider"] != "demo" || pm["amount"] != 500000.0 || pm["fee"] != 0.0 {
		t.Fatalf("багц худалдах: %d %v", code, d)
	}
	poid := d["order"].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/orders/"+poid+"/dev-pay", tt, ""); code != 200 {
		t.Fatalf("багц төлөх: %d", code)
	}
	_, p = call(t, srv, "GET", "/api/me/plan", tt, "")
	if p["plan"] != "pro" || p["active"] != true || p["commission_pct"] != 8.0 || p["days_left"].(float64) < 364 {
		t.Fatalf("Pro: %v", p)
	}
	if code, _ := call(t, srv, "POST", "/api/courses", tt, `{"title":"Шинэ","price":0}`); code != 201 {
		t.Fatalf("сунгасны дараа шууд нээгдэнэ: %d", code)
	}
	if _, ns := callList(t, srv, "GET", "/api/me/notifications", tt); countTitle(ns, "сунгагдлаа") != 1 {
		t.Fatalf("сунгасан мэдэгдэл: %v", ns)
	}

	// Pro-гийн үед зарсан: 8%. Үлдэгдэл: 10000×21% + 10000×8% = 2900.
	_, d = call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s2, "")
	oid2 := d["order"].(map[string]any)["id"].(string)
	call(t, srv, "POST", "/api/orders/"+oid2+"/dev-pay", s2, "")
	_, pay := call(t, srv, "GET", "/api/me/payout", tt, "")
	if b := pay["balance"].(map[string]any); b["earned"] != 20000.0 || b["commission"] != 2900.0 || b["net"] != 17100.0 {
		t.Fatalf("шимтгэл борлуулалт бүрээр: %v", b)
	}
	// Багцын төлбөр багшийн орлого, худалдан авалтын түүхэнд орохгүй.
	if _, sales := call(t, srv, "GET", "/api/me/sales?limit=10", tt, ""); sales["total_amount"] != 20000.0 {
		t.Fatalf("борлуулалт: %v", sales)
	}

	// Хянагч багц олгоно; багш биш хүнд олгохгүй.
	boss, _ := register(t, srv, "boss", "student")
	_, sid := register(t, srv, "stud3", "student")
	if code, _ := call(t, srv, "POST", "/api/admin/teachers/"+sid+"/plan", boss, `{"plan":"major"}`); code != http.StatusConflict {
		t.Fatalf("суралцагчид олгохгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/admin/teachers/"+tid+"/plan", tt, `{"plan":"major"}`); code != http.StatusForbidden {
		t.Fatalf("зөвхөн хянагч: %d", code)
	}
	before := p["until"].(string)
	code, g := call(t, srv, "POST", "/api/admin/teachers/"+tid+"/plan", boss, `{"plan":"major"}`)
	if code != 200 || g["plan"] != "pro" || g["until"].(string) <= before || len(g["upcoming"].([]any)) != 1 {
		t.Fatalf("олгосон Major Pro-гийн араас: %d %v", code, g)
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

// Сануулга: 7 хоног, 1 хоног, дууссан — тус бүр нэг удаа.
func TestPlanNotice(t *testing.T) {
	now := time.Now()
	u := &store.User{Role: store.RoleTeacher, CreatedAt: now.AddDate(0, -3, 5)} // 5 хоногийн дараа дуусна
	defer func(v time.Time) { store.PlanTrialFrom = v }(store.PlanTrialFrom)
	store.PlanTrialFrom = now.AddDate(-1, 0, 0)
	n, key := planNotice(u, now)
	if n == nil || !strings.HasSuffix(key, ":7") || !strings.Contains(n.Title, "Үнэгүй 3 сар") {
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
}
