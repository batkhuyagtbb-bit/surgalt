package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"surgalt/internal/store"
)

func TestConsent(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()

	// Нөхцөл, нууцлалын бодлогыг зөвшөөрөөгүй бол бүртгэхгүй.
	if code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"no@x.mn","password":"password1"}`); code != http.StatusBadRequest || !strings.Contains(fmt.Sprint(r["error"]), "зөвшөөр") {
		t.Fatalf("зөвшөөрөлгүй бүртгэл: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"noage@x.mn","password":"password1","accept_terms":true}`); code != http.StatusBadRequest || !strings.Contains(fmt.Sprint(r["error"]), "нас") {
		t.Fatalf("суралцагч насаа сонгоогүй: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"bagsh@x.mn","password":"password1","accept_terms":true,"role":"teacher"}`); code != http.StatusCreated {
		t.Fatalf("багш насаа сонгохгүй: %d %v", code, r)
	}
	code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"yes@x.mn","password":"password1","accept_terms":true,"age_group":"adult"}`)
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

// 18 хүрээгүй суралцагч: эцэг эх зөвшөөрлийн холбоосоор зөвшөөрнө (цуцалж ч болно); зөвшөөрөл ирэх хүртэл биечлэн
// уулзах цаг захиалахгүй; багш захиалгадаа 18 хүрээгүй эсэхийг харна; насаа хэлээгүй (Google г.м.) суралцагчаас асууна.
func TestGuardianConsent(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"kid@x.mn","password":"password1","accept_terms":true,"age_group":"minor","display_name":"Бат"}`)
	if code != http.StatusCreated {
		t.Fatalf("18 хүрээгүй бүртгэл: %d %v", code, r)
	}
	kid, kidID := r["token"].(string), r["user"].(map[string]any)["id"].(string)
	_, m := call(t, srv, "GET", "/api/me", kid, "")
	link, _ := m["guardian_link"].(string)
	if m["age_group"] != "minor" || m["guardian"] != nil || !strings.HasPrefix(link, "/guardian/") {
		t.Fatalf("18 хүрээгүй хэрэглэгчийн мэдээлэл: %v", m)
	}
	tok := strings.TrimPrefix(link, "/guardian/")

	day := time.Now().Add(48 * time.Hour).Truncate(time.Hour)
	at := func(h int) string { return day.Add(time.Duration(h) * time.Hour).UTC().Format(time.RFC3339) }
	slot := func(h int, mode string) string {
		code, out := callArr(t, srv, "POST", "/api/me/slots", tt, `{"starts":["`+at(h)+`"],"duration_min":60,"mode":"`+mode+`","location":"Номын сан"}`)
		if code != http.StatusCreated {
			t.Fatalf("цаг: %d %v", code, out)
		}
		return out[0].(map[string]any)["id"].(string)
	}
	off1, off2, on := slot(0, "offline"), slot(2, "offline"), slot(4, "online")
	if code, r := call(t, srv, "POST", "/api/slots/"+off1+"/book", kid, `{}`); code != http.StatusForbidden || !strings.Contains(fmt.Sprint(r["error"]), "эцэг эх") {
		t.Fatalf("зөвшөөрөлгүй биечлэн уулзахгүй: %d %v", code, r)
	}
	if code, r := call(t, srv, "POST", "/api/slots/"+on+"/book", kid, `{}`); code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("онлайн цагийг захиалж болно: %d %v", code, r)
	}

	// Зөвшөөрлийн хуудас: зөвхөн 18 хүрээгүй хүүхдийн жинхэнэ холбоос нээгдэнэ.
	page := func(path string) (int, string) {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}
	if code, body := page(link); code != http.StatusOK || !strings.Contains(body, "Бат") {
		t.Fatalf("зөвшөөрлийн хуудас: %d", code)
	}
	if code, _ := page("/guardian/" + kidID + ".xxxx"); code != http.StatusNotFound {
		t.Fatalf("хуурамч холбоос: %d", code)
	}
	for _, bad := range []string{
		`{"name":"Дорж","relation":"father","phone":"","accept":true}`,           // утасгүй
		`{"name":"Дорж","relation":"neighbor","phone":"99112233","accept":true}`, // хамаарал буруу
		`{"name":"Д","relation":"father","phone":"99112233","accept":true}`,      // нэр хэт богино
		`{"name":"Дорж","relation":"father","phone":"99112233"}`,                 // зөвшөөрөөгүй
	} {
		if code, _ := call(t, srv, "POST", "/api/guardian/"+tok, "", bad); code != http.StatusBadRequest {
			t.Fatalf("буруу зөвшөөрөл 400 (%s): %d", bad, code)
		}
	}
	if code, r := call(t, srv, "POST", "/api/guardian/"+tok, "", `{"name":"Дорж  Бат","relation":"father","phone":"9911 2233","accept":true}`); code != http.StatusOK {
		t.Fatalf("зөвшөөрөл: %d %v", code, r)
	}
	_, m = call(t, srv, "GET", "/api/me", kid, "")
	g, _ := m["guardian"].(map[string]any)
	if g == nil || g["name"] != "Дорж Бат" || g["relation"] != "father" || g["phone"] != "" || g["at"] == nil {
		t.Fatalf("зөвшөөрлийн дараа: %v", m)
	}
	if u, _ := st.UserByID(context.Background(), kidID); u.Guardian.Phone != "99112233" {
		t.Fatalf("эцэг эхийн утас хадгалагдах ёстой: %+v", u.Guardian)
	}
	if _, ns := call(t, srv, "GET", "/api/me/notifications", kid, ""); len(notifTitles(ns, NotifGuardian)) != 1 {
		t.Fatalf("хүүхдэд мэдэгдэл: %v", ns)
	}
	if code, r := call(t, srv, "POST", "/api/slots/"+off1+"/book", kid, `{}`); code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("зөвшөөрөлтэй бол биечлэн уулзаж болно: %d %v", code, r)
	}
	_, bs := callArrMap(t, srv, "GET", "/api/me/bookings", tt)
	if len(bs) != 2 || bs[0]["student_minor"] != true || bs[0]["guardian_ok"] != true {
		t.Fatalf("багшид 18 хүрээгүй эсэх харагдана: %v", bs)
	}

	// Цуцлах: хүүхэд ба биечлэн уулзалттай багшид мэдэгдэнэ; шинэ биечлэн цаг дахин хаагдана.
	if code, _ := call(t, srv, "DELETE", "/api/guardian/"+tok, "", ""); code != http.StatusOK {
		t.Fatalf("цуцлах: %d", code)
	}
	if _, m = call(t, srv, "GET", "/api/me", kid, ""); m["guardian"] != nil {
		t.Fatalf("цуцалсны дараа: %v", m)
	}
	if _, ns := call(t, srv, "GET", "/api/me/notifications", tt, ""); len(notifTitles(ns, NotifBooking)) == 0 || !strings.Contains(strings.Join(notifTitles(ns, NotifBooking), "|"), "зөвшөөрлөө цуцаллаа") {
		t.Fatalf("багшид цуцлалтын мэдэгдэл: %v", notifTitles(ns, NotifBooking))
	}
	if code, _ := call(t, srv, "POST", "/api/slots/"+off2+"/book", kid, `{}`); code != http.StatusForbidden {
		t.Fatalf("цуцалсны дараа биечлэн уулзахгүй: %d", code)
	}

	// Насанд хүрсэн суралцагчид холбоос байхгүй, хуучин холбоос нь ч ажиллахгүй.
	if _, err := st.UpdateUser(context.Background(), kidID, func(u *store.User) error { u.AgeGroup = store.AgeAdult; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, m := call(t, srv, "GET", "/api/me", kid, ""); m["guardian_link"] != nil {
		t.Fatalf("насанд хүрсэн: %v", m)
	}
	if code, _ := page(link); code != http.StatusNotFound {
		t.Fatalf("насанд хүрсэн суралцагчийн холбоос 404: %d", code)
	}

	// Насаа хэлээгүй (Google-ээр орсон г.м.) суралцагчаас зөвшөөрөхдөө асууна; нэг удаа л тохируулна.
	other, otherID := register(t, srv, "other", "student")
	if _, err := st.UpdateUser(context.Background(), otherID, func(u *store.User) error { u.AgeGroup = ""; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, m := call(t, srv, "GET", "/api/me", other, ""); m["consent_needed"] != true {
		t.Fatalf("насгүй суралцагчаас асуух ёстой: %v", m)
	}
	if code, _ := call(t, srv, "POST", "/api/me/consent", other, `{"version":"`+TermsVersion+`"}`); code != http.StatusBadRequest {
		t.Fatalf("насгүйгээр зөвшөөрөхгүй: %d", code)
	}
	if code, m := call(t, srv, "POST", "/api/me/consent", other, `{"version":"`+TermsVersion+`","age_group":"minor"}`); code != http.StatusOK || m["age_group"] != "minor" || m["consent_needed"] != false {
		t.Fatalf("нас сонгож зөвшөөрөх: %d %v", code, m)
	}
	if _, m := call(t, srv, "POST", "/api/me/consent", other, `{"version":"`+TermsVersion+`","age_group":"adult"}`); m["age_group"] != "minor" {
		t.Fatalf("насаа дараа нь өөрөө солихгүй: %v", m)
	}
}
