package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/store"
)

// arNotifs — хэрэглэгчийн тухайн төрлийн мэдэгдлүүд (шинэ нь эхэндээ).
func arNotifs(t *testing.T, srv *httptest.Server, tok, typ string) []map[string]any {
	t.Helper()
	code, items := callList(t, srv, "GET", "/api/me/notifications", tok)
	if code != 200 {
		t.Fatalf("мэдэгдэл: %d", code)
	}
	var got []map[string]any
	for _, x := range items {
		if n := x.(map[string]any); n["type"] == typ {
			got = append(got, n)
		}
	}
	return got
}

// Нууц үгтэй сургалтын нээлгэх хүсэлт: багш, хамтран заагч хоёуланд мэдэгдэл (grant: холбоос), 12 цагт нэг удаа;
// өөр багш, суралцагч өөрөө нээж чадахгүй (404); багш нээвэл нууц үг оруулсантай адил бүх хичээл нээгдэж, элсэнэ.
func TestAccessRequestCourse(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "ar_owner", "teacher")
	tc, _ := register(t, srv, "ar_co", "teacher")
	tx, _ := register(t, srv, "ar_other", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"10А анги","price":0,"published":true,"access_code":"Анги-10А"}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true,"blocks":[{"id":"t1","type":"text","text":"А"}]}`)
	if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/share", tt, `{"who":"@ar_co","mode":"edit"}`); code != http.StatusCreated {
		t.Fatalf("хамтран заагч: %d %v", code, r)
	}
	s1, u1 := register(t, srv, "ar_s1", "student")
	s2, u2 := register(t, srv, "ar_s2", "student")
	lesson := func(tok string) int {
		code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, tok, "")
		return code
	}
	if lesson(s1) == 200 {
		t.Fatal("нууц үггүй хичээл нээгдэх ёсгүй")
	}
	req := func(tok, body string) (int, map[string]any) {
		return call(t, srv, "POST", "/api/courses/"+cid+"/access-request", tok, body)
	}
	if code, _ := req("", `{"lesson_id":""}`); code != 401 {
		t.Fatalf("нэвтрээгүй: %d", code)
	}
	if code, r := req(s1, `{"lesson_id":"","message":"`+strings.Repeat("я", 301)+`"}`); code != 400 {
		t.Fatalf("301 тэмдэгтийн мессеж: %d %v", code, r)
	}
	code, r := req(s1, `{"lesson_id":"","message":"  Нууц үгээ мартчихлаа  "}`)
	if code != 200 || r["sent"] != true {
		t.Fatalf("хүсэлт: %d %v", code, r)
	}
	link := "grant:" + u1 + ":" + cid + ":"
	for name, tok := range map[string]string{"эзэмшигч": tt, "хамтран заагч": tc} {
		ns := arNotifs(t, srv, tok, "access_request")
		if len(ns) != 1 || ns[0]["link"] != link || ns[0]["body"] != "Нууц үгээ мартчихлаа" || ns[0]["actor_id"] != u1 ||
			!strings.HasPrefix(ns[0]["title"].(string), "🔑 ") || !strings.Contains(ns[0]["title"].(string), "«10А анги» сургалтыг нээх хүсэлт") {
			t.Fatalf("%s: мэдэгдэл %v", name, ns)
		}
	}
	if ns := arNotifs(t, srv, tx, "access_request"); len(ns) != 0 {
		t.Fatalf("өөр багшид мэдэгдэл очих ёсгүй: %v", ns)
	}
	// 12 цагт нэг удаа: давхар бол мэдэгдэлгүй already.
	if code, r := req(s1, `{"lesson_id":""}`); code != 200 || r["sent"] != false || r["already"] != true {
		t.Fatalf("давхар хүсэлт: %d %v", code, r)
	}
	if ns := arNotifs(t, srv, tt, "access_request"); len(ns) != 1 {
		t.Fatalf("давхар хүсэлт мэдэгдэл үүсгэх ёсгүй: %d", len(ns))
	}
	// Багш өөрөө хүсэлт илгээх шаардлагагүй.
	if code, r := req(tc, `{"lesson_id":""}`); code != 200 || r["open"] != true {
		t.Fatalf("багшийн хүсэлт: %d %v", code, r)
	}

	grant := func(tok, uid, body string) (int, map[string]any) {
		return call(t, srv, "POST", "/api/me/students/"+uid+"/grant", tok, body)
	}
	gb := `{"course_id":"` + cid + `","lesson_id":""}`
	const nf = "мэдээлэл олдсонгүй — устгагдсан байж магадгүй"
	for name, tok := range map[string]string{"өөр багш": tx, "суралцагч өөрөө": s1} {
		if code, r := grant(tok, u1, gb); code != 404 || r["error"] != nf {
			t.Fatalf("%s нээх: %d %v", name, code, r)
		}
	}
	if code, _ := grant(tt, u1, `{"course_id":"nope","lesson_id":""}`); code != 404 {
		t.Fatalf("байхгүй сургалт: %d", code)
	}
	// Хүсэлтгүй хэрэглэгч ба байхгүй хэрэглэгч — ижилхэн 404 (хэрэглэгч байгаа эсэхийг илрүүлэхгүй), элсүүлэхгүй.
	codeNoReq, noReq := grant(tt, u2, gb)
	codeGhost, ghost := grant(tt, strings.Repeat("0", 24), gb)
	if codeNoReq != 404 || codeGhost != 404 || noReq["error"] != nf || ghost["error"] != nf || len(noReq) != len(ghost) {
		t.Fatalf("хүсэлтгүй: %d %v; байхгүй хэрэглэгч: %d %v", codeNoReq, noReq, codeGhost, ghost)
	}
	if lesson(s2) == 200 || len(arNotifs(t, srv, s2, "access_granted")) != 0 {
		t.Fatal("хүсэлтгүй хэрэглэгчийг нээж, мэдэгдэх ёсгүй")
	}
	if _, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", s2, ""); a["enrolled"] == true {
		t.Fatal("хүсэлтгүй хэрэглэгчийг элсүүлэх ёсгүй")
	}
	if lesson(s1) == 200 {
		t.Fatal("татгалзсан нээлтийн дараа ч түгжээтэй байх ёстой")
	}
	if code, r := grant(tt, u1, gb); code != 200 || r["granted"] != true {
		t.Fatalf("нээх: %d %v", code, r)
	}
	if lesson(s1) != 200 {
		t.Fatal("багш нээсний дараа хичээл үзэгдэх ёстой")
	}
	if _, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, ""); a["all"] != true || a["need_code"] != false || a["enrolled"] != true {
		t.Fatalf("access: all=%v need_code=%v enrolled=%v", a["all"], a["need_code"], a["enrolled"])
	}
	ns := arNotifs(t, srv, s1, "access_granted")
	if len(ns) != 1 || ns[0]["link"] != "/c/"+cid || !strings.HasPrefix(ns[0]["title"].(string), "🔓 Багш «10А анги»") {
		t.Fatalf("суралцагчийн мэдэгдэл: %v", ns)
	}
	// Аль хэдийн нээлттэй: дахин мэдэгдэхгүй; хүсэлт ч open.
	if code, r := grant(tt, u1, gb); code != 200 || r["granted"] != false || r["open"] != true {
		t.Fatalf("давтан нээх: %d %v", code, r)
	}
	if ns := arNotifs(t, srv, s1, "access_granted"); len(ns) != 1 {
		t.Fatalf("давтан нээлт мэдэгдэл үүсгэх ёсгүй: %d", len(ns))
	}
	if code, r := req(s1, `{"lesson_id":""}`); code != 200 || r["open"] != true {
		t.Fatalf("нээлттэй сургалтын хүсэлт: %d %v", code, r)
	}
	// Хамтран заагч ч бодит хүсэлтийн хариуд нээнэ.
	if code, r := req(s2, `{"lesson_id":""}`); code != 200 || r["sent"] != true {
		t.Fatalf("s2 хүсэлт: %d %v", code, r)
	}
	if code, r := grant(tc, u2, gb); code != 200 || r["granted"] != true {
		t.Fatalf("хамтран заагч нээх: %d %v", code, r)
	}
	if lesson(s2) != 200 {
		t.Fatal("хамтран заагч нээсний дараа s2 үзэх ёстой")
	}

	// Багш нууц үгээ сольсон: өмнө нь нууц үгээр нээсэн (хүсэлтгүй) хүнийг дахин нээнэ.
	s3, u3 := register(t, srv, "ar_s3", "student")
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/unlock", s3, `{"code":"Анги-10А"}`); code != 201 {
		t.Fatalf("нууц үгээр нээх: %d", code)
	}
	call(t, srv, "PUT", "/api/courses/"+cid, tt, `{"title":"10А анги","price":0,"published":true,"access_code":"Шинэ-үг"}`)
	if lesson(s3) == 200 {
		t.Fatal("үг сольсны дараа түгжээтэй байх ёстой")
	}
	if code, r := grant(tt, u3, gb); code != 200 || r["granted"] != true || lesson(s3) != 200 {
		t.Fatalf("үг сольсны дараа дахин нээх: %d %v", code, r)
	}

	// Нууц үггүй сургалт: хүсэлт 400, хүсэлтгүй тул нээх нь 404.
	free := jmCourse(t, srv, tt, `{"title":"Нээлттэй","price":0,"published":true}`)
	if code, r := call(t, srv, "POST", "/api/courses/"+free+"/access-request", s1, `{"lesson_id":""}`); code != 400 || r["error"] != "энэ сургалтад нууц үг тавиагүй байна" {
		t.Fatalf("нууц үггүй сургалтын хүсэлт: %d %v", code, r)
	}
	if code, r := grant(tt, u1, `{"course_id":"`+free+`","lesson_id":""}`); code != 404 {
		t.Fatalf("нууц үггүй сургалтыг хүсэлтгүй нээх: %d %v", code, r)
	}
	// Нийтлээгүй нууц үгтэй сургалт: хүсэлт ба нээлт хоёулаа 404 (байхгүйтэй адил).
	draft := jmCourse(t, srv, tt, `{"title":"Ноорог","price":0,"published":false,"access_code":"Ноорог-1"}`)
	if code, r := call(t, srv, "POST", "/api/courses/"+draft+"/access-request", s1, `{"lesson_id":""}`); code != 404 || r["error"] != nf {
		t.Fatalf("нийтлээгүй сургалтын хүсэлт: %d %v", code, r)
	}
	if code, _ := grant(tt, u1, `{"course_id":"`+draft+`","lesson_id":""}`); code != 404 {
		t.Fatalf("нийтлээгүй сургалтыг нээх: %d", code)
	}
}

// «Багш гараар нээнэ» хичээлийн хүсэлт: багш нээвэл тэр суралцагчид л нээгдэнэ (423 → 200), бусдад түгжээтэй;
// буруу хичээл (manual биш, өөр сургалтынх), сургалтын эрхгүй суралцагч.
func TestAccessRequestManualLesson(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "arm_owner", "teacher")
	tx, _ := register(t, srv, "arm_other", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Дараалал","price":1000,"published":true,"drip":true}`)
	la := jmLesson(t, srv, tt, cid, `{"title":"Эхний"}`)
	lb := jmLesson(t, srv, tt, cid, `{"title":"Гарын","unlock_rule":"manual"}`)
	other := jmCourse(t, srv, tx, `{"title":"Өөр","price":0,"published":true,"drip":true}`)
	jmLesson(t, srv, tx, other, `{"title":"x","is_free":true}`)
	lo := jmLesson(t, srv, tx, other, `{"title":"Өөрийн гарын","is_free":true,"unlock_rule":"manual"}`)
	join := func(name string) (string, string) {
		tok, uid := register(t, srv, name, "student")
		_, e := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", tok, "")
		if code, _ := call(t, srv, "POST", "/api/orders/"+e["order"].(map[string]any)["id"].(string)+"/dev-pay", tok, ""); code != 200 {
			t.Fatal("төлбөр")
		}
		return tok, uid
	}
	s1, u1 := join("arm_s1")
	s2, u2 := join("arm_s2")
	s3, u3 := register(t, srv, "arm_s3", "student") // төлөөгүй
	lesson := func(tok, lid string) (int, string) {
		code, r := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, tok, "")
		st, _ := r["state"].(map[string]any)
		reason, _ := st["reason"].(string)
		return code, reason
	}
	if code, reason := lesson(s1, lb); code != http.StatusLocked || reason != "manual" {
		t.Fatalf("нээхээс өмнө: %d %s", code, reason)
	}
	req := func(tok, lid string) (int, map[string]any) {
		return call(t, srv, "POST", "/api/courses/"+cid+"/access-request", tok, `{"lesson_id":"`+lid+`"}`)
	}
	// Буруу хүсэлтүүд.
	if code, r := req(s1, la); code != 400 || r["error"] != "энэ хичээлийг багш гараар нээдэггүй" {
		t.Fatalf("manual биш хичээл: %d %v", code, r)
	}
	if code, r := req(s1, lo); code != 404 {
		t.Fatalf("өөр сургалтын хичээл: %d %v", code, r)
	}
	if code, r := req(s1, ""); code != 400 || r["error"] != "энэ сургалтад нууц үг тавиагүй байна" {
		t.Fatalf("түгжээгүй сургалтын хүсэлт: %d %v", code, r)
	}
	if code, r := req(s3, lb); code != 403 {
		t.Fatalf("сургалтын эрхгүй: %d %v", code, r)
	}
	if ns := arNotifs(t, srv, tt, "access_request"); len(ns) != 0 {
		t.Fatalf("буруу хүсэлт мэдэгдэл үүсгэх ёсгүй: %v", ns)
	}
	grant := func(tok, uid, lid string) (int, map[string]any) {
		return call(t, srv, "POST", "/api/me/students/"+uid+"/grant", tok, `{"course_id":"`+cid+`","lesson_id":"`+lid+`"}`)
	}
	// Сургалтын эрхгүй (s3), эрхтэй ч хүсэлтгүй (s1 хүсэлтээс өмнө, s2) → 404, нээгдэхгүй.
	for name, uid := range map[string]string{"эрхгүй": u3, "хүсэлтгүй s1": u1, "хүсэлтгүй s2": u2} {
		if code, r := grant(tt, uid, lb); code != 404 || r["error"] != "мэдээлэл олдсонгүй — устгагдсан байж магадгүй" {
			t.Fatalf("%s: %d %v", name, code, r)
		}
	}
	if code, reason := lesson(s1, lb); code != http.StatusLocked || reason != "manual" {
		t.Fatalf("хүсэлтгүй нээлтийн дараа: %d %s", code, reason)
	}
	if code, r := req(s1, lb); code != 200 || r["sent"] != true {
		t.Fatalf("хүсэлт: %d %v", code, r)
	}
	ns := arNotifs(t, srv, tt, "access_request")
	if len(ns) != 1 || ns[0]["link"] != "grant:"+u1+":"+cid+":"+lb || ns[0]["body"] != "Дараалал" ||
		!strings.HasPrefix(ns[0]["title"].(string), "🔐 ") || !strings.Contains(ns[0]["title"].(string), "«Гарын» хичээлийг нээх хүсэлт") {
		t.Fatalf("багшийн мэдэгдэл: %v", ns)
	}
	if code, r := req(s1, lb); code != 200 || r["already"] != true {
		t.Fatalf("давхар хүсэлт: %d %v", code, r)
	}
	if code, _ := grant(tx, u1, lb); code != 404 {
		t.Fatalf("өөр багш: %d", code)
	}
	if code, _ := grant(tt, u1, lo); code != 404 {
		t.Fatalf("өөр сургалтын хичээлийг нээх: %d", code)
	}
	if code, r := grant(tt, u1, la); code != 404 { // энэ хичээлд хүсэлт ирээгүй
		t.Fatalf("хүсэлтгүй өөр хичээлийг нээх: %d %v", code, r)
	}
	if code, reason := lesson(s1, lb); code != http.StatusLocked || reason != "manual" {
		t.Fatalf("татгалзсан нээлтийн дараа: %d %s", code, reason)
	}
	if code, r := grant(tt, u1, lb); code != 200 || r["granted"] != true {
		t.Fatalf("нээх: %d %v", code, r)
	}
	if code, _ := lesson(s1, lb); code != 200 {
		t.Fatalf("нээсний дараа s1: %d", code)
	}
	if code, reason := lesson(s2, lb); code != http.StatusLocked || reason != "manual" {
		t.Fatalf("s2 түгжээтэй хэвээр байх ёстой: %d %s", code, reason)
	}
	state := func(tok string) map[string]any {
		_, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", tok, "")
		return a["states"].(map[string]any)[lb].(map[string]any)
	}
	if st := state(s1); st["open"] != true {
		t.Fatalf("s1 access төлөв: %v", st)
	}
	if st := state(s2); st["open"] == true || st["reason"] != "manual" {
		t.Fatalf("s2 access төлөв: %v", st)
	}
	// Идэвхийн сесс (gRPC/HTTP-ийн LessonFor зам) ч нээлттэй.
	if code, r := call(t, srv, "POST", "/api/activity/start", s1, `{"course_id":"`+cid+`","lesson_id":"`+lb+`"}`); code != 200 {
		t.Fatalf("activity start: %d %v", code, r)
	}
	if code, r := req(s1, lb); code != 200 || r["open"] != true {
		t.Fatalf("нээлттэй хичээлийн хүсэлт: %d %v", code, r)
	}
	sn := arNotifs(t, srv, s1, "access_granted")
	if len(sn) != 1 || sn[0]["link"] != "/c/"+cid+"#l="+lb || !strings.Contains(sn[0]["title"].(string), "«Гарын» хичээлийг") {
		t.Fatalf("суралцагчийн мэдэгдэл: %v", sn)
	}
	if code, r := grant(tt, u1, lb); code != 200 || r["granted"] != false || r["open"] != true {
		t.Fatalf("давтан нээх: %d %v", code, r)
	}
	// Журналын «бүрмөсөн устгах» нь явцыг устгана, харин багшийн нээсэн эрх (төлсөн эрх шиг) хэвээр.
	jmPutOK(t, srv, "архивлах", tt, cid, u1, `{"archived":true}`)
	if code, r := jmReset(t, srv, tt, cid, u1, `{"confirm":true}`); code != 200 {
		t.Fatalf("устгах: %d %v", code, r)
	}
	if code, _ := lesson(s1, lb); code != 200 {
		t.Fatalf("устгалын дараа нээсэн эрх хэвээр байх ёстой: %d", code)
	}
}

// Хэрэглэгч бүрийн ерөнхий хязгаар: өөр өөр хичээлд олон хүсэлт → 429.
func TestAccessRequestRateLimit(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "arl_owner", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Олон","price":0,"published":true,"drip":true}`)
	jmLesson(t, srv, tt, cid, `{"title":"0","is_free":true}`)
	var lids []string
	for i := range 7 {
		lids = append(lids, jmLesson(t, srv, tt, cid, fmt.Sprintf(`{"title":"%d","is_free":true,"unlock_rule":"manual"}`, i+1)))
	}
	s1, _ := register(t, srv, "arl_s", "student")
	limited := false
	for _, lid := range lids {
		code, r := call(t, srv, "POST", "/api/courses/"+cid+"/access-request", s1, `{"lesson_id":"`+lid+`"}`)
		if code == http.StatusTooManyRequests {
			limited = true
			break
		}
		if code != 200 || r["sent"] != true {
			t.Fatalf("хүсэлт: %d %v", code, r)
		}
	}
	if !limited {
		t.Fatal("хязгаар ажиллах ёстой")
	}
}

func TestAccessDedupBounded(t *testing.T) {
	var d accessDedup
	now := time.Now()
	if !d.claim("a", now) || d.claim("a", now) || !d.sent("a", now) {
		t.Fatal("эхний claim л амжилттай")
	}
	d.release("a") // мэдэгдэл хүрээгүй → дахин илгээж болно
	if d.sent("a", now) || !d.claim("a", now) {
		t.Fatal("release-ийн дараа дахин claim")
	}
	if d.sent("a", now.Add(accessReqWindow+time.Second)) || !d.claim("a", now.Add(accessReqWindow+time.Second)) {
		t.Fatal("12 цагийн дараа дахин илгээж болно")
	}
	for i := range accessReqKeys + 10 {
		d.claim(fmt.Sprint("k", i), now)
	}
	if n := len(d.m); n > accessReqKeys {
		t.Fatalf("санах ой хязгааргүй өслөө: %d", n)
	}
}

// Flash card: асуулт ба санах хэсэг хоёулаа эсвэл аль нь ч биш; бүх төрөлд; /api/me/notes сүүлийн 60 + өмнөх картууд.
func TestHighlightFlashCard(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "fc_teach", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Биологи","price":0,"published":true}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Эс","is_free":true,"blocks":[{"id":"txt1","type":"text","text":"Эс бол амьдралын нэгж."}]}`)
	s1, _ := register(t, srv, "fc_s", "student")
	base := "/api/courses/" + cid + "/lessons/" + lid + "/highlights"
	for name, h := range map[string]string{
		"зөвхөн асуулт":      `"q":"Эс гэж юу вэ?"`,
		"зөвхөн санах хэсэг": `"a":"Амьдралын нэгж"`,
		"хоосон зайтай":      `"q":"Эс гэж юу вэ?","a":"   "`,
	} {
		code, r := call(t, srv, "PUT", base, s1, `{"highlights":[{"id":"h1","b":"txt1","s":0,"e":2,"c":"y","t":"Эс",`+h+`}]}`)
		if code != 400 || r["error"] != "flash card-ийн асуулт ба санах хэсгийг хоёуланг нь бөглөнө үү" {
			t.Fatalf("%s: %d %v", name, code, r)
		}
	}
	if code, _ := call(t, srv, "PUT", base, s1, `{"highlights":[{"id":"h1","b":"txt1","s":0,"e":2,"c":"y","t":"Эс","q":"`+strings.Repeat("я", 301)+`","a":"x"}]}`); code != 400 {
		t.Fatalf("301 тэмдэгтийн асуулт: %d", code)
	}
	if code, _ := call(t, srv, "PUT", base, s1, `{"highlights":[{"id":"h1","b":"txt1","s":0,"e":2,"c":"y","t":"Эс","q":"?","a":"`+strings.Repeat("я", 1001)+`"}]}`); code != 400 {
		t.Fatalf("1001 тэмдэгтийн санах хэсэг: %d", code)
	}
	ok := `{"highlights":[{"id":"h1","b":"txt1","s":0,"e":2,"c":"r","t":"Эс","q":"  Эс гэж юу вэ? ","a":" Амьдралын нэгж "},
		{"id":"v1","b":"vid1","s":0,"e":0,"c":"b","t":"0:05","k":"time","sec":5,"q":"Видеонд юу гэв?","a":"Эс хуваагдана"},
		{"id":"h2","b":"txt1","s":3,"e":6,"c":"g","t":"бол"}]}`
	code, r := call(t, srv, "PUT", base, s1, ok)
	if code != 200 {
		t.Fatalf("карттай тэмдэглэл: %d %v", code, r)
	}
	hs := r["highlights"].([]any)
	if h := hs[0].(map[string]any); h["q"] != "Эс гэж юу вэ?" || h["a"] != "Амьдралын нэгж" {
		t.Fatalf("тайрсан карт: %v", h)
	}
	if h := hs[2].(map[string]any); h["q"] != nil || h["a"] != nil {
		t.Fatalf("карт биш тэмдэглэлд q/a гарах ёсгүй: %v", h)
	}
	_, got := call(t, srv, "GET", base, s1, "")
	if h := got["highlights"].([]any)[1].(map[string]any); h["q"] != "Видеонд юу гэв?" || h["a"] != "Эс хуваагдана" {
		t.Fatalf("хадгалсан карт: %v", h)
	}

	// /api/me/notes: 70 тэмдэглэлээс эхний 5 нь карт → сүүлийн 60 + 5 карт (анхны дарааллаар).
	var items []string
	for i := range 70 {
		card := ""
		if i < 5 {
			card = fmt.Sprintf(`,"q":"Асуулт %d","a":"Хариу %d"`, i, i)
		}
		items = append(items, fmt.Sprintf(`{"id":"n%d","b":"txt1","s":0,"e":2,"c":"y","t":"Эс"%s}`, i, card))
	}
	if code, r := call(t, srv, "PUT", base, s1, `{"highlights":[`+strings.Join(items, ",")+`]}`); code != 200 {
		t.Fatalf("70 тэмдэглэл: %d %v", code, r)
	}
	_, mine := call(t, srv, "GET", "/api/me/notes", s1, "")
	notes := mine["stories"].([]any)[0].(map[string]any)["notes"].([]any)
	if len(notes) != 65 {
		t.Fatalf("story-ийн тэмдэглэл: %d", len(notes))
	}
	for i, want := range []string{"n0", "n1", "n2", "n3", "n4", "n10", "n11"} {
		if n := notes[i].(map[string]any); n["id"] != want {
			t.Fatalf("дараалал %d: %v != %s", i, n["id"], want)
		}
	}
	if n := notes[0].(map[string]any); n["q"] != "Асуулт 0" || n["a"] != "Хариу 0" {
		t.Fatalf("хуучин карт: %v", n)
	}
	if n := notes[64].(map[string]any); n["id"] != "n69" {
		t.Fatalf("сүүлийнх: %v", n)
	}
}

// Story-гийн flash card давталт → багшийн тэмдэглэлийн статистикт картын тоо, давтсан, санасан, давтсан суралцагч.
func TestStoryViewCards(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "svc_teach", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Биологи","price":0,"published":true}`)
	l1 := jmLesson(t, srv, tt, cid, `{"title":"Эс","is_free":true,"blocks":[{"id":"txt1","type":"text","text":"Эс бол амьдралын нэгж."}]}`)
	l2 := jmLesson(t, srv, tt, cid, `{"title":"Ген","is_free":true,"blocks":[{"id":"txt1","type":"text","text":"Ген бол удамшлын нэгж."}]}`)
	s1, u1 := register(t, srv, "svc_s1", "student")
	s2, _ := register(t, srv, "svc_s2", "student")
	if code, r := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+l1+"/highlights", s1, `{"highlights":[
		{"id":"h1","b":"txt1","s":0,"e":2,"c":"y","t":"Эс","q":"Эс гэж?","a":"Нэгж"},
		{"id":"h2","b":"txt1","s":3,"e":6,"c":"g","t":"бол","q":"Холбоос?","a":"бол"},
		{"id":"h3","b":"txt1","s":7,"e":12,"c":"r","t":"амьдр"}]}`); code != 200 {
		t.Fatalf("тэмдэглэл: %d %v", code, r)
	}
	sv := func(tok, body string) int {
		code, _ := call(t, srv, "POST", "/api/me/story-view", tok, body)
		return code
	}
	for name, cards := range map[string]string{
		"known > seen": `{"` + l1 + `":{"seen":1,"known":2}}`,
		"сөрөг":        `{"` + l1 + `":{"seen":1,"known":-1}}`,
		"500-аас их":   `{"` + l1 + `":{"seen":501,"known":0}}`,
	} {
		if code := sv(s1, `{"lessons":["`+l1+`"],"slides":3,"total":3,"cards":`+cards+`}`); code != 400 {
			t.Fatalf("%s: %d", name, code)
		}
	}
	// Тэмдэглэлгүй хичээл (s1-д l2) ба lessons-д байхгүй түлхүүрийг (хязгаараас хэтэрсэн ч) алгасна.
	if code := sv(s1, `{"lessons":["`+l1+`","`+l2+`"],"slides":5,"total":5,"review":true,"cards":{"`+l1+`":{"seen":3,"known":2},"`+l2+`":{"seen":999,"known":0},"ffffffffffffffffffffffff":{"seen":1,"known":9}}}`); code != 204 {
		t.Fatalf("story үзэлт: %d", code)
	}
	// Тэмдэглэлгүй хэрэглэгч дурын хичээлд үзэлт бичихгүй (204, юу ч бичигдэхгүй).
	s3, u3 := register(t, srv, "svc_s3", "student")
	if code := sv(s3, `{"lessons":["`+l1+`"],"slides":1,"total":1,"cards":{"`+l1+`":{"seen":5,"known":5}}}`); code != 204 {
		t.Fatalf("тэмдэглэлгүй story: %d", code)
	}
	if code, r := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+l2+"/highlights", s2, `{"highlights":[{"id":"g1","b":"txt1","s":0,"e":3,"c":"b","t":"Ген"}]}`); code != 200 {
		t.Fatalf("s2 тэмдэглэл: %d %v", code, r)
	}
	if code := sv(s2, `{"lessons":["`+l2+`"],"slides":2,"total":4,"cards":{"`+l2+`":{"seen":4,"known":1}}}`); code != 204 {
		t.Fatalf("s2 story: %d", code)
	}
	if code := sv(s2, `{"lessons":["`+l2+`"],"slides":1,"total":4}`); code != 204 { // картгүй үзэлт
		t.Fatalf("картгүй story: %d", code)
	}
	code, st := call(t, srv, "GET", "/api/me/courses/"+cid+"/notes-stats", tt, "")
	if code != 200 {
		t.Fatalf("статистик: %d %v", code, st)
	}
	sum := st["summary"].(map[string]any)
	if sum["cards"] != 2.0 || sum["cards_seen"] != 7.0 || sum["cards_known"] != 3.0 || sum["card_reviewers"] != 2.0 ||
		sum["story_views"] != 3.0 || sum["story_done"] != 1.0 || sum["notes"] != 4.0 || sum["viewers"] != 2.0 {
		t.Fatalf("нэгтгэл: %v", sum)
	}
	if n := len(st["students"].([]any)); n != 2 {
		t.Fatalf("суралцагчийн мөр (тэмдэглэлгүй s3 орох ёсгүй): %d", n)
	}
	for _, x := range st["students"].([]any) {
		row := x.(map[string]any)
		if row["user_id"] == u3 {
			t.Fatalf("тэмдэглэлгүй s3 мөр болж орлоо: %v", row)
		}
		want := [3]float64{0, 4, 1} // s2: карт, давтсан, санасан
		if row["user_id"] == u1 {
			want = [3]float64{2, 3, 2}
		}
		if row["cards"] != want[0] || row["cards_seen"] != want[1] || row["cards_known"] != want[2] {
			t.Fatalf("суралцагчийн мөр: %v", row)
		}
	}
	for _, x := range st["lessons"].([]any) {
		row := x.(map[string]any)
		want := [3]float64{0, 4, 1}
		if row["id"] == l1 {
			want = [3]float64{2, 3, 2}
		}
		if row["cards"] != want[0] || row["cards_seen"] != want[1] || row["cards_known"] != want[2] {
			t.Fatalf("хичээлийн мөр: %v", row)
		}
	}
	if b, _ := json.Marshal(st); strings.Contains(string(b), "Эс гэж") || strings.Contains(string(b), "Нэгж") {
		t.Fatalf("багшид картын агуулга харагдах ёсгүй: %s", b)
	}
}

func TestStoryDetailCards(t *testing.T) {
	for in, want := range map[string][2]int{
		"5/7":                    {0, 0},
		"5/7 давталт":            {0, 0},
		"5/7 карт 2/3":           {2, 3},
		"5/7 давталт карт 0/4":   {0, 4},
		"5/7 карт x/3":           {0, 0},
		"5/7 карт 4/3":           {0, 0},
		"5/7 карт":               {0, 0},
		"5/7 давталт карт 9/501": {0, 0},
	} {
		if k, s := storyDetailCards(in); k != want[0] || s != want[1] {
			t.Fatalf("%q: %d/%d", in, k, s)
		}
	}
}

// Сургалт бүрийн хязгаар: олон бүртгэлээс нэг сургалт руу (нэг багшийг) 30-аас олон хүсэлт → 429.
func TestAccessRequestCourseLimit(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "arc_owner", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Анги","price":0,"published":true,"access_code":"Анги-1"}`)
	signer := auth.NewSigner("test", time.Hour)
	sent := 0
	for i := range 32 {
		u := &store.User{Username: fmt.Sprintf("arc%d", i), Email: fmt.Sprintf("arc%d@x.mn", i), Role: store.RoleStudent, DisplayName: "Суралцагч"}
		if err := st.CreateUser(context.Background(), u); err != nil {
			t.Fatal(err)
		}
		tok, err := signer.SignUser(u.ID, string(store.RoleStudent), u.Username)
		if err != nil {
			t.Fatal(err)
		}
		code, r := call(t, srv, "POST", "/api/courses/"+cid+"/access-request", tok, `{"lesson_id":""}`)
		if code == http.StatusTooManyRequests {
			break
		}
		if code != 200 || r["sent"] != true {
			t.Fatalf("хүсэлт %d: %d %v", i, code, r)
		}
		sent++
	}
	if sent != 30 {
		t.Fatalf("сургалтын хязгаар 30 байх ёстой: %d", sent)
	}
}

// failNotifStore — мэдэгдэл хадгалахад алдаа оруулдаг store (бусад нь санах ойн store).
type failNotifStore struct {
	store.Store
	fail atomic.Bool
}

func (f *failNotifStore) AddNotifications(ctx context.Context, ns []*store.Notification) error {
	if f.fail.Load() {
		return errors.New("мэдэгдлийн сан түр ажиллахгүй")
	}
	return f.Store.AddNotifications(ctx, ns)
}

// Багшийн мэдэгдэл хадгалагдаагүй бол хүсэлт «явсан» гэж хариулахгүй, давхардлын тэмдгийг буцааж дахин илгээж болно.
func TestAccessRequestDeliveryFailure(t *testing.T) {
	mem := store.NewMemory()
	testStore = mem
	fst := &failNotifStore{Store: mem}
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{TrustProxy: true, DevPayments: true, WebhookSecret: "wh", StorageFreeMB: 10, StoragePlans: []StoragePlan{{MB: 300, Price: 5000}}}
	s := New(cfg, fst, auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	tt, _ := register(t, srv, "ard_owner", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Анги","price":0,"published":true,"access_code":"Анги-1"}`)
	s1, _ := register(t, srv, "ard_s", "student")
	req := func() (int, map[string]any) {
		return call(t, srv, "POST", "/api/courses/"+cid+"/access-request", s1, `{"lesson_id":""}`)
	}
	fst.fail.Store(true)
	if code, r := req(); code != http.StatusInternalServerError || r["sent"] != nil {
		t.Fatalf("мэдэгдэл хадгалагдаагүй: %d %v", code, r)
	}
	fst.fail.Store(false)
	if code, r := req(); code != 200 || r["sent"] != true {
		t.Fatalf("дахин оролдлого already биш, илгээгдэх ёстой: %d %v", code, r)
	}
	if ns := arNotifs(t, srv, tt, "access_request"); len(ns) != 1 {
		t.Fatalf("багшид нэг мэдэгдэл: %d", len(ns))
	}
}
