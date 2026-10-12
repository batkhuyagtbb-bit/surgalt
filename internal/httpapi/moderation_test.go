package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/store"
)

// Чатын мэдүүлэг: оролцогч бусдын мессежийг мэдүүлнэ (давтвал нэг л удаа), 1:1 ярианд хориглож болно (хоёр тал
// бичихгүй, хориглосон тал л цуцална); хянагч (ADMIN_EMAILS) жагсаалтыг харж шийдвэрлэнэ, мессежийг устгаж болно.
func TestChatReportBlock(t *testing.T) {
	srv, _ := newTestServerWith(t, func(c *Config) { c.AdminEmails = []string{"boss@x.mn"} })
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	s1, _ := register(t, srv, "stud1", "student")
	boss, _ := register(t, srv, "boss", "student") // демо горимд (DEV_PAYMENTS) нууц үгтэй бүртгэл ч хянагч болно
	if _, m := call(t, srv, "GET", "/api/me", boss, ""); m["is_admin"] != true {
		t.Fatalf("ADMIN_EMAILS-ийн хэрэглэгч хянагч: %v", m)
	}
	if _, m := call(t, srv, "GET", "/api/me", s1, ""); m["is_admin"] != nil {
		t.Fatalf("энгийн хэрэглэгч хянагч биш: %v", m)
	}

	_, d := call(t, srv, "POST", "/api/teachers/teach/chat", s1, "")
	conv := d["conversation"].(map[string]any)["id"].(string)
	send := func(tok, body string) (int, map[string]any) {
		return call(t, srv, "POST", "/api/chat/"+conv+"/messages", tok, `{"body":"`+body+`"}`)
	}
	_, mine := send(s1, "Сайн байна уу")
	code, bad := send(tt, "Муу үг")
	if code != http.StatusCreated {
		t.Fatalf("багшийн мессеж: %d %v", code, bad)
	}
	path := "/api/chat/" + conv + "/report"
	for _, c := range []struct{ body, why string }{
		{`{"message_id":"` + mine["id"].(string) + `","reason":"spam"}`, "өөрийн мессеж"},
		{`{"message_id":"` + bad["id"].(string) + `","reason":"nope"}`, "шалтгаан буруу"},
	} {
		if code, _ := call(t, srv, "POST", path, s1, c.body); code != http.StatusBadRequest {
			t.Fatalf("%s 400: %d", c.why, code)
		}
	}
	if code, _ := call(t, srv, "POST", path, "", `{"message_id":"`+bad["id"].(string)+`","reason":"harass"}`); code != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй: %d", code)
	}
	code, r := call(t, srv, "POST", path, s1, `{"message_id":"`+bad["id"].(string)+`","reason":"harass","note":"доромжилсон","block":true}`)
	if code != http.StatusOK || r["duplicate"] != false || r["block"].(map[string]any)["visitor"] != true {
		t.Fatalf("мэдүүлэг: %d %v", code, r)
	}
	if _, r := call(t, srv, "POST", path, s1, `{"message_id":"`+bad["id"].(string)+`","reason":"harass"}`); r["duplicate"] != true {
		t.Fatalf("давхар мэдүүлэг: %v", r)
	}
	_, ns := call(t, srv, "GET", "/api/me/notifications", boss, "")
	if ts := notifTitles(ns, NotifReport); len(ts) != 1 || !strings.Contains(ts[0], "Шинэ мэдүүлэг") {
		t.Fatalf("хянагчид нэг мэдэгдэл: %v", ts)
	}

	// Хориг: хоёр тал бичихгүй; хориглоогүй тал цуцалж чадахгүй, хориглосон тал цуцална.
	for _, tok := range []string{tt, s1} {
		if code, _ := send(tok, "сайн уу"); code != http.StatusForbidden {
			t.Fatalf("хоригтой ярианд бичихгүй: %d", code)
		}
	}
	if _, l := call(t, srv, "GET", "/api/chat/"+conv+"/messages", tt, ""); l["block"].(map[string]any)["visitor"] != true {
		t.Fatalf("багшид хоригийн төлөв: %v", l["block"])
	}
	call(t, srv, "DELETE", "/api/chat/"+conv+"/block", tt, "")
	if code, _ := send(tt, "сайн уу"); code != http.StatusForbidden {
		t.Fatalf("нөгөө талын хоригийг багш цуцалж чадахгүй: %d", code)
	}
	if code, b := call(t, srv, "DELETE", "/api/chat/"+conv+"/block", s1, ""); code != http.StatusOK || len(b["block"].(map[string]any)) != 0 {
		t.Fatalf("хоригоо цуцлах: %d %v", code, b)
	}
	if code, _ := send(tt, "Уучлаарай"); code != http.StatusCreated {
		t.Fatalf("цуцалсны дараа бичнэ: %d", code)
	}

	// Хянагч: жагсаалт (мэдүүлсэн мессеж тодорсон), шийдвэр + мессеж устгах, мэдүүлэгчид мэдэгдэл.
	if code, _ := call(t, srv, "GET", "/api/admin/reports", tt, ""); code != http.StatusForbidden {
		t.Fatalf("хянагч бус 403: %d", code)
	}
	_, rs := callArrMap(t, srv, "GET", "/api/admin/reports", boss)
	if len(rs) != 1 || rs[0]["reporter_name"] == "" || rs[0]["reason_label"] != "Доромжилсон, дарамталсан" || rs[0]["target_id"] == nil {
		t.Fatalf("нээлттэй мэдүүлэг: %v", rs)
	}
	ctxMsgs := rs[0]["context"].([]any)
	if last := ctxMsgs[len(ctxMsgs)-1].(map[string]any); len(ctxMsgs) < 2 || last["reported"] != true || last["body"] != "Муу үг" {
		t.Fatalf("мэдүүлгийн хүрээ: %v", ctxMsgs)
	}
	id := rs[0]["id"].(string)
	if code, r := call(t, srv, "POST", "/api/admin/reports/"+id+"/resolve", boss, `{"resolution":"Сануулсан","delete_message":true}`); code != http.StatusOK || r["status"] != "resolved" {
		t.Fatalf("шийдвэрлэх: %d %v", code, r)
	}
	if _, rs := callArrMap(t, srv, "GET", "/api/admin/reports", boss); len(rs) != 0 {
		t.Fatalf("шийдвэрлэсний дараа нээлттэй алга: %v", rs)
	}
	if _, rs := callArrMap(t, srv, "GET", "/api/admin/reports?status=resolved", boss); len(rs) != 1 || rs[0]["resolution"] != "Сануулсан" {
		t.Fatalf("шийдвэрлэсэн: %v", rs)
	}
	_, l := call(t, srv, "GET", "/api/chat/"+conv+"/messages", s1, "")
	for _, m := range l["messages"].([]any) {
		if mm := m.(map[string]any); mm["id"] == bad["id"] && mm["deleted"] != true {
			t.Fatalf("мэдүүлсэн мессеж устах ёстой: %v", mm)
		}
	}
	if _, ns := call(t, srv, "GET", "/api/me/notifications", s1, ""); len(notifTitles(ns, NotifReport)) != 1 {
		t.Fatalf("мэдүүлэгчид шийдвэрийн мэдэгдэл: %v", ns)
	}
}

// Бодит орчинд (DEV_PAYMENTS-гүй) нууц үгтэй бүртгэл хянагч болохгүй — хэн нэгэн админы имэйлээр түрүүлж
// бүртгүүлэхээс сэргийлнэ; Google/Microsoft-оор (нууц үггүй) нэвтэрсэн бүртгэл л хянагч.
func TestAdminNeedsVerifiedLogin(t *testing.T) {
	srv, st := newTestServerWith(t, func(c *Config) { c.DevPayments = false; c.AdminEmails = []string{"boss@x.mn", "real@x.mn"} })
	defer srv.Close()
	fake, _ := register(t, srv, "boss", "student")
	if code, _ := call(t, srv, "GET", "/api/admin/reports", fake, ""); code != http.StatusForbidden {
		t.Fatalf("нууц үгтэй бүртгэл хянагч биш: %d", code)
	}
	u := &store.User{Username: "real", Email: "real@x.mn", Role: store.RoleStudent, DisplayName: "Жинхэнэ хянагч"}
	if err := st.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	tok, err := auth.NewSigner("test", time.Hour).SignUser(u.ID, string(u.Role), u.DisplayName)
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := call(t, srv, "GET", "/api/admin/reports", tok, ""); code != http.StatusOK {
		t.Fatalf("баталгаажсан нэвтрэлттэй хянагч: %d", code)
	}
}
