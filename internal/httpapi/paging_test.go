package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

// registerFrom — олон хэрэглэгч бүртгэх тест: IP тус бүрээс (бүртгэлийн хязгаар IP-гаар).
func registerFrom(t *testing.T, srvURL, username, role string, n int) (string, string) {
	t.Helper()
	body := `{"username":"` + username + `","email":"` + username + `@x.mn","password":"password1","role":"` + role + `","accept_terms":true}`
	req, _ := http.NewRequest("POST", srvURL+"/api/auth/register", strings.NewReader(body))
	req.Header.Set("X-Real-IP", fmt.Sprintf("10.9.%d.%d", n/250, n%250+1))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var r map[string]any
	_ = json.NewDecoder(res.Body).Decode(&r)
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("бүртгэл %s: %d %v", username, res.StatusCode, r)
	}
	u, tok := r["user"].(map[string]any), r["token"].(string)
	if tk := asRole(t, u["id"].(string), u["username"].(string), role); tk != "" {
		tok = tk
	}
	return tok, u["id"].(string)
}

// listCall — хуудасласан жагсаалт: массив бие + X-Total-Count толгой.
func listCall(t *testing.T, srvURL, path, token string) ([]map[string]any, int) {
	t.Helper()
	req, _ := http.NewRequest("GET", srvURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		t.Fatalf("%s: %d", path, res.StatusCode)
	}
	var items []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&items); err != nil {
		t.Fatalf("%s: массив биш: %v", path, err)
	}
	total, err := strconv.Atoi(res.Header.Get("X-Total-Count"))
	if err != nil {
		t.Fatalf("%s: X-Total-Count алга", path)
	}
	return items, total
}

// Олон суралцагчтай багш: жагсаалтууд хуудаслагдаж, нэрээр хайгдана; найз мөн эсэх хуудаснаас үл хамаарна.
func TestStudentsPaging(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "pgteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Олон суралцагч","price":0,"published":true}`)
	cid := c["id"].(string)
	var toks, ids []string
	for i := 0; i < 12; i++ {
		tok, id := registerFrom(t, srv.URL, fmt.Sprintf("pgstud%02d", i), "student", i)
		call(t, srv, "PUT", "/api/me/profile", tok, fmt.Sprintf(`{"display_name":"Сурагч %02d"}`, i))
		if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", tok, ""); code != 201 {
			t.Fatalf("элсэлт %d", code)
		}
		toks, ids = append(toks, tok), append(ids, id)
	}

	items, total := listCall(t, srv.URL, "/api/me/students?limit=5", tt)
	if len(items) != 5 || total != 12 {
		t.Fatalf("багшийн жагсаалт: %d мөр, нийт %d", len(items), total)
	}
	items, total = listCall(t, srv.URL, "/api/me/students?q=%D1%81%D1%83%D1%80%D0%B0%D0%B3%D1%87%2007", tt) // «сурагч 07»
	if len(items) != 1 || total != 1 || items[0]["user"].(map[string]any)["id"] != ids[7] {
		t.Fatalf("нэрээр хайлт: %v (нийт %d)", items, total)
	}
	items, total = listCall(t, srv.URL, "/api/me/courses/"+cid+"/students?limit=3&offset=10", tt)
	if len(items) != 2 || total != 12 {
		t.Fatalf("сургалтын жагсаалтын сүүлийн хуудас: %d мөр, нийт %d", len(items), total)
	}
	if code, _ := call(t, srv, "GET", "/api/me/students?limit=5", toks[0], ""); code != http.StatusForbidden {
		t.Fatal("суралцагч багшийн жагсаалт харахгүй")
	}

	// Ангийн найзууд: өөрийгөө оруулахгүй, хязгаартай, нэрээр.
	items, total = listCall(t, srv.URL, "/api/me/classmates?limit=4", toks[0])
	if len(items) != 4 || total != 11 {
		t.Fatalf("найзууд: %d мөр, нийт %d", len(items), total)
	}
	for _, it := range items {
		if it["user"].(map[string]any)["id"] == ids[0] {
			t.Fatal("өөрийгөө найзын жагсаалтад харлаа")
		}
	}
	items, _ = listCall(t, srv.URL, "/api/me/classmates?q=%D1%81%D1%83%D1%80%D0%B0%D0%B3%D1%87%2011", toks[0]) // «сурагч 11»
	if len(items) != 1 || items[0]["via"] == "" {
		t.Fatalf("найз хайлт: %v", items)
	}
	// Эхний хуудсанд ороогүй найзтай ч хувийн яриа нээгдэнэ.
	if code, _ := call(t, srv, "POST", "/api/me/dm/"+ids[11], toks[0], ""); code != 200 {
		t.Fatalf("хуудаснаас гадуурх найзтай чат: %d", code)
	}

	// Жагсаалт кэшлэгдсэний дараа шинэ суралцагч элсвэл (жижиг жагсаалт) шууд харагдана.
	tok, _ := registerFrom(t, srv.URL, "pglate", "student", 99)
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", tok, "")
	if _, total = listCall(t, srv.URL, "/api/me/students?limit=1", tt); total != 13 {
		t.Fatalf("шинэ элсэгч харагдсангүй: нийт %d", total)
	}
	if _, total = listCall(t, srv.URL, "/api/me/classmates?limit=1", toks[0]); total != 12 {
		t.Fatalf("шинэ элсэгч найзад харагдсангүй: нийт %d", total)
	}
}

// Журнал, аналитик, даалгаврын хариу: нийт тоо, нэрээр шүүлт.
func TestTeacherViewsPaging(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "jvteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Журнал","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Хичээл","is_free":true}`)
	lid := l["id"].(string)
	for i, name := range []string{"Анар", "Болд", "Цэцэг"} {
		tok, _ := register(t, srv, fmt.Sprintf("jvs%d", i), "student")
		call(t, srv, "PUT", "/api/me/profile", tok, `{"display_name":"`+name+`"}`)
		call(t, srv, "POST", "/api/courses/"+cid+"/enroll", tok, "")
		_, ss := call(t, srv, "POST", "/api/activity/start", tok, `{"course_id":"`+cid+`","lesson_id":"`+lid+`"}`)
		if ss["session_id"] == nil {
			t.Fatalf("сесс эхэлсэнгүй: %v", ss)
		}
	}
	_, jr := call(t, srv, "GET", "/api/me/courses/"+cid+"/journal", tt, "")
	if jr["students_total"].(float64) != 3 || len(jr["students"].([]any)) != 3 {
		t.Fatalf("журнал: %v", jr["students_total"])
	}
	_, jr = call(t, srv, "GET", "/api/me/courses/"+cid+"/journal?q=%D0%B1%D0%BE%D0%BB", tt, "") // «бол»
	if jr["students_total"].(float64) != 1 || jr["students"].([]any)[0].(map[string]any)["name"] != "Болд" {
		t.Fatalf("журналын хайлт: %v", jr["students"])
	}
	if cells := jr["cells"].(map[string]any); len(cells) > 1 {
		t.Fatalf("хуудсанд ороогүй суралцагчийн нүд ирлээ: %d", len(cells))
	}
	_, an := call(t, srv, "GET", "/api/me/analytics?course="+cid, tt, "")
	if an["students_total"].(float64) != 3 || an["data"].(map[string]any)["totals"].(map[string]any)["risk"] == nil {
		t.Fatalf("аналитик: %v", an["students_total"])
	}
	_, an = call(t, srv, "GET", "/api/me/analytics?course="+cid+"&q=%D1%86%D1%8D%D1%86", tt, "") // «цэц»
	if an["students_total"].(float64) != 1 || len(an["data"].(map[string]any)["students"].([]any)) != 1 {
		t.Fatalf("аналитикийн хайлт: %v", an["students_total"])
	}
	if an["data"].(map[string]any)["totals"].(map[string]any)["students"].(float64) != 3 {
		t.Fatal("нэгтгэл бүх суралцагчаар тооцогдох ёстой")
	}
}
