package httpapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// «Чат устгах»: зөвхөн устгасан хүний жагсаалт, мессежээс алга болно; нөгөө тал хэвээр харна;
// шинэ мессеж ирвэл яриа дахин гарна (зөвхөн шинэ мессежтэй).
func TestHideChat(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "hideteach", "teacher")
	st, _ := register(t, srv, "hidestud", "student")
	_, ch := call(t, srv, "POST", "/api/teachers/hideteach/chat", st, "")
	conv := ch["conversation"].(map[string]any)["id"].(string)
	call(t, srv, "POST", "/api/chat/"+conv+"/messages", st, `{"body":"Сайн байна уу"}`)
	if _, list := callArr(t, srv, "GET", "/api/me/conversations", tt, ""); len(list) != 1 {
		t.Fatalf("багшийн жагсаалт: %v", list)
	}
	other, _ := register(t, srv, "hideother", "student")
	if code, _ := call(t, srv, "DELETE", "/api/chat/"+conv, other, ""); code != 404 {
		t.Fatalf("хөндлөнгийн хүн устгах ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "DELETE", "/api/chat/"+conv, "", ""); code != 401 {
		t.Fatalf("нэвтрээгүй: %d", code)
	}
	if code, r := call(t, srv, "DELETE", "/api/chat/"+conv, tt, ""); code != 200 || r["hidden"] != true {
		t.Fatalf("устгах: %d %v", code, r)
	}
	if _, list := callArr(t, srv, "GET", "/api/me/conversations", tt, ""); len(list) != 0 {
		t.Fatalf("устгасны дараа багшийн жагсаалтад үлдсэн: %v", list)
	}
	msgs := func(tok string) []any {
		_, m := call(t, srv, "GET", "/api/chat/"+conv+"/messages", tok, "")
		l, _ := m["messages"].([]any)
		return l
	}
	if l := msgs(tt); len(l) != 0 {
		t.Fatalf("устгасан хүнд хуучин мессеж харагдсан: %v", l)
	}
	if l := msgs(st); len(l) != 1 {
		t.Fatalf("нөгөө талд мессеж хэвээр байх ёстой: %v", l)
	}
	_, h := call(t, srv, "GET", "/api/me/home", st, "")
	if chats, _ := h["chats"].([]any); len(chats) != 1 {
		t.Fatalf("суралцагчийн жагсаалтад яриа хэвээр байх ёстой: %v", h["chats"])
	}
	time.Sleep(5 * time.Millisecond)
	call(t, srv, "POST", "/api/chat/"+conv+"/messages", st, `{"body":"Дахин асууя"}`)
	if _, list := callArr(t, srv, "GET", "/api/me/conversations", tt, ""); len(list) != 1 {
		t.Fatalf("шинэ мессежээр яриа дахин гарах ёстой: %v", list)
	}
	if l := msgs(tt); len(l) != 1 || l[0].(map[string]any)["body"] != "Дахин асууя" {
		t.Fatalf("зөвхөн шинэ мессеж харагдана: %v", l)
	}
	// Суралцагч өөрийн талаас устгавал багшид нөлөөлөхгүй.
	call(t, srv, "DELETE", "/api/chat/"+conv, st, "")
	_, h = call(t, srv, "GET", "/api/me/home", st, "")
	if chats, _ := h["chats"].([]any); len(chats) != 0 {
		t.Fatalf("суралцагчийн жагсаалтаас алга болох ёстой: %v", h["chats"])
	}
	if l := msgs(tt); len(l) != 1 {
		t.Fatalf("багшид нөлөөлөх ёсгүй: %v", l)
	}
}

// Багшийн онлайн төлөв: WebSocket холболттой үед online, салсны дараа last_seen.
func TestTeacherPresence(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "presteach", "teacher")
	state := func() map[string]any {
		code, d := call(t, srv, "GET", "/api/teachers/presteach/presence", "", "")
		if code != 200 {
			t.Fatalf("presence: %d %v", code, d)
		}
		return d
	}
	if d := state(); d["online"] != false || d["last_seen"] != nil {
		t.Fatalf("эхэндээ офлайн: %v", d)
	}
	if code, _ := call(t, srv, "GET", "/api/teachers/nobody-here/presence", "", ""); code != 404 {
		t.Fatalf("байхгүй багш: %d", code)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/chat/ws?token="+tt, nil)
	if err != nil {
		t.Fatal(err)
	}
	wait := func(want bool) map[string]any {
		for i := 0; i < 100; i++ {
			if d := state(); d["online"] == want {
				return d
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("online=%v болсонгүй", want)
		return nil
	}
	wait(true)
	conn.Close(websocket.StatusNormalClosure, "")
	if d := wait(false); d["last_seen"] == nil {
		t.Fatalf("салсны дараа last_seen байх ёстой: %v", d)
	}
}

// Журнал: идэвхтэй суралцагч цагаан толгойн дарааллаас үл хамааран дээрээ, recent=true.
func TestJournalActiveFirst(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jrteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Журнал","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true}`)
	lid := l["id"].(string)
	idle, _ := register(t, srv, "aaidle", "student")
	act, actID := register(t, srv, "zzactive", "student")
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", idle, "")
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", act, "")
	_, ss := call(t, srv, "POST", "/api/activity/start", act, `{"course_id":"`+cid+`","lesson_id":"`+lid+`","kind":"lesson"}`)
	call(t, srv, "POST", "/api/activity/beat", act, `{"session_id":"`+ss["session_id"].(string)+`","active":30}`)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/complete", act, "")
	_, j := call(t, srv, "GET", "/api/me/courses/"+cid+"/journal", tt, "")
	sts := j["students"].([]any)
	if len(sts) != 2 {
		t.Fatalf("2 суралцагч: %v", sts)
	}
	first, second := sts[0].(map[string]any), sts[1].(map[string]any)
	if first["user_id"] != actID || first["recent"] != true || second["recent"] == true {
		t.Fatalf("идэвхтэй нь дээрээ байх ёстой: %v", sts)
	}
}
