package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"surgalt/internal/store"
)

func TestConsent(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()

	// Нөхцөл, нууцлалын бодлогыг зөвшөөрөөгүй бол бүртгэхгүй.
	if code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"no@x.mn","password":"password1"}`); code != http.StatusBadRequest || !strings.Contains(fmt.Sprint(r["error"]), "зөвшөөр") {
		t.Fatalf("зөвшөөрөлгүй бүртгэл: %d %v", code, r)
	}
	code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"yes@x.mn","password":"password1","accept_terms":true}`)
	if code != http.StatusCreated {
		t.Fatalf("бүртгэл: %d %v", code, r)
	}
	u := r["user"].(map[string]any)
	if u["consent_version"] != TermsVersion || u["consent_needed"] != false {
		t.Fatalf("бүртгэлийн хариунд зөвшөөрөл тэмдэглэгдээгүй: %v", u)
	}
	tok, id := r["token"].(string), u["id"].(string)

	// Хуучин хувилбарыг зөвшөөрсөн (эсвэл Google-ээр шинээр орсон) хэрэглэгчээс дахин асууна.
	if _, err := st.UpdateUser(context.Background(), id, func(u *store.User) error { u.ConsentVersion = "2020-01-01"; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, m := call(t, srv, "GET", "/api/me", tok, ""); m["consent_needed"] != true || m["consent_version"] != "2020-01-01" {
		t.Fatalf("хуучин хувилбартай хэрэглэгч: %v", m)
	}
	// Хуучирсан хуудаснаас ирсэн хувилбарыг хүлээн авахгүй (дахин уншуулна).
	if code, _ := call(t, srv, "POST", "/api/me/consent", tok, `{"version":"2020-01-01"}`); code != http.StatusConflict {
		t.Fatalf("хуучин хувилбар: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/me/consent", "", `{"version":"`+TermsVersion+`"}`); code != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй: %d", code)
	}
	code, m := call(t, srv, "POST", "/api/me/consent", tok, `{"version":"`+TermsVersion+`"}`)
	if code != http.StatusOK || m["consent_needed"] != false || m["consent_version"] != TermsVersion {
		t.Fatalf("зөвшөөрөх: %d %v", code, m)
	}
	if _, m := call(t, srv, "GET", "/api/me", tok, ""); m["consent_needed"] != false {
		t.Fatalf("зөвшөөрсний дараа: %v", m)
	}
	// Нэвтрэхэд ч зөвшөөрлийн төлөв ирнэ (дахин /api/me асуух шаардлагагүй).
	if _, l := call(t, srv, "POST", "/api/auth/login", "", `{"email":"yes@x.mn","password":"password1"}`); l["user"].(map[string]any)["consent_version"] != TermsVersion {
		t.Fatalf("нэвтрэлтийн хариу: %v", l)
	}
}

func TestLegalPages(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	for _, c := range []struct{ path, want string }{
		{"/privacy", `id="monitoring"`},
		{"/terms", `href="/privacy#monitoring"`},
	} {
		res, err := http.Get(srv.URL + c.path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(res.Body)
		res.Body.Close()
		body := string(b)
		if res.StatusCode != http.StatusOK || !strings.Contains(body, c.want) || !strings.Contains(body, "Хувилбар "+TermsVersion) {
			t.Fatalf("%s: %d (%q байх ёстой)", c.path, res.StatusCode, c.want)
		}
	}
}
