package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
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

	if code, r := call(t, srv, "POST", "/api/auth/forgot", "", forgotBody(t, srv, "nobody@x.mn")); code != http.StatusOK || r["ok"] != true {
		t.Fatalf("бүртгэлгүй имэйлд ч ижил хариу: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/forgot", "", forgotBody(t, srv, "Bat@X.mn")); code != http.StatusOK {
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

// forgotBody — captcha авч (тестэд хариуг санах ойгоос уншиж) сэргээх хүсэлтийн бие бэлдэнэ.
func forgotBody(t *testing.T, srv *httptest.Server, email string) string {
	t.Helper()
	code, c := call(t, srv, "GET", "/api/auth/captcha", "", "")
	img, _ := c["image"].(string)
	if code != http.StatusOK || !strings.HasPrefix(img, "data:image/png;base64,") {
		t.Fatalf("captcha: %d %v", code, c)
	}
	id := c["id"].(string)
	captchas.mu.Lock()
	ans := captchas.m[id].code
	captchas.mu.Unlock()
	b, _ := json.Marshal(map[string]string{"email": email, "captcha_id": id, "captcha": ans})
	return string(b)
}

// Сэргээх хүсэлт captcha-гүй, буруу эсвэл давтан ашигласан кодоор захидал илгээхгүй; зураг PNG, 4 оронтой.
func TestForgotCaptcha(t *testing.T) {
	fm := &fakeMail{}
	srv, st := newTestServerWith(t, func(c *Config) { c.Mail = fm })
	defer srv.Close()
	_, uid := register(t, srv, "cap", "student")
	_ = st.LinkIdentity(context.Background(), uid, "google", "g-cap") // Google-тэй холбоотой: захидалд сануулна
	if code, r := call(t, srv, "POST", "/api/auth/forgot", "", `{"email":"cap@x.mn"}`); code != http.StatusBadRequest || r["code"] != "captcha" {
		t.Fatalf("captcha-гүй: %d %v", code, r)
	}
	_, c := call(t, srv, "GET", "/api/auth/captcha", "", "")
	id := c["id"].(string)
	captchas.mu.Lock()
	ans := captchas.m[id].code
	captchas.mu.Unlock()
	if len(ans) != 4 || strings.Trim(ans, "23456789") != "" {
		t.Fatalf("код: %q", ans)
	}
	raw, _ := base64.StdEncoding.DecodeString(strings.TrimPrefix(c["image"].(string), "data:image/png;base64,"))
	if cfg, err := png.DecodeConfig(bytes.NewReader(raw)); err != nil || cfg.Width < 150 || cfg.Height != 56 {
		t.Fatalf("зураг: %v %+v", err, cfg)
	}
	wrong := "2222"
	if ans == wrong {
		wrong = "3333"
	}
	if code, r := call(t, srv, "POST", "/api/auth/forgot", "", `{"email":"cap@x.mn","captcha_id":"`+id+`","captcha":"`+wrong+`"}`); code != http.StatusBadRequest || r["code"] != "captcha" {
		t.Fatalf("буруу код: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/forgot", "", `{"email":"cap@x.mn","captcha_id":"`+id+`","captcha":"`+ans+`"}`); code != http.StatusBadRequest {
		t.Fatalf("нэг удаа буруу бичсэн кодыг дахин ашиглахгүй: %d", code)
	}
	body := forgotBody(t, srv, "cap@x.mn")
	if code, _ := call(t, srv, "POST", "/api/auth/forgot", "", body); code != http.StatusOK {
		t.Fatalf("зөв код: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/forgot", "", body); code != http.StatusBadRequest {
		t.Fatalf("зөв кодыг давтан ашиглахгүй: %d", code)
	}
	if msgs := fm.wait(t, 1); len(msgs) != 1 || msgs[0].To != "cap@x.mn" || !strings.Contains(msgs[0].Text, "«Google-ээр үргэлжлүүлэх»") {
		t.Fatalf("зөвхөн зөв кодоор нэг захидал: %+v", msgs)
	}
}
