package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// callArr: JSON массив буцаадаг API.
func callArr(t *testing.T, srv *httptest.Server, method, path, token, body string) (int, []any) {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out []any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func callArrMap(t *testing.T, srv *httptest.Server, method, path, token string) (int, []map[string]any) {
	t.Helper()
	code, arr := callArr(t, srv, method, path, token, "")
	out := make([]map[string]any, len(arr))
	for i, v := range arr {
		out[i], _ = v.(map[string]any)
	}
	return code, out
}

func notifTitles(ns map[string]any, typ string) []string {
	var out []string
	for _, n := range ns["items"].([]any) {
		if m := n.(map[string]any); m["type"] == typ {
			out = append(out, m["title"].(string))
		}
	}
	return out
}

// Цаг захиалга: багш календарьт сул цаг тэмдэглэнэ → суралцагч профайлаас захиална (үнэгүй бол шууд, төлбөртэй бол
// төлбөрийн дараа) → «Шууд хичээл»-д нэмэгдэж хоёр талд мэдэгдэнэ; давхар захиалга, давхцал, цуцлалт.
func TestSlotBooking(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	s1, _ := register(t, srv, "stud1", "student")
	s2, _ := register(t, srv, "stud2", "student")
	day := time.Now().Add(48 * time.Hour).Truncate(time.Hour)
	at := func(h int) string { return day.Add(time.Duration(h) * time.Hour).UTC().Format(time.RFC3339) }

	for _, bad := range []string{
		`{"starts":["` + time.Now().Add(-time.Hour).UTC().Format(time.RFC3339) + `"],"duration_min":60,"mode":"online"}`, // өнгөрсөн
		`{"starts":["` + at(0) + `"],"duration_min":5,"mode":"online"}`,                                                  // хэт богино
		`{"starts":["` + at(0) + `"],"duration_min":60,"mode":"phone"}`,                                                  // хэлбэр буруу
		`{"starts":["` + at(0) + `"],"duration_min":60,"mode":"offline"}`,                                                // байршилгүй
		`{"starts":["` + at(0) + `","` + at(0) + `"],"duration_min":60,"mode":"online"}`,                                 // хоорондоо давхцсан
	} {
		if code, _ := call(t, srv, "POST", "/api/me/slots", tt, bad); code != http.StatusBadRequest && code != http.StatusConflict {
			t.Fatalf("буруу цаг 400/409 байх ёстой (%s): %d", bad, code)
		}
	}
	if code, _ := call(t, srv, "POST", "/api/me/slots", s1, `{"starts":["`+at(0)+`"],"duration_min":60,"mode":"online"}`); code != http.StatusForbidden {
		t.Fatalf("суралцагч цаг тэмдэглэхгүй: %d", code)
	}
	code, out := callArr(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+at(0)+`"],"duration_min":60,"mode":"online","price":0}`)
	if code != 201 || len(out) != 1 {
		t.Fatalf("онлайн үнэгүй цаг: %d %v", code, out)
	}
	freeID := out[0].(map[string]any)["id"].(string)
	code, out = callArr(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+at(2)+`"],"duration_min":45,"mode":"offline","price":30000,"location":"Номын сангийн 2 давхар"}`)
	if code != 201 {
		t.Fatalf("биечлэн төлбөртэй цаг: %d %v", code, out)
	}
	paidID := out[0].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+day.Add(30*time.Minute).UTC().Format(time.RFC3339)+`"],"duration_min":30,"mode":"online"}`); code != http.StatusConflict {
		t.Fatalf("тэмдэглэсэн цагтай давхцвал 409: %d", code)
	}

	_, p := call(t, srv, "GET", "/api/teachers/teach", "", "")
	if p["teacher"].(map[string]any)["slot_count"].(float64) != 2 {
		t.Fatalf("профайлд 2 сул цаг: %v", p["teacher"])
	}
	_, pub := call(t, srv, "GET", "/api/teachers/teach/slots", "", "")
	if sl := pub["slots"].([]any); len(sl) != 2 || sl[1].(map[string]any)["location"] != "Номын сангийн 2 давхар" || sl[0].(map[string]any)["student_id"] != nil {
		t.Fatalf("нээлттэй сул цагууд: %v", pub)
	}

	// Үнэгүй онлайн цаг: шууд баталгаажна, давхар захиалга 409.
	if code, _ := call(t, srv, "POST", "/api/slots/"+freeID+"/book", "", `{}`); code != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй хүн захиалахгүй: %d", code)
	}
	code, b := call(t, srv, "POST", "/api/slots/"+freeID+"/book", s1, `{"note":"Логарифмын бодлого"}`)
	if code != 200 || b["booked"] != true {
		t.Fatalf("үнэгүй цаг захиалах: %d %v", code, b)
	}
	if code, _ := call(t, srv, "POST", "/api/slots/"+freeID+"/book", s2, `{}`); code != http.StatusConflict {
		t.Fatalf("авсан цагийг өөр хүн захиалахгүй: %d", code)
	}
	_, ms := callArrMap(t, srv, "GET", "/api/me/meetings", tt)
	if len(ms) != 1 || !strings.HasPrefix(ms[0]["title"].(string), "Цаг захиалга: ") {
		t.Fatalf("захиалга «Шууд хичээл»-д орох ёстой: %v", ms)
	}
	_, ns := call(t, srv, "GET", "/api/me/notifications", tt, "")
	if ts := notifTitles(ns, NotifBooking); len(ts) != 1 || !strings.Contains(ts[0], "цаг захиаллаа") {
		t.Fatalf("багшид захиалгын мэдэгдэл: %v", ts)
	}
	_, h := call(t, srv, "GET", "/api/me/home", s1, "")
	var booked map[string]any
	for _, m := range h["meetings"].([]any) {
		if mm := m.(map[string]any); mm["booking"] == true {
			booked = mm
		}
	}
	if booked == nil || booked["mode"] != "online" || booked["teacher_username"] != "teach" {
		t.Fatalf("суралцагчийн нүүрт «Шууд хичээл» дотор захиалга: %v", h["meetings"])
	}

	// Төлбөртэй биечлэн цаг: төлбөрийн үеэр барина, төлсний дараа баталгаажна.
	code, b = call(t, srv, "POST", "/api/slots/"+paidID+"/book", s2, `{}`)
	if code != 201 || b["booked"] != false || b["payment"] == nil {
		t.Fatalf("төлбөртэй цаг: QR төлбөр: %d %v", code, b)
	}
	oid := b["order"].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/slots/"+paidID+"/book", s1, `{}`); code != http.StatusConflict {
		t.Fatalf("төлбөр хүлээгдэж буй цагийг өөр хүн авахгүй: %d", code)
	}
	if _, pub := call(t, srv, "GET", "/api/teachers/teach/slots", "", ""); len(pub["slots"].([]any)) != 0 {
		t.Fatalf("барьсан/авсан цаг нээлттэй жагсаалтад харагдахгүй: %v", pub)
	}
	if _, pub := call(t, srv, "GET", "/api/teachers/teach/slots", s2, ""); len(pub["slots"].([]any)) != 1 {
		t.Fatalf("өөрөө барьсан цаг төлбөрөө үргэлжлүүлэхэд өөрт нь харагдана: %v", pub)
	}
	if code, b2 := call(t, srv, "POST", "/api/slots/"+paidID+"/book", s2, `{}`); code != 201 || b2["order"].(map[string]any)["id"] != oid {
		t.Fatalf("дахин дарахад ижил захиалга: %d %v", code, b2)
	}
	if code, o := call(t, srv, "POST", "/api/orders/"+oid+"/dev-pay", s2, ""); code != 200 || o["status"] != "paid" {
		t.Fatalf("демо төлбөр: %d %v", code, o)
	}
	_, mine := call(t, srv, "GET", "/api/teachers/teach/slots", s2, "")
	if m := mine["mine"].([]any); len(m) != 1 || m[0].(map[string]any)["location"] != "Номын сангийн 2 давхар" {
		t.Fatalf("төлсний дараа захиалга баталгаажна: %v", mine)
	}
	_, ns = call(t, srv, "GET", "/api/me/notifications", s2, "")
	if ts := notifTitles(ns, NotifBooking); len(ts) != 1 || !strings.Contains(ts[0], "Цаг баталгаажлаа") {
		t.Fatalf("суралцагчид баталгаажсан мэдэгдэл: %v", ts)
	}

	// Багш захиалсан цагийг цуцална: «Шууд хичээл»-ээс хасагдаж, суралцагчид мэдэгдэнэ.
	if code, _ := call(t, srv, "DELETE", "/api/me/slots/"+freeID, tt, ""); code != http.StatusNoContent {
		t.Fatalf("цуцлах: %d", code)
	}
	if _, ms := callArrMap(t, srv, "GET", "/api/me/meetings", tt); len(ms) != 1 || !strings.Contains(ms[0]["title"].(string), "биечлэн") {
		t.Fatalf("цуцалсан уулзалт хасагдана, төлбөртэй нь үлдэнэ: %v", ms)
	}
	_, ns = call(t, srv, "GET", "/api/me/notifications", s1, "")
	if ts := notifTitles(ns, NotifBooking); len(ts) != 2 || !strings.Contains(strings.Join(ts, "|"), "цуцаллаа") {
		t.Fatalf("суралцагчид цуцалсан мэдэгдэл: %v", ts)
	}
	if _, p := call(t, srv, "GET", "/api/teachers/teach", "", ""); p["teacher"].(map[string]any)["slot_count"].(float64) != 0 {
		t.Fatalf("сул цаг үлдээгүй: %v", p["teacher"])
	}
}
