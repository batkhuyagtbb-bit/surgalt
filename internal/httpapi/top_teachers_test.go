package httpapi

import (
	"context"
	"net/http"
	"testing"

	"surgalt/internal/store"
)

// Шилдэг багш нар: үзэлт, элсэлт, сургалт, харилцаагаар 4 зэрэглэл (тэгтэйг харуулахгүй, суралцагчийг оруулахгүй).
func TestTopTeachers(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	ta, aid := register(t, srv, "topa", "teacher")
	tb, bid := register(t, srv, "topb", "teacher")
	register(t, srv, "topc", "teacher") // юу ч хийгээгүй багш
	sa, said := register(t, srv, "tops1", "student")
	sb, _ := register(t, srv, "tops2", "student")

	course := func(tok, title string) string {
		_, c := call(t, srv, "POST", "/api/courses", tok, `{"title":"`+title+`","price":0,"published":true}`)
		return c["id"].(string)
	}
	a1 := course(ta, "А1")
	course(tb, "Б1")
	b2 := course(tb, "Б2")
	course(tb, "Б3")
	for _, s := range []string{sa, sb} {
		if code, _ := call(t, srv, "POST", "/api/courses/"+a1+"/enroll", s, ""); code != 200 && code != 201 {
			t.Fatalf("элсэх: %d", code)
		}
	}
	call(t, srv, "POST", "/api/courses/"+b2+"/enroll", sa, "")
	if err := st.IncViews(ctx, map[string]int64{bid: 50, aid: 7, said: 999}, nil); err != nil {
		t.Fatal(err)
	}
	// Харилцаа: хоёр суралцагч A-д бичнэ, нэг нь B-д.
	for _, x := range []struct{ tok, teacher string }{{sa, "topa"}, {sb, "topa"}, {sa, "topb"}} {
		_, conv := call(t, srv, "POST", "/api/teachers/"+x.teacher+"/chat", x.tok, "")
		id, _ := conv["conversation"].(map[string]any)["id"].(string)
		if id == "" {
			t.Fatalf("чат: %v", conv)
		}
		if code, _ := call(t, srv, "POST", "/api/chat/"+id+"/messages", x.tok, `{"body":"Сайн байна уу"}`); code != 200 && code != 201 {
			t.Fatalf("мессеж: %d", code)
		}
	}

	code, d := call(t, srv, "GET", "/api/top-teachers", "", "")
	if code != http.StatusOK {
		t.Fatalf("top: %d %v", code, d)
	}
	boards := d["boards"].(map[string]any)
	order := func(k string) []string {
		var out []string
		for _, x := range boards[k].([]any) {
			out = append(out, x.(map[string]any)["username"].(string))
		}
		return out
	}
	eq := func(k string, want ...string) {
		got := order(k)
		if len(got) != len(want) {
			t.Fatalf("%s: %v, хүлээсэн %v", k, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: %v, хүлээсэн %v", k, got, want)
			}
		}
	}
	eq("views", "topb", "topa")    // суралцагчийн үзэлтийг оруулахгүй
	eq("students", "topa", "topb") // A: 2 элсэлт, B: 1
	eq("courses", "topb", "topa")  // B: 3, A: 1; сургалтгүй C-г харуулахгүй
	eq("talks", "topa", "topb")
	first := boards["views"].([]any)[0].(map[string]any)
	if first["rank"] != 1.0 || first["value"] != 50.0 || first["courses"] != 3.0 || first["students"] != 1.0 {
		t.Fatalf("эхний мөр: %v", first)
	}
}

// Идэвхтэй суралцагчид: сургалт, эрэмбэ, бүтээл, сахилга (сануулгагүй минут), харилцаа; өөрийн сургалт дахь үйлдэл
// тооцогдохгүй; нэр товчлогдоно; өөрийгөө нуувал жагсаалтаас алга болно.
func TestTopStudents(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	tt, tid := register(t, srv, "stteach", "teacher")
	sa, aid := register(t, srv, "stua", "student")
	_, bid := register(t, srv, "stub", "student")
	if code, _ := call(t, srv, "PUT", "/api/me/name", sa, `{"display_name":"Анужин Батболд"}`); code != 200 {
		t.Fatalf("нэр: %d", code)
	}
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"С1","price":0,"published":true}`)
	cid := c["id"].(string)
	lc, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Х1","content":"x","is_free":true}`)
	if l["id"] == nil {
		t.Fatalf("хичээл: %d %v", lc, l)
	}
	lid := l["id"].(string)
	_, c2 := call(t, srv, "POST", "/api/courses", tt, `{"title":"С2","price":0,"published":true}`)
	cid2 := c2["id"].(string)
	_, l2 := call(t, srv, "POST", "/api/courses/"+cid2+"/lessons", tt, `{"title":"Х2","content":"x","is_free":true}`)
	for _, x := range []struct{ u, c, l string }{{aid, cid, lid}, {aid, cid2, l2["id"].(string)}, {bid, cid, lid}, {tid, cid, lid}} {
		if err := st.MarkLessonViewed(ctx, x.u, x.c, x.l); err != nil {
			t.Fatal(err)
		}
	}
	st.SaveRankPoints(ctx, bid, cid, 400)
	st.SaveRankPoints(ctx, aid, cid, 100)
	st.SaveReflection(ctx, &store.Reflection{UserID: aid, CourseID: cid, LessonID: lid, TeacherID: tid, Text: "Ойлголоо", Words: 1})
	// Сахилга: A — сануулгагүй 30 мин; B — 50 мин ч сануулгатай.
	st.CreateSession(ctx, &store.StudySession{UserID: aid, CourseID: cid, LessonID: lid, TeacherID: tid, Kind: "lesson", ActiveSec: 1800, Counts: map[string]int{}})
	st.CreateSession(ctx, &store.StudySession{UserID: bid, CourseID: cid, LessonID: lid, TeacherID: tid, Kind: "lesson", ActiveSec: 3000, Counts: map[string]int{"tab_switch": 2}})
	st.AddActivityEvents(ctx, []store.ActivityEvent{{UserID: bid, CourseID: cid, LessonID: lid, TeacherID: tid, Type: "tab_switch"}})
	st.AddComment(ctx, &store.Comment{CourseID: cid, LessonID: lid, TeacherID: tid, UserID: bid, Body: "Асуулт"})

	_, d := call(t, srv, "GET", "/api/top-students", "", "")
	boards := d["boards"].(map[string]any)
	first := func(k string) map[string]any {
		b, _ := boards[k].([]any)
		if len(b) == 0 {
			t.Fatalf("%s хоосон: %v", k, boards)
		}
		return b[0].(map[string]any)
	}
	if x := first("courses"); x["name"] != "Анужин Б." || x["value"] != 2.0 || len(boards["courses"].([]any)) != 2 {
		t.Fatalf("сургалт (багшийн өөрийн сургалт тооцогдохгүй, нэр товчлогдоно): %v", boards["courses"])
	}
	if x := first("rank"); x["value"] != 400.0 || x["level"] != "Зальжин үнэг" {
		t.Fatalf("эрэмбэ: %v", x)
	}
	if x := first("creative"); x["name"] != "Анужин Б." {
		t.Fatalf("бүтээл: %v", x)
	}
	if b := boards["discipline"].([]any); len(b) != 1 || b[0].(map[string]any)["value"] != 30.0 {
		t.Fatalf("сахилга — зөвхөн сануулгагүй минут: %v", b)
	}
	if x := first("talks"); x["value"] != 1.0 {
		t.Fatalf("харилцаа: %v", x)
	}
	_, me := call(t, srv, "GET", "/api/me", sa, "")
	if first("courses")["key"] != me["rank_key"] {
		t.Fatalf("өөрийн картаа танихгүй байна")
	}
	if code, m := call(t, srv, "PUT", "/api/me/rank-hidden", sa, `{"hidden":true}`); code != 200 || m["rank_hidden"] != true {
		t.Fatalf("нуух: %d %v", code, m)
	}
	_, d = call(t, srv, "GET", "/api/top-students", "", "")
	for _, b := range d["boards"].(map[string]any) {
		for _, x := range b.([]any) {
			if x.(map[string]any)["key"] == me["rank_key"] {
				t.Fatalf("нуусан хүн харагдсаар: %v", d)
			}
		}
	}
	for in, want := range map[string]string{"Т. Батхуяг": "Т.Батхуяг", "Б.Энхжаргал": "Б.Энхжаргал", "batkhuyagtb@gmail.com": "batkhuyagtb", "Төмөр Батхуяг Б": "Төмөр Б."} {
		if got := shortName(in); got != want {
			t.Errorf("shortName(%q) = %q, хүлээсэн %q", in, got, want)
		}
	}
}
