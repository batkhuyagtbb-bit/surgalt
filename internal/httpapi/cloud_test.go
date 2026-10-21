package httpapi

import "testing"

// Үүлэн үг: суралцагчид үгээ илгээж, нийт тоо (хэдэн хүн бичсэн) үүлэнд гарна; багш зөвхөн харна.
func TestCloudWords(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Үүл","price":0,"published":true}`)
	cid := c["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Буруу","is_free":true,"blocks":[{"id":"cld1","type":"cloud","text":"Асуулт","max":5}]}`); code != 400 {
		t.Fatalf("max 1-3 байх ёстой: %d", code)
	}
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"cld1","type":"cloud","text":"Фотосинтез гэж юу вэ?","max":2}]}`)
	base := "/api/courses/" + cid + "/lessons/" + l["id"].(string) + "/cloud/cld1"
	s1, _ := register(t, srv, "stud1", "student")
	s2, _ := register(t, srv, "stud2", "student")

	if code, _ := call(t, srv, "POST", base, s1, `{"words":["  Нар! ","нар","Ургамал","гэрэл"]}`); code != 200 {
		t.Fatalf("илгээх: %d", code)
	}
	_, r := call(t, srv, "POST", base, s2, `{"words":["НАР"]}`)
	ws := r["words"].([]any)
	top := ws[0].(map[string]any)
	if top["w"] != "нар" || top["n"].(float64) != 2 || len(ws) != 2 || r["people"].(float64) != 2 {
		t.Fatalf("тоолол буруу: %v", r)
	}
	// Дахин илгээвэл өмнөхийг солино
	_, r = call(t, srv, "POST", base, s1, `{"words":["хлорофилл"]}`)
	if mine := r["mine"].([]any); len(mine) != 1 || mine[0] != "хлорофилл" {
		t.Fatalf("өмнөхийг солих ёстой: %v", r)
	}
	if code, _ := call(t, srv, "POST", base, s1, `{"words":["   ","!!!"]}`); code != 400 {
		t.Fatalf("хоосон үг 400: %d", code)
	}
	// Багш хариулахгүй, харин үр дүнг харна
	if code, _ := call(t, srv, "POST", base, tt, `{"words":["багш"]}`); code != 400 {
		t.Fatalf("багш хариулах ёсгүй: %d", code)
	}
	if code, r := call(t, srv, "GET", base, tt, ""); code != 200 || r["people"].(float64) != 2 {
		t.Fatalf("багш үр дүн харах: %d %v", code, r)
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l["id"].(string)+"/cloud/nope", s1, ""); code != 404 {
		t.Fatalf("байхгүй хэсэг 404: %d", code)
	}
}
