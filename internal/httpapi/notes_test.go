package httpapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"regexp"
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

// Кирилл нэртэй хувийн файлын гарын үсэгтэй холбоос: IIS кодчилолыг өөрчилж дамжуулсан ч (жижиг үсгийн %xx,
// давхар кодлолт) нээгдэнэ; буруу гарын үсэг татгалзагдана.
func TestSignedFileEncodings(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "Сургалтын журам.pdf")
	fw.Write([]byte("%PDF-1.4 test"))
	mw.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/me/files?visibility=private", &body)
	req.Header.Set("Authorization", "Bearer "+tt)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode >= 300 {
		t.Fatalf("файл оруулах: %v %v", err, res.StatusCode)
	}
	var info struct {
		URL string `json:"url"`
	}
	json.NewDecoder(res.Body).Decode(&info)
	res.Body.Close()
	path, query, _ := strings.Cut(info.URL, "?")
	lower := regexp.MustCompile(`%[0-9A-F]{2}`).ReplaceAllStringFunc(path, strings.ToLower)
	double := strings.ReplaceAll(path, "%", "%25")
	for name, u := range map[string]string{"эх": path, "жижиг үсэг": lower, "давхар кодлолт": double} {
		r, err := http.Get(srv.URL + u + "?" + query)
		if err != nil || r.StatusCode != 200 {
			t.Fatalf("%s: %v %v", name, err, r.StatusCode)
		}
		r.Body.Close()
	}
	if r, _ := http.Get(srv.URL + path + "?" + strings.Replace(query, "sig=", "sig=x", 1)); r.StatusCode != 403 {
		t.Fatalf("буруу гарын үсэг: %d", r.StatusCode)
	}
}

// Багшийн тэмдэглэлийн статистик: тоо (төрөл, өнгө), story үзэлт; хүрээлсэн хэсгийн зураг зөвхөн өөрийнх.
func TestNotesStats(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Биологи","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Эс","is_free":true,"blocks":[{"id":"txt1","type":"text","text":"Эс бол амьдралын нэгж."}]}`)
	lid := l["id"].(string)
	s1, u1 := register(t, srv, "stud1", "student")
	base := "/api/courses/" + cid + "/lessons/" + lid + "/highlights"
	if code, _ := call(t, srv, "PUT", base, s1, `{"highlights":[{"id":"x1","b":"doc1","s":0,"e":0,"c":"r","t":"2-р хуудас","k":"page","p":2,"box":[10,10,30,20],"img":"/note-img/`+strings.Repeat("a", 24)+`/`+strings.Repeat("b", 32)+`.webp"}]}`); code != 400 {
		t.Fatalf("бусдын зураг хориотой: %d", code)
	}
	ok := `{"highlights":[{"id":"h1","b":"txt1","s":0,"e":2,"c":"y","t":"Эс"},
		{"id":"x1","b":"doc1","s":0,"e":0,"c":"r","t":"2-р хуудас","k":"page","p":2,"box":[10,10,30,20],"img":"/note-img/` + u1 + `/` + strings.Repeat("b", 32) + `.webp"},
		{"id":"v1","b":"vid1","s":0,"e":0,"c":"r","t":"0:05","k":"time","sec":5}]}`
	if code, r := call(t, srv, "PUT", base, s1, ok); code != 200 {
		t.Fatalf("хүрээтэй тэмдэглэл: %d %v", code, r)
	}
	if code, _ := call(t, srv, "PUT", base, s1, `{"highlights":[{"id":"x2","b":"doc1","s":0,"e":0,"c":"y","t":"x","k":"page","p":2,"box":[90,10,30,20]}]}`); code != 400 {
		t.Fatalf("хуудаснаас хэтэрсэн хүрээ: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/me/story-view", s1, `{"lessons":["`+lid+`"],"slides":5,"total":5}`); code != 204 {
		t.Fatalf("story үзэлт: %d", code)
	}
	call(t, srv, "POST", "/api/me/story-view", s1, `{"lessons":["`+lid+`"],"slides":2,"total":5}`)
	if code, _ := call(t, srv, "GET", "/api/me/courses/"+cid+"/notes-stats", s1, ""); code != 403 && code != 404 {
		t.Fatalf("суралцагч статистик харах ёсгүй: %d", code)
	}
	code, st := call(t, srv, "GET", "/api/me/courses/"+cid+"/notes-stats", tt, "")
	if code != 200 {
		t.Fatalf("статистик: %d %v", code, st)
	}
	sum := st["summary"].(map[string]any)
	if sum["notes"].(float64) != 3 || sum["writers"].(float64) != 1 || sum["story_views"].(float64) != 2 || sum["story_done"].(float64) != 1 {
		t.Fatalf("нэгтгэл: %v", sum)
	}
	kinds := sum["kinds"].(map[string]any)
	if kinds["box"].(float64) != 1 || kinds["text"].(float64) != 1 || kinds["time"].(float64) != 1 || sum["colors"].(map[string]any)["r"].(float64) != 2 {
		t.Fatalf("төрөл, өнгө: %v", sum)
	}
	stu := st["students"].([]any)[0].(map[string]any)
	if stu["name"] != "stud1" || stu["notes"].(float64) != 3 || stu["story_views"].(float64) != 2 {
		t.Fatalf("суралцагчийн мөр: %v", stu)
	}
	if b, _ := json.Marshal(st); strings.Contains(string(b), "Эс бол") || strings.Contains(string(b), "note-img") {
		t.Fatalf("багшид тэмдэглэлийн агуулга харагдах ёсгүй: %s", b)
	}
}

// Урт кирилл нэртэй баримтын тасалбар IIS-ийн 260 тэмдэгтийн хязгаараас богино бөгөөд нээгдэнэ.
func TestShortMediaTicket(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	name := "ULMS_-_Сургалтын_албаны_зурагтай_алхамчилсан_заавар_дэлгэрэнгүй.pdf"
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", name)
	fw.Write([]byte("%PDF-1.4 long name"))
	mw.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/me/files?visibility=private", &body)
	req.Header.Set("Authorization", "Bearer "+tt)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode >= 300 {
		t.Fatalf("файл: %v %v", err, res.StatusCode)
	}
	var info struct {
		Path string `json:"path"`
	}
	json.NewDecoder(res.Body).Decode(&info)
	res.Body.Close()
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Журам","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"doc1","type":"file","url":"`+info.Path+`","name":"заавар.pdf"}]}`)
	s1, _ := register(t, srv, "stud", "student")
	_, got := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l["id"].(string), s1, "")
	u := got["blocks"].([]any)[0].(map[string]any)["url"].(string)
	seg := strings.Split(strings.TrimPrefix(u, "/api/media/"), "/")[0]
	if !strings.HasPrefix(u, "/api/media/") || len(seg) > 200 {
		t.Fatalf("тасалбар хэт урт (%d): %s", len(seg), u)
	}
	r, err := http.Get(srv.URL + u)
	if err != nil || r.StatusCode != 200 {
		t.Fatalf("баримт нээгдэх: %v %v", err, r.StatusCode)
	}
	r.Body.Close()
}

// Элсээгүй ч үнэгүй хичээл үзэж оноо авсан суралцагчийн нүүр хуудсанд эрэмбэ гарна.
func TestRankWithoutEnrollment(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Математик","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"txt1","type":"text","text":"Текст."}]}`)
	s1, _ := register(t, srv, "stud", "student")
	if _, h := call(t, srv, "GET", "/api/me/home", s1, ""); h["rank"] != nil {
		t.Fatalf("юу ч үзээгүй бол эрэмбэ гарахгүй: %v", h["rank"])
	}
	call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l["id"].(string), s1, "")
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+l["id"].(string)+"/complete", s1, "")
	call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, "")
	_, h := call(t, srv, "GET", "/api/me/home", s1, "")
	rk, _ := h["rank"].(map[string]any)
	if rk == nil || rk["points"].(float64) <= 0 || len(h["course_ranks"].([]any)) != 1 || len(h["courses"].([]any)) != 0 {
		t.Fatalf("элсээгүй суралцагчийн эрэмбэ: rank=%v courses=%v", h["rank"], h["courses"])
	}
}

// Багш ч бусдын сургалтад суралцагчтай адил эрэмбэ, явцтай; өөрийн сургалтаа үзсэн нь тооцогдохгүй.
func TestTeacherLearnsLikeStudent(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	ta, _ := register(t, srv, "teacha", "teacher")
	tb, _ := register(t, srv, "teachb", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", ta, `{"title":"Физик","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", ta, `{"title":"Нэг","is_free":true,"blocks":[{"id":"txt1","type":"text","text":"Текст."}]}`)
	for _, tok := range []string{tb, ta} {
		call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l["id"].(string), tok, "")
		call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+l["id"].(string)+"/complete", tok, "")
		call(t, srv, "GET", "/api/courses/"+cid+"/access", tok, "")
	}
	_, hb := call(t, srv, "GET", "/api/me/home", tb, "")
	if rk, _ := hb["rank"].(map[string]any); rk == nil || rk["points"].(float64) <= 0 || len(hb["course_ranks"].([]any)) != 1 {
		t.Fatalf("суралцаж буй багшийн эрэмбэ: rank=%v ranks=%v", hb["rank"], hb["course_ranks"])
	}
	if _, ha := call(t, srv, "GET", "/api/me/home", ta, ""); ha["rank"] != nil || len(ha["course_ranks"].([]any)) != 0 {
		t.Fatalf("өөрийн сургалт эрэмбэд орохгүй: rank=%v ranks=%v", ha["rank"], ha["course_ranks"])
	}
}
