package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

// Багш хооронд хичээл хуваалцах: код → өөр багшийн сургалтад хуулбар (hidden), файл нь хүлээн авагчийн сан руу
// хуулагдана; буруу/хуучин код, эзэмшигч биш хүн татгалзагдана.
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
	if code, _ := call(t, srv, "POST", "/api/courses/"+cida+"/lessons/"+lida+"/share", tb, ""); code != http.StatusNotFound {
		t.Fatalf("өөр багш код авч болохгүй: %d", code)
	}
	code, sh := call(t, srv, "POST", "/api/courses/"+cida+"/lessons/"+lida+"/share", ta, "")
	if code != 200 || sh["code"] == "" || sh["title"] != "Хуваалцах хичээл" {
		t.Fatalf("хуваалцах код: %d %v", code, sh)
	}
	_, cb := call(t, srv, "POST", "/api/courses", tb, `{"title":"Миний сургалт","price":0}`)
	cidb := cb["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cidb+"/lessons/import", tb, `{"code":"muu-kod"}`); code != 400 {
		t.Fatalf("буруу код: %d", code)
	}
	code, nl := call(t, srv, "POST", "/api/courses/"+cidb+"/lessons/import", tb, `{"code":"`+sh["code"].(string)+`","section":"Бүлэг 1"}`)
	if code != 201 || nl["title"] != "Хуваалцах хичээл" || nl["hidden"] != true || nl["section"] != "Бүлэг 1" {
		t.Fatalf("импорт: %d %v", code, nl)
	}
	blocks := nl["blocks"].([]any)
	if len(blocks) != 3 {
		t.Fatalf("блок: %d", len(blocks))
	}
	fu := blocks[1].(map[string]any)["url"].(string)
	if !strings.HasPrefix(fu, "/files/"+idb+"/private/") || strings.Contains(fu, ida) || !strings.HasSuffix(fu, "lecture.pdf") {
		t.Fatalf("файл хүлээн авагчийн сан руу хуулагдах ёстой: %s", fu)
	}
	if blocks[2].(map[string]any)["quiz"] == nil {
		t.Fatal("асуулт хуулагдаагүй")
	}
	// Эх багшийн файл хэвээр, хүлээн авагчийн санд шинэ файл нэмэгдсэн.
	_, fa := call(t, srv, "GET", "/api/me/files", ta, "")
	_, fb := call(t, srv, "GET", "/api/me/files", tb, "")
	if len(fa["files"].([]any)) != 1 || len(fb["files"].([]any)) != 1 {
		t.Fatalf("файлын сан: эх %d, хүлээн авагч %d", len(fa["files"].([]any)), len(fb["files"].([]any)))
	}
	// Код дахин ашиглагдана (хугацаандаа) — хоёр дахь хуулбар.
	if code, _ := call(t, srv, "POST", "/api/courses/"+cidb+"/lessons/import", tb, `{"code":"`+sh["code"].(string)+`"}`); code != 201 {
		t.Fatalf("дахин импорт: %d", code)
	}
	// Эх хичээл устсан бол код хүчингүй.
	call(t, srv, "DELETE", "/api/courses/"+cida+"/lessons/"+lida, ta, "")
	if code, _ := call(t, srv, "POST", "/api/courses/"+cidb+"/lessons/import", tb, `{"code":"`+sh["code"].(string)+`"}`); code != http.StatusNotFound {
		t.Fatalf("устсан хичээлийн код: %d", code)
	}
}
