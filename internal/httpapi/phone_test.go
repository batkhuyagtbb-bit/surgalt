package httpapi

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestCleanPhone(t *testing.T) {
	for in, want := range map[string]string{"9911 2233": "99112233", "+976 9911-2233": "99112233", "97699112233": "99112233",
		"(+1) 415.555.0100": "+14155550100", "": ""} {
		if got, ok := cleanPhone(in); !ok || got != want {
			t.Fatalf("cleanPhone(%q) = %q %v, хүлээсэн %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"12345", "9911-223", "99112233a", "+12", "+1234567890123456", "++99112233"} {
		if _, ok := cleanPhone(in); ok {
			t.Fatalf("cleanPhone(%q) буруу дугаарыг зөвшөөрөв", in)
		}
	}
}

// Багшийн утас: нээлттэй профайл (JSON, HTML) зөвхөн эхний 4 оронтой; дарж нээхэд бүтэн дугаар өгч, багшид
// нэг хүн тутамд нэг мэдэгдэл очиж статистикт тоологдоно; багш өөрөө нээвэл тоологдохгүй.
func TestTeacherPhoneReveal(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	if code, _ := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Бат","phone":"12345"}`); code != http.StatusBadRequest {
		t.Fatalf("буруу дугаар 400 байх ёстой: %d", code)
	}
	if code, u := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Бат","phone":"+976 8811-4455"}`); code != 200 || u["phone"] != "88114455" {
		t.Fatalf("дугаар хадгалах: %d %v", code, u["phone"])
	}
	// Зураг солих зэрэг хэсэгчилсэн хадгалалт (phone ирээгүй) дугаарыг устгахгүй.
	if _, u := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Бат багш"}`); u["phone"] != "88114455" {
		t.Fatalf("phone ирээгүй үед хэвээр үлдэх ёстой: %v", u["phone"])
	}
	if _, me := call(t, srv, "GET", "/api/me", tt, ""); me["phone"] != "88114455" {
		t.Fatalf("эзэмшигч өөрийн дугаараа харна: %v", me["phone"])
	}

	_, p := call(t, srv, "GET", "/api/teachers/teach", "", "")
	if pt := p["teacher"].(map[string]any); pt["phone_hint"] != "8811" || pt["phone"] != nil {
		t.Fatalf("нээлттэй профайлд зөвхөн эхний 4 орон: %v", pt)
	}
	res, err := http.Get(srv.URL + "/t/teach")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if html := string(page); !strings.Contains(html, "8811 ••••") || strings.Contains(html, "88114455") || strings.Contains(html, "8811 4455") {
		t.Fatal("профайлын HTML-д бүтэн дугаар гарч болохгүй, эхний 4 орон харагдана")
	}

	st, _ := register(t, srv, "stud", "student")
	for range 2 { // нэг зочин хоёр дарсан ч нэг л удаа тоологдоно
		if code, d := call(t, srv, "POST", "/api/teachers/teach/phone", "", `{"vid":"guest-1"}`); code != 200 || d["display"] != "8811 4455" || d["tel"] != "+97688114455" {
			t.Fatalf("дугаар нээх: %d %v", code, d)
		}
	}
	call(t, srv, "POST", "/api/teachers/teach/phone", st, "")
	call(t, srv, "POST", "/api/teachers/teach/phone", tt, "") // багш өөрөө — тоологдохгүй
	_, ns := call(t, srv, "GET", "/api/me/notifications", tt, "")
	var titles []string
	for _, n := range ns["items"].([]any) {
		if m := n.(map[string]any); m["type"] == NotifPhoneView {
			titles = append(titles, m["title"].(string))
		}
	}
	if len(titles) != 2 || !strings.Contains(strings.Join(titles, "|"), "Зочин утасны дугаарыг тань харлаа") {
		t.Fatalf("багшид зочин ба суралцагч тус бүр нэг мэдэгдэл: %v", titles)
	}
	_, v := call(t, srv, "GET", "/api/me/visits", tt, "")
	if tot := v["totals"].(map[string]any); tot["phone"].(float64) != 2 || tot["views"].(float64) != 0 {
		t.Fatalf("статистик: утас харсан 2, үзэлтэд тооцохгүй: %v", tot)
	}
	if rec := v["recent"].([]any); len(rec) != 2 || rec[0].(map[string]any)["kind"] != "phone" {
		t.Fatalf("сүүлийн зочдод утас харсан мөр: %v", rec)
	}

	if code, _ := call(t, srv, "POST", "/api/teachers/stud/phone", "", ""); code != http.StatusNotFound {
		t.Fatalf("багш биш эсвэл дугааргүй — 404: %d", code)
	}
	req, _ := http.NewRequest("POST", srv.URL+"/api/teachers/teach/phone", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Googlebot/2.1)")
	if res, err := http.DefaultClient.Do(req); err != nil || res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("робот дугаар авч чадахгүй: %v %v", res.StatusCode, err)
	}
}
