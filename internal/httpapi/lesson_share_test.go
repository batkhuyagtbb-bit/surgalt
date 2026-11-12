package httpapi

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

// Багш хооронд хуваалцах: хичээлийг имэйлээр «харах, хадгалах» эрхээр → хүлээн авагч урьдчилан харж, өөрийн сургалтад
// хадгална (файл нь өөрийнх нь сан руу хуулагдана); сургалтыг «хамт хөгжүүлэх» эрхээр → Course.Teachers-т орж,
// буцаахад гарна. Буруу имэйл, өөртөө, давхар хуваалцалт татгалзагдана.
func TestLessonShare(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	ta, ida := register(t, srv, "sharer", "teacher")
	tb, idb := register(t, srv, "taker", "teacher")
	_, ca := call(t, srv, "POST", "/api/courses", ta, `{"title":"Эх сургалт","price":0,"published":true}`)
	cida := ca["id"].(string)
	pdf := uploadFile(t, srv, ta, "lecture.pdf", []byte("%PDF-1.4 test"))
	_, la := call(t, srv, "POST", "/api/courses/"+cida+"/lessons", ta,
		`{"title":"Хуваалцах хичээл","is_free":true,"blocks":[{"id":"t1","type":"text","text":"Сайн уу"},{"id":"f1","type":"file","url":"`+pdf+`","name":"lecture.pdf"},{"id":"q1","type":"quiz","quiz":{"kind":"single","question":"2+2?","options":["3","4"],"correct":[1],"points":1}}]}`)
	lida := la["id"].(string)
	share := func(tok, body string) (int, map[string]any) { return call(t, srv, "POST", "/api/courses/"+cida+"/share", tok, body) }
	if code, _ := share(tb, `{"who":"sharer@x.mn"}`); code != http.StatusNotFound {
		t.Fatalf("өөр багш хуваалцаж болохгүй: %d", code)
	}
	if code, _ := share(ta, `{"who":"nobody@x.mn"}`); code != http.StatusNotFound {
		t.Fatalf("байхгүй имэйл: %d", code)
	}
	if code, _ := share(ta, `{"who":"sharer@x.mn"}`); code != http.StatusBadRequest {
		t.Fatalf("өөртөө: %d", code)
	}
	if code, _ := share(ta, `{"who":"taker@x.mn","mode":"edit","lesson_id":"`+lida+`"}`); code != http.StatusBadRequest {
		t.Fatalf("хичээлээр хамт хөгжүүлэх эрх өгөхгүй: %d", code)
	}
	code, sh := share(ta, `{"who":"taker@x.mn","lesson_id":"`+lida+`"}`)
	if code != 201 || sh["mode"] != "view" || sh["lesson_title"] != "Хуваалцах хичээл" || sh["to"].(map[string]any)["username"] != "taker" {
		t.Fatalf("хичээл хуваалцах: %d %v", code, sh)
	}
	if code, _ := share(ta, `{"who":"taker@x.mn","lesson_id":"`+lida+`"}`); code != http.StatusConflict {
		t.Fatalf("давхар: %d", code)
	}
	_, ls := call(t, srv, "GET", "/api/courses/"+cida+"/shares", ta, "")
	if len(ls["shares"].([]any)) != 1 {
		t.Fatalf("сургалтын хуваалцалтууд: %v", ls)
	}
	// Хүлээн авагч: жагсаалт, урьдчилан харах (медиа гарын үсэгтэй), хадгалах.
	_, mine := call(t, srv, "GET", "/api/me/shared", tb, "")
	ms := mine["shares"].([]any)
	if len(ms) != 1 || ms[0].(map[string]any)["from"].(map[string]any)["username"] != "sharer" {
		t.Fatalf("надад хуваалцсан: %v", mine)
	}
	sid := ms[0].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "GET", "/api/me/shared/"+sid, ta, ""); code != http.StatusNotFound {
		t.Fatalf("бусдын хуваалцалтыг харахгүй: %d", code)
	}
	code, v := call(t, srv, "GET", "/api/me/shared/"+sid, tb, "")
	l := v["lesson"].(map[string]any)
	if code != 200 || l["title"] != "Хуваалцах хичээл" || !strings.Contains(l["blocks"].([]any)[1].(map[string]any)["url"].(string), "sig=") {
		t.Fatalf("урьдчилан харах: %d %v", code, v)
	}
	_, cb := call(t, srv, "POST", "/api/courses", tb, `{"title":"Миний сургалт","price":0}`)
	cidb := cb["id"].(string)
	code, sv := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", tb, `{"course_id":"`+cidb+`","section":"Бүлэг 1"}`)
	if code != 201 || sv["count"].(float64) != 1 {
		t.Fatalf("хадгалах: %d %v", code, sv)
	}
	nl := sv["lessons"].([]any)[0].(map[string]any)
	if nl["hidden"] != true || nl["section"] != "Бүлэг 1" || len(nl["blocks"].([]any)) != 3 {
		t.Fatalf("хуулбар: %v", nl)
	}
	fu := nl["blocks"].([]any)[1].(map[string]any)["url"].(string)
	if !strings.HasPrefix(fu, "/files/"+idb+"/private/") || strings.Contains(fu, ida) || !strings.HasSuffix(fu, "lecture.pdf") {
		t.Fatalf("файл хүлээн авагчийн сан руу хуулагдах ёстой: %s", fu)
	}
	if code, _ := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", tb, `{"course_id":"`+cida+`"}`); code != http.StatusNotFound {
		t.Fatalf("бусдын сургалтад хадгалахгүй: %d", code)
	}
	// Сургалтаар «хамт хөгжүүлэх»: Teachers-т орно, студийн жагсаалтад shared гарна, буцаахад гарна.
	code, es := share(ta, `{"who":"@taker","mode":"edit"}`)
	if code != 201 || es["mode"] != "edit" {
		t.Fatalf("хамт хөгжүүлэх: %d %v", code, es)
	}
	_, co := callArrMap(t, srv, "GET", "/api/me/courses?shared=1", tb)
	found := false
	for _, m := range co {
		if m["id"] == cida && m["shared"] == true {
			found = true
		}
	}
	if !found {
		t.Fatalf("хамт хөгжүүлэгчийн жагсаалтад сургалт алга: %v", co)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cida+"/lessons", tb, `{"title":"Хамтарч нэмсэн","is_free":true}`); code != 201 {
		t.Fatalf("хамт хөгжүүлэгч хичээл нэмэх: %d", code)
	}
	if code, _ := call(t, srv, "DELETE", "/api/courses/"+cida+"/shares/"+es["id"].(string), ta, ""); code != 200 {
		t.Fatalf("буцаах: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cida+"/lessons", tb, `{"title":"Дахин","is_free":true}`); code == 201 {
		t.Fatal("буцаасны дараа хичээл нэмж болохгүй")
	}
	// Эх хичээл устсан бол хуваалцалт жагсаалтаас алга болно.
	call(t, srv, "DELETE", "/api/courses/"+cida+"/lessons/"+lida, ta, "")
	_, mine2 := call(t, srv, "GET", "/api/me/shared", tb, "")
	if len(mine2["shares"].([]any)) != 0 {
		t.Fatalf("устсан хичээлийн хуваалцалт: %v", mine2)
	}
}

// Сургалтаар хуваалцсан («харах, хадгалах»): хүлээн авагч хичээлүүдийг (хаалттай нь ч) бүлэг, дарааллаараа харж,
// өөрийн шинэ сургалт болгоно — тайлбар, үнэ, тохиргоо, бүлэг, дараалал, хичээлийн нээлттэй/хаалттай төлөв хэвээр,
// файл нь өөрийнх нь сан руу; дараа нь өөрөө дараалал, бүлгийг сольж зохион байгуулна. Сонгосон хичээлүүдийг л авч
// болно; өөрийн сургалтгүй ч болно. «Зөвхөн багцаар» хичээлийг үнэгүй сургалтад хадгалахад үнэгүй болно.
func TestSharedCourseFork(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	ta, _ := register(t, srv, "owner", "teacher")
	tb, idb := register(t, srv, "forker", "teacher")
	_, ca := call(t, srv, "POST", "/api/courses", ta, `{"title":"Математик","description":"Алгебрын үндэс","price":50000,"drip":true,"certificate":true,"cover":"t03","published":true}`)
	cida := ca["id"].(string)
	pdf := uploadFile(t, srv, ta, "notes.pdf", []byte("%PDF-1.4 test"))
	mk := func(body string) string {
		code, l := call(t, srv, "POST", "/api/courses/"+cida+"/lessons", ta, body)
		if code != 201 {
			t.Fatalf("хичээл үүсгэх: %d %v", code, l)
		}
		return l["id"].(string)
	}
	l1 := mk(`{"title":"Танилцуулга","is_free":true,"section":"1-р бүлэг"}`)
	l2 := mk(`{"title":"Тэгшитгэл","section":"1-р бүлэг","blocks":[{"id":"f1","type":"file","url":"` + pdf + `","name":"notes.pdf"}]}`) // зөвхөн багцаар
	l3 := mk(`{"title":"Шалгалт","is_free":true,"section":"2-р бүлэг","exam":{"time_min":30,"attempts":1,"pass_pct":60}}`)
	l4 := mk(`{"title":"Бэлтгэж буй","is_free":true,"section":"2-р бүлэг","hidden":true}`)
	code, sh := call(t, srv, "POST", "/api/courses/"+cida+"/share", ta, `{"who":"forker@x.mn"}`)
	if code != 201 {
		t.Fatalf("хуваалцах: %d %v", code, sh)
	}
	sid := sh["id"].(string)
	// Жагсаалтад бүх хичээлийн тоо (хаалттай нь ч), харахад бүх хичээл эх дарааллаараа бүлэгтэйгээ.
	_, mine := call(t, srv, "GET", "/api/me/shared", tb, "")
	if n := mine["shares"].([]any)[0].(map[string]any)["lesson_count"]; n != float64(4) {
		t.Fatalf("хичээлийн тоо: %v", n)
	}
	_, v := call(t, srv, "GET", "/api/me/shared/"+sid, tb, "")
	rows := v["lessons"].([]any)
	if len(rows) != 4 || rows[0].(map[string]any)["id"] != l1 || rows[3].(map[string]any)["hidden"] != true || rows[2].(map[string]any)["kind"] != "exam" || rows[1].(map[string]any)["section"] != "1-р бүлэг" {
		t.Fatalf("хуваалцсан сургалтын хичээлүүд: %v", rows)
	}
	// Өөрийн сургалтгүй ч шинэ сургалт болгоно.
	code, fk := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", tb, `{"new_course":true,"title":"  Миний математик  "}`)
	if code != 201 || fk["count"].(float64) != 4 || fk["course_id"] == "" {
		t.Fatalf("өөрийн сургалт болгох: %d %v", code, fk)
	}
	nc := fk["course_id"].(string)
	_, my := call(t, srv, "GET", "/api/me/courses/"+nc, tb, "")
	co := my["course"].(map[string]any)
	if co["teacher_id"] != idb || co["title"] != "Миний математик" || co["description"] != "Алгебрын үндэс" || co["price"] != float64(50000) || co["drip"] != true || co["certificate"] != true || co["cover"] != "t03" || co["published"] != false {
		t.Fatalf("шинэ сургалтын тохиргоо: %v", co)
	}
	if my["shared"] == true {
		t.Fatal("хуулбар сургалт хүлээн авагчийнх байх ёстой")
	}
	ls := my["lessons"].([]any)
	want := []struct {
		title, section string
		hidden         bool
	}{{"Танилцуулга", "1-р бүлэг", false}, {"Тэгшитгэл", "1-р бүлэг", false}, {"Шалгалт", "2-р бүлэг", false}, {"Бэлтгэж буй", "2-р бүлэг", true}}
	ids := make([]string, len(ls))
	for i, x := range ls {
		l := x.(map[string]any)
		ids[i] = l["id"].(string)
		h, _ := l["hidden"].(bool)
		if l["title"] != want[i].title || l["section"] != want[i].section || h != want[i].hidden || l["position"] != float64(i+1) {
			t.Fatalf("хуулбар хичээл %d: %v", i, l)
		}
	}
	if fu := ls[1].(map[string]any)["blocks"].([]any)[0].(map[string]any)["url"].(string); !strings.Contains(fu, "/files/"+idb+"/") {
		t.Fatalf("файл хүлээн авагчийн сан руу: %s", fu)
	}
	if ls[2].(map[string]any)["exam"] == nil {
		t.Fatal("шалгалтын тохиргоо хуулагдах ёстой")
	}
	// Хүлээн авагч өөрөө зохион байгуулна: дараалал, бүлгийг солих.
	if code, r := call(t, srv, "PUT", "/api/courses/"+nc+"/lesson-order", tb, `{"items":[{"id":"`+ids[3]+`","section":"Эхлэл"},{"id":"`+ids[0]+`","section":"Эхлэл"},{"id":"`+ids[1]+`","section":""},{"id":"`+ids[2]+`","section":""}]}`); code != 200 {
		t.Fatalf("дараалал солих: %d %v", code, r)
	}
	_, my2 := call(t, srv, "GET", "/api/me/courses/"+nc, tb, "")
	moved := map[string]map[string]any{}
	for _, x := range my2["lessons"].([]any) {
		moved[x.(map[string]any)["id"].(string)] = x.(map[string]any)
	}
	if moved[ids[3]]["position"] != float64(1) || moved[ids[3]]["section"] != "Эхлэл" {
		t.Fatalf("зохион байгуулсны дараа: %v", moved[ids[3]])
	}
	// Эх сургалт өөрчлөгдөхгүй.
	_, src := call(t, srv, "GET", "/api/me/courses/"+cida, ta, "")
	if sl := src["lessons"].([]any); len(sl) != 4 || sl[0].(map[string]any)["id"] != l1 {
		t.Fatalf("эх сургалт өөрчлөгдсөн: %v", sl)
	}
	// Сонгосон хичээлүүд (эх дарааллаар) — үнэгүй сургалтад «зөвхөн багцаар» хичээл үнэгүй болж, хаалттай нэмэгдэнэ.
	_, cb := call(t, srv, "POST", "/api/courses", tb, `{"title":"Үнэгүй сургалт","price":0}`)
	code, part := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", tb, `{"course_id":"`+cb["id"].(string)+`","lesson_ids":["`+l3+`","`+l2+`"]}`)
	if code != 201 || part["count"].(float64) != 2 {
		t.Fatalf("сонгосон хичээлүүд: %d %v", code, part)
	}
	pl := part["lessons"].([]any)
	if pl[0].(map[string]any)["title"] != "Тэгшитгэл" || pl[0].(map[string]any)["is_free"] != true || pl[0].(map[string]any)["hidden"] != true || pl[1].(map[string]any)["title"] != "Шалгалт" {
		t.Fatalf("сонгосон хичээлийн хуулбар: %v", pl)
	}
	if code, _ := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", tb, `{"new_course":true,"lesson_ids":["nope"]}`); code != http.StatusNotFound {
		t.Fatalf("байхгүй хичээл сонгох: %d", code)
	}
	_ = l4
	// Бусдын хуваалцалтаар сургалт үүсгэхгүй.
	if code, _ := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", ta, `{"new_course":true}`); code != http.StatusNotFound {
		t.Fatalf("бусдын хуваалцалт: %d", code)
	}
}

// Хүлээн авагчийн файлын сан хүрэлцэхгүй бол хуулахаас өмнө ойлгомжтой 413 (өмнө нь дундуур нь «дотоод алдаа» өгч,
// хагас хуулбар санг дүүргэдэг байв): юу ч хуулагдахгүй, сургалт үүсэхгүй; багтах хэмжээгээр сонговол хадгалагдана.
// Урьдчилан харахад хичээл бүрийн файлын хэмжээ, хүлээн авагчийн сан харагдана.
func TestSharedSaveQuota(t *testing.T) {
	srv, _ := newTestServer(t) // үнэгүй сан 10MB
	defer srv.Close()
	ta, _ := register(t, srv, "bigowner", "teacher")
	tb, _ := register(t, srv, "smalltaker", "teacher")
	uploadFile(t, srv, tb, "mine.pdf", append([]byte("%PDF-1.4 "), bytes.Repeat([]byte("m"), 5<<20)...)) // 5MB ашигласан
	_, ca := call(t, srv, "POST", "/api/courses", ta, `{"title":"Видео сургалт","price":0}`)
	cid := ca["id"].(string)
	for i, name := range []string{"a.pdf", "b.pdf"} {
		p := uploadFile(t, srv, ta, name, append([]byte("%PDF-1.4 "), bytes.Repeat([]byte{byte('a' + i)}, 4<<20)...))
		if code, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", ta, `{"title":"Хичээл `+name+`","is_free":true,"blocks":[{"id":"f","type":"file","url":"`+p+`","name":"`+name+`"}]}`); code != 201 {
			t.Fatalf("хичээл: %d %v", code, l)
		}
	}
	_, sh := call(t, srv, "POST", "/api/courses/"+cid+"/share", ta, `{"who":"smalltaker@x.mn"}`)
	sid := sh["id"].(string)
	_, v := call(t, srv, "GET", "/api/me/shared/"+sid, tb, "")
	rows, st := v["lessons"].([]any), v["storage"].(map[string]any)
	if b := rows[0].(map[string]any)["bytes"].(float64); b < 4<<20 || st["quota"].(float64) != 10<<20 || st["used"].(float64) < 5<<20 {
		t.Fatalf("хэмжээ, сан: %v %v", rows, st)
	}
	usage := func() float64 { _, s := call(t, srv, "GET", "/api/me/storage", tb, ""); return s["used"].(float64) }
	before := usage()
	code, e := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", tb, `{"new_course":true}`)
	if code != http.StatusRequestEntityTooLarge || !strings.Contains(e["error"].(string), "Файлын сан хүрэлцэхгүй") {
		t.Fatalf("багтахгүй: %d %v", code, e)
	}
	if _, cs := callArrMap(t, srv, "GET", "/api/me/courses", tb); len(cs) != 0 || usage() != before {
		t.Fatalf("юу ч үлдэх ёсгүй: %v, %v → %v", cs, before, usage())
	}
	_, cb := call(t, srv, "POST", "/api/courses", tb, `{"title":"Миний","price":0}`)
	if code, _ := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", tb, `{"course_id":"`+cb["id"].(string)+`"}`); code != http.StatusRequestEntityTooLarge || usage() != before {
		t.Fatalf("байгаа сургалтад багтахгүй: %d", code)
	}
	one := rows[0].(map[string]any)["id"].(string)
	if code, r := call(t, srv, "POST", "/api/me/shared/"+sid+"/save", tb, `{"new_course":true,"lesson_ids":["`+one+`"]}`); code != 201 || r["count"].(float64) != 1 {
		t.Fatalf("багтах хэмжээгээр: %d %v", code, r)
	}
}
