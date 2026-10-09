package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// Профайлын үзэлт: хэн, хэдэн удаа, хэр удаан, хаанаас, ямар төхөөрөмжөөр — багшид статистик.
func TestProfileVisitStats(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, me := call(t, srv, "GET", "/api/me", tt, "")
	username := me["username"].(string)
	visit := func(token, ua, body string) (int, map[string]any) {
		req, _ := http.NewRequest("POST", srv.URL+"/api/views", bytes.NewBufferString(body))
		req.Header.Set("User-Agent", ua)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	const phone = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile/15E148"
	const pc = "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15"
	// Зочин утаснаас Facebook-ээс ирж 42 сек байсан.
	code, a := visit("", phone, `{"kind":"profile","id":"`+username+`","vid":"abc123xyz","ref":"l.facebook.com"}`)
	if code != 200 || a["visit"] == nil || a["t"] == nil {
		t.Fatalf("үзэлт бүртгэх: %d %v", code, a)
	}
	end := "/api/views/" + a["visit"].(string) + "/end"
	if code, _ := call(t, srv, "POST", end, "", `{"t":"bad","seconds":99}`); code != 404 {
		t.Fatalf("буруу гарын үсэгтэй хугацаа: %d", code)
	}
	call(t, srv, "POST", end, "", `{"t":"`+a["t"].(string)+`","seconds":42}`)
	call(t, srv, "POST", end, "", `{"t":"`+a["t"].(string)+`","seconds":30}`) // хугацаа буурахгүй
	// Нэвтэрсэн суралцагч компьютероос шууд орж 6 сек байсан (буцаж гарсан).
	s1, _ := register(t, srv, "stud", "student")
	_, b := visit(s1, pc, `{"kind":"profile","id":"`+username+`"}`)
	call(t, srv, "POST", "/api/views/"+b["visit"].(string)+"/end", "", `{"t":"`+b["t"].(string)+`","seconds":6}`)
	// Ижил зочин дахин орсон — үзэлт нэмэгдэнэ, давтагдаагүй зочин хэвээр.
	visit("", phone, `{"kind":"profile","id":"`+username+`","vid":"abc123xyz","ref":"google.com"}`)
	// Багш өөрийнхийгөө, робот — тоологдохгүй.
	if code, _ := visit(tt, pc, `{"kind":"profile","id":"`+username+`"}`); code != 204 {
		t.Fatalf("багш өөрийнхийгөө үзвэл тоологдохгүй: %d", code)
	}
	if code, _ := visit("", "Googlebot/2.1 (+http://www.google.com/bot.html)", `{"kind":"profile","id":"`+username+`"}`); code != 204 {
		t.Fatalf("робот тоологдохгүй: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/me/visits", s1, ""); code != 403 {
		t.Fatalf("суралцагч статистик харахгүй: %d", code)
	}
	_, st := call(t, srv, "GET", "/api/me/visits?days=7", tt, "")
	tot := st["totals"].(map[string]any)
	if tot["views"].(float64) != 3 || tot["unique"].(float64) != 2 || tot["members"].(float64) != 1 || tot["avg_sec"].(float64) != 24 || tot["total_sec"].(float64) != 48 || tot["bounce_pct"].(float64) != 50 {
		t.Fatalf("нийт: %v", tot)
	}
	if len(st["daily"].([]any)) != 7 {
		t.Fatalf("7 өдрийн цуваа: %v", st["daily"])
	}
	refs := map[string]float64{}
	for _, x := range st["referrers"].([]any) {
		m := x.(map[string]any)
		refs[m["name"].(string)] = m["count"].(float64)
	}
	if refs["Facebook"] != 1 || refs["Google"] != 1 || refs["Шууд"] != 1 {
		t.Fatalf("эх сурвалж: %v", refs)
	}
	recent := st["recent"].([]any)
	first := recent[len(recent)-1].(map[string]any) // хамгийн эхний үзэлт
	if len(recent) != 3 || first["name"] != "Зочин" || first["device"] != "mobile" || first["seconds"].(float64) != 42 {
		t.Fatalf("сүүлийн зочид: %v", recent)
	}
}
