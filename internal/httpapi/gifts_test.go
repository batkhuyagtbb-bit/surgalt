package httpapi

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
)

// Системийн бэлэг: зоос онооноос (10 оноо = 1 зоос), хайрцаг эрэмбээр нээгдэнэ, зоос хүрэхгүй бол нээхгүй,
// давхардсан дүрд зоос буцаана, магадлал 100% болно.
func TestGiftBoxes(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	if code, _ := call(t, srv, "GET", "/api/me/gifts", "", ""); code != 401 {
		t.Fatalf("нэвтрээгүй: %d", code)
	}
	tok, uid := register(t, srv, "gift", "student")
	if _, err := st.SaveRankPoints(context.Background(), uid, "c1", 450); err != nil { // 45 зоос, «Зальжин үнэг» эрэмбэ
		t.Fatal(err)
	}
	_, g := call(t, srv, "GET", "/api/me/gifts", tok, "")
	if g["coins"].(float64) != 45 || g["level"].(float64) != 4 || g["rank"].(map[string]any)["name"] != "Зальжин үнэг" || g["rank"].(map[string]any)["era"] != 2.0 {
		t.Fatalf("төлөв: %v %v %v", g["coins"], g["level"], g["rank"])
	}
	boxes := g["boxes"].([]any)
	unlocked := map[string]bool{}
	for _, b := range boxes {
		m := b.(map[string]any)
		unlocked[m["id"].(string)] = m["unlocked"].(bool)
		var sum float64
		for _, c := range m["chances"].([]any) {
			sum += c.(map[string]any)["pct"].(float64)
		}
		if math.Abs(sum-100) > 0.5 {
			t.Fatalf("%s магадлал %v", m["id"], sum)
		}
	}
	if !unlocked["wood"] || !unlocked["silver"] || unlocked["gold"] || unlocked["legend"] {
		t.Fatalf("нээгдсэн хайрцаг: %v", unlocked)
	}
	if len(g["animals"].([]any)) != 18 || g["collected"].(float64) != 0 {
		t.Fatalf("цуглуулга: %v", g["collected"])
	}
	// Хүрээгүй эрэмбийн нэр нууц: дүр (зөвхөн түвшин, оноо), хайрцгийн түгжээ (оноо, үе), алдааны мессеж.
	an := g["animals"].([]any)
	if a4, a5 := an[4].(map[string]any), an[5].(map[string]any); a4["name"] != "Зальжин үнэг" || a5["name"] != "" || a5["emoji"] != "" || a5["points"].(float64) != 560 {
		t.Fatalf("хүрээгүй дүр нууцлагдаагүй: %v %v", a4, a5)
	}
	if an[9].(map[string]any)["era_name"] != "" || an[3].(map[string]any)["era_name"] == "" {
		t.Fatalf("үеийн нэр: %v", an[9])
	}
	for _, b := range boxes {
		m := b.(map[string]any)
		if _, leak := m["unlock_name"]; leak || (m["id"] == "gold" && (m["unlock_points"].(float64) != 750 || m["unlock_era"].(float64) != 3)) {
			t.Fatalf("хайрцгийн түгжээ: %v", m)
		}
	}
	if code, e := call(t, srv, "POST", "/api/me/gifts/open", tok, `{"box":"gold"}`); code != 403 || strings.Contains(fmt.Sprint(e), "Одой дархан") || !strings.Contains(fmt.Sprint(e), "750 оноонд") {
		t.Fatalf("түгжээтэй хайрцаг: %d %v", code, e)
	}
	if code, _ := call(t, srv, "POST", "/api/me/gifts/open", tok, `{"box":"nope"}`); code != 400 {
		t.Fatalf("байхгүй хайрцаг: %d", code)
	}
	code, o := call(t, srv, "POST", "/api/me/gifts/open", tok, `{"box":"wood"}`)
	a := o["animal"].(map[string]any)
	if code != 200 || o["new"] != true || o["coins"].(float64) != 35 || a["level"].(float64) > 5 || a["cheer"] == "" || a["emoji"] == "" {
		t.Fatalf("модон хайрцаг: %d %v", code, o)
	}
	code, o = call(t, srv, "POST", "/api/me/gifts/open", tok, `{"box":"silver"}`)
	a = o["animal"].(map[string]any)
	if lvl := a["level"].(float64); code != 200 || lvl < 3 || lvl > 8 || o["coins"].(float64) != 10+o["refund"].(float64) {
		t.Fatalf("мөнгөн хайрцаг: %d %v", code, o)
	}
	if code, _ := call(t, srv, "POST", "/api/me/gifts/open", tok, `{"box":"silver"}`); code != 409 {
		t.Fatalf("зоос хүрэлцэхгүй: %d", code)
	}
	_, g = call(t, srv, "GET", "/api/me/gifts", tok, "")
	if g["spent"].(float64) != 35 || len(g["recent"].([]any)) != 2 || g["collected"].(float64) < 1 {
		t.Fatalf("нээлтийн дараа: %v", g)
	}
	// Давхардал: зөвхөн нэг амьтантай хайрцаг байсан ч гэсэн (жин) — шинэ амьтан 2 дахин илүү магадлалтай.
	w := boxWeights(giftBoxes[0], map[int]int{0: 1})
	if w[0] != 6 || w[1] != 10 {
		t.Fatalf("жин: %v", w)
	}
}
