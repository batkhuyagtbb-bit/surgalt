package httpapi

import (
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
