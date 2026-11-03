package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/verifymn"
)

// fakeVerify — Verify.MN-ийн хуурамч сервер: сесс бүрийн төлвийг тест өөрчилнө.
type fakeVerify struct {
	*httptest.Server
	mu     sync.Mutex
	status map[string]string
	bodies []map[string]string
	auth   string
}

func newFakeVerify(t *testing.T) *fakeVerify {
	f := &fakeVerify{status: map[string]string{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.auth = r.Header.Get("Authorization")
		switch {
		case r.Method == "POST" && r.URL.Path == "/sessions":
			var b map[string]string
			_ = json.NewDecoder(r.Body).Decode(&b)
			if len(b["phone"]) != 8 {
				w.WriteHeader(400)
				_, _ = io.WriteString(w, `{"message":"invalid phone"}`)
				return
			}
			id := "s" + string(rune('a'+len(f.bodies)))
			f.bodies = append(f.bodies, b)
			f.status[id] = verifymn.Pending
			_ = json.NewEncoder(w).Encode(map[string]any{"sessionId": id, "phone": b["phone"], "shortcode": "144773", "text": b["text"],
				"smsUri": "sms:144773?body=" + b["text"], "displayInstruction": "144773 руу илгээнэ үү", "expiresAt": time.Now().Add(5 * time.Minute)})
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/sessions/"):
			id := strings.TrimPrefix(r.URL.Path, "/sessions/")
			st, ok := f.status[id]
			if !ok {
				w.WriteHeader(404)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"sessionId": id, "sessionStatus": st, "callbackStatus": "PENDING", "expiresAt": time.Now().Add(time.Minute)})
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeVerify) set(id, st string) { f.mu.Lock(); f.status[id] = st; f.mu.Unlock() }
func (f *fakeVerify) last() map[string]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.bodies[len(f.bodies)-1]
}

// Нэвтэрсэн хэрэглэгч утсаа заавал баталгаажуулна (SMS-ээр); тэр дугаараараа нууц үгээ мартсан үед сэргээнэ.
// Токеныг өөр хүн ашиглахгүй, баталгаажаагүй/хугацаа дууссан/бүртгэлгүй дугаарт сэргээх холбоос өгөхгүй.
func TestPhoneVerifyAndReset(t *testing.T) {
	fv := newFakeVerify(t)
	st := newTestStore(t)
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{TrustProxy: true}, st, auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	tok, _ := register(t, srv, "phu", "student")
	other, _ := register(t, srv, "pho", "teacher")
	// Verify.MN тохируулаагүй: асуухгүй, сэргээх арга зөвхөн имэйл.
	if _, me := call(t, srv, "GET", "/api/me", tok, ""); me["phone_required"] != false {
		t.Fatalf("Verify.MN-гүй үед асуухгүй: %v", me["phone_required"])
	}
	if _, rc := call(t, srv, "GET", "/api/auth/recovery", "", ""); rc["phone"] != false {
		t.Fatalf("recovery: %v", rc)
	}
	s.Verify = verifymn.New(fv.URL, "vrf_test")
	if _, me := call(t, srv, "GET", "/api/me", tok, ""); me["phone_required"] != true || me["phone_verified"] != "" {
		t.Fatalf("нэвтэрмэгц утас асууна: %v %v", me["phone_required"], me["phone_verified"])
	}
	if _, rc := call(t, srv, "GET", "/api/auth/recovery", "", ""); rc["phone"] != true {
		t.Fatalf("recovery: %v", rc)
	}

	if code, _ := call(t, srv, "POST", "/api/me/phone/verify", tok, `{"phone":"+1 555 0100"}`); code != http.StatusBadRequest {
		t.Fatalf("гадаад дугаар: %d", code)
	}
	code, v := call(t, srv, "POST", "/api/me/phone/verify", tok, `{"phone":"9911-2233"}`)
	if code != 200 || v["phone"] != "99112233" || v["shortcode"] != "144773" || len(v["text"].(string)) != 6 || !strings.HasPrefix(v["sms_uri"].(string), "sms:144773") {
		t.Fatalf("сесс: %d %v", code, v)
	}
	if b := fv.last(); b["phone"] != "99112233" || b["text"] != v["text"] || !strings.HasPrefix(b["responseSms"], "surgalt.mn:") || !strings.HasSuffix(b["callback"], "/api/verify/callback") || fv.auth != "Bearer vrf_test" {
		t.Fatalf("Verify.MN-д илгээсэн: %v %q", b, fv.auth)
	}
	vt := v["token"].(string)
	if _, r := call(t, srv, "POST", "/api/me/phone/check", tok, `{"token":"`+vt+`"}`); r["status"] != "pending" {
		t.Fatalf("хүлээгдэж буй: %v", r)
	}
	fv.set("sa", verifymn.Verified)
	if code, _ := call(t, srv, "POST", "/api/me/phone/check", other, `{"token":"`+vt+`"}`); code != http.StatusForbidden {
		t.Fatalf("өөр хүний токен: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/me/phone/check", tok, `{"token":"хуурамч"}`); code != http.StatusBadRequest {
		t.Fatalf("хуурамч токен: %d", code)
	}
	if _, r := call(t, srv, "POST", "/api/me/phone/check", tok, `{"token":"`+vt+`"}`); r["status"] != "verified" || r["user"].(map[string]any)["phone_verified"] != "99112233" {
		t.Fatalf("баталгаажлаа: %v", r)
	}
	if _, me := call(t, srv, "GET", "/api/me", tok, ""); me["phone_required"] != false || me["phone_verified"] != "99112233" {
		t.Fatalf("дахин асуухгүй: %v %v", me["phone_required"], me["phone_verified"])
	}

	// Нууц үг мартсан: captcha → SMS → баталгаажмагц сэргээх холбоос.
	if code, r := call(t, srv, "POST", "/api/auth/forgot/phone", "", `{"phone":"99112233"}`); code != http.StatusBadRequest || r["code"] != "captcha" {
		t.Fatalf("captcha-гүй: %d %v", code, r)
	}
	body := strings.Replace(forgotBody(t, srv, "x@x.mn"), `"email":"x@x.mn"`, `"phone":"9911 2233"`, 1)
	code, fp := call(t, srv, "POST", "/api/auth/forgot/phone", "", body)
	if code != 200 || fp["token"] == nil {
		t.Fatalf("утсаар сэргээх: %d %v", code, fp)
	}
	ft := fp["token"].(string)
	if code, _ := call(t, srv, "POST", "/api/me/phone/check", tok, `{"token":"`+ft+`"}`); code != http.StatusBadRequest {
		t.Fatalf("сэргээх токеныг баталгаажуулахад ашиглахгүй: %d", code)
	}
	if _, r := call(t, srv, "POST", "/api/auth/forgot/phone/check", "", `{"token":"`+ft+`"}`); r["status"] != "pending" || r["accounts"] != nil {
		t.Fatalf("SMS ирээгүй үед холбоос өгөхгүй: %v", r)
	}
	fv.set("sb", verifymn.Verified)
	_, r := call(t, srv, "POST", "/api/auth/forgot/phone/check", "", `{"token":"`+ft+`"}`)
	accts, _ := r["accounts"].([]any)
	if r["status"] != "verified" || len(accts) != 1 || accts[0].(map[string]any)["username"] != "phu" || accts[0].(map[string]any)["email"] != "ph***@x.mn" {
		t.Fatalf("бүртгэл: %v", r)
	}
	rt := accts[0].(map[string]any)["token"].(string)
	if code, rr := call(t, srv, "POST", "/api/auth/reset", "", `{"token":"`+rt+`","password":"newpass123"}`); code != 200 || rr["token"] == nil {
		t.Fatalf("шинэ нууц үг: %d %v", code, rr)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/login", "", `{"email":"phu@x.mn","password":"newpass123"}`); code != 200 {
		t.Fatalf("шинэ нууц үгээр нэвтэрнэ: %d", code)
	}

	// Бүртгэлгүй дугаар баталгаажсан ч холбоос өгөхгүй; хугацаа дууссан.
	body = strings.Replace(forgotBody(t, srv, "x@x.mn"), `"email":"x@x.mn"`, `"phone":"88001122"`, 1)
	_, fp = call(t, srv, "POST", "/api/auth/forgot/phone", "", body)
	fv.set("sc", verifymn.Verified)
	if _, r := call(t, srv, "POST", "/api/auth/forgot/phone/check", "", `{"token":"`+fp["token"].(string)+`"}`); r["status"] != "none" || r["accounts"] != nil {
		t.Fatalf("бүртгэлгүй дугаар: %v", r)
	}
	body = strings.Replace(forgotBody(t, srv, "x@x.mn"), `"email":"x@x.mn"`, `"phone":"88001133"`, 1)
	_, fp = call(t, srv, "POST", "/api/auth/forgot/phone", "", body)
	fv.set("sd", verifymn.Expired)
	if _, r := call(t, srv, "POST", "/api/auth/forgot/phone/check", "", `{"token":"`+fp["token"].(string)+`"}`); r["status"] != "expired" {
		t.Fatalf("хугацаа дууссан: %v", r)
	}
	// Түр хадгалсан дугаар (SMS-гүй): дахин асуухгүй; утсаар сэргээхэд тэр дугаараас SMS ирмэгц баталгаажна.
	pend, _ := register(t, srv, "php", "student")
	if code, _ := call(t, srv, "PUT", "/api/me/phone", pend, `{"phone":"12"}`); code != http.StatusBadRequest {
		t.Fatalf("буруу дугаар: %d", code)
	}
	if code, me := call(t, srv, "PUT", "/api/me/phone", pend, `{"phone":"8800 4455"}`); code != 200 || me["phone_pending"] != "88004455" || me["phone_required"] != false || me["phone_verified"] != "" {
		t.Fatalf("түр хадгалах: %d %v", code, me)
	}
	body = strings.Replace(forgotBody(t, srv, "x@x.mn"), `"email":"x@x.mn"`, `"phone":"88004455"`, 1)
	_, fp = call(t, srv, "POST", "/api/auth/forgot/phone", "", body)
	fv.set("se", verifymn.Verified)
	_, r = call(t, srv, "POST", "/api/auth/forgot/phone/check", "", `{"token":"`+fp["token"].(string)+`"}`)
	if accts, _ := r["accounts"].([]any); r["status"] != "verified" || len(accts) != 1 || accts[0].(map[string]any)["username"] != "php" {
		t.Fatalf("түр хадгалсан дугаараар сэргээх: %v", r)
	}
	if _, me := call(t, srv, "GET", "/api/me", pend, ""); me["phone_verified"] != "88004455" || me["phone_pending"] != "" {
		t.Fatalf("SMS ирснээр баталгаажна: %v %v", me["phone_verified"], me["phone_pending"])
	}
	if got := maskEmail("google_1@users.noreply.surgalt"); got != "" {
		t.Errorf("орлуулсан имэйлийг харуулахгүй: %q", got)
	}
}
