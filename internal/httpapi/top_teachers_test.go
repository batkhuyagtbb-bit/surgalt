package httpapi

import (
	"context"
	"net/http"
	"testing"
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
