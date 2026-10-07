package httpapi

import (
	"fmt"
	"strings"
	"testing"
)

// Рубрикаар үнэлэх даалгавар: мөр (шалгуур) × багана (түвшин), оноог сервер бодно; дүн суралцагч ба журналд орно.
func TestRubricAssignment(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Эссэ","price":0,"published":true}`)
	cid := c["id"].(string)
	lessons := "/api/courses/" + cid + "/lessons"
	// Буруу рубрикууд татгалзагдана.
	for _, bad := range []string{
		`{"grading":"rubric","rubric":{"levels":["Сайн"],"criteria":[{"id":"crit","name":"А","points":[1]}]}}`,
		`{"grading":"rubric","rubric":{"levels":["Сайн","Муу"],"criteria":[]}}`,
		`{"grading":"rubric","rubric":{"levels":["Сайн","Муу"],"criteria":[{"id":"crit","name":"А","points":[2000,0]}]}}`,
		`{"grading":"rubric","rubric":{"levels":["Сайн",""],"criteria":[{"id":"crit","name":"А","points":[1,0]}]}}`,
		`{"grading":"rubric","rubric":{"levels":["Сайн","Муу"],"criteria":[{"id":"crit","name":"А","points":[0,0]}]}}`,
		`{"grading":"magic"}`,
	} {
		if code, r := call(t, srv, "POST", lessons, tt, `{"title":"x","is_free":true,"assignment":`+bad+`}`); code != 400 {
			t.Fatalf("буруу рубрик зөвшөөрөгдлөө (%d %v): %s", code, r, bad)
		}
	}
	rubric := `{"levels":["Маш сайн","Сайн","Дунд"],"criteria":[
		{"id":"crit1","name":"Агуулга","points":[10,6,2],"desc":["Бүрэн, гүнзгий","Гол санаа бий",""]},
		{"id":"c2","name":"Хэлбэр","points":[5,3,1]}]}`
	code, l := call(t, srv, "POST", lessons, tt, `{"title":"Эссэ бичих","is_free":true,"blocks":[{"id":"t001","type":"text","text":"Эссэ бич"}],"assignment":{"due":{},"grading":"rubric","rubric":`+rubric+`}}`)
	if code != 201 {
		t.Fatalf("рубриктэй даалгавар: %d %v", code, l)
	}
	a := l["assignment"].(map[string]any)
	crits := a["rubric"].(map[string]any)["criteria"].([]any)
	if a["max_score"].(float64) != 15 || crits[1].(map[string]any)["id"] != "c200" || len(crits[1].(map[string]any)["desc"].([]any)) != 3 {
		t.Fatalf("дээд оноо 15 (10+5), ID засагдаж, тайлбар баганын тоогоор: %v", a)
	}
	lid := l["id"].(string)
	s1, uid := register(t, srv, "stud", "student")
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")
	if code, r := call(t, srv, "POST", lessons+"/"+lid+"/submit", s1, `{"text":"Миний эссэ"}`); code != 200 && code != 201 {
		t.Fatalf("илгээх: %d %v", code, r)
	}
	grade := lessons + "/" + lid + "/submissions/" + uid
	if code, r := call(t, srv, "PUT", grade, tt, `{"rubric":{"crit1":1},"feedback":"x"}`); code != 400 || !strings.Contains(fmt.Sprint(r), "Хэлбэр") {
		t.Fatalf("шалгуур дутуу бол татгалзана: %d %v", code, r)
	}
	if code, _ := call(t, srv, "PUT", grade, tt, `{"rubric":{"crit1":1,"c200":7}}`); code != 400 {
		t.Fatal("байхгүй түвшин татгалзагдана")
	}
	code, g := call(t, srv, "PUT", grade, tt, `{"score":999,"rubric":{"crit1":1,"c200":0},"feedback":"Агуулгаа гүнзгийрүүл"}`)
	if code != 200 || g["score"].(float64) != 11 || g["rubric"].(map[string]any)["c200"].(float64) != 0 {
		t.Fatalf("оноог сервер бодно (6+5=11): %d %v", code, g)
	}
	// Суралцагчийн хэсэгт: нийт дүн ба шалгуур бүрийн сонголт, мэдэгдэл.
	_, info := call(t, srv, "GET", lessons+"/"+lid+"/assignment", s1, "")
	sub := info["submission"].(map[string]any)
	if sub["score"].(float64) != 11 || sub["rubric"].(map[string]any)["crit1"].(float64) != 1 || info["assignment"].(map[string]any)["rubric"] == nil {
		t.Fatalf("суралцагчид рубрикийн дүн: %v", info)
	}
	_, ns := call(t, srv, "GET", "/api/me/notifications", s1, "")
	if !strings.Contains(fmt.Sprint(ns), "11/15") {
		t.Fatalf("дүнгийн мэдэгдэл 11/15: %v", ns)
	}
	// Журналд: нүдэнд оноо ба сонголт, хичээлд рубрикийн хүснэгт.
	_, jr := call(t, srv, "GET", "/api/me/courses/"+cid+"/journal", tt, "")
	cell := jr["cells"].(map[string]any)[uid].(map[string]any)[lid].(map[string]any)["asg"].(map[string]any)
	var jl map[string]any
	for _, x := range jr["lessons"].([]any) {
		if x.(map[string]any)["id"] == lid {
			jl = x.(map[string]any)
		}
	}
	if cell["score"].(float64) != 11 || cell["max"].(float64) != 15 || cell["rubric"].(map[string]any)["crit1"].(float64) != 1 || jl["rubric"] == nil {
		t.Fatalf("журналд рубрикийн дүн: %v %v", cell, jl)
	}
	// Тоогоор үнэлэх даалгавар хэвээр ажиллана (рубрик илгээсэн ч тоогоор).
	_, l2 := call(t, srv, "POST", lessons, tt, `{"title":"Тоогоор","is_free":true,"assignment":{"due":{},"max_score":20}}`)
	lid2 := l2["id"].(string)
	call(t, srv, "POST", lessons+"/"+lid2+"/submit", s1, `{"text":"хариу"}`)
	if code, g2 := call(t, srv, "PUT", lessons+"/"+lid2+"/submissions/"+uid, tt, `{"score":17,"rubric":{"x":1}}`); code != 200 || g2["score"].(float64) != 17 || g2["rubric"] != nil {
		t.Fatalf("тоогоор үнэлэх: %d %v", code, g2)
	}
}
