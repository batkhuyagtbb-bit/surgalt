package httpapi

import "testing"

// Мэдэгдэл: үүсгэсэн хүний нэр, зураг (actor) ирнэ; дээр нь дарсныг нэг бүрчлэн уншсан болгоно;
// хоосон хүсэлт бүгдийг уншсан болгоно.
func TestNotificationActorAndRead(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "ntteach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Мэдэгдэл","price":0,"published":true}`)
	cid := c["id"].(string)
	var sids []string
	for _, u := range []string{"ntstud1", "ntstud2"} {
		st, sid := register(t, srv, u, "student")
		call(t, srv, "PUT", "/api/me/name", st, `{"display_name":"Сараа `+u+`"}`)
		call(t, srv, "POST", "/api/courses/"+cid+"/enroll", st, "")
		sids = append(sids, sid)
	}
	list := func() ([]map[string]any, float64) {
		_, d := call(t, srv, "GET", "/api/me/notifications", tt, "")
		raw, _ := d["items"].([]any)
		out := make([]map[string]any, len(raw))
		for i, x := range raw {
			out[i] = x.(map[string]any)
		}
		return out, d["unread"].(float64)
	}
	items, unread := list()
	if len(items) != 2 || unread != 2 {
		t.Fatalf("2 элсэлтийн мэдэгдэл: %v %v", items, unread)
	}
	for _, n := range items {
		a, _ := n["actor"].(map[string]any)
		if a == nil || n["actor_id"] == "" || a["id"] != n["actor_id"] || a["display_name"] == "" {
			t.Fatalf("actor байх ёстой: %v", n)
		}
	}
	if a := items[0]["actor"].(map[string]any); a["id"] != sids[1] || a["display_name"] != "Сараа ntstud2" {
		t.Fatalf("хамгийн сүүлийнх нь 2-р суралцагч: %v", a)
	}
	// Нэгийг нь уншсан болгоно → нөгөө нь уншаагүй хэвээр.
	if code, _ := call(t, srv, "POST", "/api/me/notifications/read", tt, `{"id":"`+items[0]["id"].(string)+`"}`); code != 204 {
		t.Fatalf("нэгийг уншсан: %d", code)
	}
	items, unread = list()
	if unread != 1 || items[0]["read"] != true || items[1]["read"] == true {
		t.Fatalf("зөвхөн дарсан нь уншсан: %v %v", unread, items)
	}
	if code, _ := call(t, srv, "POST", "/api/me/notifications/read", tt, `{"id":`); code != 400 {
		t.Fatalf("буруу JSON: %d", code)
	}
	// Бүгдийг уншсан болгох (хоосон хүсэлт — өмнөх клиенттэй нийцтэй).
	if code, _ := call(t, srv, "POST", "/api/me/notifications/read", tt, ""); code != 204 {
		t.Fatalf("бүгдийг уншсан: %d", code)
	}
	if items, unread = list(); unread != 0 || items[1]["read"] != true {
		t.Fatalf("бүгд уншсан байх ёстой: %v %v", unread, items)
	}
	if code, _ := call(t, srv, "POST", "/api/me/notifications/read", "", `{"id":"x"}`); code != 401 {
		t.Fatalf("нэвтрээгүй: %d", code)
	}
}
