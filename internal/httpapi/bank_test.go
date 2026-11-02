package httpapi

import (
	"context"
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
	"surgalt/internal/khanbank"
	"surgalt/internal/store"
)

// fakeCGW — Хаан банкны Corporate Gateway-ийн хуурамч сервер: mode = ok | reject | fail.
type fakeCGW struct {
	*httptest.Server
	mu    sync.Mutex
	mode  string
	calls []map[string]any
	paths []string
}

func newFakeCGW(t *testing.T) *fakeCGW {
	f := &fakeCGW{mode: "ok"}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/auth/token":
			_, _ = io.WriteString(w, `{"access_token":"tok","access_token_expires_in":1799,"developer_email":"dev@x.mn"}`)
		case r.URL.Path == "/v1/accounts":
			_, _ = io.WriteString(w, `{"accounts":[{"number":"9000000001","type":"L","currency":"MNT","status":"OPEN"},{"number":"5021712565","type":"D","currency":"MNT","status":"OPEN","balance":5000000,"avalaibleBalance":4900000,"name":"ХАРИЛЦАХ"}]}`)
		case r.URL.Path == "/v1/accounts/cam":
			_, _ = io.WriteString(w, `{"iban":"MN170015001105123456","name":"Б*т Б**д","msgId":"7"}`)
		case strings.HasPrefix(r.URL.Path, "/v1/transfer/"):
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			f.mu.Lock()
			f.calls, f.paths = append(f.calls, b), append(f.paths, r.URL.Path)
			mode := f.mode
			f.mu.Unlock()
			switch mode {
			case "reject":
				w.WriteHeader(400)
				_, _ = io.WriteString(w, `{"code":"CGW014","message":"Хүлээн авагчийн нэр таарахгүй байна"}`)
			case "fail":
				w.WriteHeader(504)
				_, _ = io.WriteString(w, `{"fault":{"faultstring":"Gateway Timeout"}}`)
			default:
				_, _ = io.WriteString(w, `{"uuid":"u1","journalNo":7148753,"systemDate":"2026/11/03"}`)
			}
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeCGW) set(mode string) { f.mu.Lock(); f.mode = mode; f.mu.Unlock() }
func (f *fakeCGW) last() (string, map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return "", nil
	}
	return f.paths[len(f.paths)-1], f.calls[len(f.calls)-1]
}

// Татах хүсэлтийг Хаан банкаар шилжүүлэх: банк татгалзвал хүлээгдэж буй руу (шалтгаантай), хариу тодорхойгүй бол
// «шалгах» (дахин автоматаар илгээхгүй) → хянагч хуулгаар шалгаад дахин илгээнэ → шилжүүлсэн (журналтай);
// Хаан банкны IBAN → банк доторх шилжүүлэг; автомат горим (данс 24 цагаас өмнө солигдсон бол хүлээнэ).
func TestBankPayouts(t *testing.T) {
	cgw := newFakeCGW(t)
	st := newTestStore(t)
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{TrustProxy: true, DevPayments: true, AdminEmails: []string{"boss@x.mn"}, Bank: BankConfig{TranPassword: "pw", AutoMax: 1000000}},
		st, auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.Khan = khanbank.New(cgw.URL+"/v1", "ck", "sk")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	ctx := context.Background()

	tt, tid := register(t, srv, "bankteach", "teacher")
	boss, _ := register(t, srv, "boss", "student")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Курс","price":30000,"published":true}`)
	cid := c["id"].(string)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","content":"x"}`)
	st1, _ := register(t, srv, "bankstud", "student")
	_, d := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", st1, "")
	call(t, srv, "POST", "/api/orders/"+d["order"].(map[string]any)["id"].(string)+"/dev-pay", st1, "")

	if code, _ := call(t, srv, "PUT", "/api/me/payout/account", tt, `{"bank":"golomt","account_no":"1105 123 456","account_name":"Бат  Болд","schedule":"manual"}`); code != 200 {
		t.Fatalf("данс: %d", code)
	}
	_, p := call(t, srv, "POST", "/api/me/payout/withdraw", tt, `{"amount":10000}`)
	pid := p["id"].(string)
	if p["status"] != "pending" {
		t.Fatalf("автомат горимгүй үед хүлээгдэнэ: %v", p)
	}
	// Хянагчийн банкны мэдээлэл — багшид харагдахгүй.
	if code, _ := call(t, srv, "GET", "/api/admin/bank", tt, ""); code != http.StatusForbidden {
		t.Fatalf("банкны мэдээлэл зөвхөн хянагчид: %d", code)
	}
	_, b := call(t, srv, "GET", "/api/admin/bank", boss, "")
	if b["configured"] != true || b["test"] != true || b["from_account"] != "5021712565" || b["available"] != 4900000.0 || b["login"] != "dev@x.mn" || b["tran_password"] != true {
		t.Fatalf("банк: %v", b)
	}
	if _, h := call(t, srv, "GET", "/api/admin/payouts/"+pid+"/holder", boss, ""); h["name"] != "Б*т Б**д" || h["entered"] != "Бат Болд" {
		t.Fatalf("ДНС нэр: %v", h)
	}

	// 1) Банк татгалзсан: хүлээгдэж буй руу, шалтгаантай.
	cgw.set("reject")
	_, r1 := call(t, srv, "POST", "/api/admin/payouts/"+pid+"/send", boss, "")
	if r1["status"] != "pending" || !strings.Contains(r1["note"].(string), "нэр таарахгүй") {
		t.Fatalf("татгалзсан: %v", r1)
	}
	path, body := cgw.last()
	if path != "/v1/transfer/interbank" || body["toBank"] != "150000" || body["toBankBic"] != "GLMTMNUB" || body["toAccount"] != "1105123456" ||
		body["toAccountName"] != "Бат Болд" || body["fromAccount"] != "5021712565" || body["amount"] != 10000.0 || body["loginName"] != "dev@x.mn" || body["tranPassword"] != "pw" {
		t.Fatalf("банк хоорондын шилжүүлгийн бие: %s %v", path, body)
	}
	// 2) Хариу тодорхойгүй: «шалгах», давтан илгээхгүй.
	cgw.set("fail")
	_, r2 := call(t, srv, "POST", "/api/admin/payouts/"+pid+"/send", boss, "")
	if r2["status"] != "review" {
		t.Fatalf("тодорхойгүй: %v", r2)
	}
	if code, _ := call(t, srv, "POST", "/api/admin/payouts/"+pid+"/send", boss, ""); code != http.StatusConflict {
		t.Fatalf("шалгах төлөвтэйг дахин илгээхгүй: %d", code)
	}
	if bal, _ := st.TeacherBalance(ctx, tid); bal.Pending != 10000 {
		t.Fatalf("шалгах төлөвтэй дүн үлдэгдлээс хасагдсан хэвээр: %+v", bal)
	}
	_, open := callAny(t, srv, "GET", "/api/admin/payouts", boss)
	if len(open) != 1 || open[0].(map[string]any)["status"] != "review" {
		t.Fatalf("шийдвэрлэгдээгүй жагсаалт: %v", open)
	}
	// 3) Хуулгаар шилжээгүйг батлаад дахин илгээнэ → шилжүүлсэн.
	call(t, srv, "POST", "/api/admin/payouts/"+pid, boss, `{"status":"pending","note":"хуулгад байхгүй"}`)
	cgw.set("ok")
	_, r3 := call(t, srv, "POST", "/api/admin/payouts/"+pid+"/send", boss, "")
	if r3["status"] != "paid" || r3["journal"] != "7148753" || !strings.Contains(r3["note"].(string), "журнал 7148753") {
		t.Fatalf("шилжүүлсэн: %v", r3)
	}
	if bal, _ := st.TeacherBalance(ctx, tid); bal.Paid != 10000 || bal.Pending != 0 {
		t.Fatalf("шилжүүлсний дараа үлдэгдэл: %+v", bal)
	}
	if _, ns := callList(t, srv, "GET", "/api/me/notifications", tt); countTitle(ns, "данс руу шилжлээ") != 1 {
		t.Fatalf("багшид мэдэгдэл: %v", ns)
	}

	// Хаан банкны IBAN → банк доторх шилжүүлэг (дансны дугаар руу хөрвүүлнэ); автомат горим.
	s.cfg.Bank.Auto = true
	call(t, srv, "PUT", "/api/me/payout/account", tt, `{"bank":"khan","account_no":"MN12 0005 0000 1234 5678","account_name":"Бат Болд","schedule":"manual"}`)
	_, p2 := call(t, srv, "POST", "/api/me/payout/withdraw", tt, `{"amount":5000}`)
	time.Sleep(300 * time.Millisecond)
	if ps, _ := st.Payouts(ctx, tid, store.PayoutPending, 10); len(ps) != 1 || ps[0].ID != p2["id"] {
		t.Fatalf("данс саяхан солигдсон: хянагч батлах хүртэл хүлээнэ: %+v", ps)
	}
	call(t, srv, "POST", "/api/admin/payouts/"+p2["id"].(string), boss, `{"status":"rejected","note":"туршилт"}`)
	a, _ := st.PayoutAccount(ctx, tid)
	a.UpdatedAt = time.Now().Add(-48 * time.Hour)
	if err := st.SetPayoutAccount(ctx, a); err != nil {
		t.Fatal(err)
	}
	_, p3 := call(t, srv, "POST", "/api/me/payout/withdraw", tt, `{"amount":5000}`)
	var got *store.Payout
	for i := 0; i < 50; i++ {
		ps, _ := st.Payouts(ctx, tid, store.PayoutPaid, 10)
		for j := range ps {
			if ps[j].ID == p3["id"] {
				got = &ps[j]
			}
		}
		if got != nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if got == nil || got.ProcessedBy != "Хаан банк (автомат)" {
		t.Fatalf("автомат шилжүүлэг: %+v", got)
	}
	if path, body := cgw.last(); path != "/v1/transfer/domestic" || body["toAccount"] != "12345678" || body["toBank"] != nil {
		t.Fatalf("банк доторх: %s %v", path, body)
	}
}

// callAny — массив буцаадаг API.
func callAny(t *testing.T, srv *httptest.Server, method, path, token string) (int, []any) {
	req, _ := http.NewRequest(method, srv.URL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out []any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}
