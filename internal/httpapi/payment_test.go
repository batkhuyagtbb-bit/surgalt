package httpapi

import (
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
	"surgalt/internal/qpay"
	"surgalt/internal/qpay/qpaytest"
)

func newQPayServer(t *testing.T) (*httptest.Server, *qpaytest.Server) {
	t.Helper()
	fake := qpaytest.New()
	t.Cleanup(fake.Close)
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{TrustProxy: true, StorageFreeMB: 10, StoragePlans: []StoragePlan{{MB: 300, Price: 5000}}},
		newTestStore(t), auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	s.QPay = qpay.New(fake.URL, qpaytest.User, qpaytest.Pass, "TEST_INVOICE")
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return srv, fake
}

// Хичээл дангаар: QPay QR → банкны апп-аар төлөх → QPay callback → эрх нээгдэнэ (мэдэгдэл нэг л удаа).
func TestQPayLessonPayment(t *testing.T) {
	srv, fake := newQPayServer(t)
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Төлбөртэй","price":50000,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Логарифм","price":15000,"content":"x"}`)
	lid := l["id"].(string)
	s1, _ := register(t, srv, "stud", "student")

	code, d := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/buy", s1, "")
	if code != 201 {
		t.Fatalf("худалдан авах: %d %v", code, d)
	}
	pay, oid := d["payment"].(map[string]any), d["order"].(map[string]any)["id"].(string)
	if pay["provider"] != "qpay" || !strings.HasPrefix(pay["qr"].(string), "data:image/png;base64,") || pay["dev_pay"] != false {
		t.Fatalf("QPay QR ирэх ёстой: %v", pay)
	}
	if urls, _ := pay["urls"].([]any); len(urls) != 2 {
		t.Fatalf("банкны апп-ын холбоос: %v", pay["urls"])
	}
	invs := fake.Invoices()
	if len(invs) != 1 || invs[0].Amount != 15000 || invs[0].SenderNo != oid || !strings.Contains(invs[0].Callback, "/api/payments/qpay?o="+oid+"&t=") {
		t.Fatalf("нэхэмжлэх: %+v", invs)
	}
	// Дахин дарахад ижил захиалга, ижил нэхэмжлэх (QPay-д давхар үүсгэхгүй).
	if _, d2 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/buy", s1, ""); d2["order"].(map[string]any)["id"] != oid || len(fake.Invoices()) != 1 {
		t.Fatalf("давхар нэхэмжлэх үүсэв: %v %d", d2, len(fake.Invoices()))
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, s1, ""); code != http.StatusPaymentRequired {
		t.Fatalf("төлөөгүй хичээл 402: %d", code)
	}
	if _, o := call(t, srv, "GET", "/api/orders/"+oid, s1, ""); o["status"] != "pending" {
		t.Fatalf("төлөөгүй захиалга: %v", o)
	}
	fake.Pay(invs[0].ID)
	// Буруу гарын үсэгтэй callback — үл тооцно.
	if res, err := http.Get(srv.URL + "/api/payments/qpay?o=" + oid + "&t=bad"); err != nil || res.StatusCode != 404 {
		t.Fatalf("буруу callback: %v %v", res.StatusCode, err)
	}
	cb, _ := url.Parse(invs[0].Callback)
	for range 2 { // QPay давтан мэдэгдэж болно — нэг л удаа биелнэ
		res, err := http.Get(srv.URL + cb.RequestURI())
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if res.StatusCode != 200 || string(body) != "SUCCESS" {
			t.Fatalf("callback: %d %s", res.StatusCode, body)
		}
	}
	if _, o := call(t, srv, "GET", "/api/orders/"+oid, s1, ""); o["status"] != "paid" {
		t.Fatalf("төлөгдсөн байх ёстой: %v", o)
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, s1, ""); code != 200 {
		t.Fatalf("төлсний дараа хичээл нээгдэнэ: %d", code)
	}
	_, ns := call(t, srv, "GET", "/api/me/notifications", s1, "")
	paidN := 0
	for _, n := range ns["items"].([]any) {
		if strings.Contains(n.(map[string]any)["title"].(string), "амжилттай") {
			paidN++
		}
	}
	if paidN != 1 {
		t.Fatalf("төлбөрийн мэдэгдэл нэг л удаа: %d %v", paidN, ns)
	}
}

// Callback ирээгүй ч (локал орчин, сүлжээ) захиалгын төлөвийг асуухад QPay-ээс шалгаж нээнэ; токен хүчингүй бол дахин нэвтэрнэ.
func TestQPayPollingCoursePayment(t *testing.T) {
	srv, fake := newQPayServer(t)
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Багц","price":89000,"published":true}`)
	cid := c["id"].(string)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","content":"x"}`)
	s1, _ := register(t, srv, "stud", "student")
	code, d := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")
	if code != 201 || d["payment"].(map[string]any)["provider"] != "qpay" {
		t.Fatalf("багц: %d %v", code, d)
	}
	oid := d["order"].(map[string]any)["id"].(string)
	fake.ExpireToken()
	fake.Pay(fake.Invoices()[0].ID)
	if _, o := call(t, srv, "GET", "/api/orders/"+oid, s1, ""); o["status"] != "paid" {
		t.Fatalf("асуухад QPay-ээс шалгаж төлөгдсөн болно: %v", o)
	}
	if fake.Auths != 2 {
		t.Fatalf("хүчингүй токены дараа дахин нэвтрэх: %d", fake.Auths)
	}
	if _, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, ""); a["enrolled"] != true {
		t.Fatalf("багц төлсний дараа элссэн: %v", a)
	}
}

// QPay тохируулаагүй, DEV_PAYMENTS: QR нь демо төлбөрийн хуудас руу; утсаар (нэвтрэлтгүй) төлөхөд нээгдэнэ.
func TestDemoPaymentQR(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Демо","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Төлбөртэй хичээл","price":9900,"content":"x"}`)
	lid := l["id"].(string)
	s1, _ := register(t, srv, "stud", "student")
	_, d := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/buy", s1, "")
	pay, oid := d["payment"].(map[string]any), d["order"].(map[string]any)["id"].(string)
	link, _ := pay["pay_url"].(string)
	if pay["provider"] != "demo" || !strings.HasPrefix(pay["qr"].(string), "data:image/png;base64,") || !strings.Contains(link, "/pay/"+oid+"?t=") {
		t.Fatalf("демо QR: %v", pay)
	}
	u, _ := url.Parse(link)
	res, err := http.Get(srv.URL + u.RequestURI())
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(page), "Төлбөртэй хичээл") || !strings.Contains(string(page), "9,900") {
		t.Fatalf("демо төлбөрийн хуудас: %d", res.StatusCode)
	}
	if res, _ := http.Get(srv.URL + "/pay/" + oid + "?t=bad"); res.StatusCode != 404 {
		t.Fatalf("буруу гарын үсэгтэй хуудас 404: %d", res.StatusCode)
	}
	if code, _ := call(t, srv, "POST", "/api/orders/"+oid+"/dev-pay?t=bad", "", ""); code != 401 {
		t.Fatalf("гарын үсэггүй, нэвтрэлтгүй төлөх боломжгүй: %d", code)
	}
	if code, o := call(t, srv, "POST", "/api/orders/"+oid+"/dev-pay?"+u.RawQuery, "", ""); code != 200 || o["status"] != "paid" {
		t.Fatalf("утсаар демо төлөх: %d %v", code, o)
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, s1, ""); code != 200 {
		t.Fatalf("төлсний дараа нээгдэнэ: %d", code)
	}
}
