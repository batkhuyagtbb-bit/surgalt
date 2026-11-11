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
