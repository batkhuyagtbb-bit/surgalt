package httpapi

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"surgalt/internal/store"
)

// Давтах popup (review_due.go): сүүлийн 7 хоногт илгээсэн сануулгаас тэр хичээлд илгээснээс хойш эргэж ороогүй нь —
// үе шат өсөхөөр, шинээс, 5 хүртэл. Сесс (last_at ≥ sent_at), story үзсэн, «давтлаа» дарсан бол алга болно; сануулгаас
// өмнө дууссан сесс тоологдохгүй, 19:00-ийг дамжин суралцсан нь тоологдоно; 7 хоногоос хуучин, нуусан хичээл, эрхгүй
// хичээл, багшид — юу ч үгүй. Серверийн кэш (60 сек) сесс эхлэх, «давтлаа»-д хүчингүй болно.
func TestReviewsDue(t *testing.T) {
	srv, s, st := newPlanServer(t)
	mem, ok := st.(*store.Memory)
	if !ok {
		t.Skip("сануулгын цагийг өөрчлөх нь санах ойн store-д л")
	}
	ctx := context.Background()
	tt, tid := register(t, srv, "rvd_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Давтлагын сургалт","price":0,"published":true}`)
	ids := map[string]string{}
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		ids[n] = jmLesson(t, srv, tt, cid, `{"title":"Хичээл `+n+`","is_free":true}`)
	}
	paid := jmCourse(t, srv, tt, `{"title":"Төлбөртэй","price":10000,"published":true}`)
	pl := jmLesson(t, srv, tt, paid, `{"title":"Төлбөртэй хичээл"}`)
	su, uid := register(t, srv, "rvd_s", "student")
	jmEnroll(t, srv, su, cid)
	for _, n := range []string{"a", "b", "c", "d", "e", "f", "g", "h"} {
		if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+ids[n], su, ""); code != http.StatusOK {
			t.Fatalf("хичээл үзэх %s: %d", n, code)
		}
	}
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+ids["h"]+"/visibility", tt, `{"hidden":true}`); code != http.StatusOK {
		t.Fatalf("хичээл нуух: %d", code)
	}

	// Хоосон: {"items":[]}.
	res, body := rawCall(t, srv, "GET", "/api/me/reviews/due", su, nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"items":[]`) {
		t.Fatalf("хоосон хариу: %d %s", res.StatusCode, body)
	}
	if code, _ := call(t, srv, "GET", "/api/me/reviews/due", "", ""); code != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй: %d", code)
	}

	stages := map[string]int{"a": 1, "d": 1, "b": 3, "c": 7, "e": 14, "f": 30, "g": 1, "h": 1}
	var items []store.ReviewItem
	for n, stg := range stages {
		items = append(items, store.ReviewItem{UserID: uid, CourseID: cid, LessonID: ids[n], Stage: stg})
	}
	items = append(items, store.ReviewItem{UserID: uid, CourseID: paid, LessonID: pl, Stage: 1}, // эрхгүй
		store.ReviewItem{UserID: tid, CourseID: cid, LessonID: ids["a"], Stage: 1}) // багш
	if got, err := st.ClaimReviews(ctx, items); err != nil || len(got) != len(items) {
		t.Fatalf("сануулга тэмдэглэх: %d %v", len(got), err)
	}
	now := time.Now()
	sentAt := map[string]time.Time{"a": now.Add(-time.Hour), "d": now.Add(-2 * time.Hour), "b": now.Add(-3 * time.Hour),
		"c": now.Add(-4 * time.Hour), "e": now.Add(-5 * time.Hour), "f": now.Add(-6 * time.Hour), "g": now.Add(-8 * 24 * time.Hour), "h": now.Add(-time.Hour)}
	for n, at := range sentAt {
		if !mem.SetReviewSentAt(uid, ids[n], stages[n], at) {
			t.Fatalf("сануулгын цаг %s", n)
		}
	}
	mem.SetReviewSentAt(uid, pl, 1, now.Add(-30*time.Minute))

	// Сануулга store-д шууд бичсэн тул (кэш) эхлээд хоосон хэвээр — кэшийг ашиглаж буйн нотолгоо; дараа нь цэвэрлэнэ.
	due := func(tok string) []map[string]any {
		t.Helper()
		code, r := call(t, srv, "GET", "/api/me/reviews/due", tok, "")
		if code != http.StatusOK {
			t.Fatalf("due: %d %v", code, r)
		}
		var out []map[string]any
		for _, x := range r["items"].([]any) {
			out = append(out, x.(map[string]any))
		}
		return out
	}
	if len(due(su)) != 0 {
		t.Fatal("60 секундын кэш ашиглагдаагүй")
	}
	fresh := func() { s.reviewDueChanged(uid) }
	fresh()
	names := func(list []map[string]any) string {
		byID := map[string]string{}
		for n, id := range ids {
			byID[id] = n
		}
		var out []string
		for _, it := range list {
			out = append(out, byID[it["lesson_id"].(string)])
		}
		return strings.Join(out, ",")
	}

	// Эрэмбэ: үе шат өсөхөөр, ижил үе шатад шинэ нь эхэнд; 5 хүртэл (f — 30 хоног, сүүлд тасарна); g (8 хоногийн өмнө),
	// h (нуусан), төлбөртэй (эрхгүй) орохгүй.
	list := due(su)
	if got := names(list); got != "a,d,b,c,e" {
		t.Fatalf("эрэмбэ: %s %v", got, list)
	}
	first := list[0]
	if first["course_id"] != cid || first["course_title"] != "Давтлагын сургалт" || first["lesson_title"] != "Хичээл a" || prNum(first["stage"]) != 1 ||
		first["url"] != "/c/"+cid+"#l="+ids["a"] || first["viewed_at"] == "" || first["sent_at"] == "" {
		t.Fatalf("мөрийн талбарууд: %v", first)
	}
	if va, _ := time.Parse(time.RFC3339Nano, first["viewed_at"].(string)); va.IsZero() || va.After(time.Now()) {
		t.Fatalf("viewed_at: %v", first["viewed_at"])
	}
	if len(due(tt)) != 0 {
		t.Fatal("багшид давтах popup гарлаа")
	}

	// a: сесс эхлүүлсэн (кэш шууд хүчингүй) → алга. d: story үзсэн → алга. b: «давтлаа» → алга.
	if code, r := call(t, srv, "POST", "/api/activity/start", su, `{"course_id":"`+cid+`","lesson_id":"`+ids["a"]+`","kind":"lesson"}`); code != http.StatusOK || r["session_id"] == nil {
		t.Fatalf("сесс эхлүүлэх: %d %v", code, r)
	}
	if got := names(due(su)); got != "d,b,c,e,f" {
		t.Fatalf("сесс эхэлсний дараа (кэш хүчингүй): %s", got)
	}
	if err := st.AddActivityEvents(ctx, []store.ActivityEvent{{UserID: uid, CourseID: cid, LessonID: ids["d"], TeacherID: tid, Type: storyViewType, Detail: "3/3"}}); err != nil {
		t.Fatal(err)
	}
	fresh()
	if code, r := call(t, srv, "POST", "/api/me/reviews/"+ids["b"]+"/done", su, ""); code != http.StatusNoContent {
		t.Fatalf("давтлаа: %d %v", code, r)
	}
	if got := names(due(su)); got != "c,e,f" {
		t.Fatalf("давтсаны дараа: %s", got)
	}
	evs, _ := st.ActivityEvents(ctx, store.ActivityFilter{UserID: uid, Type: reviewDoneType}, 10)
	if len(evs) != 1 || evs[0].LessonID != ids["b"] || evs[0].CourseID != cid || evs[0].TeacherID != tid {
		t.Fatalf("review_done үйл явдал: %+v", evs)
	}

	// last_at: e — сануулгаас өмнө эхэлж, түүнээс хойш үргэлжилсэн (19:00-ийг дамжсан) сесс → давтсан;
	// f — сануулгаас өмнө дууссан сесс → давтаагүй.
	se, sf := sentAt["e"], sentAt["f"]
	_ = st.CreateSession(ctx, &store.StudySession{UserID: uid, CourseID: cid, LessonID: ids["e"], TeacherID: tid, Kind: "lesson",
		StartedAt: se.Add(-10 * time.Minute), LastAt: se.Add(5 * time.Minute)})
	_ = st.CreateSession(ctx, &store.StudySession{UserID: uid, CourseID: cid, LessonID: ids["f"], TeacherID: tid, Kind: "lesson",
		StartedAt: sf.Add(-20 * time.Minute), LastAt: sf.Add(-10 * time.Minute)})
	fresh()
	if got := names(due(su)); got != "c,f" {
		t.Fatalf("last_at: %s", got)
	}
	// Сануулгаас ӨМНӨ (last_at) сесс тоологдохгүй: c-д сесс эхлүүлээд сануулгын цагийг түүнээс хойш болгоно.
	if code, _ := call(t, srv, "POST", "/api/activity/start", su, `{"course_id":"`+cid+`","lesson_id":"`+ids["c"]+`","kind":"lesson"}`); code != http.StatusOK {
		t.Fatalf("сесс c: %d", code)
	}
	mem.SetReviewSentAt(uid, ids["c"], 7, time.Now().Add(time.Minute))
	fresh()
	if got := names(due(su)); got != "c,f" {
		t.Fatalf("сануулгаас өмнөх сесс: %s", got)
	}
	// Хуучин үе шатын сануулгын дараах давталт: хичээлийн СҮҮЛИЙН сануулгаас хойш давтаагүй бол хугацаа нь болсон хэвээр.
	// a-д 3 хоногийн сануулга (одоо) — өмнөх сесс нь түүнээс өмнө тул a дахин гарна, үе шат 3.
	if got, _ := st.ClaimReviews(ctx, []store.ReviewItem{{UserID: uid, CourseID: cid, LessonID: ids["a"], Stage: 3}}); len(got) != 1 {
		t.Fatal("a-ийн 3 хоногийн сануулга")
	}
	mem.SetReviewSentAt(uid, ids["a"], 3, time.Now().Add(time.Minute))
	fresh()
	list = due(su)
	if got := names(list); got != "a,c,f" || prNum(list[0]["stage"]) != 3 {
		t.Fatalf("шинэ үе шат: %s %v", names(list), list)
	}
}

// «Давтлаа» (S3): хичээл олдохгүй, нуусан, эрхгүй, хаагдсан (auto_block) бол 404; due жагсаалтад байхгүй бол 204 (юу ч
// бичихгүй); due бол бичнэ; багшид юу ч бичихгүй (204).
func TestReviewDoneAccess(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	tt, tid := register(t, srv, "rvx_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Нээлттэй","price":0,"published":true}`)
	open := jmLesson(t, srv, tt, cid, `{"title":"Нээлттэй","is_free":true}`)
	other := jmLesson(t, srv, tt, cid, `{"title":"Хаагдсан","is_free":true}`)
	hidden := jmLesson(t, srv, tt, cid, `{"title":"Нууц","is_free":true}`)
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+hidden+"/visibility", tt, `{"hidden":true}`); code != http.StatusOK {
		t.Fatalf("нуух: %d", code)
	}
	paid := jmCourse(t, srv, tt, `{"title":"Төлбөртэй","price":10000,"published":true}`)
	pl := jmLesson(t, srv, tt, paid, `{"title":"Төлбөртэй"}`)
	su, uid := register(t, srv, "rvx_s", "student")
	done := func(tok, lid string) (int, map[string]any) {
		return call(t, srv, "POST", "/api/me/reviews/"+lid+"/done", tok, "")
	}
	written := func() int {
		evs, _ := st.ActivityEvents(ctx, store.ActivityFilter{Type: reviewDoneType}, 100)
		return len(evs)
	}
	// Хаагдсан хичээл: auto_block үйл явдал.
	if err := st.AddActivityEvents(ctx, []store.ActivityEvent{{UserID: uid, CourseID: cid, LessonID: other, TeacherID: tid, Type: "auto_block", Detail: "5 удаа"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.ClaimReviews(ctx, []store.ReviewItem{{UserID: uid, CourseID: cid, LessonID: open, Stage: 1}, {UserID: uid, CourseID: cid, LessonID: other, Stage: 1}}); err != nil {
		t.Fatal(err)
	}
	for name, lid := range map[string]string{"байхгүй": "nolesson", "нуусан": hidden, "эрхгүй": pl, "хаагдсан": other} {
		if code, r := done(su, lid); code != http.StatusNotFound {
			t.Fatalf("%s: %d %v", name, code, r)
		}
	}
	if code, _ := done("", open); code != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй: %d", code)
	}
	if code, _ := done(tt, open); code != http.StatusNoContent || written() != 0 {
		t.Fatalf("багш: %d (%d бичигдсэн)", code, written())
	}
	if code, _ := done(su, open); code != http.StatusNoContent || written() != 1 {
		t.Fatalf("due хичээл: %d (%d бичигдсэн)", code, written())
	}
	// Дахин: due-д байхгүй болсон тул бичихгүй (үйл явдлаар хориг түлхэж гаргахаас сэргийлнэ).
	for range 5 {
		if code, _ := done(su, open); code != http.StatusNoContent {
			t.Fatalf("давтан: %d", code)
		}
	}
	if n := written(); n != 1 {
		t.Fatalf("due биш үед бичигдлээ: %d", n)
	}
	evs, _ := st.ActivityEvents(ctx, store.ActivityFilter{Type: reviewDoneType}, 10)
	if len(evs) != 1 || evs[0].UserID != uid || evs[0].LessonID != open {
		t.Fatalf("зөвхөн суралцагчийн тэмдэг бичигдэнэ: %+v", evs)
	}
	// Хязгаар (burst 10): дээрх 10 дуудлагаар дууссан тул шинэ сануулгатай хичээлд ч чимээгүй 204, бичихгүй.
	for range 5 {
		done(su, open)
	}
	if _, err := st.ClaimReviews(ctx, []store.ReviewItem{{UserID: uid, CourseID: cid, LessonID: open, Stage: 3}}); err != nil {
		t.Fatal(err)
	}
	if code, _ := done(su, open); code != http.StatusNoContent || written() != 1 {
		t.Fatalf("хязгаар хэтэрсэн: %d (%d бичигдсэн)", code, written())
	}
}
