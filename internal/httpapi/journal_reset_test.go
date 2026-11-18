package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"surgalt/internal/store"
)

// B11: төлсөн хоцролт, оролцооны эрхтэй суралцагчийн өгөгдлийг устгасны дараа журналд «эхлээгүй» (recent, last, avg,
// оноо, үзсэн нүдгүй), сургалтын явц 0 — гэхдээ эрх ажилласаар (need_pay false, шалгалт эхэлнэ, нүүр хуудсанд «нээлттэй»).
func TestResetPassesKeepNotStarted(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "rpn_t", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Төлбөртэй эрх","price":0,"published":true}`)
	cid := c["id"].(string)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	_, l1 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Хоцорсон даалгавар","is_free":true,"assignment":{"resubmit":true,"due":{"at":"`+past+`","late":"paid","late_fee":2000}}}`)
	_, l2 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Төлбөртэй шалгалт","is_free":true,"blocks":[{"id":"q001","type":"quiz","quiz":{"question":"?","options":["a","b"],"correct":[0]}}],"exam":{"pass_pct":50,"due":{"fee":1500}}}`)
	lLate, lFee := l1["id"].(string), l2["id"].(string)
	s1, uid := register(t, srv, "rpn_s", "student")
	if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, ""); code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("элсэлт: %d %v", code, r)
	}
	base := func(lid string) string { return "/api/courses/" + cid + "/lessons/" + lid }
	for _, lid := range []string{lLate, lFee} {
		code, o := call(t, srv, "POST", base(lid)+"/late-pay", s1, "")
		ord, _ := o["order"].(map[string]any)
		if code != http.StatusCreated || ord == nil {
			t.Fatalf("захиалга: %d %v", code, o)
		}
		if code, r := call(t, srv, "POST", "/api/orders/"+ord["id"].(string)+"/dev-pay", s1, ""); code != 200 {
			t.Fatalf("төлбөр: %d %v", code, r)
		}
	}
	if code, r := call(t, srv, "POST", base(lLate)+"/submit", s1, `{"text":"Хоцорсон ч илгээлээ"}`); code != 200 {
		t.Fatalf("даалгавар: %d %v", code, r)
	}
	if code, r := call(t, srv, "PUT", "/api/me/courses/"+cid+"/journal/students/"+uid, tt, `{"archived":true}`); code != 200 {
		t.Fatalf("архивлах: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", "/api/me/courses/"+cid+"/journal/students/"+uid+"/reset", tt, `{"confirm":true}`); code != 200 {
		t.Fatalf("устгах: %d %v", code, r)
	}

	_, j := call(t, srv, "GET", "/api/me/courses/"+cid+"/journal", tt, "")
	var row map[string]any
	for _, x := range j["students"].([]any) {
		if m := x.(map[string]any); m["user_id"] == uid {
			row = m
		}
	}
	rank, _ := row["rank"].(map[string]any)
	if row == nil || row["recent"] == true || row["last"] != nil || row["avg"].(float64) != 0 || row["done"].(float64) != 0 || rank["points"].(float64) != 0 {
		t.Fatalf("журналд «эхлээгүй» байх ёстой: %v", row)
	}
	if cells, _ := j["cells"].(map[string]any); cells[uid] != nil {
		t.Fatalf("үзсэн нүд байх ёсгүй: %v", cells[uid])
	}
	_, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, "")
	if a["done"].(float64) != 0 {
		t.Fatalf("сургалтын явц 0: %v", a["done"])
	}
	// Эрх ажилласаар.
	_, info := call(t, srv, "GET", base(lLate)+"/assignment", s1, "")
	if due, _ := info["due"].(map[string]any); due == nil || due["late"] != true || due["need_pay"] == true {
		t.Fatalf("төлсөн хоцролтын эрх: %v", info["due"])
	}
	if code, r := call(t, srv, "POST", base(lLate)+"/submit", s1, `{"text":"Дахин илгээв"}`); code != 200 {
		t.Fatalf("төлсөн хоцролттой даалгавар илгээх: %d %v", code, r)
	}
	_, home := call(t, srv, "GET", "/api/me/home", s1, "")
	for _, x := range home["tasks"].([]any) {
		if tk := x.(map[string]any); tk["status"] == "need_pay" {
			t.Fatalf("нүүр хуудсанд төлбөр нэхэх ёсгүй: %v", tk)
		}
	}
	if code, r := call(t, srv, "POST", base(lFee)+"/exam/start", s1, ""); code != 200 && code != 201 {
		t.Fatalf("төлсөн оролцооны эрхээр шалгалт эхлэх: %d %v", code, r)
	}
}

// R2: санах ойд ачаалаагүй самбарын хадгалсан оролцооны хэмжүүрээс ("_m") суралцагч хасагдана (бусдынх хэвээр) —
// устгах өөр өгөгдөлгүй (Wiped=false) үед ч; багшийн хяналтад харагдахгүй.
func TestResetScrubsBoardMetrics(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	tt, _ := register(t, srv, "rbm_t", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Самбар","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"brd1","type":"board","text":""}]}`)
	lid := l["id"].(string)
	s1, uid := register(t, srv, "rbm_s", "student")
	s2, uid2 := register(t, srv, "rbm_s2", "student")
	for _, tok := range []string{s1, s2} {
		call(t, srv, "POST", "/api/courses/"+cid+"/enroll", tok, "")
	}
	day := time.Now().Format("2006-01-02")
	met := `{"u":{"` + uid + `":{"n":"А","t":-1,"d":{"` + day + `":[60,30,5,2,0]}},"` + uid2 + `":{"n":"Б","t":-1,"d":{"` + day + `":[90,40,7,1,0]}}}}`
	if err := st.SaveBoardRooms(ctx, []store.BoardRoomData{{LessonID: lid, BlockID: "brd1", Room: boardMetricsRoom, Data: []byte(met)}}); err != nil {
		t.Fatal(err)
	}
	if code, r := call(t, srv, "PUT", "/api/me/courses/"+cid+"/journal/students/"+uid, tt, `{"archived":true}`); code != 200 {
		t.Fatalf("архивлах: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", "/api/me/courses/"+cid+"/journal/students/"+uid+"/reset", tt, `{"confirm":true}`); code != 200 {
		t.Fatalf("устгах: %d %v", code, r)
	}
	rows, err := st.BoardRoomsNamed(ctx, []string{lid}, boardMetricsRoom)
	if err != nil || len(rows) != 1 {
		t.Fatalf("хэмжүүр: %v %v", rows, err)
	}
	var m boardMetrics
	if err := json.Unmarshal(rows[0].Data, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.U[uid]; ok {
		t.Fatalf("устгасан суралцагчийн самбарын хэмжүүр үлдлээ: %s", rows[0].Data)
	}
	if u := m.U[uid2]; u == nil || u.Days[day][0] != 90 {
		t.Fatalf("бусдын хэмжүүр хөндөгдлөө: %s", rows[0].Data)
	}
	if _, n := call(t, srv, "GET", "/api/me/notifications", s1, ""); len(n["items"].([]any)) != 0 {
		t.Fatalf("устгах өгөгдөлгүй үед мэдэгдэх ёсгүй: %v", n["items"])
	}
}

// B5, B2(c), B2(d): багшийг устгахгүй (400, хязгаарт тооцогдохгүй); нэг суралцагчийг 10 минутын дотор дахин устгахгүй
// (429); устгах өгөгдөлгүй суралцагчийн мөр шинэчлэгдэх боловч мэдэгдэл очихгүй.
func TestResetTargetCooldownNoData(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, tid := register(t, srv, "rtc_t", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Хүлээлт","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true}`)
	lid := l["id"].(string)
	s1, uid := register(t, srv, "rtc_s", "student")
	s2, uid2 := register(t, srv, "rtc_s2", "student")
	for _, tok := range []string{s1, s2} {
		call(t, srv, "POST", "/api/courses/"+cid+"/enroll", tok, "")
	}
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/complete", s1, "")
	path := func(u string) string { return "/api/me/courses/" + cid + "/journal/students/" + u }
	for range 4 {
		if code, r := call(t, srv, "POST", path(tid)+"/reset", tt, `{"confirm":true}`); code != 400 || r["error"] != "сургалтын багшийг журналд засах боломжгүй" {
			t.Fatalf("багшийг устгах: %d %v", code, r)
		}
	}
	for _, u := range []string{uid, uid2} {
		if code, r := call(t, srv, "PUT", path(u), tt, `{"archived":true,"alias":"Нэр"}`); code != 200 {
			t.Fatalf("архивлах: %d %v", code, r)
		}
	}
	if code, r := call(t, srv, "POST", path(uid)+"/reset", tt, `{"confirm":true}`); code != 200 {
		t.Fatalf("устгах: %d %v", code, r)
	}
	call(t, srv, "PUT", path(uid), tt, `{"archived":true}`)
	if code, r := call(t, srv, "POST", path(uid)+"/reset", tt, `{"confirm":true}`); code != http.StatusTooManyRequests ||
		r["error"] != "энэ суралцагчийн өгөгдлийг саяхан устгасан — 10 минутын дараа дахин оролдоно уу" {
		t.Fatalf("саяхан устгасан: %d %v", code, r)
	}
	// Өгөгдөлгүй (зөвхөн элссэн) суралцагч: 200, архив/нэр арилна, мэдэгдэлгүй.
	if code, r := call(t, srv, "POST", path(uid2)+"/reset", tt, `{"confirm":true}`); code != 200 || r["reset_at"] == nil {
		t.Fatalf("өгөгдөлгүй суралцагч: %d %v", code, r)
	}
	_, n := call(t, srv, "GET", "/api/me/notifications", s2, "")
	for _, x := range n["items"].([]any) {
		if x.(map[string]any)["type"] == "course_reset" {
			t.Fatalf("устгах юмгүй үед мэдэгдэх ёсгүй: %v", x)
		}
	}
	_, j := call(t, srv, "GET", "/api/me/courses/"+cid+"/journal", tt, "")
	for _, x := range j["students"].([]any) {
		if m := x.(map[string]any); m["user_id"] == uid2 && (m["archived"] == true || m["alias"] != nil) {
			t.Fatalf("өгөгдөлгүй суралцагчийн мөр шинэчлэгдээгүй: %v", m)
		}
	}
}
