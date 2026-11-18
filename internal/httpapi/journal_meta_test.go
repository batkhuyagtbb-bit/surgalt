package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"surgalt/internal/store"
)

// ---- Журнал: багшийн өгсөн нэр (alias), бүлэг, архив, суралцахуйн өгөгдөл устгах — HTTP түвшний тест ----
// Дизайн: journal-design.md §3 (API гэрээ), §4.3 (устгах/үлдээх хүснэгт), §P (бүтээгдэхүүний шийдвэр).

// jmLessonBlocks — устгалын тестийн «бүх төрлийн» хичээл: текст (тэмдэглэл), асуулт, үүлэн үг, цагаан самбар.
const jmLessonBlocks = `[{"id":"t001","type":"text","text":"Фотосинтез бол ургамлын хоол хийх үйл явц."},` +
	`{"id":"q001","type":"quiz","quiz":{"question":"?","options":["a","b"],"correct":[0]}},` +
	`{"id":"cld1","type":"cloud","text":"Юу сурав?","max":2},` +
	`{"id":"brd1","type":"board","text":""}]`

const jmExamQuiz = `"blocks":[{"id":"q001","type":"quiz","quiz":{"question":"?","options":["a","b"],"correct":[0]}}]`

func jmID(t *testing.T, code int, m map[string]any, what string) string {
	t.Helper()
	id, _ := m["id"].(string)
	if (code != http.StatusCreated && code != http.StatusOK) || id == "" {
		t.Fatalf("%s үүссэнгүй: %d %v", what, code, m)
	}
	return id
}

func jmCourse(t *testing.T, srv *httptest.Server, tok, body string) string {
	t.Helper()
	code, c := call(t, srv, "POST", "/api/courses", tok, body)
	return jmID(t, code, c, "сургалт")
}

func jmLesson(t *testing.T, srv *httptest.Server, tok, cid, body string) string {
	t.Helper()
	code, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tok, body)
	return jmID(t, code, l, "хичээл")
}

func jmEnroll(t *testing.T, srv *httptest.Server, tok, cid string) {
	t.Helper()
	if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", tok, ""); code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("элсэлт: %d %v", code, r)
	}
}

func jmPut(t *testing.T, srv *httptest.Server, tok, cid, uid, body string) (int, map[string]any) {
	return call(t, srv, "PUT", "/api/me/courses/"+cid+"/journal/students/"+uid, tok, body)
}

func jmRename(t *testing.T, srv *httptest.Server, tok, cid, body string) (int, map[string]any) {
	return call(t, srv, "PUT", "/api/me/courses/"+cid+"/journal/groups", tok, body)
}

func jmReset(t *testing.T, srv *httptest.Server, tok, cid, uid, body string) (int, map[string]any) {
	return call(t, srv, "POST", "/api/me/courses/"+cid+"/journal/students/"+uid+"/reset", tok, body)
}

// jmOK — PUT/POST-ийн хариу 200 байх ёстой.
func jmOK(t *testing.T, what string, code int, r map[string]any) map[string]any {
	t.Helper()
	if code != http.StatusOK {
		t.Fatalf("%s: %d %v", what, code, r)
	}
	return r
}

func jmPutOK(t *testing.T, srv *httptest.Server, what, tok, cid, uid, body string) map[string]any {
	t.Helper()
	code, r := jmPut(t, srv, tok, cid, uid, body)
	return jmOK(t, what, code, r)
}

func jmRenameOK(t *testing.T, srv *httptest.Server, what, tok, cid, body string) map[string]any {
	t.Helper()
	code, r := jmRename(t, srv, tok, cid, body)
	return jmOK(t, what, code, r)
}

func jmResetOK(t *testing.T, srv *httptest.Server, what, tok, cid, uid, body string) map[string]any {
	t.Helper()
	code, r := jmReset(t, srv, tok, cid, uid, body)
	return jmOK(t, what, code, r)
}

// jmFail — тодорхой статус ба алдааны мессеж.
func jmFail(t *testing.T, what string, code int, r map[string]any, want int, msg string) {
	t.Helper()
	if code != want {
		t.Fatalf("%s: %d хүлээсэн, %d %v", what, want, code, r)
	}
	if msg != "" && r["error"] != msg {
		t.Fatalf("%s: мессеж %q хүлээсэн, %v", what, msg, r["error"])
	}
}

// jmGet — журнал (200 байх ёстой).
func jmGet(t *testing.T, srv *httptest.Server, tok, cid, query string) map[string]any {
	t.Helper()
	path := "/api/me/courses/" + cid + "/journal"
	if query != "" {
		path += "?" + query
	}
	code, j := call(t, srv, "GET", path, tok, "")
	if code != http.StatusOK {
		t.Fatalf("журнал %s: %d %v", query, code, j)
	}
	return j
}

// jmRow — журналын хуудсанд тухайн суралцагчийн мөр (байхгүй бол nil).
func jmRow(j map[string]any, uid string) map[string]any {
	sts, _ := j["students"].([]any)
	for _, x := range sts {
		if m := x.(map[string]any); m["user_id"] == uid {
			return m
		}
	}
	return nil
}

// jmIDs — журналын хуудсан дахь суралцагчдын ID-ийн олонлог.
func jmIDs(j map[string]any) map[string]bool {
	out := map[string]bool{}
	sts, _ := j["students"].([]any)
	for _, x := range sts {
		out[x.(map[string]any)["user_id"].(string)] = true
	}
	return out
}

// jmGroups — "нэр:тоо,нэр:тоо" (хариуны дарааллаар).
func jmGroups(m map[string]any) string {
	gs, ok := m["groups"].([]any)
	if !ok {
		return fmt.Sprintf("<groups алга: %v>", m["groups"])
	}
	parts := make([]string, 0, len(gs))
	for _, x := range gs {
		g := x.(map[string]any)
		parts = append(parts, fmt.Sprintf("%v:%v", g["name"], g["count"]))
	}
	return strings.Join(parts, ",")
}

// jmCounts — бүлэг, бүлэггүй, архивын тоо нэг мөрөөр.
func jmCounts(m map[string]any) string {
	return fmt.Sprintf("groups=[%s] ungrouped=%v archived=%v", jmGroups(m), m["ungrouped"], m["archived"])
}

func jmLen(v any) int {
	a, _ := v.([]any)
	return len(a)
}

func jmNum(v any) float64 {
	f, _ := v.(float64)
	return f
}

// jmLearn — суралцагч хичээлийг бүрэн судална: үзэх, сесс + цохилт, асуулт, дуусгах, дүгнэлт, видео үзэлт, тэмдэглэл,
// үүлэн үг. Сессийн ID-г буцаана.
func jmLearn(t *testing.T, srv *httptest.Server, tok, cid, lid string) string {
	t.Helper()
	base := "/api/courses/" + cid + "/lessons/" + lid
	if code, r := call(t, srv, "GET", base, tok, ""); code != http.StatusOK {
		t.Fatalf("хичээл үзэх: %d %v", code, r)
	}
	code, ss := call(t, srv, "POST", "/api/activity/start", tok, `{"course_id":"`+cid+`","lesson_id":"`+lid+`","kind":"lesson"}`)
	sid, _ := ss["session_id"].(string)
	if code != http.StatusOK || sid == "" {
		t.Fatalf("сесс: %d %v", code, ss)
	}
	if code, r := call(t, srv, "POST", "/api/activity/beat", tok, `{"session_id":"`+sid+`","active":4}`); code != http.StatusOK {
		t.Fatalf("цохилт: %d %v", code, r)
	}
	steps := []struct{ path, body string }{
		{base + "/quiz/q001", `{"answer":[0],"ms":3000}`},
		{base + "/complete", ``},
		{base + "/reflect", `{"text":"Энэ хичээлээр фотосинтезийн үйл явцыг сайн ойлголоо."}`},
		{base + "/progress/q001", `{"duration":55,"buckets":{"0":1,"1":2}}`},
		{base + "/cloud/cld1", `{"words":["нар"]}`},
	}
	for _, s := range steps {
		if code, r := call(t, srv, "POST", s.path, tok, s.body); code != http.StatusOK {
			t.Fatalf("%s: %d %v", s.path, code, r)
		}
	}
	if code, r := call(t, srv, "PUT", base+"/highlights", tok, `{"highlights":[{"id":"h1","b":"t001","s":0,"e":10,"c":"y","t":"Фотосинтез","n":"гол"}]}`); code != http.StatusOK {
		t.Fatalf("тэмдэглэл: %d %v", code, r)
	}
	return sid
}

// jmAccess — суралцагчийн сургалтын төлөв (GET /api/courses/{id}/access).
func jmAccess(t *testing.T, srv *httptest.Server, tok, cid string) map[string]any {
	t.Helper()
	code, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", tok, "")
	if code != http.StatusOK {
		t.Fatalf("access: %d %v", code, a)
	}
	return a
}

func jmHighlights(t *testing.T, srv *httptest.Server, tok, cid, lid string) int {
	t.Helper()
	code, r := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid+"/highlights", tok, "")
	if code != http.StatusOK {
		t.Fatalf("тэмдэглэл унших: %d %v", code, r)
	}
	return jmLen(r["highlights"])
}

func jmCloudMine(t *testing.T, srv *httptest.Server, tok, cid, lid string) int {
	t.Helper()
	code, r := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid+"/cloud/cld1", tok, "")
	if code != http.StatusOK {
		t.Fatalf("үүлэн үг унших: %d %v", code, r)
	}
	return jmLen(r["mine"])
}

// jmDetail — багшийн суралцагчийн дэлгэрэнгүй шинжилгээ (сургалтаар).
func jmDetail(t *testing.T, srv *httptest.Server, tok, cid, uid string) map[string]any {
	t.Helper()
	code, d := call(t, srv, "GET", "/api/me/analytics/students/"+uid+"?course="+cid, tok, "")
	if code != http.StatusOK {
		t.Fatalf("шинжилгээ: %d %v", code, d)
	}
	return d
}

func jmNotifs(t *testing.T, srv *httptest.Server, tok string) []map[string]any {
	t.Helper()
	_, items := callList(t, srv, "GET", "/api/me/notifications", tok)
	out := make([]map[string]any, 0, len(items))
	for _, x := range items {
		out = append(out, x.(map[string]any))
	}
	return out
}

func jmArchive(t *testing.T, srv *httptest.Server, tok, cid, uid string) {
	t.Helper()
	jmPutOK(t, srv, "архивлах", tok, cid, uid, `{"archived":true}`)
}

func jmMetas(t *testing.T, st store.Store, cid string) map[string]store.CourseStudentMeta {
	t.Helper()
	ms, err := st.CourseStudentMetas(context.Background(), cid)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

// ---------------------------------------------------------------------------------------------------------------

// Нэр (alias): зөвхөн тухайн сургалтын журналд; name = alias, account_name = бүртгэлийн нэр; хайлт alias, бүртгэлийн нэр,
// username-аар; хоосон зай нэгтгэгдэнэ; бүртгэлийн нэртэй ижил бол хадгалахгүй; "" арилгана; хэсэгчилсэн засвар бусад
// талбарыг хөндөхгүй; суралцагчийн бүртгэлийн нэр хэзээ ч өөрчлөгдөхгүй.
func TestJournalAlias(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jalias_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Нэрийн журнал","price":0,"published":true}`)
	jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	s1, uid1 := register(t, srv, "bat2015", "student")
	s2, uid2 := register(t, srv, "dorj2015", "student")
	jmEnroll(t, srv, s1, cid)
	jmEnroll(t, srv, s2, cid)
	cid2 := jmCourse(t, srv, tt, `{"title":"Өөр сургалт","price":0,"published":true}`)
	jmLesson(t, srv, tt, cid2, `{"title":"Нэг","is_free":true}`)
	jmEnroll(t, srv, s1, cid2)

	// Нэр өгөх: хоосон зай нэгтгэгдэнэ.
	r := jmPutOK(t, srv, "нэр өгөх", tt, cid, uid1, `{"alias":"  Бат   Эрдэнэ  "}`)
	stu := r["student"].(map[string]any)
	if stu["user_id"] != uid1 || stu["name"] != "Бат Эрдэнэ" || stu["alias"] != "Бат Эрдэнэ" || stu["account_name"] != "bat2015" {
		t.Fatalf("засварын хариу: %v", stu)
	}
	for _, k := range []string{"group", "archived", "archived_at"} {
		if _, ok := stu[k]; ok {
			t.Fatalf("хоосон %q талбар omitempty байх ёстой: %v", k, stu)
		}
	}
	if jmCounts(r) != "groups=[] ungrouped=2 archived=0" {
		t.Fatalf("засварын хариуны тоо: %s", jmCounts(r))
	}
	j := jmGet(t, srv, tt, cid, "")
	row := jmRow(j, uid1)
	if row == nil || row["name"] != "Бат Эрдэнэ" || row["alias"] != "Бат Эрдэнэ" || row["account_name"] != "bat2015" || row["username"] != "bat2015" {
		t.Fatalf("журналын мөр: %v", row)
	}
	if o := jmRow(j, uid2); o == nil || o["name"] != "dorj2015" || o["account_name"] != "dorj2015" || o["alias"] != nil {
		t.Fatalf("нэргүй суралцагчийн мөр: %v", o)
	}
	// Нэр зөвхөн энэ сургалтад: өөр сургалтын журналд бүртгэлийн нэрээр.
	if o := jmRow(jmGet(t, srv, tt, cid2, ""), uid1); o == nil || o["name"] != "bat2015" || o["alias"] != nil {
		t.Fatalf("өөр сургалтын журналд нэр гарах ёсгүй: %v", o)
	}
	// Бүртгэлийн нэр өөрчлөгдөхгүй.
	if u, err := st.UserByID(context.Background(), uid1); err != nil || u.DisplayName != "bat2015" {
		t.Fatalf("бүртгэлийн нэр өөрчлөгдлөө: %v %v", u, err)
	}

	// Хайлт: alias (жижиг үсгээр), бүртгэлийн нэр, username; AND шүүлт.
	for q, want := range map[string]string{"эрдэнэ": uid1, "БАТ": uid1, "bat20": uid1, "dorj": uid2} {
		j := jmGet(t, srv, tt, cid, "q="+q)
		if ids := jmIDs(j); len(ids) != 1 || !ids[want] || jmNum(j["students_total"]) != 1 {
			t.Fatalf("хайлт %q: %v total=%v", q, ids, j["students_total"])
		}
	}
	if j := jmGet(t, srv, tt, cid, "q=байхгүйнэр"); jmLen(j["students"]) != 0 || jmNum(j["students_total"]) != 0 || jmCounts(j) != "groups=[] ungrouped=2 archived=0" {
		t.Fatalf("олдохгүй хайлт: тоо нь бүх суралцагчаар байх ёстой: %v %s", j["students"], jmCounts(j))
	}

	// Хэсэгчилсэн засвар: бүлэг өгөхөд нэр хэвээр; нэр солиход бүлэг хэвээр.
	r = jmPutOK(t, srv, "бүлэг", tt, cid, uid1, `{"group":"5А"}`)
	if s := r["student"].(map[string]any); s["alias"] != "Бат Эрдэнэ" || s["group"] != "5А" {
		t.Fatalf("бүлэг өгөхөд нэр алдагдлаа: %v", s)
	}
	r = jmPutOK(t, srv, "нэр солих", tt, cid, uid1, `{"alias":"Бат"}`)
	if s := r["student"].(map[string]any); s["alias"] != "Бат" || s["group"] != "5А" || s["name"] != "Бат" {
		t.Fatalf("нэр солиход бүлэг алдагдлаа: %v", s)
	}

	// Бүртгэлийн нэртэй ижил (хоосон зайтай ч) → тусдаа нэр хадгалахгүй.
	r = jmPutOK(t, srv, "бүртгэлийн нэртэй ижил", tt, cid, uid1, `{"alias":"  bat2015 "}`)
	if s := r["student"].(map[string]any); s["alias"] != nil || s["name"] != "bat2015" || s["group"] != "5А" {
		t.Fatalf("бүртгэлийн нэртэй ижил нэр хадгалагдах ёсгүй: %v", s)
	}
	if m := jmMetas(t, st, cid)[uid1]; m.Alias != "" || m.Group != "5А" {
		t.Fatalf("хадгалсан мөр: %+v", m)
	}
	// "" → арилгана.
	jmPutOK(t, srv, "нэр өгөх", tt, cid, uid2, `{"alias":"Дорж"}`)
	r = jmPutOK(t, srv, "нэр арилгах", tt, cid, uid2, `{"alias":""}`)
	if s := r["student"].(map[string]any); s["alias"] != nil || s["name"] != "dorj2015" || s["account_name"] != "dorj2015" {
		t.Fatalf("нэр арилгах: %v", s)
	}
	if row := jmRow(jmGet(t, srv, tt, cid, ""), uid2); row["name"] != "dorj2015" || row["alias"] != nil {
		t.Fatalf("арилгасны дараа журнал: %v", row)
	}
	if _, ok := jmMetas(t, st, cid)[uid2]; ok {
		t.Fatalf("бүх талбар хоосон мөр «мэдээлэлгүй» гэж алгасагдах ёстой: %v", jmMetas(t, st, cid))
	}
}

// Нэрийн шалгалт: талбаргүй, удирдах/формат тэмдэгт (хоосон зай нэгтгэхээс ӨМНӨ, түүхий утга дээр), урт (нэгтгэсний
// дараах rune-ээр), буруу төрөл, үл мэдэх талбар; шалгалтын дараалал (талбар → багш → гишүүнчлэл → утга). Алдаатай
// хүсэлт юу ч бичихгүй.
func TestJournalAliasValidation(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, tid := register(t, srv, "jval_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Шалгалт","price":0,"published":true}`)
	jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	s1, uid := register(t, srv, "jval_s", "student")
	jmEnroll(t, srv, s1, cid)
	_, stranger := register(t, srv, "jval_x", "student")
	jmPutOK(t, srv, "эхний нэр", tt, cid, uid, `{"alias":"Анхны","group":"5А"}`)

	const (
		noField  = "өөрчлөх талбар алга"
		badAlias = "нэрэнд зөвшөөрөгдөөгүй тэмдэгт байна"
		longName = "нэр 80 тэмдэгтээс хэтрэхгүй байх ёстой"
		badGroup = "бүлгийн нэрэнд зөвшөөрөгдөөгүй тэмдэгт байна"
		longGrp  = "бүлгийн нэр 40 тэмдэгтээс хэтрэхгүй байх ёстой"
	)
	ya := func(n int) string { return strings.Repeat("я", n) }
	cases := []struct {
		name, body string
		code       int
		msg        string
	}{
		{"хоосон объект", `{}`, 400, noField},
		{"null нэр", `{"alias":null}`, 400, noField},
		{"мөр шилжүүлэх (нэгтгэхэд алга болох байсан)", `{"alias":"Бат\nДорж"}`, 400, badAlias},
		{"таб", `{"alias":"Бат\tДорж"}`, 400, badAlias},
		{"NUL", `{"alias":"Бат\u0000"}`, 400, badAlias},
		{"DEL", `{"alias":"Бат\u007f"}`, 400, badAlias},
		{"C1 удирдах", `{"alias":"Бат\u0085"}`, 400, badAlias},
		{"чиглэл эргүүлэх RLO (Cf)", `{"alias":"Бат\u202eДорж"}`, 400, badAlias},
		{"RLM (Cf)", `{"alias":"\u200fБат"}`, 400, badAlias},
		{"үл харагдах зай ZWSP (Cf)", `{"alias":"Бат\u200b"}`, 400, badAlias},
		{"zero-width joiner (Cf)", `{"alias":"Ба\u200dт"}`, 400, badAlias},
		{"зөөлөн зураас (Cf)", `{"alias":"Ба\u00adт"}`, 400, badAlias},
		{"BOM (Cf)", `{"alias":"\ufeffБат"}`, 400, badAlias},
		{"81 rune", `{"alias":"` + ya(81) + `"}`, 400, longName},
		{"бүлэгт мөр шилжүүлэх", `{"group":"5\nА"}`, 400, badGroup},
		{"бүлэгт Cf", `{"group":"5\u2066А"}`, 400, badGroup},
		{"бүлэг 41 rune", `{"group":"` + ya(41) + `"}`, 400, longGrp},
		{"нэр зөв, бүлэг буруу — юу ч бичихгүй", `{"alias":"Шинэ","group":"5\tА"}`, 400, badGroup},
		{"нэр буруу, бүлэг зөв — юу ч бичихгүй", `{"alias":"Шинэ\r","group":"6Б"}`, 400, badAlias},
		{"архивтай хамт буруу нэр — архивлахгүй", `{"alias":"\u200eX","archived":true}`, 400, badAlias},
		{"буруу төрөл (тоо)", `{"alias":5}`, 400, ""},
		{"буруу төрөл (archived)", `{"archived":"yes"}`, 400, ""},
		{"үл мэдэх талбар", `{"alias":"Бат","nick":"x"}`, 400, ""},
		{"JSON биш", `alias=Бат`, 400, ""},
	}
	for _, c := range cases {
		code, r := jmPut(t, srv, tt, cid, uid, c.body)
		jmFail(t, c.name, code, r, c.code, c.msg)
	}
	if m := jmMetas(t, st, cid)[uid]; m.Alias != "Анхны" || m.Group != "5А" || m.Archived {
		t.Fatalf("алдаатай хүсэлтүүд мөрийг өөрчиллөө: %+v", m)
	}

	// Хил: яг 80 / 40 rune; хоосон зай нэгтгэсний дараа тоологдоно (түүхий нь 80-аас урт ч).
	for _, ok := range []struct{ body, alias, group string }{
		{`{"alias":"` + ya(80) + `"}`, ya(80), "5А"},
		{`{"alias":"   ` + ya(40) + `      ` + ya(39) + `   "}`, ya(40) + " " + ya(39), "5А"},
		{`{"group":"  ` + ya(20) + `    ` + ya(19) + ` "}`, ya(40) + " " + ya(39), ya(20) + " " + ya(19)},
	} {
		r := jmPutOK(t, srv, "хилийн утга", tt, cid, uid, ok.body)
		if s := r["student"].(map[string]any); s["alias"] != ok.alias || s["group"] != ok.group {
			t.Fatalf("хилийн утга %s: %v", ok.body[:20], s)
		}
	}

	// Шалгалтын дараалал: талбаргүй нь багшаас өмнө; багш нь гишүүнчлэлээс өмнө; гишүүнчлэл нь утгаас өмнө.
	code, r := jmPut(t, srv, tt, cid, tid, `{}`)
	jmFail(t, "эзэмшигч + талбаргүй", code, r, 400, noField)
	code, r = jmPut(t, srv, tt, cid, tid, `{"alias":"Багш\n"}`)
	jmFail(t, "эзэмшигч + буруу нэр", code, r, 400, "сургалтын багшийг журналд засах боломжгүй")
	code, r = jmPut(t, srv, tt, cid, stranger, `{"alias":"Бат\n"}`)
	jmFail(t, "гишүүн биш + буруу нэр", code, r, 404, "энэ суралцагч таны сургалтад байхгүй байна")
	code, r = jmPut(t, srv, tt, cid, "no-such-user", `{"alias":"Бат"}`)
	jmFail(t, "байхгүй хэрэглэгч", code, r, 404, "энэ суралцагч таны сургалтад байхгүй байна")
	if _, ok := jmMetas(t, st, cid)[stranger]; ok {
		t.Fatal("гишүүн бишид мөр бичигдлээ")
	}
}

// Бүлэг: том жижиг үсэг үл харгалзан нэгтгэгдэнэ (байгаа бичлэг ялна, архивласан суралцагчийнх ч); тоо нь архивлаагүй
// бүх суралцагчаар, жижиг үсгээр эрэмбэлсэн; ?group= (EqualFold), ?nogroup=1, q-тэй AND; "" нь бүлгээс гаргана;
// бүлэг далд (зөвхөн архивласан гишүүнтэй бол жагсаалтад алга).
func TestJournalGroups(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jgrp_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Бүлэгтэй","price":0,"published":true}`)
	jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	var uid [4]string
	for i, n := range []string{"jgrp_a", "jgrp_b", "jgrp_c", "jgrp_d"} {
		var tok string
		tok, uid[i] = register(t, srv, n, "student")
		jmEnroll(t, srv, tok, cid)
	}
	A, B, C, D := uid[0], uid[1], uid[2], uid[3]
	put := func(u, body string) map[string]any {
		t.Helper()
		code, r := jmPut(t, srv, tt, cid, u, body)
		return jmOK(t, body, code, r)
	}
	grp := func(r map[string]any) any { return r["student"].(map[string]any)["group"] }

	if r := put(A, `{"group":"  5А  "}`); grp(r) != "5А" || jmCounts(r) != "groups=[5А:1] ungrouped=3 archived=0" {
		t.Fatalf("A: %v %s", grp(r), jmCounts(r))
	}
	if r := put(B, `{"group":"5а"}`); grp(r) != "5А" || jmCounts(r) != "groups=[5А:2] ungrouped=2 archived=0" {
		t.Fatalf("B (жижиг үсгээр) байгаа бичлэгт нэгдэх ёстой: %v %s", grp(r), jmCounts(r))
	}
	if r := put(C, `{"group":"Ахлах"}`); jmCounts(r) != "groups=[5А:2,Ахлах:1] ungrouped=1 archived=0" {
		t.Fatalf("C: %s", jmCounts(r))
	}
	if r := put(D, `{"group":"5Б"}`); jmCounts(r) != "groups=[5А:2,5Б:1,Ахлах:1] ungrouped=0 archived=0" {
		t.Fatalf("D (эрэмбэ жижиг үсгээр): %s", jmCounts(r))
	}
	put(D, `{"group":""}`)

	j := jmGet(t, srv, tt, cid, "")
	if jmCounts(j) != "groups=[5А:2,Ахлах:1] ungrouped=1 archived=0" || jmNum(j["students_total"]) != 4 {
		t.Fatalf("журналын тоо: %s total=%v", jmCounts(j), j["students_total"])
	}
	if jmRow(j, A)["group"] != "5А" || jmRow(j, D)["group"] != nil {
		t.Fatalf("мөрийн бүлэг: %v %v", jmRow(j, A), jmRow(j, D))
	}
	filters := []struct {
		q    string
		want []string
	}{
		{"group=5%D0%B0", []string{A, B}}, // "5а" — том жижиг үсэг үл харгалзана
		{"group=%20%205А%20", []string{A, B}},
		{"group=Ахлах", []string{C}},
		{"nogroup=1", []string{D}},
		{"group=5А&q=jgrp_b", []string{B}},
		{"nogroup=1&q=jgrp_a", nil},
		{"group=Байхгүй", nil},
	}
	for _, f := range filters {
		j := jmGet(t, srv, tt, cid, f.q)
		ids := jmIDs(j)
		if len(ids) != len(f.want) || int(jmNum(j["students_total"])) != len(f.want) {
			t.Fatalf("шүүлт %s: %v total=%v, хүлээсэн %v", f.q, ids, j["students_total"], f.want)
		}
		for _, w := range f.want {
			if !ids[w] {
				t.Fatalf("шүүлт %s: %s алга: %v", f.q, w, ids)
			}
		}
		if jmCounts(j) != "groups=[5А:2,Ахлах:1] ungrouped=1 archived=0" {
			t.Fatalf("шүүлт %s: тоо бүх суралцагчаар байх ёстой: %s", f.q, jmCounts(j))
		}
	}

	// Архивласан суралцагч бүлгийн тоонд орохгүй, бүлгээ хадгална; бүлгийн цорын ганц гишүүн архивлагдвал бүлэг алга.
	if r := put(B, `{"archived":true}`); jmCounts(r) != "groups=[5А:1,Ахлах:1] ungrouped=1 archived=1" {
		t.Fatalf("B архивлахад: %s", jmCounts(r))
	}
	if j := jmGet(t, srv, tt, cid, "archived=1&group=5а"); len(jmIDs(j)) != 1 || jmRow(j, B)["group"] != "5А" {
		t.Fatalf("архивласан B бүлгээ хадгалах ёстой: %v", j["students"])
	}
	if r := put(A, `{"archived":true}`); jmCounts(r) != "groups=[Ахлах:1] ungrouped=1 archived=2" {
		t.Fatalf("5А-ийн бүх гишүүн архивт: бүлэг жагсаалтаас алга болох ёстой: %s", jmCounts(r))
	}
	if j := jmGet(t, srv, tt, cid, "group=5А"); len(jmIDs(j)) != 0 {
		t.Fatalf("архивгүй жагсаалтад 5А хоосон: %v", j["students"])
	}
	// Архивласан гишүүдийн бичлэг ч нэгтгэлд тооцогдоно (том жижиг үсэг).
	if r := put(D, `{"group":"5а"}`); grp(r) != "5А" || jmCounts(r) != "groups=[5А:1,Ахлах:1] ungrouped=0 archived=2" {
		t.Fatalf("архивласан гишүүдийн бүлгийн бичлэг: %v %s", grp(r), jmCounts(r))
	}
	// "" → бүлгээс гаргана.
	if r := put(C, `{"group":""}`); grp(r) != nil || jmCounts(r) != "groups=[5А:1] ungrouped=1 archived=2" {
		t.Fatalf("бүлгээс гаргах: %v %s", grp(r), jmCounts(r))
	}
}

// 100 бүлгийн хязгаар: архивласан суралцагчийн бүлэг ч тоологдоно, том жижиг үсэг үл харгалзан, засаж буй суралцагчийн
// одоогийн бүлгийг хасаж; байгаа бүлэгт нэгдэх нь хязгаарт хамаарахгүй; алдаатай хүсэлт юу ч бичихгүй.
func TestJournalGroupsCap(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	tt, tid := register(t, srv, "jcap_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Олон бүлэг","price":0,"published":true}`)
	jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	sa, A := register(t, srv, "jcap_a", "student")
	sb, B := register(t, srv, "jcap_b", "student")
	jmEnroll(t, srv, sa, cid)
	jmEnroll(t, srv, sb, cid)
	// 98 тусдаа бүлэг (G98 нь архивласан суралцагчийнх) + "dup"/"DUP" (нэг бүлэг) = 99 бүлэг.
	seed := func(uid, g string, archived bool) {
		_, err := st.UpdateCourseStudentMeta(ctx, cid, uid, tid, func(c *store.CourseStudentMeta, _ map[string]store.CourseStudentMeta) error {
			c.Group, c.Archived = g, archived
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 98; i++ {
		seed(fmt.Sprintf("syn-%03d", i), fmt.Sprintf("G%02d", i), i == 98)
	}
	seed("syn-dup1", "dup", false)
	seed("syn-dup2", "DUP", false)
	const capMsg = "нэг сургалтад 100 хүртэл бүлэг үүсгэх боломжтой"

	r := jmPutOK(t, srv, "100 дахь бүлэг", tt, cid, A, `{"group":"Шинэ1"}`)
	if r["student"].(map[string]any)["group"] != "Шинэ1" {
		t.Fatalf("A: %v", r["student"])
	}
	code, r := jmPut(t, srv, tt, cid, B, `{"group":"Шинэ2"}`)
	jmFail(t, "101 дэх бүлэг", code, r, 400, capMsg)
	code, r = jmPut(t, srv, tt, cid, B, `{"alias":"Бямба","group":"Шинэ2"}`)
	jmFail(t, "101 дэх бүлэг (нэртэй хамт)", code, r, 400, capMsg)
	if m, ok := jmMetas(t, st, cid)[B]; ok {
		t.Fatalf("хязгаарын алдаа юу ч бичих ёсгүй: %+v", m)
	}
	// Байгаа бүлэгт (архивласан суралцагчийнх ч) нэгдэх нь болно, байгаа бичлэгээр.
	r = jmPutOK(t, srv, "архивласны бүлэгт", tt, cid, B, `{"group":"g98"}`)
	if g := r["student"].(map[string]any)["group"]; g != "G98" {
		t.Fatalf("архивласан суралцагчийн бүлгийн бичлэг: %v", g)
	}
	r = jmPutOK(t, srv, "dup бүлэгт", tt, cid, B, `{"group":"Dup"}`)
	if g := r["student"].(map[string]any)["group"]; g != "dup" && g != "DUP" {
		t.Fatalf("байгаа бичлэгүүдийн нэг байх ёстой: %v", g)
	}
	// A өөрийн цорын ганц бүлгийг шинээр солино: өөрийн одоогийн бүлэг тоологдохгүй (99 + шинэ = 100).
	r = jmPutOK(t, srv, "өөрийн бүлгийг солих", tt, cid, A, `{"group":"Шинэ3"}`)
	if g := r["student"].(map[string]any)["group"]; g != "Шинэ3" {
		t.Fatalf("A: %v", g)
	}
	// B-ийн одоогийн бүлэг ("dup"/"DUP") хасагдсан ч нөгөө бичлэг нь ижил бүлэг хэвээр тоологдоно → 101.
	code, r = jmPut(t, srv, tt, cid, B, `{"group":"Шинэ4"}`)
	jmFail(t, "том жижиг үсгийн давхар бүлэг ганцаар тоологдоно", code, r, 400, capMsg)
}

// Бүлгийн нэр солих / тараах: архивласан гишүүд ч хамт; ижил бичлэгээр → changed 0; зөвхөн том жижиг үсэг → бичсэнээр;
// өөр байгаа бүлэгтэй таарвал нэгтгэж тэр бичлэг ялна; байхгүй бүлэг 404; хоосон from 400; to-г шалгана; "" тараана.
func TestJournalGroupRenameDissolve(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jren_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Нэр солих","price":0,"published":true}`)
	jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	var uid [5]string
	for i := range uid {
		var tok string
		tok, uid[i] = register(t, srv, fmt.Sprintf("jren_s%d", i), "student")
		jmEnroll(t, srv, tok, cid)
	}
	A, B, C, D, E := uid[0], uid[1], uid[2], uid[3], uid[4]
	for _, x := range []struct{ u, body string }{{A, `{"group":"5А"}`}, {B, `{"group":"5А"}`}, {C, `{"group":"6В"}`}, {E, `{"group":"5А","archived":true}`}} {
		jmPutOK(t, srv, x.body, tt, cid, x.u, x.body)
	}
	_ = D
	rename := func(body string) map[string]any {
		t.Helper()
		code, r := jmRename(t, srv, tt, cid, body)
		return jmOK(t, body, code, r)
	}
	archGroup := func() any { return jmRow(jmGet(t, srv, tt, cid, "archived=1"), E)["group"] }

	r := rename(`{"from":"5а","to":"  5Б  "}`)
	if jmNum(r["changed"]) != 3 || jmCounts(r) != "groups=[5Б:2,6В:1] ungrouped=1 archived=1" {
		t.Fatalf("нэр солих (архивласан гишүүн ч): %v %s", r["changed"], jmCounts(r))
	}
	if g := archGroup(); g != "5Б" {
		t.Fatalf("архивласан гишүүний бүлэг солигдох ёстой: %v", g)
	}
	if r := rename(`{"from":"5Б","to":"5Б"}`); jmNum(r["changed"]) != 0 || jmCounts(r) != "groups=[5Б:2,6В:1] ungrouped=1 archived=1" {
		t.Fatalf("ижил бичлэг → changed 0: %v %s", r["changed"], jmCounts(r))
	}
	if r := rename(`{"from":"5Б","to":"5б"}`); jmNum(r["changed"]) != 3 || jmCounts(r) != "groups=[5б:2,6В:1] ungrouped=1 archived=1" {
		t.Fatalf("зөвхөн том жижиг үсэг → бичсэнээр: %v %s", r["changed"], jmCounts(r))
	}
	// Өөр байгаа бүлэгтэй таарвал нэгтгэнэ, тэр бүлгийн бичлэг ялна.
	if r := rename(`{"from":"5б","to":"6в"}`); jmNum(r["changed"]) != 3 || jmCounts(r) != "groups=[6В:3] ungrouped=1 archived=1" {
		t.Fatalf("нэгтгэх: %v %s", r["changed"], jmCounts(r))
	}
	if g := archGroup(); g != "6В" {
		t.Fatalf("нэгтгэсний дараа архивласан гишүүн: %v", g)
	}

	errs := []struct {
		name, body string
		code       int
		msg        string
	}{
		{"байхгүй бүлэг", `{"from":"Байхгүй","to":"7Г"}`, 404, "ийм бүлэг олдсонгүй — хуудсаа сэргээнэ үү"},
		{"хоосон from", `{"from":"   ","to":"7Г"}`, 400, "бүлгийн нэрээ оруулна уу"},
		{"from алга", `{"to":"7Г"}`, 400, "бүлгийн нэрээ оруулна уу"},
		{"to-д удирдах тэмдэгт", `{"from":"6В","to":"7\nГ"}`, 400, "бүлгийн нэрэнд зөвшөөрөгдөөгүй тэмдэгт байна"},
		{"to-д Cf", `{"from":"6В","to":"7\u200bГ"}`, 400, "бүлгийн нэрэнд зөвшөөрөгдөөгүй тэмдэгт байна"},
		{"to 41 rune", `{"from":"6В","to":"` + strings.Repeat("я", 41) + `"}`, 400, "бүлгийн нэр 40 тэмдэгтээс хэтрэхгүй байх ёстой"},
		{"үл мэдэх талбар", `{"from":"6В","to":"7Г","all":true}`, 400, ""},
	}
	for _, e := range errs {
		code, r := jmRename(t, srv, tt, cid, e.body)
		jmFail(t, e.name, code, r, e.code, e.msg)
	}
	if j := jmGet(t, srv, tt, cid, ""); jmCounts(j) != "groups=[6В:3] ungrouped=1 archived=1" {
		t.Fatalf("алдаатай хүсэлтүүд бүлгийг өөрчиллөө: %s", jmCounts(j))
	}

	// Тараах: бүх гишүүн (архивласан ч) бүлэггүй болно; дахин тараах → 404.
	if r := rename(`{"from":"6В","to":""}`); jmNum(r["changed"]) != 4 || jmCounts(r) != "groups=[] ungrouped=4 archived=1" {
		t.Fatalf("тараах: %v %s", r["changed"], jmCounts(r))
	}
	if g := archGroup(); g != nil {
		t.Fatalf("архивласан гишүүн ч бүлэггүй болох ёстой: %v", g)
	}
	code, r2 := jmRename(t, srv, tt, cid, `{"from":"6В","to":""}`)
	jmFail(t, "тараасан бүлгийг дахин", code, r2, 404, "ийм бүлэг олдсонгүй — хуудсаа сэргээнэ үү")
}

// Архив: журналаас нууж, өгөгдлийг (нүд, явц) хадгална; нэр, бүлэг хэвээр; ?archived=1 ижил хэлбэрээр нүдтэйгээ;
// давтан архивлахад archived_at өөрчлөгдөхгүй; сэргээхэд хариунд archived/archived_at түлхүүр байхгүй (omitempty).
func TestJournalArchive(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jarc_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Архив","price":0,"published":true}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true,"blocks":`+jmLessonBlocks+`}`)
	s1, A := register(t, srv, "jarc_a", "student")
	s2, B := register(t, srv, "jarc_b", "student")
	jmEnroll(t, srv, s1, cid)
	jmEnroll(t, srv, s2, cid)
	jmLearn(t, srv, s1, cid, lid)

	jmPutOK(t, srv, "нэр, бүлэг", tt, cid, A, `{"alias":"Бат","group":"5А"}`)
	r := jmPutOK(t, srv, "архивлах", tt, cid, A, `{"archived":true}`)
	stu := r["student"].(map[string]any)
	at, _ := stu["archived_at"].(string)
	if stu["archived"] != true || at == "" || stu["alias"] != "Бат" || stu["group"] != "5А" || stu["name"] != "Бат" {
		t.Fatalf("архивлах хариу: %v", stu)
	}
	if ts, err := time.Parse(time.RFC3339Nano, at); err != nil || time.Since(ts) > time.Minute {
		t.Fatalf("archived_at: %q %v", at, err)
	}
	if jmCounts(r) != "groups=[] ungrouped=1 archived=1" {
		t.Fatalf("архивлах хариуны тоо: %s", jmCounts(r))
	}
	j := jmGet(t, srv, tt, cid, "")
	if ids := jmIDs(j); ids[A] || !ids[B] || jmNum(j["students_total"]) != 1 || jmNum(j["archived"]) != 1 {
		t.Fatalf("үндсэн журналд архивласан суралцагч гарах ёсгүй: %v %v", ids, j)
	}
	if _, ok := j["cells"].(map[string]any)[A]; ok {
		t.Fatal("үндсэн журналд архивласан суралцагчийн нүд гарах ёсгүй")
	}
	ja := jmGet(t, srv, tt, cid, "archived=1")
	row := jmRow(ja, A)
	if ids := jmIDs(ja); len(ids) != 1 || row == nil || jmNum(ja["students_total"]) != 1 {
		t.Fatalf("архивын жагсаалт: %v", ja["students"])
	}
	if row["archived"] != true || row["archived_at"] != at || row["alias"] != "Бат" || row["group"] != "5А" || row["name"] != "Бат" || row["account_name"] != "jarc_a" || jmNum(row["done"]) != 1 {
		t.Fatalf("архивын мөр (өгөгдөл хадгалагдсан): %v", row)
	}
	if c, _ := ja["cells"].(map[string]any)[A].(map[string]any)[lid].(map[string]any); c == nil || c["done"] != true {
		t.Fatalf("архивын жагсаалтад нүд байх ёстой: %v", ja["cells"])
	}
	if jmLen(ja["lessons"]) != 1 {
		t.Fatalf("архивын жагсаалт ижил хэлбэртэй (lessons): %v", ja["lessons"])
	}

	// Давтан архивлах → archived_at хэвээр.
	time.Sleep(5 * time.Millisecond)
	r = jmPutOK(t, srv, "давтан архивлах", tt, cid, A, `{"archived":true}`)
	if r["student"].(map[string]any)["archived_at"] != at {
		t.Fatalf("давтан архивлахад archived_at өөрчлөгдөх ёсгүй: %v → %v", at, r["student"])
	}

	// Сэргээх: archived, archived_at түлхүүргүй; нэр, бүлэг, өгөгдөл хэвээр.
	r = jmPutOK(t, srv, "сэргээх", tt, cid, A, `{"archived":false}`)
	stu = r["student"].(map[string]any)
	if _, ok := stu["archived"]; ok {
		t.Fatalf("сэргээсэн хариунд archived түлхүүр байх ёсгүй: %v", stu)
	}
	if _, ok := stu["archived_at"]; ok {
		t.Fatalf("сэргээсэн хариунд archived_at түлхүүр байх ёсгүй: %v", stu)
	}
	if stu["alias"] != "Бат" || stu["group"] != "5А" || jmCounts(r) != "groups=[5А:1] ungrouped=1 archived=0" {
		t.Fatalf("сэргээх: %v %s", stu, jmCounts(r))
	}
	j = jmGet(t, srv, tt, cid, "")
	row = jmRow(j, A)
	if row == nil || row["archived"] != nil || row["archived_at"] != nil || row["alias"] != "Бат" || jmNum(row["done"]) != 1 || jmNum(j["students_total"]) != 2 {
		t.Fatalf("сэргээсний дараа журнал: %v", row)
	}
	if _, ok := j["cells"].(map[string]any)[A]; !ok {
		t.Fatal("сэргээсний дараа нүд буцаж гарах ёстой")
	}
	if ja := jmGet(t, srv, tt, cid, "archived=1"); jmLen(ja["students"]) != 0 || jmNum(ja["archived"]) != 0 {
		t.Fatalf("архив хоосон байх ёстой: %v", ja["students"])
	}
	// Архивлаагүйг сэргээх нь юу ч өөрчлөхгүй (200).
	jmPutOK(t, srv, "архивлаагүйг сэргээх", tt, cid, B, `{"archived":false}`)
}

// Эрх: нэвтрээгүй 401, суралцагч 403, өөр багш 404 (бүх 4 зам); хамтран заагч багш нэр/бүлэг/архив/нэр солихыг хийж
// чадна, устгал 403 (эзэмшигч л); багшийг журналд засахгүй; гишүүн биш 404; нэр, бүлэг суралцагчид хэзээ ч харагдахгүй.
func TestJournalAuth(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, tid := register(t, srv, "jauth_owner", "teacher")
	tc, coID := register(t, srv, "jauth_co", "teacher")
	tx, _ := register(t, srv, "jauth_other", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Эрхийн сургалт","price":0,"published":true}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true,"blocks":`+jmLessonBlocks+`}`)
	s1, uid := register(t, srv, "jauth_s", "student")
	jmEnroll(t, srv, s1, cid)
	_, outsider := register(t, srv, "jauth_out", "student")
	// Хамт хөгжүүлэх эрх → Course.Teachers.
	if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/share", tt, `{"who":"@jauth_co","mode":"edit"}`); code != http.StatusCreated {
		t.Fatalf("хамтран заагч нэмэх: %d %v", code, r)
	}
	if co, _ := st.CourseByID(context.Background(), cid); len(co.Teachers) != 1 || co.Teachers[0] != coID {
		t.Fatalf("Course.Teachers: %v", co.Teachers)
	}

	base := "/api/me/courses/" + cid + "/journal"
	routes := []struct{ method, path, body string }{
		{"GET", base, ""},
		{"PUT", base + "/students/" + uid, `{"alias":"X"}`},
		{"PUT", base + "/groups", `{"from":"5А","to":"5Б"}`},
		{"POST", base + "/students/" + uid + "/reset", `{"confirm":true}`},
	}
	for _, rt := range routes {
		if code, r := call(t, srv, rt.method, rt.path, "", rt.body); code != http.StatusUnauthorized {
			t.Fatalf("нэвтрээгүй %s %s: %d %v", rt.method, rt.path, code, r)
		}
		code, r := call(t, srv, rt.method, rt.path, s1, rt.body)
		jmFail(t, "суралцагч "+rt.method+" "+rt.path, code, r, 403, "энэ үйлдлийг зөвхөн багш хийх боломжтой")
		code, r = call(t, srv, rt.method, rt.path, tx, rt.body)
		jmFail(t, "өөр багш "+rt.method+" "+rt.path, code, r, 404, "мэдээлэл олдсонгүй — устгагдсан байж магадгүй")
	}
	code, r := call(t, srv, "GET", "/api/me/courses/nope/journal", tt, "")
	jmFail(t, "байхгүй сургалт", code, r, 404, "")
	if m := jmMetas(t, st, cid); len(m) != 0 {
		t.Fatalf("татгалзсан хүсэлтүүд мөр бичлээ: %v", m)
	}

	// Хамтран заагч: A, B, C болон нэр солих — зөвшөөрнө.
	if j := jmGet(t, srv, tc, cid, ""); jmRow(j, uid) == nil {
		t.Fatalf("хамтран заагч журнал харах: %v", j["students"])
	}
	jmPutOK(t, srv, "хамтран заагч нэр/бүлэг", tc, cid, uid, `{"alias":"НууцНэрЖ","group":"НууцБүлэгЖ"}`)
	jmRenameOK(t, srv, "хамтран заагч бүлгийн нэр солих", tc, cid, `{"from":"НууцБүлэгЖ","to":"НууцБүлэгЖ2"}`)
	jmPutOK(t, srv, "хамтран заагч архивлах", tc, cid, uid, `{"archived":true}`)
	// Устгал: хамтран заагчид 403 (архивласан, confirm-тэй ч), өгөгдөл хөндөгдөхгүй.
	code, r = jmReset(t, srv, tc, cid, uid, `{"confirm":true}`)
	jmFail(t, "хамтран заагч устгах", code, r, 403, "зөвхөн сургалтын эзэмшигч өгөгдөл устгах эрхтэй")
	code, r = jmReset(t, srv, tc, cid, uid, `{}`)
	jmFail(t, "хамтран заагч (confirm-гүй) — эзэмшигчийн шалгалт эхэнд", code, r, 403, "зөвхөн сургалтын эзэмшигч өгөгдөл устгах эрхтэй")
	if m := jmMetas(t, st, cid)[uid]; !m.Archived || m.ResetAt != nil || m.Alias != "НууцНэрЖ" {
		t.Fatalf("403-ийн дараа мөр өөрчлөгдсөн: %+v", m)
	}

	// Багшийг (эзэмшигч, хамтран заагч) журналд засахгүй; гишүүн биш 404.
	for _, who := range []string{tid, coID} {
		code, r = jmPut(t, srv, tt, cid, who, `{"alias":"Багш"}`)
		jmFail(t, "багшийг засах", code, r, 400, "сургалтын багшийг журналд засах боломжгүй")
		code, r = jmPut(t, srv, tc, cid, who, `{"archived":true}`)
		jmFail(t, "хамтран заагч багшийг архивлах", code, r, 400, "сургалтын багшийг журналд засах боломжгүй")
	}
	code, r = jmPut(t, srv, tt, cid, outsider, `{"group":"5А"}`)
	jmFail(t, "гишүүн биш", code, r, 404, "энэ суралцагч таны сургалтад байхгүй байна")

	// Нэр, бүлэг суралцагчид хэзээ ч харагдахгүй (архивтай ч, сэргээсэн ч).
	jmPutOK(t, srv, "сэргээх", tt, cid, uid, `{"archived":false}`)
	for _, p := range []string{"/api/courses/" + cid + "/access", "/api/courses/" + cid, "/api/courses/" + cid + "/lessons/" + lid,
		"/api/me", "/api/me/home", "/api/me/enrollments", "/api/me/notifications", "/api/courses/" + cid + "/lessons/" + lid + "/cloud/cld1"} {
		res, body := rawCall(t, srv, "GET", p, s1, nil)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("суралцагч %s: %d %s", p, res.StatusCode, body)
		}
		if s := string(body); strings.Contains(s, "НууцНэрЖ") || strings.Contains(s, "НууцБүлэгЖ") {
			t.Fatalf("суралцагчид %s хариунд нэр/бүлэг задарлаа: %s", p, s)
		}
	}

	// Эзэмшигч устгаж чадна (хамтран заагч архивласан ч).
	jmArchive(t, srv, tt, cid, uid)
	code, r = jmReset(t, srv, tt, cid, uid, `{"confirm":true}`)
	jmOK(t, "эзэмшигч устгах", code, r)
}

// Устгал (D) бүтэн урсгал: архивласан суралцагчийн энэ сургалт дахь бүх суралцахуйн өгөгдөл (§4.3 wipe — Memory дэмждэг
// бүгд) устаж, суралцагчид харагдах замаар баталгаажна; хориг, давтах эрх, оролдлогын тоо шинээр эхэлнэ; журналд «эхлээгүй»
// байдлаар, нэр/бүлэг/архивгүй буцаж гарна; багшийн самбар үлдэнэ.
func TestJournalResetFlow(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	tt, tid := register(t, srv, "jrf_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Бүтэн устгал","price":0,"published":true}`)
	l1 := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true,"blocks":`+jmLessonBlocks+`}`)
	l2 := jmLesson(t, srv, tt, cid, `{"title":"Шалгалт","is_free":true,"exam":{"pass_pct":50,"attempts":1},`+jmExamQuiz+`}`)
	l3 := jmLesson(t, srv, tt, cid, `{"title":"Даалгавар","is_free":true,"assignment":{"max_score":10}}`)
	l4 := jmLesson(t, srv, tt, cid, `{"title":"Хяналт","is_free":true}`)
	s1, uid := register(t, srv, "jrf_s", "student")
	jmEnroll(t, srv, s1, cid)
	lbase := func(lid string) string { return "/api/courses/" + cid + "/lessons/" + lid }

	// --- Өгөгдөл үүсгэх ---
	jmLearn(t, srv, s1, cid, l1) // явц, асуулт+лог, сесс, дүгнэлт, видео, тэмдэглэл, үүлэн үг
	// Цагаан самбар: багшийнх (main) + суралцагчийн хувийн самбар.
	board := lbase(l1) + "/board/brd1"
	if code, r := call(t, srv, "POST", board+"/ops", tt, `{"room":"main","cid":"tch01","ops":[{"t":"add","o":{"id":"teach1","type":"rect","left":1}}]}`); code != 200 {
		t.Fatalf("багшийн самбар: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", board+"/state", tt, `{"mode":"solo"}`); code != 200 {
		t.Fatalf("хувиараа горим: %d %v", code, r)
	}
	_, g := call(t, srv, "GET", board, s1, "")
	if g["room"] != "u:"+uid {
		t.Fatalf("хувийн самбар: %v", g)
	}
	if code, r := call(t, srv, "POST", board+"/ops", s1, `{"room":"u:`+uid+`","cid":"s1aaa","ops":[{"t":"add","o":{"id":"mine01","type":"textbox","text":"Сайн уу"}}]}`); code != 200 {
		t.Fatalf("хувийн самбарт бичих: %d %v", code, r)
	}
	// Шалгалт: 1 оролдлого → багш дахин нээв (+1, teacher_unblock үйл явдал).
	_, ex := call(t, srv, "POST", lbase(l2)+"/exam/start", s1, "")
	aid := ex["attempt"].(map[string]any)["id"].(string)
	if code, r := call(t, srv, "POST", lbase(l2)+"/exam/submit", s1, `{"attempt_id":"`+aid+`","answers":{"q001":{"answer":[0]}}}`); code != 200 {
		t.Fatalf("шалгалт илгээх: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", "/api/me/students/"+uid+"/unblock", tt, `{"lesson_id":"`+l2+`"}`); code != 200 || jmNum(r["unblocked"]) != 1 {
		t.Fatalf("шалгалт дахин нээх: %d %v", code, r)
	}
	// Даалгавар + дүн.
	if code, r := call(t, srv, "POST", lbase(l3)+"/submit", s1, `{"text":"Миний ажил"}`); code != 200 {
		t.Fatalf("даалгавар: %d %v", code, r)
	}
	if code, r := call(t, srv, "PUT", lbase(l3)+"/submissions/"+uid, tt, `{"score":8,"feedback":"Сайн"}`); code != 200 {
		t.Fatalf("дүгнэх: %d %v", code, r)
	}
	// Хяналт: auto_block → хичээл хаагдана.
	_, ss := call(t, srv, "POST", "/api/activity/start", s1, `{"course_id":"`+cid+`","lesson_id":"`+l4+`","kind":"lesson"}`)
	call(t, srv, "POST", "/api/activity/beat", s1, `{"session_id":"`+ss["session_id"].(string)+`","active":3,"events":[{"type":"auto_block","detail":"5 удаа гарсан"}],"end":"auto_block"}`)
	if code, _ := call(t, srv, "GET", lbase(l4), s1, ""); code != http.StatusLocked {
		t.Fatalf("хаагдсан хичээл 423 байх ёстой: %d", code)
	}
	// Давтах сануулгын тэмдэг.
	if got, _ := st.ClaimReviews(ctx, []store.ReviewItem{{UserID: uid, CourseID: cid, LessonID: l1, Stage: 1}}); len(got) != 1 {
		t.Fatalf("сануулгын тэмдэг: %v", got)
	}

	// --- Устгахаас өмнөх байдал ---
	a := jmAccess(t, srv, s1, cid) // эрэмбийн оноог хадгална (autoAward)
	if jmNum(a["done"]) < 2 || a["blocks"].(map[string]any)[l4] == nil || jmNum(a["rank"].(map[string]any)["points"]) <= 0 {
		t.Fatalf("устгахаас өмнө: done=%v blocks=%v rank=%v", a["done"], a["blocks"], a["rank"])
	}
	if pts, _ := st.RankPointsSum(ctx, uid); pts <= 0 {
		t.Fatalf("эрэмбийн оноо хадгалагдаагүй: %d", pts)
	}
	_, ei := call(t, srv, "GET", lbase(l2)+"/exam", s1, "")
	if jmLen(ei["attempts"]) != 1 || jmNum(ei["left"]) != 1 {
		t.Fatalf("устгахаас өмнө шалгалт: %v", ei)
	}
	d := jmDetail(t, srv, tt, cid, uid)
	for _, k := range []string{"sessions", "exams", "quiz_logs", "reflections", "watches"} {
		if jmLen(d[k]) == 0 {
			t.Fatalf("устгахаас өмнө шинжилгээнд %s алга: %v", k, d[k])
		}
	}
	jmGet(t, srv, tt, cid, "")                 // журналыг кэшлэнэ
	if jmCloudMine(t, srv, s1, cid, l1) != 1 { // үүлэн үгийн нэгтгэл (2 сек кэш) устгалын өмнөх агшинд ачаалагдсан
		t.Fatal("устгахаас өмнө үүлэн үг алга")
	}

	// --- Архивлаад устгах ---
	jmPutOK(t, srv, "нэр/бүлэг", tt, cid, uid, `{"alias":"Бат","group":"5А"}`)
	jmArchive(t, srv, tt, cid, uid)
	before := time.Now().Add(-time.Second)
	code, r := jmReset(t, srv, tt, cid, uid, `{"confirm":true}`)
	jmOK(t, "устгах", code, r)
	ra, _ := r["reset_at"].(string)
	if ts, err := time.Parse(time.RFC3339Nano, ra); r["ok"] != true || err != nil || ts.Before(before) || time.Since(ts) > time.Minute {
		t.Fatalf("устгалын хариу: %v", r)
	}
	if jmCounts(r) != "groups=[] ungrouped=1 archived=0" {
		t.Fatalf("устгалын хариуны тоо: %s", jmCounts(r))
	}
	// Эрэмбийн оноо (сургалтын) устсан — /access дахин тооцохоос өмнө шалгана.
	if pts, _ := st.RankPointsSum(ctx, uid); pts != 0 {
		t.Fatalf("rank_points устаагүй: %d", pts)
	}
	m := jmMetas(t, st, cid)[uid]
	if m.Alias != "" || m.Group != "" || m.Archived || m.ArchivedAt != nil || m.ResetAt == nil || m.ResetBy != tid {
		t.Fatalf("устгасны дараах мөр: %+v", m)
	}

	// Журнал: «эхлээгүй», нэр/бүлэг/архивгүй, нүдгүй (суралцагч хичээл нээж явц үүсгэхээс өмнө шалгана).
	j := jmGet(t, srv, tt, cid, "")
	row := jmRow(j, uid)
	if row == nil || row["enrolled"] != true || jmNum(row["done"]) != 0 || jmNum(row["active"]) != 0 || row["alias"] != nil || row["group"] != nil || row["archived"] != nil || row["name"] != "jrf_s" {
		t.Fatalf("устгасны дараа журналын мөр: %v", row)
	}
	if _, ok := j["cells"].(map[string]any)[uid]; ok {
		t.Fatalf("устгасны дараа нүд үлдлээ: %v", j["cells"])
	}
	if jmCounts(j) != "groups=[] ungrouped=1 archived=0" {
		t.Fatalf("журналын тоо: %s", jmCounts(j))
	}

	// --- Суралцагчид харагдах замаар ---
	a = jmAccess(t, srv, s1, cid)
	if a["enrolled"] != true || jmNum(a["done"]) != 0 || len(a["progress"].(map[string]any)) != 0 || len(a["blocks"].(map[string]any)) != 0 || jmNum(a["rank"].(map[string]any)["points"]) != 0 {
		t.Fatalf("устгасны дараа access: enrolled=%v done=%v progress=%v blocks=%v rank=%v", a["enrolled"], a["done"], a["progress"], a["blocks"], a["rank"])
	}
	if code, r := call(t, srv, "GET", lbase(l4), s1, ""); code != 200 {
		t.Fatalf("хориг (activity_events + кэш) арилах ёстой: %d %v", code, r)
	}
	_, ei = call(t, srv, "GET", lbase(l2)+"/exam", s1, "")
	if jmLen(ei["attempts"]) != 0 || jmNum(ei["left"]) != 1 {
		t.Fatalf("шалгалт эхнээс (оролдлого, дахин нээсэн эрх устсан): attempts=%v left=%v", ei["attempts"], ei["left"])
	}
	if _, info := call(t, srv, "GET", lbase(l3)+"/assignment", s1, ""); info["submission"] != nil {
		t.Fatalf("даалгаврын хариу, дүн устах ёстой: %v", info["submission"])
	}
	if n := jmHighlights(t, srv, s1, cid, l1); n != 0 {
		t.Fatalf("тэмдэглэл устах ёстой: %d", n)
	}
	// Сангаас устсан ч HTTP хариу s.clouds (2 сек) кэшээс ирж болно — хоёуланг нь шалгана (Errorf: бусад шалгалт үргэлжилнэ).
	if cw, _ := st.CloudWords(ctx, l1, "cld1"); len(cw[uid]) != 0 {
		t.Errorf("cloud_answers сангаас устах ёстой: %v", cw)
	}
	if code, cl := call(t, srv, "GET", lbase(l1)+"/cloud/cld1", s1, ""); code != 200 || jmLen(cl["mine"]) != 0 || jmNum(cl["people"]) != 0 {
		t.Errorf("үүлэн үг устах ёстой (суралцагчид устгалын дараа шууд харагдах хариу): %d %v", code, cl)
	}
	_, g = call(t, srv, "GET", board, s1, "")
	if objs := fmt.Sprint(g["objects"]); g["room"] != "u:"+uid || strings.Contains(objs, "mine01") || !strings.Contains(objs, "teach1") {
		t.Fatalf("хувийн самбар шинээр (багшийнхаас) эхлэх ёстой: %v", g)
	}
	if _, g := call(t, srv, "GET", board+"?peek=1&room=main", tt, ""); !strings.Contains(fmt.Sprint(g["objects"]), "teach1") {
		t.Fatalf("багшийн самбар (main) үлдэх ёстой: %v", g)
	}
	d = jmDetail(t, srv, tt, cid, uid)
	for _, k := range []string{"sessions", "exams", "quiz_logs", "reflections", "watches"} {
		if jmLen(d[k]) != 0 {
			t.Fatalf("устгасны дараа шинжилгээнд %s үлдлээ: %v", k, d[k])
		}
	}
	if got, _ := st.ClaimReviews(ctx, []store.ReviewItem{{UserID: uid, CourseID: cid, LessonID: l1, Stage: 1}}); len(got) != 1 {
		t.Fatalf("review_reminders тэмдэг устах ёстой (дахин claim болно): %v", got)
	}
	// Шинээр эхэлж чадна.
	if code, r := call(t, srv, "POST", lbase(l2)+"/exam/start", s1, ""); code != 200 && code != 201 {
		t.Fatalf("шалгалтыг дахин эхлэх: %d %v", code, r)
	}
}

// Устгал үлдээх ёстойг хадгална (§4.3 keep): элсэлт, дангаар худалдсан хичээл, захиалга, төлсөн хоцролт/оролцооны эрх
// (сэргээгдэнэ — төлөгдөөгүйнх үгүй), нууц үгээр нээсэн, шууд хичээлийн эрх, эрэмбийн түвшин, бэлгийн хайрцаг,
// хэлэлцүүлгийн сэтгэгдэл, лайк, мэдэгдэл.
func TestJournalResetKeepsPurchases(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	tt, tid := register(t, srv, "jrk_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Худалдан авалт","price":0,"published":true}`)
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	lFree := jmLesson(t, srv, tt, cid, `{"title":"Үнэгүй","is_free":true,"discussion":true,"blocks":`+jmLessonBlocks+`}`)
	lPaid := jmLesson(t, srv, tt, cid, `{"title":"Төлбөртэй","price":5000}`)
	lLate := jmLesson(t, srv, tt, cid, `{"title":"Хоцорсон даалгавар","is_free":true,"assignment":{"resubmit":true,"due":{"at":"`+past+`","late":"paid","late_fee":2000}}}`)
	lUnpaid := jmLesson(t, srv, tt, cid, `{"title":"Төлөөгүй хоцролт","is_free":true,"assignment":{"due":{"at":"`+past+`","late":"paid","late_fee":1000}}}`)
	lFee := jmLesson(t, srv, tt, cid, `{"title":"Төлбөртэй шалгалт","is_free":true,`+jmExamQuiz+`,"exam":{"pass_pct":50,"due":{"fee":1500}}}`)
	s1, uid := register(t, srv, "jrk_s", "student")
	jmEnroll(t, srv, s1, cid)
	lbase := func(lid string) string { return "/api/courses/" + cid + "/lessons/" + lid }
	pay := func(path string) string {
		t.Helper()
		code, o := call(t, srv, "POST", path, s1, "")
		ord, _ := o["order"].(map[string]any)
		if code != http.StatusCreated || ord == nil {
			t.Fatalf("захиалга %s: %d %v", path, code, o)
		}
		oid := ord["id"].(string)
		if code, r := call(t, srv, "POST", "/api/orders/"+oid+"/dev-pay", s1, ""); code != 200 {
			t.Fatalf("төлбөр: %d %v", code, r)
		}
		return oid
	}
	oPaid := pay(lbase(lPaid) + "/buy")
	oLate := pay(lbase(lLate) + "/late-pay")
	oFee := pay(lbase(lFee) + "/late-pay")
	if code, o := call(t, srv, "POST", lbase(lUnpaid)+"/late-pay", s1, ""); code != http.StatusCreated || o["order"] == nil {
		t.Fatalf("төлөгдөөгүй захиалга: %d %v", code, o)
	}
	// Шууд хичээл (төлбөртэй) худалдаж авах.
	m := &store.Meeting{TeacherID: tid, CourseID: cid, Title: "Давтлага", StartsAt: time.Now().Add(2 * time.Hour), DurationMin: 60, MeetURL: "https://meet.google.com/abc-defg-hij"}
	if err := st.CreateMeeting(ctx, m); err != nil {
		t.Fatal(err)
	}
	if code, r := call(t, srv, "PUT", "/api/me/meetings/"+m.ID, tt, `{"price":15000}`); code != 200 {
		t.Fatalf("шууд хичээлийн үнэ: %d %v", code, r)
	}
	oMeet := pay("/api/meetings/" + m.ID + "/buy")
	// Суралцахуйн өгөгдөл (устах ёстой) — худалдсан хичээлүүд дээр ч.
	jmLearn(t, srv, s1, cid, lFree)
	if code, r := call(t, srv, "GET", lbase(lPaid), s1, ""); code != 200 {
		t.Fatalf("худалдсан хичээл: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", lbase(lLate)+"/submit", s1, `{"text":"Хоцорсон ч илгээлээ"}`); code != 200 {
		t.Fatalf("хоцорсон даалгавар (төлсөн): %d %v", code, r)
	}
	_, fe := call(t, srv, "POST", lbase(lFee)+"/exam/start", s1, "")
	fa, _ := fe["attempt"].(map[string]any)
	if fa == nil {
		t.Fatalf("төлбөртэй шалгалт эхлэх: %v", fe)
	}
	call(t, srv, "POST", lbase(lFee)+"/exam/submit", s1, `{"attempt_id":"`+fa["id"].(string)+`","answers":{"q001":{"answer":[0]}}}`)
	// Хэлэлцүүлэг, лайк.
	if code, r := call(t, srv, "POST", lbase(lFree)+"/discussion", s1, `{"body":"Энэ хэсгийг тайлбарлана уу?"}`); code != http.StatusCreated {
		t.Fatalf("сэтгэгдэл: %d %v", code, r)
	}
	if _, lk := call(t, srv, "POST", lbase(lFree)+"/like", s1, `{"target":"lesson"}`); lk["liked"] != true {
		t.Fatalf("лайк: %v", lk)
	}
	// Эрэмбийн түвшин, бэлгийн хайрцаг (rank_levels, box_opens).
	if err := st.SetUserRankLevel(ctx, uid, 9); err != nil {
		t.Fatal(err)
	}
	if err := st.AddBoxOpen(ctx, &store.BoxOpen{ID: "box1", UserID: uid, Box: "gold", Animal: 3, Cost: 10, At: time.Now()}); err != nil {
		t.Fatal(err)
	}
	notifsBefore := len(jmNotifs(t, srv, s1))
	if notifsBefore == 0 {
		t.Fatal("суралцагчид өмнө нь мэдэгдэл (худалдан авалт) ирсэн байх ёстой")
	}

	jmArchive(t, srv, tt, cid, uid)
	code, r := jmReset(t, srv, tt, cid, uid, `{"confirm":true}`)
	jmOK(t, "устгах", code, r)

	// Элсэлт, худалдсан хичээл.
	a := jmAccess(t, srv, s1, cid)
	if a["enrolled"] != true || jmNum(a["done"]) != 0 {
		t.Fatalf("элсэлт хэвээр, явц 0: %v %v", a["enrolled"], a["done"])
	}
	if bought := fmt.Sprint(a["lessons"]); !strings.Contains(bought, lPaid) {
		t.Fatalf("lesson_access хэвээр байх ёстой: %v", a["lessons"])
	}
	if code, r := call(t, srv, "GET", lbase(lPaid), s1, ""); code != 200 {
		t.Fatalf("худалдсан хичээл нээлттэй хэвээр: %d %v", code, r)
	}
	// Захиалгууд хэвээр (төлсөн).
	for _, oid := range []string{oPaid, oLate, oFee, oMeet} {
		code, o := call(t, srv, "GET", "/api/orders/"+oid, s1, "")
		if code != 200 || !strings.Contains(fmt.Sprint(o), "paid") {
			t.Fatalf("захиалга %s: %d %v", oid, code, o)
		}
	}
	// Төлсөн хоцролтын эрх сэргэсэн: хариу устсан ч дахин илгээхэд 402 биш.
	_, info := call(t, srv, "GET", lbase(lLate)+"/assignment", s1, "")
	if info["submission"] != nil {
		t.Fatalf("даалгаврын хариу устах ёстой: %v", info["submission"])
	}
	if due, _ := info["due"].(map[string]any); due == nil || due["late"] != true || due["need_pay"] == true {
		t.Fatalf("төлсөн хоцролтын эрх сэргэх ёстой: %v", info["due"])
	}
	if code, r := call(t, srv, "POST", lbase(lLate)+"/submit", s1, `{"text":"Дахин илгээв"}`); code != 200 {
		t.Fatalf("төлсөн хоцролттой даалгаврыг дахин илгээх: %d %v", code, r)
	}
	// Төлөөгүй хоцролтын эрх сэргээгдэхгүй.
	if code, r := call(t, srv, "POST", lbase(lUnpaid)+"/submit", s1, `{"text":"x y z"}`); code != http.StatusPaymentRequired {
		t.Fatalf("төлөгдөөгүй захиалгаар эрх олгох ёсгүй: %d %v", code, r)
	}
	// Оролцооны төлбөрийн эрх сэргэсэн: оролдлого устсан, дахин эхэлнэ (402 биш).
	_, ei := call(t, srv, "GET", lbase(lFee)+"/exam", s1, "")
	if jmLen(ei["attempts"]) != 0 {
		t.Fatalf("шалгалтын оролдлого устах ёстой: %v", ei["attempts"])
	}
	if code, r := call(t, srv, "POST", lbase(lFee)+"/exam/start", s1, ""); code != 200 && code != 201 {
		t.Fatalf("төлсөн оролцооны эрх сэргэх ёстой: %d %v", code, r)
	}
	// Шууд хичээлийн эрх.
	_, mt := call(t, srv, "GET", "/api/courses/"+cid+"/meetings", s1, "")
	if mm := mt["meetings"].([]any)[0].(map[string]any); mm["access"] != true || mm["meet_url"] == nil {
		t.Fatalf("meeting_access хэвээр байх ёстой: %v", mm)
	}
	// Хэлэлцүүлэг, лайк хэвээр.
	_, dsc := call(t, srv, "GET", lbase(lFree)+"/discussion", s1, "")
	if jmNum(dsc["count"]) != 1 || dsc["liked"] != true || jmNum(dsc["likes"]) != 1 {
		t.Fatalf("сэтгэгдэл, лайк хэвээр байх ёстой: %v", dsc)
	}
	// Эрэмбийн түвшин, хайрцаг хэвээр.
	if lvl, _ := st.UserRankLevel(ctx, uid); lvl != 9 {
		t.Fatalf("rank_levels хэвээр байх ёстой: %d", lvl)
	}
	if bs, _ := st.BoxOpens(ctx, uid); len(bs) != 1 {
		t.Fatalf("box_opens хэвээр байх ёстой: %v", bs)
	}
	// Мэдэгдлүүд хэвээр (+ устгалын мэдэгдэл).
	if n := len(jmNotifs(t, srv, s1)); n != notifsBefore+1 {
		t.Fatalf("мэдэгдэл: өмнө %d, одоо %d (+1 course_reset хүлээсэн)", notifsBefore, n)
	}

	// Нууц үгтэй сургалт: course_unlocks хэвээр — устгасны дараа ч бүх хичээл нээлттэй, дахин нууц үг шаардахгүй.
	lcid := jmCourse(t, srv, tt, `{"title":"Нууц үгтэй","price":0,"published":true,"access_code":"Анги-10А"}`)
	ll := jmLesson(t, srv, tt, lcid, `{"title":"Нууц хичээл","is_free":false,"price":0}`)
	if code, r := call(t, srv, "POST", "/api/courses/"+lcid+"/unlock", s1, `{"code":"Анги-10А"}`); code != http.StatusCreated {
		t.Fatalf("нууц үгээр нээх: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", "/api/courses/"+lcid+"/lessons/"+ll+"/complete", s1, ""); code != 200 {
		t.Fatalf("нууц сургалтын хичээл дуусгах: %d %v", code, r)
	}
	jmArchive(t, srv, tt, lcid, uid)
	code, r = jmReset(t, srv, tt, lcid, uid, `{"confirm":true}`)
	jmOK(t, "нууц сургалтад устгах", code, r)
	la := jmAccess(t, srv, s1, lcid)
	if la["all"] != true || la["need_code"] != false || la["enrolled"] != true || jmNum(la["done"]) != 0 {
		t.Fatalf("course_unlocks хэвээр, явц 0: all=%v need_code=%v enrolled=%v done=%v", la["all"], la["need_code"], la["enrolled"], la["done"])
	}
	if code, r := call(t, srv, "GET", "/api/courses/"+lcid+"/lessons/"+ll, s1, ""); code != 200 {
		t.Fatalf("нууц сургалтын хичээл нээлттэй хэвээр: %d %v", code, r)
	}
}

// Устгалын хүрээ: зөвхөн (энэ сургалт, энэ суралцагч). Нэг сургалтын өөр суралцагч, нэг суралцагчийн өөр сургалт (нэг
// багшийнх ба өөр багшийнх) — явц, тэмдэглэл, үүлэн үг, самбар, шалгалт, шинжилгээ, эрэмбийн оноо, сануулга, журналын
// мэдээлэл бүгд хэвээр.
func TestJournalResetScope(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	ctx := context.Background()
	tt, _ := register(t, srv, "jrs_t", "teacher")
	t2, _ := register(t, srv, "jrs_t2", "teacher")
	c1 := jmCourse(t, srv, tt, `{"title":"Нэг","price":0,"published":true}`)
	c2 := jmCourse(t, srv, tt, `{"title":"Хоёр","price":0,"published":true}`)
	d1 := jmCourse(t, srv, t2, `{"title":"Өөр багшийн","price":0,"published":true}`)
	l1 := jmLesson(t, srv, tt, c1, `{"title":"Нэг","is_free":true,"blocks":`+jmLessonBlocks+`}`)
	e1 := jmLesson(t, srv, tt, c1, `{"title":"Шалгалт","is_free":true,"exam":{"pass_pct":50,"attempts":3},`+jmExamQuiz+`}`)
	m1 := jmLesson(t, srv, tt, c2, `{"title":"Хоёр","is_free":true,"blocks":`+jmLessonBlocks+`}`)
	n1 := jmLesson(t, srv, t2, d1, `{"title":"Гурав","is_free":true,"blocks":`+jmLessonBlocks+`}`)
	s1, u1 := register(t, srv, "jrs_s1", "student")
	s2, u2 := register(t, srv, "jrs_s2", "student")
	for _, x := range []struct{ tok, cid string }{{s1, c1}, {s2, c1}, {s1, c2}, {s1, d1}} {
		jmEnroll(t, srv, x.tok, x.cid)
	}
	board := "/api/courses/" + c1 + "/lessons/" + l1 + "/board/brd1"
	call(t, srv, "POST", board+"/state", tt, `{"mode":"solo"}`)
	type pair struct{ tok, uid, cid, lid, teacher string }
	pairs := []pair{{s1, u1, c1, l1, tt}, {s2, u2, c1, l1, tt}, {s1, u1, c2, m1, tt}, {s1, u1, d1, n1, t2}}
	for _, p := range pairs {
		jmLearn(t, srv, p.tok, p.cid, p.lid)
	}
	for _, x := range []struct{ tok, uid, obj string }{{s1, u1, "own_s1"}, {s2, u2, "own_s2"}} {
		call(t, srv, "GET", board, x.tok, "")
		if code, r := call(t, srv, "POST", board+"/ops", x.tok, `{"room":"u:`+x.uid+`","cid":"x`+x.obj[4:]+`aa","ops":[{"t":"add","o":{"id":"`+x.obj+`","type":"rect"}}]}`); code != 200 {
			t.Fatalf("хувийн самбар %s: %d %v", x.obj, code, r)
		}
	}
	for _, tok := range []string{s1, s2} {
		_, ex := call(t, srv, "POST", "/api/courses/"+c1+"/lessons/"+e1+"/exam/start", tok, "")
		call(t, srv, "POST", "/api/courses/"+c1+"/lessons/"+e1+"/exam/submit", tok, `{"attempt_id":"`+ex["attempt"].(map[string]any)["id"].(string)+`","answers":{}}`)
	}
	for _, p := range pairs {
		jmAccess(t, srv, p.tok, p.cid)
	}
	// Эрэмбийн оноог тодорхой утгаар (дараа нь /access дуудахгүй).
	for _, x := range []struct {
		uid, cid string
		pts      int
	}{{u1, c1, 40}, {u1, c2, 30}, {u1, d1, 20}, {u2, c1, 25}} {
		if _, err := st.SaveRankPoints(ctx, x.uid, x.cid, x.pts); err != nil {
			t.Fatal(err)
		}
	}
	reviews := []store.ReviewItem{{UserID: u1, CourseID: c1, LessonID: l1, Stage: 1}, {UserID: u2, CourseID: c1, LessonID: l1, Stage: 1},
		{UserID: u1, CourseID: c2, LessonID: m1, Stage: 1}, {UserID: u1, CourseID: d1, LessonID: n1, Stage: 1}}
	if got, _ := st.ClaimReviews(ctx, reviews); len(got) != 4 {
		t.Fatalf("сануулга: %v", got)
	}
	// Журналын мэдээлэл: өөр суралцагч, өөр сургалтад.
	jmPutOK(t, srv, "s2 нэр", tt, c1, u2, `{"alias":"Дорж","group":"5А"}`)
	jmPutOK(t, srv, "s1 c2 нэр", tt, c2, u1, `{"alias":"Бат2","group":"6Б","archived":true}`)

	jmArchive(t, srv, tt, c1, u1)
	code, r := jmReset(t, srv, tt, c1, u1, `{"confirm":true}`)
	jmOK(t, "устгах", code, r)
	if jmCounts(r) != "groups=[5А:1] ungrouped=1 archived=0" {
		t.Fatalf("устгалын хариуны тоо: %s", jmCounts(r))
	}

	// Эрэмбийн оноо: зөвхөн (u1, c1) устана.
	if p, _ := st.RankPointsSum(ctx, u1); p != 50 {
		t.Fatalf("u1 эрэмбийн оноо 30+20 хүлээсэн: %d", p)
	}
	if p, _ := st.RankPointsSum(ctx, u2); p != 25 {
		t.Fatalf("u2 эрэмбийн оноо хөндөгдлөө: %d", p)
	}
	// Сануулга: зөвхөн (u1, c1) дахин claim болно.
	if got, _ := st.ClaimReviews(ctx, reviews); len(got) != 1 || got[0].UserID != u1 || got[0].CourseID != c1 {
		t.Fatalf("зөвхөн устгасан хосын сануулга чөлөөлөгдөх ёстой: %+v", got)
	}

	// Устгасан хос.
	if a := jmAccess(t, srv, s1, c1); jmNum(a["done"]) != 0 {
		t.Fatalf("(u1, c1) явц устах ёстой: %v", a["done"])
	}
	if jmHighlights(t, srv, s1, c1, l1) != 0 {
		t.Fatal("(u1, c1) тэмдэглэл устах ёстой")
	}
	if cw, _ := st.CloudWords(ctx, l1, "cld1"); len(cw[u1]) != 0 || len(cw[u2]) != 1 {
		t.Errorf("cloud_answers сан: u1 устаж, u2 үлдэх ёстой: %v", cw)
	}
	if n := jmCloudMine(t, srv, s1, c1, l1); n != 0 {
		t.Errorf("(u1, c1) үүлэн үг устах ёстой (HTTP хариу): mine=%d", n)
	}
	// Хөндөгдөх ёсгүй хосууд.
	for _, p := range pairs[1:] {
		what := fmt.Sprintf("(%s, %s)", p.uid[:6], p.cid[:6])
		if a := jmAccess(t, srv, p.tok, p.cid); jmNum(a["done"]) != 1 {
			t.Fatalf("%s явц хөндөгдлөө: %v", what, a["done"])
		}
		if jmHighlights(t, srv, p.tok, p.cid, p.lid) != 1 || jmCloudMine(t, srv, p.tok, p.cid, p.lid) != 1 {
			t.Fatalf("%s тэмдэглэл/үүлэн үг хөндөгдлөө", what)
		}
		d := jmDetail(t, srv, p.teacher, p.cid, p.uid)
		for _, k := range []string{"sessions", "quiz_logs", "reflections", "watches"} {
			if jmLen(d[k]) == 0 {
				t.Fatalf("%s шинжилгээнд %s алга болсон", what, k)
			}
		}
	}
	if _, ei := call(t, srv, "GET", "/api/courses/"+c1+"/lessons/"+e1+"/exam", s2, ""); jmLen(ei["attempts"]) != 1 || jmNum(ei["left"]) != 2 {
		t.Fatalf("u2-ийн шалгалтын оролдлого хөндөгдлөө: %v", ei)
	}
	if _, ei := call(t, srv, "GET", "/api/courses/"+c1+"/lessons/"+e1+"/exam", s1, ""); jmLen(ei["attempts"]) != 0 || jmNum(ei["left"]) != 3 {
		t.Fatalf("u1-ийн шалгалтын оролдлого устах ёстой: %v", ei)
	}
	if _, g := call(t, srv, "GET", board+"?peek=1&room=u:"+u2, tt, ""); !strings.Contains(fmt.Sprint(g["objects"]), "own_s2") {
		t.Fatalf("u2-ийн хувийн самбар хөндөгдлөө: %v", g)
	}
	if _, g := call(t, srv, "GET", board, s1, ""); strings.Contains(fmt.Sprint(g["objects"]), "own_s1") {
		t.Fatalf("u1-ийн хувийн самбар устах ёстой: %v", g)
	}
	// Журнал: u2 нүд, нэр, бүлэгтэйгээ; c2-д u1 архивтай, нэртэй хэвээр.
	j := jmGet(t, srv, tt, c1, "")
	if row := jmRow(j, u2); row == nil || row["alias"] != "Дорж" || row["group"] != "5А" || jmNum(row["done"]) != 1 {
		t.Fatalf("u2-ийн журналын мөр: %v", row)
	}
	if _, ok := j["cells"].(map[string]any)[u2]; !ok {
		t.Fatal("u2-ийн нүд алга болсон")
	}
	if mc2 := jmMetas(t, st, c2)[u1]; mc2.Alias != "Бат2" || mc2.Group != "6Б" || !mc2.Archived || mc2.ResetAt != nil {
		t.Fatalf("өөр сургалтын журналын мэдээлэл хөндөгдлөө: %+v", mc2)
	}
	if row := jmRow(jmGet(t, srv, tt, c2, "archived=1"), u1); row == nil || jmNum(row["done"]) != 1 {
		t.Fatalf("c2-ийн журналд u1 өгөгдөлтэйгээ байх ёстой: %v", row)
	}
	if row := jmRow(jmGet(t, srv, t2, d1, ""), u1); row == nil || jmNum(row["done"]) != 1 {
		t.Fatalf("өөр багшийн журналд u1 өгөгдөлтэйгээ байх ёстой: %v", row)
	}
}

// Идэвхтэй сесс, шалгалтын оролдлого: устгасны дараа хуучин сессийн цохилт 404 «сесс олдсонгүй» (клиент шинээр эхэлнэ),
// хуучин оролдлогын илгээлт 404; шинэ сесс, оролдлого ажиллана; идэвхтэй хугацаа 0-ээс эхэлнэ.
func TestJournalResetActiveSession(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jra_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Идэвхтэй","price":0,"published":true}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	eid := jmLesson(t, srv, tt, cid, `{"title":"Шалгалт","is_free":true,"exam":{"pass_pct":50,"attempts":1},`+jmExamQuiz+`}`)
	s1, uid := register(t, srv, "jra_s", "student")
	jmEnroll(t, srv, s1, cid)
	_, ss := call(t, srv, "POST", "/api/activity/start", s1, `{"course_id":"`+cid+`","lesson_id":"`+lid+`","kind":"lesson"}`)
	sid := ss["session_id"].(string)
	if code, r := call(t, srv, "POST", "/api/activity/beat", s1, `{"session_id":"`+sid+`","active":4}`); code != 200 {
		t.Fatalf("цохилт: %d %v", code, r)
	}
	_, ex := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+eid+"/exam/start", s1, "")
	aid := ex["attempt"].(map[string]any)["id"].(string)
	if row := jmRow(jmGet(t, srv, tt, cid, ""), uid); row == nil || jmNum(row["active"]) <= 0 {
		t.Fatalf("устгахаас өмнө идэвхтэй хугацаа: %v", row)
	}

	jmArchive(t, srv, tt, cid, uid)
	jmResetOK(t, srv, "устгах", tt, cid, uid, `{"confirm":true}`)

	code, r := call(t, srv, "POST", "/api/activity/beat", s1, `{"session_id":"`+sid+`","active":4}`)
	jmFail(t, "хуучин сессийн цохилт", code, r, 404, "сесс олдсонгүй")
	code, r = call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+eid+"/exam/submit", s1, `{"attempt_id":"`+aid+`","answers":{"q001":{"answer":[0]}}}`)
	jmFail(t, "хуучин оролдлогын илгээлт", code, r, 404, "оролдлого олдсонгүй")
	if row := jmRow(jmGet(t, srv, tt, cid, ""), uid); row == nil || jmNum(row["active"]) != 0 {
		t.Fatalf("устгасны дараа идэвхтэй хугацаа 0: %v", row)
	}
	// Шинэ сесс, шинэ оролдлого.
	code, ss = call(t, srv, "POST", "/api/activity/start", s1, `{"course_id":"`+cid+`","lesson_id":"`+lid+`","kind":"lesson"}`)
	if code != 200 || ss["session_id"] == sid {
		t.Fatalf("шинэ сесс: %d %v", code, ss)
	}
	if code, r := call(t, srv, "POST", "/api/activity/beat", s1, `{"session_id":"`+ss["session_id"].(string)+`","active":2}`); code != 200 {
		t.Fatalf("шинэ сессийн цохилт: %d %v", code, r)
	}
	code, ex = call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+eid+"/exam/start", s1, "")
	if (code != 200 && code != 201) || ex["attempt"].(map[string]any)["id"] == aid {
		t.Fatalf("шинэ оролдлого: %d %v", code, ex)
	}
}

// Дараалсан нээлт: эхний хичээлээ үзсэнээр нээгдсэн (10 минут кэштэй) хичээл устгасны дараа дахин түгжигдэнэ — явц
// (drip base) ба dripOpen кэш хоёулаа арилна; суралцагч эхнээс нь дахин нээнэ.
func TestJournalResetDrip(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jrd_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Дараалсан","price":0,"published":true,"drip":true}`)
	l1 := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	l2 := jmLesson(t, srv, tt, cid, `{"title":"Хоёр","price":1000,"unlock_after_h":0,"blocks":[{"id":"t001","type":"text","text":"Хоёр дахь хичээл"}]}`)
	s1, uid := register(t, srv, "jrd_s", "student")
	jmEnroll(t, srv, s1, cid)
	_, o := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+l2+"/buy", s1, "")
	call(t, srv, "POST", "/api/orders/"+o["order"].(map[string]any)["id"].(string)+"/dev-pay", s1, "")
	hl := "/api/courses/" + cid + "/lessons/" + l2 + "/highlights" // lessonForUser → dripOpen кэш
	if code, _ := call(t, srv, "GET", hl, s1, ""); code != http.StatusLocked {
		t.Fatalf("эхнийхээ үзээгүй бол 2 түгжээтэй: %d", code)
	}
	call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l1, s1, "")
	if code, r := call(t, srv, "GET", hl, s1, ""); code != 200 {
		t.Fatalf("эхнийхээ үзсэний дараа 2 нээгдэнэ (кэшлэгдэнэ): %d %v", code, r)
	}

	jmArchive(t, srv, tt, cid, uid)
	jmResetOK(t, srv, "устгах", tt, cid, uid, `{"confirm":true}`)

	if code, r := call(t, srv, "GET", hl, s1, ""); code != http.StatusLocked {
		t.Fatalf("устгасны дараа dripOpen кэш арилж 2 түгжигдэх ёстой: %d %v", code, r)
	}
	if code, r := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l2, s1, ""); code != http.StatusLocked || r["state"].(map[string]any)["reason"] != "prev" {
		t.Fatalf("устгасны дараа 2 «prev» шалтгаанаар түгжээтэй: %d %v", code, r)
	}
	if st := jmAccess(t, srv, s1, cid)["states"].(map[string]any)[l2].(map[string]any); st["open"] == true || st["reason"] != "prev" {
		t.Fatalf("access төлөв: %v", st)
	}
	// Эхнээс нь: 1-ийг үзвэл 2 дахин нээгдэнэ.
	call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+l1, s1, "")
	if code, r := call(t, srv, "GET", hl, s1, ""); code != 200 {
		t.Fatalf("дахин үзсэний дараа нээгдэнэ: %d %v", code, r)
	}
}

// Элсээгүй зочин (үнэгүй хичээл үзсэн): журналд гарна, архивлаж устгаж болно; устгасны дараа элсэлтгүй тул журнал,
// архивын жагсаалт, тоо (устгалын хариуных ч — кэшлэсэн нэгтгэлээр тооцсон) бүгдээс алга болж, дахин засах 404.
func TestJournalResetVisitorDisappears(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jrv_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Нээлттэй","price":0,"published":true}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true,"blocks":`+jmLessonBlocks+`}`)
	sv, vid := register(t, srv, "jrv_visitor", "student")
	se, eidU := register(t, srv, "jrv_member", "student")
	jmEnroll(t, srv, se, cid)
	jmLearn(t, srv, sv, cid, lid) // элсээгүй
	if a := jmAccess(t, srv, sv, cid); a["enrolled"] != false {
		t.Fatalf("зочин элсээгүй байх ёстой: %v", a["enrolled"])
	}
	j := jmGet(t, srv, tt, cid, "")
	if row := jmRow(j, vid); row == nil || row["enrolled"] != false || jmNum(row["done"]) != 1 {
		t.Fatalf("зочин журналд харагдах ёстой: %v", row)
	}
	if jmCounts(j) != "groups=[] ungrouped=2 archived=0" {
		t.Fatalf("тоо: %s", jmCounts(j))
	}
	jmPutOK(t, srv, "зочинд бүлэг", tt, cid, vid, `{"group":"Зочид"}`)
	jmArchive(t, srv, tt, cid, vid)
	jmGet(t, srv, tt, cid, "archived=1") // кэш: устгахаас өмнөх нэгтгэл

	code, r := jmReset(t, srv, tt, cid, vid, `{"confirm":true}`)
	jmOK(t, "зочныг устгах", code, r)
	if jmCounts(r) != "groups=[] ungrouped=1 archived=0" {
		t.Fatalf("устгалын хариуны тоонд зочин орох ёсгүй: %s", jmCounts(r))
	}
	j = jmGet(t, srv, tt, cid, "")
	if ids := jmIDs(j); ids[vid] || !ids[eidU] || jmNum(j["students_total"]) != 1 || jmCounts(j) != "groups=[] ungrouped=1 archived=0" {
		t.Fatalf("зочин журналаас алга болох ёстой: %v %s", ids, jmCounts(j))
	}
	if ja := jmGet(t, srv, tt, cid, "archived=1"); jmLen(ja["students"]) != 0 {
		t.Fatalf("зочин архиваас ч алга: %v", ja["students"])
	}
	if j := jmGet(t, srv, tt, cid, "q=jrv_visitor"); jmLen(j["students"]) != 0 {
		t.Fatalf("хайлтаар ч олдохгүй: %v", j["students"])
	}
	code, r = jmPut(t, srv, tt, cid, vid, `{"alias":"Зочин"}`)
	jmFail(t, "устгасан зочныг засах", code, r, 404, "энэ суралцагч таны сургалтад байхгүй байна")
}

// Устгал: мэдэгдэл ба ачааллаас хамгаалалт.
//   - Өгөгдөлтэй суралцагчид яг нэг course_reset мэдэгдэл (гарчиг, бие, /c/<id> холбоос); өөр хэнд ч үгүй.
//   - Эзэмшигч тус бүрт 3 устгал (ratelimit 1/60 сек, burst 3); 4 дэх нь 429 бөгөөд юу ч өөрчлөхгүй.
//   - confirm-гүй (400), архивлаагүй (409), саяхан устгасан (429 cooldown) хариу хязгаарт тооцогдохгүй — хязгаар
//     дууссан ч cooldown-ийн мессеж ирнэ (шалгалт хязгаараас өмнө).
//   - Нэг (сургалт, суралцагч)-ийг 10 минутын дотор дахин устгахгүй (429).
//   - Өөр эзэмшигчийн хязгаар тусдаа; өгөгдөлгүй суралцагчийн устгал амжилттай (архив, нэр арилна) ч мэдэгдэлгүй.
func TestJournalResetNotifiesAndLimits(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jrn_t", "teacher")
	t2, _ := register(t, srv, "jrn_t2", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Мэдэгдэлтэй","price":0,"published":true}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true}`)
	cid2 := jmCourse(t, srv, t2, `{"title":"Өөр эзэмшигч","price":0,"published":true}`)
	lid2 := jmLesson(t, srv, t2, cid2, `{"title":"Нэг","is_free":true}`)
	const (
		rateMsg     = "хэт олон хүсэлт ирлээ — түр хүлээгээд дахин оролдоно уу"
		cooldownMsg = "энэ суралцагчийн өгөгдлийг саяхан устгасан — 10 минутын дараа дахин оролдоно уу"
		notArchived = "эхлээд суралцагчийг архивлана уу — зөвхөн архивласан суралцагчийн өгөгдлийг устгана"
	)
	// 4 суралцагч (өгөгдөлтэй: хичээл үзэж, дуусгасан) cid-д; s1 cid2-т ч өгөгдөлтэй; s5 cid2-т зөвхөн элссэн (өгөгдөлгүй).
	var toks, uids [4]string
	for i := range toks {
		toks[i], uids[i] = register(t, srv, fmt.Sprintf("jrn_s%d", i+1), "student")
		jmEnroll(t, srv, toks[i], cid)
		if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/complete", toks[i], ""); code != 200 {
			t.Fatalf("дуусгах: %d %v", code, r)
		}
	}
	s1, u1 := toks[0], uids[0]
	jmEnroll(t, srv, s1, cid2)
	call(t, srv, "POST", "/api/courses/"+cid2+"/lessons/"+lid2+"/complete", s1, "")
	s5, u5 := register(t, srv, "jrn_s5", "student")
	jmEnroll(t, srv, s5, cid2)

	resetNotifs := func(tok, link string) []map[string]any {
		t.Helper()
		var out []map[string]any
		for _, n := range jmNotifs(t, srv, tok) {
			if n["type"] == "course_reset" && (link == "" || n["link"] == link) {
				out = append(out, n)
			}
		}
		return out
	}

	// Хязгаарт тооцогдохгүй хариунууд: confirm-гүй 400, архивлаагүй 409 (тус бүр 5 удаа — хуучин дүрмээр хязгаар дуусах байсан).
	for range 5 {
		code, r := jmReset(t, srv, tt, cid, u1, `{"confirm":false}`)
		jmFail(t, "confirm false", code, r, 400, "устгахыг баталгаажуулна уу")
		code, r = jmReset(t, srv, tt, cid, u1, `{"confirm":true}`)
		jmFail(t, "архивлаагүй", code, r, 409, notArchived)
	}
	for _, u := range uids {
		jmArchive(t, srv, tt, cid, u)
	}

	// 1 дэх устгал + яг тодорхой мэдэгдэл.
	jmResetOK(t, srv, "1 дэх устгал", tt, cid, u1, `{"confirm":true}`)
	got := resetNotifs(s1, "")
	if len(got) != 1 || got[0]["title"] != "🔄 Сургалтын явц шинээр эхэллээ" || got[0]["link"] != "/c/"+cid ||
		got[0]["body"] != "«Мэдэгдэлтэй» сургалтын явцыг багш тань шинэчиллээ. Хичээлээ эхнээс нь үзээрэй." {
		t.Fatalf("course_reset мэдэгдэл (яг нэг): %v", got)
	}
	for _, tok := range []string{toks[1], toks[2], toks[3], s5, tt, t2} {
		if n := resetNotifs(tok, ""); len(n) != 0 {
			t.Fatalf("өөр хүнд course_reset очих ёсгүй: %v", n)
		}
	}

	// Cooldown: s1-ийг дахин архивлаад устгах → 429 (3 удаа — хязгаарт тооцогдохгүй), юу ч өөрчлөгдөхгүй.
	jmArchive(t, srv, tt, cid, u1)
	for range 3 {
		code, r := jmReset(t, srv, tt, cid, u1, `{"confirm":true}`)
		jmFail(t, "саяхан устгасныг дахин", code, r, http.StatusTooManyRequests, cooldownMsg)
	}
	if m := jmMetas(t, st, cid)[u1]; !m.Archived {
		t.Fatalf("cooldown 429 үед архив хэвээр байх ёстой: %+v", m)
	}
	if n := resetNotifs(s1, ""); len(n) != 1 {
		t.Fatalf("cooldown 429 үед мэдэгдэл нэмэгдэх ёсгүй: %v", n)
	}

	// 2, 3 дахь устгал (өөр суралцагчид) амжилттай.
	jmResetOK(t, srv, "2 дахь устгал", tt, cid, uids[1], `{"confirm":true}`)
	jmResetOK(t, srv, "3 дахь устгал", tt, cid, uids[2], `{"confirm":true}`)
	for _, i := range []int{1, 2} {
		if n := resetNotifs(toks[i], "/c/"+cid); len(n) != 1 {
			t.Fatalf("s%d мэдэгдэл: %v", i+1, n)
		}
	}

	// 4 дэх → 429 хязгаар; юу ч өөрчлөгдөхгүй (архивтаа, явц, нэр хэвээр, мэдэгдэлгүй).
	jmPutOK(t, srv, "s4 нэр", tt, cid, uids[3], `{"alias":"Дөрөв"}`)
	code, r := jmReset(t, srv, tt, cid, uids[3], `{"confirm":true}`)
	jmFail(t, "4 дэх устгал", code, r, http.StatusTooManyRequests, rateMsg)
	if m := jmMetas(t, st, cid)[uids[3]]; !m.Archived || m.Alias != "Дөрөв" || m.ResetAt != nil {
		t.Fatalf("хязгаарын 429 үед мөр өөрчлөгдлөө: %+v", m)
	}
	if a := jmAccess(t, srv, toks[3], cid); jmNum(a["done"]) != 1 || a["enrolled"] != true {
		t.Fatalf("хязгаарын 429 үед өгөгдөл хөндөгдлөө: done=%v", a["done"])
	}
	if ja := jmGet(t, srv, tt, cid, "archived=1"); jmRow(ja, uids[3]) == nil {
		t.Fatal("хязгаарын 429 үед суралцагч архивтаа үлдэх ёстой")
	}
	if n := resetNotifs(toks[3], ""); len(n) != 0 {
		t.Fatalf("хязгаарын 429 үед мэдэгдэл очих ёсгүй: %v", n)
	}
	// Хязгаар дууссан ч cooldown болон архивлаагүйн шалгалт эхэнд: тус тусын мессеж.
	code, r = jmReset(t, srv, tt, cid, u1, `{"confirm":true}`)
	jmFail(t, "хязгаар дууссан үед cooldown", code, r, http.StatusTooManyRequests, cooldownMsg)
	code, r = jmReset(t, srv, tt, cid, uids[1], `{"confirm":true}`) // устгасны дараа архивгүй
	jmFail(t, "хязгаар дууссан үед архивлаагүй", code, r, 409, notArchived)

	// Өөр эзэмшигчийн хязгаар тусдаа: өгөгдөлтэй s1 (cid2) → мэдэгдэлтэй.
	jmArchive(t, srv, t2, cid2, u1)
	jmResetOK(t, srv, "өөр эзэмшигч устгах", t2, cid2, u1, `{"confirm":true}`)
	if n := resetNotifs(s1, "/c/"+cid2); len(n) != 1 || n[0]["body"] != "«Өөр эзэмшигч» сургалтын явцыг багш тань шинэчиллээ. Хичээлээ эхнээс нь үзээрэй." {
		t.Fatalf("өөр сургалтын мэдэгдэл: %v", n)
	}
	// Өгөгдөлгүй суралцагч: 200, архив/нэр арилж, reset тэмдэг тавигдана, мэдэгдэлгүй.
	jmPutOK(t, srv, "s5 архив, нэр, бүлэг", t2, cid2, u5, `{"archived":true,"alias":"Тав","group":"5А"}`)
	r = jmResetOK(t, srv, "өгөгдөлгүй устгал", t2, cid2, u5, `{"confirm":true}`)
	if r["ok"] != true || r["reset_at"] == nil {
		t.Fatalf("өгөгдөлгүй устгалын хариу: %v", r)
	}
	if m := jmMetas(t, st, cid2)[u5]; m.Archived || m.Alias != "" || m.Group != "" || m.ResetAt == nil {
		t.Fatalf("өгөгдөлгүй устгалын дараах мөр: %+v", m)
	}
	if row := jmRow(jmGet(t, srv, t2, cid2, ""), u5); row == nil || row["archived"] != nil || row["alias"] != nil || row["group"] != nil {
		t.Fatalf("өгөгдөлгүй суралцагч журналд энгийнээр буцах ёстой: %v", row)
	}
	if n := resetNotifs(s5, ""); len(n) != 0 {
		t.Fatalf("устгах өгөгдөлгүй үед мэдэгдэх ёсгүй: %v", n)
	}
}

// Урьдчилсан нөхцөл: архивлаагүй → 409 (өгөгдөл хөндөгдөхгүй); confirm алга/false/буруу төрөл/хоосон бие/үл мэдэх талбар
// → 400 (өгөгдөл хөндөгдөхгүй, архивтаа); амжилттай устгасны дараа дахин устгахад дахин архивлах шаардлагатай.
func TestJournalResetRequiresArchiveAndConfirm(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jrr_t", "teacher")
	cid := jmCourse(t, srv, tt, `{"title":"Нөхцөл","price":0,"published":true}`)
	lid := jmLesson(t, srv, tt, cid, `{"title":"Нэг","is_free":true,"blocks":`+jmLessonBlocks+`}`)
	s1, uid := register(t, srv, "jrr_s", "student")
	jmEnroll(t, srv, s1, cid)
	_, outsider := register(t, srv, "jrr_x", "student")
	jmLearn(t, srv, s1, cid, lid)
	jmPutOK(t, srv, "нэр", tt, cid, uid, `{"alias":"Бат"}`)
	const notArchived = "эхлээд суралцагчийг архивлана уу — зөвхөн архивласан суралцагчийн өгөгдлийг устгана"
	intact := func(what string) {
		t.Helper()
		if a := jmAccess(t, srv, s1, cid); jmNum(a["done"]) != 1 {
			t.Fatalf("%s: өгөгдөл хөндөгдөх ёсгүй: done=%v", what, a["done"])
		}
		if jmHighlights(t, srv, s1, cid, lid) != 1 {
			t.Fatalf("%s: тэмдэглэл хөндөгдөх ёсгүй", what)
		}
	}

	code, r := jmReset(t, srv, tt, cid, uid, `{"confirm":true}`)
	jmFail(t, "архивлаагүй", code, r, 409, notArchived)
	intact("архивлаагүй")
	if m := jmMetas(t, st, cid)[uid]; m.Alias != "Бат" || m.ResetAt != nil {
		t.Fatalf("409-ийн дараа мөр өөрчлөгдлөө: %+v", m)
	}

	jmArchive(t, srv, tt, cid, uid)
	for _, c := range []struct{ name, body, msg string }{
		{"хоосон объект", `{}`, "устгахыг баталгаажуулна уу"},
		{"confirm false", `{"confirm":false}`, "устгахыг баталгаажуулна уу"},
		{"confirm null", `{"confirm":null}`, "устгахыг баталгаажуулна уу"},
		{"confirm мөр", `{"confirm":"true"}`, ""},
		{"confirm тоо", `{"confirm":1}`, ""},
		{"хоосон бие", ``, ""},
		{"үл мэдэх талбар", `{"confirm":true,"force":true}`, ""},
		{"илүү өгөгдөл", `{"confirm":true}{"confirm":true}`, ""},
	} {
		code, r := jmReset(t, srv, tt, cid, uid, c.body)
		jmFail(t, c.name, code, r, 400, c.msg)
	}
	intact("400-ийн дараа")
	if m := jmMetas(t, st, cid)[uid]; !m.Archived || m.ResetAt != nil {
		t.Fatalf("400-ийн дараа архивтаа үлдэх ёстой: %+v", m)
	}

	jmResetOK(t, srv, "устгах", tt, cid, uid, `{"confirm":true}`)
	if a := jmAccess(t, srv, s1, cid); jmNum(a["done"]) != 0 {
		t.Fatalf("устгасны дараа явц 0: %v", a["done"])
	}
	code, r = jmReset(t, srv, tt, cid, uid, `{"confirm":true}`)
	jmFail(t, "дахин устгах (архивгүй)", code, r, 409, notArchived)
	code, r = jmReset(t, srv, tt, cid, outsider, `{"confirm":true}`)
	jmFail(t, "гишүүн биш (мөргүй)", code, r, 409, notArchived)
	if _, ok := jmMetas(t, st, cid)[outsider]; ok {
		t.Fatal("гишүүн бишид мөр бичигдлээ")
	}
}
