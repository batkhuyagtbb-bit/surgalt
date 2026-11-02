package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
)

// Багийн чат (баг доторх, баг ↔ багш, бүх багт), дуут өрөө (баг, тайз, гар өргөх),
// оролцооны хэмжүүр → багшийн статистик.
func TestBoardChatVoiceStats(t *testing.T) {
	st := newTestStore(t)
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	hub := chat.NewHub()
	s := New(Config{TrustProxy: true, DevPayments: true}, st, auth.NewSigner("test", time.Hour), hub, fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	tt, _ := register(t, srv, "bcteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Баг","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"brd1","type":"board","text":""}]}`)
	base := "/api/courses/" + cid + "/lessons/" + l["id"].(string) + "/board/brd1"
	s1, u1 := register(t, srv, "bcstud1", "student")
	s2, u2 := register(t, srv, "bcstud2", "student")
	s3, u3 := register(t, srv, "bcstud3", "student")
	for _, tok := range []string{tt, s1, s2, s3} {
		if code, _ := call(t, srv, "GET", base, tok, ""); code != 200 {
			t.Fatalf("самбарт нэгдэх: %d", code)
		}
	}
	// s2-ийн WebSocket суваг: зөвхөн өөрийн багийн чат, дуу ирэх ёстой.
	ws2 := chat.NewClient()
	hub.Subscribe(ws2, chat.UserKey(u2))
	defer hub.Unsubscribe(ws2, chat.UserKey(u2))
	drain := func() (evs []boardEvent) {
		for {
			select {
			case b := <-ws2.Send:
				var e boardEvent
				if json.Unmarshal(b, &e) == nil && (e.Ev == "chat" || e.Ev == "voice" || e.Ev == "rtc") {
					evs = append(evs, e)
				}
			default:
				return
			}
		}
	}

	// «Багаар» горимоос бусад үед суралцагч чатлахгүй.
	call(t, srv, "POST", base+"/state", tt, `{"mode":"team","teams":2,"assign":{"`+u1+`":0,"`+u2+`":0,"`+u3+`":1}}`)
	call(t, srv, "POST", base+"/state", tt, `{"mode":"solo"}`)
	if code, _ := call(t, srv, "POST", base+"/chat", s1, `{"ch":"t:0","body":"сайн уу"}`); code != 403 {
		t.Fatalf("хувиараа горимд багийн чат хаалттай: %d", code)
	}
	call(t, srv, "POST", base+"/state", tt, `{"mode":"team"}`)
	drain()

	// Баг доторх чат: гишүүд бичнэ, өөр багийнх бичиж/уншиж чадахгүй.
	if code, r := call(t, srv, "POST", base+"/chat", s1, `{"ch":"t:0","body":"  Сайн уу, багийнхан  "}`); code != 200 || r["msgs"].([]any)[0].(map[string]any)["body"] != "Сайн уу, багийнхан" {
		t.Fatalf("багийн чат: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", base+"/chat", s3, `{"ch":"t:0","body":"орж болох уу"}`); code != 403 {
		t.Fatalf("өөр багийн чатад бичих ёсгүй: %d", code)
	}
	for _, bad := range []string{`{"ch":"t:0","body":""}`, `{"ch":"t:00","body":"x"}`, `{"ch":"x:0","body":"x"}`, `{"ch":"all","body":"x"}`, `{"ch":"t:0","body":"` + strings.Repeat("ю", 1001) + `"}`} {
		if code, _ := call(t, srv, "POST", base+"/chat", s1, bad); code != 400 && code != 403 {
			t.Fatalf("буруу хүсэлт %s: %d", bad, code)
		}
	}
	call(t, srv, "POST", base+"/chat", s1, `{"ch":"b:0","body":"Багшаа, асуулт байна"}`)
	if evs := drain(); len(evs) != 2 || evs[0].Room != "t:0" || evs[1].Room != "b:0" || evs[0].Msg.UID != u1 {
		t.Fatalf("s2 багийнхаа хоёр мессежийг шууд авах ёстой: %+v", evs)
	}
	_, g := call(t, srv, "GET", base+"/chat", s3, "")
	if ch := g["chats"].(map[string]any); g["team"].(float64) != 1 || len(ch) != 2 || len(ch["t:1"].([]any)) != 0 || ch["t:0"] != nil {
		t.Fatalf("s3 зөвхөн өөрийн багийн (хоосон) сувгийг харна: %v", g)
	}
	_, g = call(t, srv, "GET", base+"/chat", tt, "")
	if ch := g["chats"].(map[string]any); len(ch["t:0"].([]any)) != 1 || len(ch["b:0"].([]any)) != 1 {
		t.Fatalf("багш бүх багийн чатыг харна: %v", g)
	}

	// Багш бүх багт нэг дор; суралцагч устгаж чадахгүй, багш устгана.
	if code, r := call(t, srv, "POST", base+"/chat", tt, `{"ch":"all","body":"5 минут үлдлээ"}`); code != 200 || len(r["msgs"].([]any)) != 2 {
		t.Fatalf("бүх багт: %d %v", code, r)
	}
	_, g = call(t, srv, "GET", base+"/chat", s3, "")
	if b1 := g["chats"].(map[string]any)["b:1"].([]any); len(b1) != 1 || b1[0].(map[string]any)["t"] != true {
		t.Fatalf("s3 багшийн мессежийг авна: %v", g)
	}
	_, g = call(t, srv, "GET", base+"/chat", s2, "")
	mid := g["chats"].(map[string]any)["t:0"].([]any)[0].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", base+"/chat", s2, `{"ch":"t:0","del":"`+mid+`"}`); code != 403 {
		t.Fatalf("суралцагч устгах ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/chat", tt, `{"ch":"t:0","del":"`+mid+`"}`); code != 200 {
		t.Fatalf("багш устгах: %d", code)
	}
	if _, g = call(t, srv, "GET", base+"/chat", s2, ""); len(g["chats"].(map[string]any)["t:0"].([]any)) != 0 {
		t.Fatalf("устгасан мессеж харагдах ёсгүй: %v", g)
	}
	call(t, srv, "POST", base+"/state", tt, `{"lock":true}`)
	if code, _ := call(t, srv, "POST", base+"/chat", s1, `{"ch":"t:0","body":"x"}`); code != 403 {
		t.Fatalf("түр зогсоосон үед чатлах ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/voice", s1, `{"to":"t:0","seq":1,"d":"`+voiceChunk()+`"}`); code != 403 {
		t.Fatalf("түр зогсоосон үед багтаа ярих ёсгүй: %d", code)
	}
	call(t, srv, "POST", base+"/state", tt, `{"lock":false}`)
	drain()

	// Дуут өрөө: багийнхан хоорондоо; бүх ангид зөвхөн багш эсвэл «ярих эрх» авсан нь.
	for i := range 6 {
		if code, _ := call(t, srv, "POST", base+"/voice", s1, `{"to":"t:0","seq":`+string(rune('1'+i))+`,"d":"`+voiceChunk()+`"}`); code != 204 {
			t.Fatalf("багтаа ярих: %d", code)
		}
	}
	if code, _ := call(t, srv, "POST", base+"/voice", s3, `{"to":"t:0","seq":1,"d":"`+voiceChunk()+`"}`); code != 403 {
		t.Fatalf("өөр багт ярих ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/voice", s3, `{"to":"all","seq":1,"d":"`+voiceChunk()+`"}`); code != 403 {
		t.Fatalf("эрхгүй бол бүх ангид ярих ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/voice", s1, `{"to":"t:0","seq":1,"d":"%%%"}`); code != 400 {
		t.Fatalf("буруу base64: %d", code)
	}
	if evs := drain(); len(evs) != 6 || evs[0].Voice.UID != u1 || evs[0].Voice.To != "t:0" {
		t.Fatalf("s2 багийнхаа дууг авах ёстой: %d", len(evs))
	}
	// Нөөц зам зөвхөн «only» сонсогчид; өөр багийнхныг нэмсэн ч очихгүй.
	call(t, srv, "POST", base+"/voice", s1, `{"to":"t:0","seq":9,"d":"`+voiceChunk()+`","only":["`+u3+`"]}`)
	if evs := drain(); len(evs) != 0 {
		t.Fatalf("only-д ороогүй s2-д очих ёсгүй: %+v", evs)
	}
	// WebRTC дохио: багийнхан хоорондоо, багш бүгдтэй; өөр багийнхантай холбогдохгүй.
	if code, _ := call(t, srv, "POST", base+"/voice", s1, `{"sig":{"to":"`+u2+`","t":"offer","id":"lnk001","sdp":"v=0"}}`); code != 200 {
		t.Fatalf("багийнхандаа offer: %d", code)
	}
	if evs := drain(); len(evs) != 1 || evs[0].Rtc == nil || evs[0].Rtc.From != u1 || evs[0].Rtc.T != "offer" || evs[0].Rtc.SDP != "v=0" {
		t.Fatalf("s2 offer авах ёстой: %+v", evs)
	}
	if code, _ := call(t, srv, "POST", base+"/voice", s3, `{"sig":{"to":"`+u2+`","t":"offer","id":"lnk002","sdp":"v=0"}}`); code != 403 {
		t.Fatalf("өөр багийнхантай холбогдох ёсгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/voice", tt, `{"sig":{"to":"`+u3+`","t":"offer","id":"lnk003","sdp":"v=0"}}`); code != 200 {
		t.Fatalf("багш дурын суралцагчтай: %d", code)
	}
	if code, _ := call(t, srv, "POST", base+"/voice", s1, `{"sig":{"to":"`+u2+`","t":"hack","id":"lnk001","sdp":"v=0"}}`); code != 400 {
		t.Fatalf("буруу дохио: %d", code)
	}
	call(t, srv, "POST", base+"/voice", s1, `{"talk":1500}`)
	if code, _ := call(t, srv, "POST", base+"/voice", s1, `{"talk":60000}`); code != 200 { // өнгөрсөн хугацаанаас их бол хасагдана
		t.Fatalf("talk: %d", code)
	}
	call(t, srv, "POST", base+"/voice", s3, `{"hand":true}`)
	_, g = call(t, srv, "GET", base, tt, "")
	if hands := g["state"].(map[string]any)["hands"].([]any); len(hands) != 1 || hands[0] != u3 {
		t.Fatalf("гар өргөсөн: %v", g["state"])
	}
	call(t, srv, "POST", base+"/state", tt, `{"stage":{"`+u3+`":true}}`)
	_, g = call(t, srv, "GET", base, tt, "")
	if stt := g["state"].(map[string]any); stt["hands"] != nil || len(stt["stage"].([]any)) != 1 {
		t.Fatalf("ярих эрх өгөхөд гар буух ёстой: %v", stt)
	}
	if code, _ := call(t, srv, "POST", base+"/voice", s3, `{"to":"all","seq":1,"d":"`+voiceChunk()+`"}`); code != 204 {
		t.Fatalf("ярих эрхтэй бол бүх ангид: %d", code)
	}
	if evs := drain(); len(evs) != 1 || evs[0].Voice.UID != u3 || evs[0].Voice.To != "all" {
		t.Fatalf("тайзны дууг бүгд сонсоно: %+v", evs)
	}
	if code, _ := call(t, srv, "POST", base+"/voice", tt, `{"to":"t:1","seq":1,"d":"`+voiceChunk()+`"}`); code != 204 {
		t.Fatalf("багш нэг багт хандаж ярих: %d", code)
	}
	if evs := drain(); len(evs) != 0 {
		t.Fatalf("1-р багт хандсан дуу 0-р багийн s2-д очих ёсгүй: %+v", evs)
	}
	call(t, srv, "POST", base+"/voice", s1, `{"mic":false}`)
	_, g = call(t, srv, "GET", base, tt, "")
	for _, p := range g["peers"].([]any) {
		if pm := p.(map[string]any); pm["uid"] == u1 && pm["mic"] == true {
			t.Fatalf("микрофон унтраасан: %v", pm)
		}
	}

	// Засвар → хэмжүүр; хадгалаад статистикт.
	call(t, srv, "POST", base+"/ops", s1, `{"room":"t:0","cid":"s1aaa","ops":[{"t":"add","o":{"id":"team01","type":"ellipse"}},{"t":"add","o":{"id":"team02","type":"rect"}}]}`)
	s.flushBoards(context.Background(), true)
	_, a := call(t, srv, "GET", "/api/me/analytics?days=7", tt, "")
	data := a["data"].(map[string]any)
	var x1, x3 map[string]any
	for _, v := range data["students"].([]any) {
		switch m := v.(map[string]any); m["user_id"] {
		case u1:
			x1 = m
		case u3:
			x3 = m
		}
	}
	if x1 == nil || x1["board_msgs"].(float64) != 2 || x1["board_ops"].(float64) != 2 || x1["board_voice_sec"].(float64) != 1 ||
		x1["board_active_sec"].(float64) < 5 || x1["board_team"].(float64) != 0 || x1["collab_score"].(float64) <= 0 {
		t.Fatalf("s1-ийн хэмжүүр: %v", x1)
	}
	if x3 == nil || x3["board_msgs"].(float64) != 0 || x3["board_team"].(float64) != 1 {
		t.Fatalf("s3-ийн хэмжүүр: %v", x3)
	}
	if tot := data["totals"].(map[string]any); tot["board_msgs"].(float64) != 2 || tot["board_students"].(float64) < 2 {
		t.Fatalf("нийт: %v", tot)
	}
	_ = s3
}

// voiceChunk — 200мс-ийн (3200 дээж) ADPCM хэсэг: 3 байт толгой + 1600 байт.
func voiceChunk() string { return base64.StdEncoding.EncodeToString(make([]byte, 1603)) }

// Сонсогчид: тайз — бүгд; баг — гишүүд, мөн тэр багийн самбарыг харж буй багш.
func TestBoardVoiceListeners(t *testing.T) {
	now := time.Now()
	bs := &boardSess{state: boardState{Mode: "team", Teams: 2, Assign: map[string]int{"a": 0, "b": 0, "c": 1}}, peers: map[string]*boardPeer{
		"a": {UID: "a", Room: "t:0", at: now}, "b": {UID: "b", Room: "main", at: now}, "c": {UID: "c", Room: "t:1", at: now},
		"old": {UID: "old", Room: "t:0", at: now.Add(-time.Hour)}, "T": {UID: "T", Room: "t:1", Teacher: true, at: now}}}
	got := func(sender, to string) []string {
		ids := bs.voiceListeners(sender, to, now)
		slices.Sort(ids)
		return ids
	}
	if ids := got("a", "t:0"); !slices.Equal(ids, []string{"b"}) {
		t.Fatalf("0-р баг: %v", ids)
	}
	if ids := got("c", "t:1"); !slices.Equal(ids, []string{"T"}) {
		t.Fatalf("1-р багийг харж буй багш сонсоно: %v", ids)
	}
	if ids := got("T", "all"); !slices.Equal(ids, []string{"a", "b", "c"}) {
		t.Fatalf("тайз: %v", ids)
	}
	if !bs.canSpeak("a", false, "t:0") || bs.canSpeak("a", false, "t:1") || bs.canSpeak("a", false, "all") || !bs.canSpeak("T", true, "all") || bs.canSpeak("T", true, "b:0") {
		t.Fatal("ярих эрх буруу")
	}
}

// wsWait — хэрэглэгчийн WebSocket-ийг нээгээд, typ төрлийн дараагийн үйл явдлыг хүлээх функц буцаана.
func wsWait(t *testing.T, srv *httptest.Server, token string) func(typ string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/api/chat/ws?token="+token, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close(websocket.StatusNormalClosure, ""); cancel() })
	time.Sleep(50 * time.Millisecond) // бүртгэл (Subscribe) дуусах
	return func(typ string) map[string]any {
		t.Helper()
		for {
			_, b, err := conn.Read(ctx)
			if err != nil {
				t.Fatalf("WebSocket-оор %q ирсэнгүй: %v", typ, err)
			}
			var m map[string]any
			if json.Unmarshal(b, &m) == nil && m["type"] == typ {
				return m
			}
		}
	}
}

// Дуу, чатын хүсэлт харж буй самбарыг солихгүй: багийн самбар (t:0) дээр ярьж/бичиж буй багш
// «main» руу шилжиж, бүх хүнд оролцогчдын жагсаалт дахин очиж, тэр багийн сонсогчдоос гарах ёсгүй.
func TestBoardVoiceKeepsRoom(t *testing.T) {
	st := newTestStore(t)
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	hub := chat.NewHub()
	s := New(Config{TrustProxy: true}, st, auth.NewSigner("test", time.Hour), hub, fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	tt, tid := register(t, srv, "krteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Баг","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"brd1","type":"board","text":""}]}`)
	base := "/api/courses/" + cid + "/lessons/" + l["id"].(string) + "/board/brd1"
	s1, u1 := register(t, srv, "krstud1", "student")
	for _, tok := range []string{tt, s1} {
		call(t, srv, "GET", base, tok, "")
	}
	call(t, srv, "POST", base+"/state", tt, `{"mode":"team","teams":2,"assign":{"`+u1+`":0}}`)
	call(t, srv, "POST", base+"/presence", tt, `{"room":"t:0"}`)
	ws := chat.NewClient()
	hub.Subscribe(ws, chat.UserKey(u1))
	defer hub.Unsubscribe(ws, chat.UserKey(u1))
	peersEv := func() (n int) {
		for {
			select {
			case b := <-ws.Send:
				var e boardEvent
				if json.Unmarshal(b, &e) == nil && e.Ev == "peers" {
					n++
				}
			default:
				return
			}
		}
	}
	peersEv()
	call(t, srv, "POST", base+"/voice", tt, `{"mic":true}`)
	for seq := 1; seq <= 5; seq++ {
		call(t, srv, "POST", base+"/voice", tt, `{"to":"t:0","seq":`+strconv.Itoa(seq)+`,"d":"`+voiceChunk()+`"}`)
	}
	call(t, srv, "POST", base+"/voice", tt, `{"talk":800}`)
	call(t, srv, "POST", base+"/chat", tt, `{"ch":"b:0","body":"сайн уу"}`)
	if n := peersEv(); n != 1 { // зөвхөн микрофон асаасан нь (Mic тэмдэг)
		t.Fatalf("өрөө солигдож оролцогчдын жагсаалт давтан илгээгдэв: %d", n)
	}
	s.boards.mu.Lock()
	bs := s.boards.m[l["id"].(string)+"/brd1"]
	s.boards.mu.Unlock()
	bs.mu.Lock()
	room := bs.peers[tid].Room
	ls := bs.voiceListeners(u1, "t:0", time.Now())
	bs.mu.Unlock()
	if room != "t:0" || !slices.Contains(ls, tid) {
		t.Fatalf("багш багийн самбар дээрээ үлдэж, багийн дууг сонсох ёстой: %q %v", room, ls)
	}
}
