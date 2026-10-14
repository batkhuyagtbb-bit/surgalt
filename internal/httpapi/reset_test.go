package httpapi

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/mail"
)

// fakeMail — илгээсэн захидлыг цуглуулна.
type fakeMail struct {
	mu   sync.Mutex
	sent []mail.Message
}

func (f *fakeMail) Send(_ context.Context, m mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
}

func (f *fakeMail) wait(t *testing.T, n int) []mail.Message {
	t.Helper()
	for i := 0; i < 100; i++ {
		f.mu.Lock()
		if len(f.sent) >= n {
			out := append([]mail.Message(nil), f.sent...)
			f.mu.Unlock()
			return out
		}
		f.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%d захидал хүлээсэн", n)
	return nil
}

var resetLinkRe = regexp.MustCompile(`/reset#t=([A-Za-z0-9_.-]+)`)

// Нууц үг сэргээх: бүртгэлтэй имэйл рүү 1 цагийн нэг удаагийн холбоос очно (бүртгэлгүй бол ч ижил хариу);
// солимогц өмнөх нэвтрэлт хүчингүй, хуучин нууц үг ажиллахгүй, холбоосыг давтан ашиглахгүй.
func TestPasswordReset(t *testing.T) {
	fm := &fakeMail{}
	srv, _ := newTestServerWith(t, func(c *Config) { c.Mail = fm })
	defer srv.Close()
	old, uid := register(t, srv, "bat", "student")

	if code, r := call(t, srv, "POST", "/api/auth/forgot", "", `{"email":"nobody@x.mn"}`); code != http.StatusOK || r["ok"] != true {
		t.Fatalf("бүртгэлгүй имэйлд ч ижил хариу: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/forgot", "", `{"email":"Bat@X.mn"}`); code != http.StatusOK {
		t.Fatalf("сэргээх хүсэлт: %d", code)
	}
	msgs := fm.wait(t, 1)
	time.Sleep(50 * time.Millisecond)
	if fm.wait(t, 1); len(fm.sent) != 1 || msgs[0].To != "bat@x.mn" || !strings.Contains(msgs[0].Subject, "нууц үг") {
		t.Fatalf("зөвхөн бүртгэлтэй хаяг руу нэг захидал: %+v", fm.sent)
	}
	m := resetLinkRe.FindStringSubmatch(msgs[0].Text)
	if m == nil || !strings.Contains(msgs[0].HTML, m[0]) {
		t.Fatalf("захидалд сэргээх холбоос: %q", msgs[0].Text)
	}
	tok := m[1]

	if code, _ := call(t, srv, "POST", "/api/auth/reset", "", `{"token":"`+tok+`","password":"12345"}`); code != http.StatusBadRequest {
		t.Fatalf("богино нууц үг: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/reset", "", `{"token":"хуурамч.токен","password":"newpass123"}`); code != http.StatusBadRequest {
		t.Fatalf("хуурамч токен: %d", code)
	}
	expired, _ := auth.NewSigner("test", time.Hour).Seal(resetClaim{K: resetKind, UID: uid, FP: "x"}, -time.Minute)
	if code, _ := call(t, srv, "POST", "/api/auth/reset", "", `{"token":"`+expired+`","password":"newpass123"}`); code != http.StatusBadRequest {
		t.Fatalf("хугацаа дууссан токен: %d", code)
	}
	code, r := call(t, srv, "POST", "/api/auth/reset", "", `{"token":"`+tok+`","password":"newpass123"}`)
	if code != http.StatusOK || r["token"] == nil {
		t.Fatalf("нууц үг солих: %d %v", code, r)
	}
	if code, _ := call(t, srv, "GET", "/api/me", old, ""); code != http.StatusUnauthorized {
		t.Fatalf("өмнөх нэвтрэлт хүчингүй болох ёстой: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/me", r["token"].(string), ""); code != http.StatusOK {
		t.Fatalf("шинэ нэвтрэлт ажиллана: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/login", "", `{"email":"bat@x.mn","password":"password1"}`); code == http.StatusOK {
		t.Fatal("хуучин нууц үг ажиллахгүй")
	}
	if code, _ := call(t, srv, "POST", "/api/auth/login", "", `{"email":"bat@x.mn","password":"newpass123"}`); code != http.StatusOK {
		t.Fatalf("шинэ нууц үгээр нэвтэрнэ: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/reset", "", `{"token":"`+tok+`","password":"another123"}`); code != http.StatusBadRequest {
		t.Fatalf("холбоосыг давтан ашиглахгүй: %d", code)
	}
}

// SMTP тохируулаагүй бодит орчинд сэргээх хүсэлт ажиллахгүй гэдгийг шууд хэлнэ (захидал хүлээлгэхгүй).
func TestPasswordResetNoMail(t *testing.T) {
	srv, _ := newTestServerWith(t, func(c *Config) { c.DevPayments = false })
	defer srv.Close()
	register(t, srv, "bat", "student")
	if code, _ := call(t, srv, "POST", "/api/auth/forgot", "", `{"email":"bat@x.mn"}`); code != http.StatusServiceUnavailable {
		t.Fatalf("SMTP-гүй: %d", code)
	}
}
