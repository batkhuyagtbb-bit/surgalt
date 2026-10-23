package httpapi

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"strings"
	"testing"
)

// Видео (мөч), баримт (хуудас) дээрх тэмдэглэл; номын тэмдэглэл; бүх тэмдэглэлээс «story».
func TestMediaNotesAndStories(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Биологи","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Фотосинтез","is_free":true,"blocks":[{"id":"txt1","type":"text","text":"Ургамал."}]}`)
	base := "/api/courses/" + cid + "/lessons/" + l["id"].(string) + "/highlights"
	s1, _ := register(t, srv, "stud1", "student")

	ok := `{"highlights":[{"id":"h1","b":"txt1","s":0,"e":7,"c":"y","t":"Ургамал"},
		{"id":"v1","b":"vid1","s":0,"e":0,"c":"r","t":"3:25","k":"time","sec":205.4,"n":"томьёо"},
		{"id":"d1","b":"doc1","s":0,"e":0,"c":"b","t":"5-р хуудас","k":"page","p":5}]}`
	if code, r := call(t, srv, "PUT", base, s1, ok); code != 200 {
		t.Fatalf("медиа тэмдэглэл хадгалах: %d %v", code, r)
	}
	for _, bad := range []string{
		`{"highlights":[{"id":"v1","b":"vid1","s":0,"e":0,"c":"y","t":"x","k":"time","sec":-1}]}`,
		`{"highlights":[{"id":"v1","b":"vid1","s":0,"e":0,"c":"y","t":"x","k":"time","sec":3,"p":2}]}`,
		`{"highlights":[{"id":"d1","b":"doc1","s":0,"e":0,"c":"y","t":"x","k":"page","p":0}]}`,
		`{"highlights":[{"id":"d1","b":"doc1","s":0,"e":5,"c":"y","t":"x","k":"page","p":2}]}`,
		`{"highlights":[{"id":"d1","b":"doc1","s":0,"e":0,"c":"y","t":"x","k":"pin","p":2}]}`,
		`{"highlights":[{"id":"h1","b":"txt1","s":0,"e":7,"c":"y","t":"x","sec":4}]}`,
	} {
		if code, _ := call(t, srv, "PUT", base, s1, bad); code != 400 {
			t.Fatalf("буруу тэмдэглэл 400 байх ёстой: %d %s", code, bad)
		}
	}

	// Ном: зөвхөн хуудасны тэмдэглэл
	_, b := call(t, srv, "POST", "/api/me/books", tt, `{"title":"Ургамал судлал","price":0}`)
	bid := b["id"].(string)
	var pg bytes.Buffer
	_ = png.Encode(&pg, image.NewNRGBA(image.Rect(0, 0, 120, 170)))
	for _, n := range []string{"1", "2"} {
		r, _ := http.NewRequest("POST", srv.URL+"/api/me/books/"+bid+"/pages/"+n, bytes.NewReader(pg.Bytes()))
		r.Header.Set("Authorization", "Bearer "+tt)
		if res, err := http.DefaultClient.Do(r); err != nil || res.StatusCode != 204 {
			t.Fatalf("номын хуудас: %v %v", err, res.StatusCode)
		}
	}
	if code, _ := call(t, srv, "POST", "/api/me/books/"+bid+"/pages-done", tt, `{"total":2}`); code != 200 {
		t.Fatalf("хуудас бэлэн: %d", code)
	}
	req, _ := http.NewRequest("PUT", srv.URL+"/api/me/books/"+bid, strings.NewReader(`{"title":"Ургамал судлал","price":0,"published":true}`))
	req.Header.Set("Authorization", "Bearer "+tt)
	if res, err := http.DefaultClient.Do(req); err != nil || res.StatusCode != 200 {
		t.Fatalf("ном нийтлэх: %v %v", err, res.StatusCode)
	}
	nb := "/api/books/" + bid + "/notes"
	if code, _ := call(t, srv, "PUT", nb, s1, `{"highlights":[{"id":"n1","b":"book","s":0,"e":7,"c":"y","t":"Ургамал"}]}`); code != 400 {
		t.Fatalf("номд текстийн тодруулга хориотой: %d", code)
	}
	if code, _ := call(t, srv, "PUT", nb, s1, `{"highlights":[{"id":"n1","b":"book","s":0,"e":0,"c":"g","t":"2-р хуудас","k":"page","p":2,"n":"чухал"}]}`); code != 200 {
		t.Fatalf("номын тэмдэглэл: %d", code)
	}
	if code, _ := call(t, srv, "GET", nb, "", ""); code != 401 {
		t.Fatalf("нэвтрээгүй: %d", code)
	}

	// Story: хичээл, ном тус бүр (зөвхөн өөрт)
	_, st := call(t, srv, "GET", "/api/me/notes", s1, "")
	stories := st["stories"].([]any)
	if len(stories) != 2 {
		t.Fatalf("2 story байх ёстой: %v", st)
	}
	titles := map[string]int{}
	for _, x := range stories {
		m := x.(map[string]any)
		titles[m["title"].(string)] = len(m["notes"].([]any))
		if m["kind"] == "lesson" && (m["sub"] != "Биологи" || !strings.Contains(m["url"].(string), "#l=")) {
			t.Fatalf("хичээлийн story: %v", m)
		}
	}
	if titles["Фотосинтез"] != 3 || titles["Ургамал судлал"] != 1 {
		t.Fatalf("story-ийн тэмдэглэлүүд: %v", titles)
	}
	if _, o := call(t, srv, "GET", "/api/me/notes", tt, ""); len(o["stories"].([]any)) != 0 {
		t.Fatalf("багш бусдын тэмдэглэлийг харах ёсгүй: %v", o)
	}
}
