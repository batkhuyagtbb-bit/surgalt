package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// Үүлэн үг WebSocket-оор: эрхтэй суралцагч бүртгүүлж шинэ хариултын нэгтгэлийг авна; эрхгүй (төлбөртэй хичээл)
// бүртгүүлж чадахгүй; олон хариулт зэрэг ирэхэд түлхэлт нэгтгэгдэнэ.
func TestCloudPush(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "pushteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Үүл","price":5000,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Үнэгүй","is_free":true,"blocks":[{"id":"cld1","type":"cloud","text":"Юу вэ?","max":2}]}`)
	_, lp := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Төлбөртэй","is_free":false,"blocks":[{"id":"cld2","type":"cloud","text":"Юу вэ?","max":2}]}`)
	lid, lpid := l["id"].(string), lp["id"].(string)
	viewer, _ := register(t, srv, "pushview", "student")
	writer, _ := register(t, srv, "pushwrite", "student")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/chat/ws?token="+viewer, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	send := func(v any) {
		b, _ := json.Marshal(v)
		if err := conn.Write(ctx, websocket.MessageText, b); err != nil {
			t.Fatal(err)
		}
	}
	send(map[string]string{"t": "sub", "topic": "cloud", "course": cid, "lesson": lpid, "block": "cld2"}) // эрхгүй — бүртгэгдэхгүй
	send(map[string]string{"t": "sub", "topic": "cloud", "course": cid, "lesson": lid, "block": "cld1"})
	time.Sleep(300 * time.Millisecond)

	for i, w := range []string{"нар", "гэрэл", "ургамал"} { // 3 хариулт зэрэг — эхнийх шууд, бусад нь нэгтгэгдэнэ
		if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/cloud/cld1", writer, `{"words":["`+w+`"]}`); code != 200 {
			t.Fatalf("хариулт %d: %d", i, code)
		}
	}
	got := []map[string]any{}
	deadline := time.Now().Add(cloudPushEvery + 3*time.Second)
	for time.Now().Before(deadline) {
		rctx, rc := context.WithDeadline(ctx, deadline)
		_, data, err := conn.Read(rctx)
		rc()
		if err != nil {
			break
		}
		var ev map[string]any
		if json.Unmarshal(data, &ev) == nil && ev["type"] == "cloud" {
			got = append(got, ev)
			if ev["block"] == "cld2" {
				t.Fatal("эрхгүй хичээлийн үүлэн үг ирлээ")
			}
		}
	}
	if len(got) == 0 || len(got) > 2 {
		t.Fatalf("түлхэлт 1-2 удаа ирэх ёстой (нэгтгэсэн), ирсэн: %d", len(got))
	}
	last := got[len(got)-1]
	ws, _ := last["words"].([]any)
	if len(ws) != 1 || last["people"].(float64) != 1 || ws[0].(map[string]any)["w"] != "ургамал" {
		t.Fatalf("сүүлийн нэгтгэл: %v", last)
	}
	if _, has := last["mine"]; has {
		t.Fatal("түлхэлтэд «миний үг» байх ёсгүй")
	}
}

// Хичээл нээх нэгтгэсэн хүсэлт: include-ээр тодруулга, хэлэлцүүлэг хамт ирнэ.
func TestLessonInclude(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "incteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Нэгтгэл","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Хичээл","is_free":true,"discussion":true,"blocks":[{"id":"t1","type":"text","text":"Фотосинтез"}]}`)
	lid := l["id"].(string)
	st, _ := register(t, srv, "incstud", "student")
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/discussion", st, `{"body":"Сайн хичээл"}`)
	call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+lid+"/highlights", st, `{"highlights":[{"id":"h1","b":"t1","s":0,"e":5,"t":"Фотос","c":"y"}]}`)

	_, plain := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, st, "")
	if plain["_highlights"] != nil || plain["_discussion"] != nil || plain["title"] != "Хичээл" {
		t.Fatalf("include-гүй хариу өөрчлөгдсөн: %v", plain)
	}
	code, b := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid+"?include=highlights,discussion", st, "")
	if code != 200 || b["title"] != "Хичээл" || b["id"] != lid {
		t.Fatalf("нэгтгэсэн хариу: %d %v", code, b)
	}
	d, _ := b["_discussion"].(map[string]any)
	if d == nil || d["count"].(float64) != 1 {
		t.Fatalf("хэлэлцүүлэг алга: %v", b["_discussion"])
	}
	h, _ := b["_highlights"].(map[string]any)
	if h == nil || len(h["highlights"].([]any)) != 1 {
		t.Fatalf("тодруулга алга: %v", b["_highlights"])
	}
}
