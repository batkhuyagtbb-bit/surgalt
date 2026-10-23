package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
)

// Цагаан самбар: багш заана → хувиараа (ажлын хуудас хуулагдана) → багаар; түгжсэн объект, түр зогсоох,
// зургийн эх сурвалжийн шалгалт, зураг оруулах, санах ойгоос буулгаад дахин ачаалах.
func TestBoard(t *testing.T) {
	st := newTestStore(t)
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{TrustProxy: true, DevPayments: true}, st, auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Самбар","price":0,"published":true}`)
	cid := c["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Буруу","is_free":true,"blocks":[{"id":"brd1","type":"board","text":"Зур","max":20}]}`); code != 400 {
		t.Fatalf("багийн тоо 2-12 байх ёстой: %d", code)
	}
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"brd1","type":"board","text":""}]}`)
	lid := l["id"].(string)
	base := "/api/courses/" + cid + "/lessons/" + lid + "/board/brd1"
	s1, _ := register(t, srv, "stud1", "student")
	s2, _ := register(t, srv, "stud2", "student")

	// Багш заана: суралцагч зөвхөн харна.
	code, g := call(t, srv, "GET", base, s1, "")
	if code != 200 || g["room"] != "main" || g["write"] != false || g["title"] != "Цагаан самбар" {
		t.Fatalf("суралцагч нэгдэх: %d %v", code, g)
	}
	uid1 := g["me"].(map[string]any)["uid"].(string)
	if code, _ := call(t, srv, "POST", base+"/ops", s1, `{"room":"main","cid":"abcd1","ops":[{"t":"add","o":{"id":"obj001","type":"rect"}}]}`); code != 403 {
		t.Fatalf("суралцагч багшийн самбарт бичих ёсгүй: %d", code)
	}
	ops := `{"room":"main","cid":"tch01","ops":[{"t":"add","o":{"id":"wsheet","type":"rect","left":1,"lk":true}},{"t":"add","o":{"id":"free01","type":"path","path":[["M",0,0]]}}]}`
	if code, r := call(t, srv, "POST", base+"/ops", tt, ops); code != 200 || r["v"].(float64) != 1 {
		t.Fatalf("багш бичих: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", base+"/ops", tt, `{"room":"main","cid":"tch01","ops":[{"t":"add","o":{"id":"img001","type":"image","src":"https://evil.example/x.png"}}]}`); code != 400 {
		t.Fatalf("гадаад зураг хориотой: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/ops", tt, `{"room":"main","cid":"tch01","ops":[{"t":"add","o":{"id":"scr001","type":"script"}}]}`); code != 400 {
		t.Fatalf("үл мэдэх төрөл: %d", code)
	}
	if _, g = call(t, srv, "GET", base, s1, ""); len(g["objects"].([]any)) != 2 {
		t.Fatalf("суралцагч багшийн самбарыг харах: %v", g)
	}

	// Хувиараа: шинэ самбар багшийнхаас эхэлнэ; түгжсэн объектыг суралцагч устгаж чадахгүй.
	if code, _ := call(t, srv, "POST", base+"/state", s1, `{"mode":"solo"}`); code != 403 {
		t.Fatalf("суралцагч горим солих ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/state", tt, `{"mode":"solo"}`); code != 200 {
		t.Fatalf("горим: %d", code)
	}
	_, g = call(t, srv, "GET", base, s1, "")
	if g["room"] != "u:"+uid1 || g["write"] != true || len(g["objects"].([]any)) != 2 {
		t.Fatalf("хувийн самбар (ажлын хуудастай): %v", g)
	}
	room1 := "u:" + uid1
	call(t, srv, "POST", base+"/ops", s1, `{"room":"`+room1+`","cid":"s1aaa","ops":[{"t":"del","id":"wsheet"},{"t":"del","id":"free01"},{"t":"add","o":{"id":"mine01","type":"textbox","text":"Сайн уу"}}]}`)
	_, g = call(t, srv, "GET", base+"?peek=1&room="+room1, tt, "")
	if objs := g["objects"].([]any); len(objs) != 2 || !strings.Contains(objMust(t, objs[0]), "wsheet") {
		t.Fatalf("түгжсэн объект үлдэж, чөлөөт нь устах ёстой: %v", g["objects"])
	}
	if code, _ := call(t, srv, "GET", base+"?peek=1&room="+room1, s2, ""); code != 403 {
		t.Fatalf("бусдын самбарыг харах ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/ops", s2, `{"room":"`+room1+`","cid":"s2aaa","ops":[{"t":"clear"}]}`); code != 403 {
		t.Fatalf("бусдын самбарт бичих ёсгүй: %d", code)
	}

	// Түр зогсоох: суралцагч бичиж чадахгүй.
	call(t, srv, "POST", base+"/state", tt, `{"lock":true}`)
	if code, _ := call(t, srv, "POST", base+"/ops", s1, `{"room":"`+room1+`","cid":"s1aaa","ops":[{"t":"add","o":{"id":"mine02","type":"rect"}}]}`); code != 403 {
		t.Fatalf("зогсоосон үед бичих ёсгүй: %d", code)
	}
	call(t, srv, "POST", base+"/state", tt, `{"lock":false}`)

	// Багаар: самбар дээр байгаа хүүхдүүд багт хуваагдана; хоёулаа нэг багт бол нэг самбар.
	call(t, srv, "GET", base, s2, "")
	if code, r := call(t, srv, "POST", base+"/state", tt, `{"mode":"team","teams":2,"shuffle":true}`); code != 200 || len(r["state"].(map[string]any)["assign"].(map[string]any)) != 2 {
		t.Fatalf("багт хуваах: %d %v", code, r)
	}
	_, g = call(t, srv, "GET", base, s1, "")
	team := g["room"].(string)
	if !strings.HasPrefix(team, "t:") || g["write"] != true {
		t.Fatalf("багийн самбар: %v", g)
	}
	if code, _ := call(t, srv, "POST", base+"/ops", s1, `{"room":"`+team+`","cid":"s1aaa","ops":[{"t":"add","o":{"id":"team01","type":"ellipse"}}]}`); code != 200 {
		t.Fatalf("багийн самбарт бичих: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/state", tt, `{"distribute":true}`); code != 200 {
		t.Fatalf("тараах: %d", code)
	}
	if _, g = call(t, srv, "GET", base+"?peek=1&room="+team, tt, ""); len(g["objects"].([]any)) != 2 {
		t.Fatalf("тараасны дараа багшийн самбартай ижил: %v", g["objects"])
	}

	// Зураг: PNG → WebP, /board-img/-ээр үйлчилнэ.
	img := image.NewNRGBA(image.Rect(0, 0, 40, 30))
	img.Set(5, 5, color.NRGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	req, _ := http.NewRequest("POST", srv.URL+base+"/image", &buf)
	req.Header.Set("Authorization", "Bearer "+tt)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("зураг оруулах: %v %v", err, res.StatusCode)
	}
	var up struct {
		URL  string `json:"url"`
		W, H int
	}
	err = json.NewDecoder(res.Body).Decode(&up)
	res.Body.Close()
	if err != nil || !strings.HasPrefix(up.URL, "/board-img/"+lid+"/") || up.W != 40 {
		t.Fatalf("зургийн хариу: %v %+v", err, up)
	}
	if r2, err := http.Get(srv.URL + up.URL); err != nil || r2.StatusCode != 200 || r2.Header.Get("Content-Type") != "image/webp" {
		t.Fatalf("самбарын зураг үйлчлэх: %v %v", err, r2)
	}
	if code, _ := call(t, srv, "POST", base+"/ops", tt, `{"room":"main","cid":"tch01","ops":[{"t":"add","o":{"id":"img002","type":"image","src":"`+up.URL+`"}}]}`); code != 200 {
		t.Fatalf("өөрийн зураг зөвшөөрөгдөх ёстой: %d", code)
	}

	// Хадгалаад санах ойгоос буулгасан ч бүгд сэргэнэ.
	s.flushBoards(context.Background(), true)
	s.boards.mu.Lock()
	s.boards.m = nil
	s.boards.mu.Unlock()
	_, g = call(t, srv, "GET", base+"?peek=1&room=main", tt, "")
	if len(g["objects"].([]any)) != 3 {
		t.Fatalf("хадгалсан самбар сэргэх: %v", g["objects"])
	}
	_, g = call(t, srv, "GET", base, s1, "")
	if g["room"] != team || g["state"].(map[string]any)["mode"] != "team" {
		t.Fatalf("ангийн төлөв сэргэх: %v", g)
	}
}

func objMust(t *testing.T, o any) string {
	m, ok := o.(map[string]any)
	if !ok {
		t.Fatalf("объект биш: %v", o)
	}
	id, _ := m["id"].(string)
	return id
}
