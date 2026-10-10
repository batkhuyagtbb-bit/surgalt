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

// Давталт: улирал (13 долоо хоног) — цаг бүр долоо хоног бүр, нэг цувралд; өмнө тэмдэглэсэн цагтай давхцах долоо
// хоногийг алгасна; «энэ цагаас хойших бүгд»-ийг хасахад захиалсан цаг үлдэнэ.
func TestSlotRepeatWeekly(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	st, _ := register(t, srv, "stud", "student")
	first := time.Now().Add(72 * time.Hour).Truncate(time.Hour)
	iso := func(wk int) string { return first.AddDate(0, 0, 7*wk).UTC().Format(time.RFC3339) }
	if code, _ := callArr(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+iso(2)+`"],"duration_min":60,"mode":"online"}`); code != 201 {
		t.Fatalf("тусдаа цаг: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+iso(0)+`"],"duration_min":60,"mode":"online","weeks":60}`); code != http.StatusBadRequest {
		t.Fatalf("53-аас их долоо хоног 400: %d", code)
	}
	code, out := callArr(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+iso(0)+`"],"duration_min":60,"mode":"online","weeks":13}`)
	if code != 201 || len(out) != 12 {
		t.Fatalf("13 долоо хоногоос давхцсан 1-ийг алгасаж 12 цаг: %d %d", code, len(out))
	}
	series := out[0].(map[string]any)["series_id"]
	for i, o := range out {
		m := o.(map[string]any)
		wk := i
		if i >= 2 {
			wk = i + 1 // 3 дахь долоо хоног (wk=2) алгасагдсан
		}
		if m["series_id"] != series || series == "" {
			t.Fatalf("нэг цувралд байх ёстой: %v", m)
		}
		if at, _ := time.Parse(time.RFC3339, m["starts_at"].(string)); !at.Equal(first.AddDate(0, 0, 7*wk)) {
			t.Fatalf("%d дэх цаг долоо хоног бүр ижил цагт: %v", i, at)
		}
	}
	if code, b := call(t, srv, "POST", "/api/slots/"+out[3].(map[string]any)["id"].(string)+"/book", st, `{}`); code != 200 || b["booked"] != true {
		t.Fatalf("давталтын нэгийг захиалах: %d %v", code, b)
	}
	code, d := call(t, srv, "DELETE", "/api/me/slots/"+out[2].(map[string]any)["id"].(string)+"?series=1", tt, "")
	if code != 200 || d["deleted"].(float64) != 9 {
		t.Fatalf("цааших сул давталтыг хасна (захиалсныг үлдээнэ): %d %v", code, d)
	}
	from := first.In(mnLoc).Format("2006-01-02")
	if _, left := callArr(t, srv, "GET", "/api/me/slots?from="+from+"&days=62", tt, ""); len(left) != 4 {
		t.Fatalf("үлдэх ёстой: эхний 2 давталт, тусдаа цаг, захиалсан цаг: %d", len(left))
	}
}

// Багш захиалгуудаа жагсаалтаар харж, сонгосныг (эсвэл бүгдийг) нэг дор цуцална: суралцагч бүрт нэг мэдэгдэл,
// «Шууд хичээл»-ээс хасагдана; бусдын цагийг цуцалж чадахгүй.
func TestCancelBookingsBulk(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	other, _ := register(t, srv, "other", "teacher")
	s1, _ := register(t, srv, "stud1", "student")
	s2, _ := register(t, srv, "stud2", "student")
	base := time.Now().Add(48 * time.Hour).Truncate(time.Hour)
	at := func(h int) string { return base.Add(time.Duration(h) * time.Hour).UTC().Format(time.RFC3339) }
	_, out := callArr(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+at(0)+`","`+at(1)+`","`+at(2)+`","`+at(3)+`"],"duration_min":60,"mode":"online"}`)
	_, foreign := callArr(t, srv, "POST", "/api/me/slots", other, `{"starts":["`+at(0)+`"],"duration_min":60,"mode":"online"}`)
	id := func(i int) string { return out[i].(map[string]any)["id"].(string) }
	for i, who := range []string{s1, s1, s2} {
		if code, _ := call(t, srv, "POST", "/api/slots/"+id(i)+"/book", who, `{}`); code != 200 {
			t.Fatalf("захиалга %d: %d", i, code)
		}
	}
	if _, bs := callArr(t, srv, "GET", "/api/me/bookings", tt, ""); len(bs) != 3 {
		t.Fatalf("багшид 3 захиалга (сул цаг орохгүй): %d", len(bs))
	}
	if code, _ := call(t, srv, "GET", "/api/me/bookings", s1, ""); code != http.StatusForbidden {
		t.Fatalf("суралцагч багшийн захиалгыг харахгүй: %d", code)
	}
	foreignID := foreign[0].(map[string]any)["id"].(string)
	code, d := call(t, srv, "POST", "/api/me/slots/cancel", tt, `{"ids":["`+id(0)+`","`+id(1)+`","`+foreignID+`"]}`)
	if code != 200 || d["cancelled"].(float64) != 2 {
		t.Fatalf("сонгосон 2-ыг цуцална, бусдын цагийг үл тооно: %d %v", code, d)
	}
	_, ns := call(t, srv, "GET", "/api/me/notifications", s1, "")
	var cancels []string
	for _, x := range notifTitles(ns, NotifBooking) {
		if strings.Contains(x, "цуцаллаа") {
			cancels = append(cancels, x)
		}
	}
	if len(cancels) != 1 || !strings.Contains(cancels[0], "2 уулзалтыг цуцаллаа") {
		t.Fatalf("нэг суралцагчид нэгтгэсэн нэг мэдэгдэл: %v", cancels)
	}
	if _, ms := callArrMap(t, srv, "GET", "/api/me/meetings", tt); len(ms) != 1 {
		t.Fatalf("цуцалсан уулзалтууд «Шууд хичээл»-ээс хасагдана: %v", ms)
	}
	if _, bs := callArr(t, srv, "GET", "/api/me/bookings", tt, ""); len(bs) != 1 {
		t.Fatalf("1 захиалга үлдэнэ: %d", len(bs))
	}
	if code, d := call(t, srv, "POST", "/api/me/slots/cancel", tt, `{"ids":["`+id(2)+`"]}`); code != 200 || d["cancelled"].(float64) != 1 {
		t.Fatalf("үлдсэнийг цуцлах: %d %v", code, d)
	}
	if code, _ := call(t, srv, "POST", "/api/me/slots/cancel", tt, `{"ids":[]}`); code != http.StatusBadRequest {
		t.Fatalf("хоосон жагсаалт 400: %d", code)
	}
}

func TestCleanMeetURL(t *testing.T) {
	for in, want := range map[string]string{
		"meet.google.com/abc-defg-hij":                "https://meet.google.com/abc-defg-hij",
		" https://meet.google.com/abc-defg-hij?a=1":   "https://meet.google.com/abc-defg-hij?a=1",
		"https://us02web.zoom.us/j/123456":            "https://us02web.zoom.us/j/123456",
		"https://teams.microsoft.com/l/meetup-join/x": "https://teams.microsoft.com/l/meetup-join/x",
	} {
		if got, ok := cleanMeetURL(in); !ok || got != want {
			t.Fatalf("cleanMeetURL(%q) = %q %v, хүлээсэн %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "http://meet.google.com/abc-defg-hij", "javascript:alert(1)", "https://evil.com/meet.google.com/abc",
		"https://meet.google.com/", "https://user:pw@meet.google.com/abc-defg-hij", "https://meet.google.com.evil.com/abc"} {
		if _, ok := cleanMeetURL(in); ok {
			t.Fatalf("cleanMeetURL(%q) зөвшөөрөх ёсгүй", in)
		}
	}
}

// Google Meet холбоогүй багш meet.new-ээс хуулсан холбоосоо онлайн захиалгад тавина: «Шууд хичээл», суралцагчийн
// нүүр шинэчлэгдэж мэдэгдэл очно; шууд хичээлийг ч гараар тавьсан холбоосоор товлоно.
func TestManualMeetLink(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	other, _ := register(t, srv, "other", "teacher")
	st, _ := register(t, srv, "stud", "student")
	at := time.Now().Add(48 * time.Hour).Truncate(time.Hour).UTC()
	_, out := callArr(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+at.Format(time.RFC3339)+`","`+at.Add(time.Hour).Format(time.RFC3339)+`"],"duration_min":60,"mode":"online"}`)
	booked, free := out[0].(map[string]any)["id"].(string), out[1].(map[string]any)["id"].(string)
	call(t, srv, "POST", "/api/slots/"+booked+"/book", st, `{}`)
	link := `{"meet_url":"meet.google.com/abc-defg-hij"}`
	if code, _ := call(t, srv, "PUT", "/api/me/slots/"+booked+"/meet", tt, `{"meet_url":"https://evil.com/x"}`); code != http.StatusBadRequest {
		t.Fatalf("буруу холбоос 400: %d", code)
	}
	if code, _ := call(t, srv, "PUT", "/api/me/slots/"+booked+"/meet", other, link); code != http.StatusNotFound {
		t.Fatalf("бусдын захиалгад холбоос тавихгүй: %d", code)
	}
	if code, _ := call(t, srv, "PUT", "/api/me/slots/"+free+"/meet", tt, link); code != http.StatusNotFound {
		t.Fatalf("захиалаагүй цагт холбоос тавихгүй: %d", code)
	}
	if code, sl := call(t, srv, "PUT", "/api/me/slots/"+booked+"/meet", tt, link); code != 200 || sl["meet_url"] != "https://meet.google.com/abc-defg-hij" {
		t.Fatalf("холбоос тавих: %d %v", code, sl)
	}
	if _, ms := callArrMap(t, srv, "GET", "/api/me/meetings", tt); len(ms) != 1 || ms[0]["meet_url"] != "https://meet.google.com/abc-defg-hij" {
		t.Fatalf("«Шууд хичээл»-д холбоос: %v", ms)
	}
	_, h := call(t, srv, "GET", "/api/me/home", st, "")
	found := false
	for _, m := range h["meetings"].([]any) {
		if mm := m.(map[string]any); mm["booking"] == true && mm["meet_url"] == "https://meet.google.com/abc-defg-hij" {
			found = true
		}
	}
	_, ns := call(t, srv, "GET", "/api/me/notifications", st, "")
	if ts := notifTitles(ns, NotifBooking); !found || !strings.Contains(strings.Join(ts, "|"), "Уулзалтын холбоос ирлээ") {
		t.Fatalf("суралцагчийн нүүрт холбоос, мэдэгдэл: %v %v", found, ts)
	}

	// Шууд хичээл: Google тохируулаагүй ч гараар тавьсан холбоосоор товлоно; холбоосгүй бол Google шаардана.
	start := time.Now().Add(72 * time.Hour).UTC().Format(time.RFC3339)
	if code, m := call(t, srv, "POST", "/api/me/meetings", tt, `{"title":"Давтлага","starts_at":"`+start+`","duration_min":60,"meet_url":"https://meet.google.com/xyz-abcd-efg"}`); code != 201 || m["meet_url"] != "https://meet.google.com/xyz-abcd-efg" {
		t.Fatalf("гараар тавьсан холбоосоор товлох: %d %v", code, m)
	}
	if code, _ := call(t, srv, "POST", "/api/me/meetings", tt, `{"title":"Давтлага","starts_at":"`+start+`","duration_min":60}`); code != http.StatusServiceUnavailable {
		t.Fatalf("холбоосгүй бол Google Meet шаардана: %d", code)
	}
}
