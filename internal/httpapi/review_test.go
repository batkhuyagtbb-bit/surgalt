package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"surgalt/internal/store"
)

// Давтах сануулга: анх үзсэнээс 1, 3, 7, 14, 30 хоногийн дараа орой 19:00-оос хойш; нэг сургалтынх нэгтгэгдэнэ; нэг л
// удаа; нуусан хичээл, багшид сануулахгүй.
func TestReviewReminders(t *testing.T) {
	srv, s, st := newPlanServer(t)
	mem, ok := st.(*store.Memory)
	if !ok {
		t.Skip("үзсэн цагийг хойшлуулах нь санах ойн store-д л")
	}
	ctx := context.Background()
	tt, tid := register(t, srv, "revteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Математик","price":10000,"published":true}`)
	cid := c["id"].(string)
	lesson := func(title string) string {
		_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"`+title+`","content":"x"}`)
		return l["id"].(string)
	}
	a, b, h := lesson("Бутархай"), lesson("Тэгшитгэл"), lesson("Нууц")
	su, uid := register(t, srv, "revstud", "student")
	_, d := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", su, "")
	if code, _ := call(t, srv, "POST", "/api/orders/"+d["order"].(map[string]any)["id"].(string)+"/dev-pay", su, ""); code != 200 {
		t.Fatalf("элсэх: %d", code)
	}
	for _, l := range []string{a, b, h} {
		if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l, su, ""); code != 200 {
			t.Fatalf("хичээл үзэх: %d", code)
		}
	}
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+h+"/visibility", tt, `{"hidden":true}`); code != 200 {
		t.Fatalf("хичээл нуух: %d", code)
	}
	day := 24 * time.Hour
	mem.BackdateProgress(uid, a, day)
	mem.BackdateProgress(uid, b, 7*day)
	mem.BackdateProgress(uid, h, 3*day)

	reviews := func() []map[string]any {
		_, ns := callList(t, srv, "GET", "/api/me/notifications", su)
		var out []map[string]any
		for _, x := range ns {
			if n := x.(map[string]any); n["type"] == "review" {
				out = append(out, n)
			}
		}
		return out
	}
	l := time.Now().In(mnLoc)
	at := func(hour int) time.Time { return time.Date(l.Year(), l.Month(), l.Day(), hour, 0, 0, 0, mnLoc) }

	s.reviewTick(ctx, at(10)) // өглөө — илгээхгүй
	if n := len(reviews()); n != 0 {
		t.Fatalf("19:00-оос өмнө сануулав: %d", n)
	}
	s.reviewTick(ctx, at(20))
	s.reviewTick(ctx, at(21)) // өдөрт нэг удаа
	rs := reviews()
	if len(rs) != 1 {
		t.Fatalf("нэг сургалтын 2 хичээл нэг мэдэгдэлд: %v", rs)
	}
	r := rs[0]
	title, body := r["title"].(string), r["body"].(string)
	if !strings.Contains(title, "2 хичээл") || r["link"] != "/c/"+cid+"#l="+a || !strings.Contains(body, "«Бутархай» (1 хоног)") ||
		!strings.Contains(body, "«Тэгшитгэл» (7 хоног)") || strings.Contains(body, "Нууц") {
		t.Fatalf("сануулгын агуулга: %v", r)
	}
	if n, err := s.sendReviews(ctx, time.Now()); err != nil || n != 0 {
		t.Fatalf("давтан сануулав: %d %v", n, err)
	}
	// 3 дахь өдөр: зөвхөн «Бутархай» (7 хоногийнх нэгэнт илгээгдсэн).
	mem.BackdateProgress(uid, a, 2*day)
	if n, err := s.sendReviews(ctx, time.Now()); err != nil || n != 1 {
		t.Fatalf("3 хоногийн сануулга: %d %v", n, err)
	}
	if rs := reviews(); len(rs) != 2 || rs[0]["title"] != "🧠 Давтах цаг: «Бутархай»" || !strings.Contains(rs[0]["body"].(string), "3 хоног") {
		t.Fatalf("3 хоногийн сануулгын агуулга: %v", rs)
	}
	// Багшид (өөрийн сургалт) сануулахгүй.
	_, tn := callList(t, srv, "GET", "/api/me/notifications", tt)
	for _, x := range tn {
		if x.(map[string]any)["type"] == "review" {
			t.Fatalf("багшид давтах сануулга очлоо (%s)", tid)
		}
	}
}
