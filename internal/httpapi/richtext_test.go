package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

// Word шиг текстийн HTML: хэлбэржүүлэлт үлдэж, скрипт, event, javascript: холбоос, бусдын файл хасагдана.
func TestSanitizeRich(t *testing.T) {
	const tid = "t1"
	own := "/files/t1/private/0123456789abcdef__zurag.webp"
	in := `<h1 style="text-align:center;color:#dc2626;position:fixed">Гарчиг</h1>` +
		`<p onclick="alert(1)" style="font-size:x-large;margin-left:2em;background:url(x)">Сайн <b>тод</b> <font color="red">улаан</font> H<sub>2</sub>O x<sup>2</sup></p>` +
		`<script>alert(1)</script><style>p{}</style><iframe src="https://evil"></iframe>` +
		`<a href="javascript:alert(1)">муу</a> <a href="https://surgalt.mn" onmouseover="x">сайн</a>` +
		`<img src="` + own + `?exp=1&sig=abc" class="rt-center evil" style="width:50%;height:9px" onerror="alert(1)">` +
		`<img src="/files/t2/private/0123456789abcdef__other.webp"><img src="/files/t1/../t2/x.webp"><img src="data:image/png;base64,AAAA"><img src="javascript:x">` +
		`<table><tr><td colspan="2" style="width:40%">A</td><td rowspan="x">B</td></tr></table>` +
		`<o:p>Word</o:p><ul><li>нэг<li>хоёр</ul><p>неспалсан`
	out, text, imgs := sanitizeRich(in, tid)
	for _, bad := range []string{"script", "alert", "onclick", "onerror", "onmouseover", "javascript", "iframe", "evil", "position", "url(", "t2", "data:", "height", "<style", "o:p", "rowspan", "?exp", "<h1", "<font"} {
		if strings.Contains(out, bad) {
			t.Errorf("%q хасагдаагүй:\n%s", bad, out)
		}
	}
	for _, good := range []string{`<h2 style="text-align:center;color:#dc2626">Гарчиг</h2>`, `style="font-size:x-large;margin-left:2em"`, `<b>тод</b>`, `<span style="color:red">улаан</span>`,
		`<sub>2</sub>`, `<sup>2</sup>`, `<a href="https://surgalt.mn" target="_blank" rel="noopener nofollow">сайн</a>`, `<img src="` + own + `" class="rt-center" style="width:50%">`,
		`<td colspan="2" style="width:40%">A</td>`, `<li>нэг`, `<p>неспалсан</p>`} {
		if !strings.Contains(out, good) {
			t.Errorf("%q алга:\n%s", good, out)
		}
	}
	if imgs != 1 {
		t.Errorf("зураг %d", imgs)
	}
	if !strings.HasPrefix(text, "Гарчиг\n") || !strings.Contains(text, "Сайн тод улаан H2O x2") || strings.Contains(text, "alert") || !strings.Contains(text, "нэг\nхоёр") {
		t.Errorf("цэвэр текст: %q", text)
	}
	if o, _, _ := sanitizeRich(`<p style="line-height:1.5;margin-bottom:1.6em;margin-top:0px">a</p><p style="line-height:99;margin-bottom:url(x)">b</p>`, tid); o != `<p style="line-height:1.5;margin-bottom:1.6em;margin-top:0px">a</p><p>b</p>` {
		t.Errorf("зай: %s", o)
	}
	if o, _, _ := sanitizeRich(`<b><i>хаагаагүй`, tid); o != "<b><i>хаагаагүй</i></b>" {
		t.Errorf("хаах: %s", o)
	}
	if o, _, _ := sanitizeRich(`<svg><script>x</script><a>y</a></svg>z<p>&lt;b&gt;</p>`, tid); o != "z<p>&lt;b&gt;</p>" {
		t.Errorf("svg/escape: %s", o)
	}
}

// Хичээлд хадгалж, суралцагч ба багшид гарын үсэгтэй зурагтай HTML ирнэ; хоосон, зөвхөн зурагтай текст.
func TestRichTextLesson(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, tid := register(t, srv, "rich", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Word","price":0,"published":true}`)
	cid := c["id"].(string)
	img := "/files/" + tid + "/private/0123456789abcdef__zurag.webp"
	body := `{"title":"Хэлбэр","is_free":true,"blocks":[{"id":"t1","type":"text","html":"<p style=\"color:#16a34a\">Ногоон <b>тод</b></p><script>x()</script>"},{"id":"t2","type":"text","html":"<p><img src=\"` + img + `?sig=old\"></p>"}]}`
	code, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, body)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("хадгалах: %d %v", code, l)
	}
	if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Хоосон","blocks":[{"id":"t3","type":"text","html":"<p> <br></p><script>a</script>"}]}`); code != http.StatusBadRequest {
		t.Fatalf("хоосон HTML: %d %v", code, r)
	}
	_, mine := call(t, srv, "GET", "/api/me/courses/"+cid, tt, "")
	bs := mine["lessons"].([]any)[0].(map[string]any)["blocks"].([]any)
	b0, b1 := bs[0].(map[string]any), bs[1].(map[string]any)
	if b0["html"] != `<p style="color:#16a34a">Ногоон <b>тод</b></p>` || b0["text"] != "Ногоон тод" {
		t.Fatalf("текст: %v", b0)
	}
	if h := b1["html"].(string); !strings.Contains(h, `src="`+img+`?`) || strings.Contains(h, "sig=old") || b1["text"] != nil {
		t.Fatalf("зураг багшид гарын үсэгтэй: %v", b1)
	}
	s1, _ := register(t, srv, "richs", "student")
	_, view := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l["id"].(string), s1, "")
	vb := view["blocks"].([]any)
	if h := vb[1].(map[string]any)["html"].(string); !strings.Contains(h, `src="`+img+`?`) {
		t.Fatalf("суралцагчид зураг: %s", h)
	}
}
