package httpapi

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// Сургалтын толгойн зургийн загвар (t01–t20): үүсгэх/засахад хадгалагдана, буруу ID татгалзагдана, cover-гүй
// хэсэгчилсэн хадгалалт зургийг арилгахгүй, нийтийн хуудсанд SVG нь орно, SVG файлууд үйлчилнэ.
func TestCourseCover(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "cov", "teacher")
	code, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Физик","price":0,"published":true,"cover":"t04"}`)
	if code != 201 || c["cover"] != "t04" {
		t.Fatalf("зурагтай үүсгэх: %d %v", code, c)
	}
	cid := c["id"].(string)
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Физик","price":0,"published":true,"cover":"t99"}`); code != 400 {
		t.Fatalf("буруу загвар: %d", code)
	}
	if _, u := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Физик 2","price":0,"published":true}`); u["cover"] != "t04" {
		t.Fatalf("cover-гүй хадгалалт зургийг арилгахгүй: %v", u)
	}
	if _, u := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Физик 2","price":0,"published":true,"cover":"t11"}`); u["cover"] != "t11" {
		t.Fatalf("солих: %v", u)
	}
	res, err := http.Get(srv.URL + "/c/" + cid)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "/static/covers/t11.svg") {
		t.Fatalf("нийтийн хуудас: %d, зураг олдсонгүй", res.StatusCode)
	}
	for _, id := range []string{"t01", "t11", "t20"} {
		r, err := http.Get(srv.URL + "/static/covers/" + id + ".svg")
		if err != nil || r.StatusCode != 200 || !strings.Contains(r.Header.Get("Content-Type"), "svg") {
			t.Fatalf("%s.svg: %v %d %s", id, err, r.StatusCode, r.Header.Get("Content-Type"))
		}
		r.Body.Close()
	}
	if _, u := call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"Физик 2","price":0,"published":true,"cover":""}`); u["cover"] != "" {
		t.Fatalf("зураггүй болгох: %v", u)
	}
}

// Сургалтын товч тайлбар 50 үгэнд багтана.
func TestCourseDescriptionWords(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "desc", "teacher")
	long := strings.Repeat("үг ", 51)
	if code, r := call(t, srv, "POST", "/api/courses", tt, `{"title":"Урт","price":0,"description":"`+long+`"}`); code != 400 || !strings.Contains(r["error"].(string), "50 үг") {
		t.Fatalf("51 үг: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/courses", tt, `{"title":"Яг","price":0,"description":"`+strings.Repeat("үг ", 50)+`"}`); code != 201 {
		t.Fatalf("50 үг: %d", code)
	}
}

// Толгойн зургийн загвар профайлын нүүр зураг ч болно; буруу загвар татгалзагдана.
func TestProfileCoverTemplate(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "pcov", "teacher")
	code, me := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Б. Багш","cover_url":"/static/covers/t13.svg"}`)
	if code != 200 || me["cover_url"] != "/static/covers/t13.svg" {
		t.Fatalf("загвар нүүр зураг: %d %v", code, me)
	}
	if code, _ := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Б. Багш","cover_url":"/static/covers/t99.svg"}`); code != 400 {
		t.Fatalf("буруу загвар: %d", code)
	}
	if code, _ := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Б. Багш","cover_url":"/static/app.css"}`); code != 400 {
		t.Fatalf("өөр static файл: %d", code)
	}
}
