package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/oauth"
	"surgalt/internal/store"
)

func fbSigned(secret, userID string) string {
	p := base64.RawURLEncoding.EncodeToString([]byte(`{"algorithm":"HMAC-SHA256","issued_at":1700000000,"user_id":"` + userID + `"}`))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(p))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) + "." + p
}

// Facebook-ийн өгөгдөл устгах хүсэлт: зөвхөн Facebook-ээр нэвтэрдэг бүртгэл бүхэлдээ устана, нууц үгтэй бүртгэлээс
// зөвхөн холболт, Facebook-ийн зураг; сургалттай багшийнх хэсэгчлэн. Төлвийг /data-deletion?code= харуулна.
func TestFacebookDataDeletion(t *testing.T) {
	st := newTestStore(t)
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{TrustProxy: true, PublicURL: "https://surgalt.mn"}, st, auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.OAuth = oauth.NewRegistry("https://surgalt.mn", oauth.Credentials{}, oauth.Credentials{}, oauth.Credentials{ClientID: "fbapp", ClientSecret: "fbsecret"}, oauth.Credentials{})
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	ctx := context.Background()
	fbPic := "https://scontent.xx.fbcdn.net/v/p.jpg"

	only, _, err := s.loginWithIdentity(ctx, oauth.Identity{Provider: "facebook", Subject: "fb1", Email: "only@x.mn", Name: "Зөвхөн FB", AvatarURL: fbPic}, store.RoleStudent)
	if err != nil {
		t.Fatal(err)
	}
	_, pwID := register(t, srv, "pwuser", "student")
	if err := st.LinkIdentity(ctx, pwID, "facebook", "fb2"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateUser(ctx, pwID, func(u *store.User) error { u.AvatarURL = fbPic; return nil }); err != nil {
		t.Fatal(err)
	}
	teach, _, err := s.loginWithIdentity(ctx, oauth.Identity{Provider: "facebook", Subject: "fb3", Email: "t@x.mn", Name: "Багш"}, store.RoleTeacher)
	if err != nil {
		t.Fatal(err)
	}
	tok, _ := s.tokens.SignUser(teach.ID, string(store.RoleTeacher), teach.DisplayName)
	if code, _ := call(t, srv, "POST", "/api/courses", tok, `{"title":"Сургалт","price":0,"published":true}`); code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("сургалт: %d", code)
	}

	post := func(sr string) (int, map[string]string) {
		res, err := http.PostForm(srv.URL+"/auth/facebook/data-deletion", url.Values{"signed_request": {sr}})
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]string
		_ = json.NewDecoder(res.Body).Decode(&out)
		return res.StatusCode, out
	}
	page := func(code string) string {
		res, err := http.Get(srv.URL + "/data-deletion?code=" + url.QueryEscape(code))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		if res.StatusCode != 200 {
			t.Fatalf("хуудас: %d", res.StatusCode)
		}
		return string(b)
	}

	if code, _ := post(fbSigned("wrong", "fb1")); code != http.StatusBadRequest {
		t.Fatalf("буруу гарын үсэг: %d", code)
	}
	if _, err := st.UserByID(ctx, only.ID); err != nil {
		t.Fatal("буруу хүсэлтээр устах ёсгүй")
	}
	want := map[string]struct {
		st, text string
	}{"fb1": {"d", "Устгагдсан"}, "fb2": {"u", "Facebook холболт устгагдсан"}, "fb3": {"p", "Хэсэгчлэн устгагдсан"}, "fb404": {"n", "Хадгалсан мэдээлэл алга"}}
	for _, fb := range []string{"fb1", "fb2", "fb3", "fb404"} {
		code, r := post(fbSigned("fbsecret", fb))
		if code != http.StatusOK || r["confirmation_code"] == "" || r["url"] != "https://surgalt.mn/data-deletion?code="+r["confirmation_code"] {
			t.Fatalf("%s: %d %v", fb, code, r)
		}
		if r["confirmation_code"][0] != want[fb].st[0] {
			t.Fatalf("%s: төлөв %q", fb, r["confirmation_code"])
		}
		if html := page(r["confirmation_code"]); !strings.Contains(html, want[fb].text) || !strings.Contains(html, r["confirmation_code"]) {
			t.Fatalf("%s: хуудсанд төлөв алга", fb)
		}
	}
	if _, err := st.UserByID(ctx, only.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("зөвхөн FB бүртгэл устах ёстой: %v", err)
	}
	pw, err := st.UserByID(ctx, pwID)
	if err != nil || pw.AvatarURL != "" {
		t.Fatalf("нууц үгтэй бүртгэл үлдэж, FB зураг арилах ёстой: %v %q", err, pw.AvatarURL)
	}
	for _, fb := range []string{"fb2", "fb3"} {
		if _, err := st.UserByIdentity(ctx, "facebook", fb); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("%s холболт салах ёстой: %v", fb, err)
		}
	}
	if _, err := st.UserByID(ctx, teach.ID); err != nil {
		t.Fatal("сургалттай багшийн бүртгэл үлдэх ёстой")
	}
	if html := page("d123-xxxxxxxxxxxx"); !strings.Contains(html, "Код танигдсангүй") {
		t.Fatal("хуурамч код танигдах ёсгүй")
	}
	if html := page(""); !strings.Contains(html, "Хэрхэн устгуулах") || strings.Contains(html, "Баталгаажуулах код:") {
		t.Fatal("заавар хуудас")
	}
}
