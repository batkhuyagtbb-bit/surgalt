package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

// Даалгаврын хариуны дүрэм: текст засварлагчийн HTML цэвэрлэгдэж, цэвэр текст нь үгийн тоонд; таб сольсон тоо хадгалагдаж
// багшид харагдана; хариуг нэг л удаа илгээнэ (resubmit зөвшөөрвөл шинэчилнэ); үгийн баримжаа 0-5000.
func TestAssignmentAnswerRules(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "arules", "teacher")
	s1, _ := register(t, srv, "astud", "student")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Даалгавар","price":0,"published":true}`)
	cid := c["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Эссе","is_free":true,"assignment":{"max_score":100,"words":9000}}`); code != 400 {
		t.Fatalf("үгийн баримжаа 5000-аас их: %d", code)
	}
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Эссе","is_free":true,"assignment":{"max_score":100,"words":300}}`)
	lid := l["id"].(string)
	if a := l["assignment"].(map[string]any); a["words"].(float64) != 300 || a["resubmit"] != false {
		t.Fatalf("даалгаврын тохиргоо: %v", a)
	}
	base := "/api/courses/" + cid + "/lessons/" + lid
	code, r := call(t, srv, "POST", base+"/submit", s1, `{"html":"<h2>Гарчиг</h2><p>Нэг <b>хоёр</b> гурав<script>alert(1)</script></p><img src=x onerror=alert(1)>","tab_switches":4}`)
	sub := r["submission"].(map[string]any)
	if code != 200 || strings.Contains(sub["html"].(string), "script") || strings.Contains(sub["html"].(string), "onerror") || !strings.Contains(sub["html"].(string), "<b>хоёр</b>") {
		t.Fatalf("HTML цэвэрлэлт: %d %v", code, sub["html"])
	}
	if !strings.Contains(sub["text"].(string), "Нэг хоёр гурав") || sub["tab_switches"].(float64) != 4 {
		t.Fatalf("цэвэр текст, таб: %v", sub)
	}
	if code, _ := call(t, srv, "POST", base+"/submit", s1, `{"text":"дахин"}`); code != http.StatusConflict {
		t.Fatalf("хариуг нэг л удаа: %d", code)
	}
	// Багш дүгнэхдээ таб сольсон тоо, HTML-ийг харна.
	_, subs := call(t, srv, "GET", base+"/submissions", tt, "")
	first := subs["submissions"].([]any)[0].(map[string]any)
	if first["tab_switches"].(float64) != 4 || !strings.Contains(first["html"].(string), "<h2>") {
		t.Fatalf("багшийн харагдац: %v", first)
	}
	// Дахин илгээхийг зөвшөөрвөл шинэчилнэ.
	call(t, srv, "PUT", base, tt, `{"title":"Эссе","is_free":true,"assignment":{"max_score":100,"words":300,"resubmit":true}}`)
	if code, _ := call(t, srv, "POST", base+"/submit", s1, `{"text":"шинэчилсэн хариу"}`); code != 200 {
		t.Fatalf("resubmit зөвшөөрсөн: %d", code)
	}
}
