package httpapi

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"surgalt/internal/auth"
)

// Хичээлийг (үнэгүйг ч) зөвхөн бүртгэлтэй, нэвтэрсэн хэрэглэгч үзнэ: агуулга нийтийн хуудас, JSON-д орохгүй.
func TestLessonsNeedLogin(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	s1, _ := register(t, srv, "stud1", "student")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Үнэгүй сургалт","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Танилцуулга","content":"Нууц агуулга 42","is_free":true}`)
	lid := l["id"].(string)

	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, "", ""); code != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй хүн үнэгүй хичээл үзэхгүй: %d", code)
	}
	for _, path := range []string{"/api/courses/" + cid, "/c/" + cid} {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != http.StatusOK || strings.Contains(string(b), "Нууц агуулга 42") || !strings.Contains(string(b), "Танилцуулга") {
			t.Fatalf("%s: гарчиг харагдаж, агуулга нуугдах ёстой (%d)", path, res.StatusCode)
		}
	}
	if code, r := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, s1, ""); code != http.StatusOK || r["content"] != "Нууц агуулга 42" {
		t.Fatalf("нэвтэрсэн суралцагч үнэгүй хичээлийг үзнэ: %d %v", code, r)
	}
}

// Чатыг зөвхөн бүртгэлтэй, нэвтэрсэн хэрэглэгч бичнэ: зочны токен олгохгүй, хуучин зочны токен ч ажиллахгүй.
func TestChatNeedsLogin(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	register(t, srv, "teach", "teacher")
	s1, _ := register(t, srv, "stud1", "student")
	if code, _ := call(t, srv, "POST", "/api/auth/guest", "", `{"name":"Зочин"}`); code == http.StatusCreated || code == http.StatusOK {
		t.Fatalf("зочны чатын токен олгохгүй: %d", code)
	}
	guest, _, err := auth.NewSigner("test", time.Hour).SignGuest("Зочин")
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := call(t, srv, "POST", "/api/teachers/teach/chat", guest, ""); code != http.StatusUnauthorized {
		t.Fatalf("зочин чат эхлүүлэхгүй: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/chat/ws?token="+guest, "", ""); code != http.StatusUnauthorized {
		t.Fatalf("зочин чатын сувагт холбогдохгүй: %d", code)
	}
	code, d := call(t, srv, "POST", "/api/teachers/teach/chat", s1, "")
	if code != http.StatusOK {
		t.Fatalf("нэвтэрсэн суралцагч чатлана: %d %v", code, d)
	}
	conv := d["conversation"].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/chat/"+conv+"/messages", guest, `{"body":"сайн уу"}`); code != http.StatusUnauthorized {
		t.Fatalf("зочин мессеж бичихгүй: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/chat/"+conv+"/messages", s1, `{"body":"Сайн байна уу"}`); code != http.StatusCreated {
		t.Fatalf("суралцагч мессеж бичнэ: %d", code)
	}
}
