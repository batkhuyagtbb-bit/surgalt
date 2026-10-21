package httpapi

import (
	"strings"
	"testing"
)

// Маркерын тодруулга зөвхөн тухайн суралцагчид харагдана: өөр суралцагч, багш харахгүй.
func TestHighlightsPrivate(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Маркер","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"txt1","type":"text","text":"Фотосинтез бол ургамлын хоол хийх үйл явц."}]}`)
	base := "/api/courses/" + cid + "/lessons/" + l["id"].(string) + "/highlights"
	s1, _ := register(t, srv, "stud1", "student")
	s2, _ := register(t, srv, "stud2", "student")

	body := `{"highlights":[{"id":"h1","b":"txt1","s":0,"e":10,"c":"y","t":"Фотосинтез","n":"гол ойлголт"}]}`
	if code, _ := call(t, srv, "PUT", base, s1, body); code != 200 {
		t.Fatalf("хадгалах: %d", code)
	}
	_, got := call(t, srv, "GET", base, s1, "")
	hs := got["highlights"].([]any)
	if len(hs) != 1 || hs[0].(map[string]any)["n"] != "гол ойлголт" {
		t.Fatalf("өөрийн тодруулга: %v", got)
	}
	for name, tok := range map[string]string{"өөр суралцагч": s2, "багш": tt} {
		if _, o := call(t, srv, "GET", base, tok, ""); len(o["highlights"].([]any)) != 0 {
			t.Fatalf("%s бусдын тодруулгыг харах ёсгүй: %v", name, o)
		}
	}
	if code, _ := call(t, srv, "GET", base, "", ""); code != 401 {
		t.Fatalf("нэвтрээгүй хүн: %d", code)
	}
	for _, bad := range []string{
		`{"highlights":[{"id":"h1","b":"txt1","s":0,"e":10,"c":"purple","t":"x"}]}`,
		`{"highlights":[{"id":"h1","b":"txt1","s":5,"e":5,"c":"y","t":"x"}]}`,
		`{"highlights":[{"id":"h1","b":"txt1","s":0,"e":3,"c":"y","t":"x","n":"` + strings.Repeat("я", 1001) + `"}]}`,
	} {
		if code, _ := call(t, srv, "PUT", base, s1, bad); code != 400 {
			t.Fatalf("буруу өгөгдөл 400 байх ёстой: %d %s", code, bad[:60])
		}
	}
	if code, _ := call(t, srv, "PUT", base, s1, `{"highlights":[]}`); code != 200 {
		t.Fatalf("бүгдийг арилгах: %d", code)
	}
	if _, o := call(t, srv, "GET", base, s1, ""); len(o["highlights"].([]any)) != 0 {
		t.Fatalf("арилгасны дараа хоосон: %v", o)
	}
}
