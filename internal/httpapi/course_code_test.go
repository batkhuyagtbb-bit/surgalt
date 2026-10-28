package httpapi

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// Нууц үгтэй сургалт: нууц үг нийтэд гарахгүй, зөвхөн зөв үгээр элсэж бүх хичээл (үнэгүй ч) нээгдэнэ.
func TestLockedCourse(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "lockteach", "teacher")
	if code, r := call(t, srv, "POST", "/api/courses", tt, `{"title":"10А анги","price":0,"published":true,"access_code":"  Анги-10А  "}`); code != 201 || r["access_code"] != "Анги-10А" || r["locked"] != true {
		t.Fatalf("сургалт: %d %v", code, r)
	}
	code, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Нууц","price":5000,"published":true,"access_code":"abc"}`)
	if code != 400 {
		t.Fatalf("3 тэмдэгттэй нууц үг: %d %v", code, c)
	}
	_, c = call(t, srv, "POST", "/api/courses", tt, `{"title":"Математик","price":0,"published":true,"access_code":"Анги-10А"}`)
	cid := c["id"].(string)
	_, l1 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Үнэгүй","is_free":true,"blocks":[{"id":"t1","type":"text","text":"А"}]}`)
	code, l2 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Үнэгүй биш","is_free":false,"price":0,"blocks":[{"id":"t2","type":"text","text":"Б"}]}`)
	if code != 201 && code != 200 {
		t.Fatalf("нууц үгтэй сургалтад үнэгүй биш 0₮ хичээл: %d %v", code, l2)
	}
	// Нийтэд: нууц үг хаана ч алга, түгжээтэй гэж харагдана.
	res, _ := http.Get(srv.URL + "/c/" + cid)
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	res2, _ := http.Get(srv.URL + "/api/courses/" + cid)
	pub, _ := io.ReadAll(res2.Body)
	res2.Body.Close()
	for _, b := range [][]byte{page, pub} {
		if strings.Contains(strings.ToLower(string(b)), "анги-10а") {
			t.Fatal("нууц үг нийтэд гарч байна")
		}
	}
	if !strings.Contains(string(pub), `"locked":true`) || !strings.Contains(string(page), "Нууц үгээр нээх") {
		t.Fatal("түгжээтэй гэж харагдах ёстой")
	}
	_, mine := call(t, srv, "GET", "/api/me/courses/"+cid, tt, "")
	if mine["course"].(map[string]any)["access_code"] != "Анги-10А" {
		t.Fatalf("багшид нууц үг харагдах ёстой: %v", mine["course"])
	}

	s1, _ := register(t, srv, "lockstud", "student")
	lesson := func(tok, lid string) int {
		code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, tok, "")
		return code
	}
	if lesson(s1, l1["id"].(string)) == 200 {
		t.Fatal("нууц үгтэй сургалтын үнэгүй хичээл ч түгжээтэй байх ёстой")
	}
	if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, ""); code != 403 || r["need_code"] != true {
		t.Fatalf("нууц үггүй элсэх: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/unlock", s1, `{"code":"буруу үг"}`); code != 403 {
		t.Fatalf("буруу нууц үг: %d", code)
	}
	if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/unlock", s1, `{"code":"  анги-10а "}`); code != 201 || r["enrolled"] != true {
		t.Fatalf("зөв нууц үг (том жижиг, зай хамаагүй): %d %v", code, r)
	}
	if lesson(s1, l1["id"].(string)) != 200 || lesson(s1, l2["id"].(string)) != 200 {
		t.Fatal("нээгдсэний дараа бүх хичээл үзэгдэх ёстой")
	}
	if _, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, ""); a["all"] != true {
		t.Fatalf("all: %v", a["all"])
	}
	// Таах оролдлого хязгаартай.
	s2, _ := register(t, srv, "guesser", "student")
	limited := false
	for i := 0; i < 10; i++ {
		if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/unlock", s2, `{"code":"guess`+string(rune('a'+i))+`"}`); code == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("олон буруу оролдлого хязгаарлагдах ёстой")
	}
	// «Нийтлэх» г.м access_code-гүй хадгалалт нууц үгийг арилгахгүй; "" бол түгжээ авагдана.
	call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Математик","price":0,"published":true}`)
	if _, m := call(t, srv, "GET", "/api/me/courses/"+cid, tt, ""); m["course"].(map[string]any)["access_code"] != "Анги-10А" {
		t.Fatal("хэсэгчилсэн хадгалалтад нууц үг хэвээр байх ёстой")
	}
	call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Математик","price":0,"published":true,"access_code":""}`)
	s3, _ := register(t, srv, "freestud", "student")
	if lesson(s3, l1["id"].(string)) != 200 {
		t.Fatal("нууц үг арилсны дараа үнэгүй хичээл нээлттэй байх ёстой")
	}
}

// Заавал багшаас нууц үг авна: түгжихээс өмнө элссэн хүн ч, багш үгээ сольсон үед ч одоогийн үгийг оруулна.
func TestLockedCourseStrict(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "strictteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Анги","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"t1","type":"text","text":"А"}]}`)
	lid := l["id"].(string)
	s1, _ := register(t, srv, "earlybird", "student")
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")
	lesson := func() int { code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, s1, ""); return code }
	if lesson() != 200 {
		t.Fatal("түгжихээс өмнө үнэгүй хичээл нээлттэй")
	}
	call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Анги","price":0,"published":true,"access_code":"Нэгдүгээр"}`)
	if lesson() == 200 {
		t.Fatal("өмнө элссэн ч нууц үг шаардах ёстой")
	}
	if _, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, ""); a["need_code"] != true || a["all"] == true {
		t.Fatalf("need_code: %v all: %v", a["need_code"], a["all"])
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, ""); code != 403 {
		t.Fatalf("элсэлтээр тойрч болохгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/unlock", s1, `{"code":"нэгдүгээр"}`); code != 201 || lesson() != 200 {
		t.Fatalf("нууц үгээр нээх: %d", code)
	}
	call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Анги","price":0,"published":true,"access_code":"Хоёрдугаар"}`)
	if lesson() == 200 {
		t.Fatal("багш үгээ сольсон бол шинэ үгийг оруулах ёстой")
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/unlock", s1, `{"code":"Хоёрдугаар"}`); code != 201 || lesson() != 200 {
		t.Fatalf("шинэ үгээр нээх: %d", code)
	}
}
