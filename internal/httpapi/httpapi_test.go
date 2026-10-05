package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/chat"
	"surgalt/internal/files"
	"surgalt/internal/store"
)

func newTestServer(t *testing.T) (*httptest.Server, store.Store) {
	st := store.NewMemory()
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	s := New(Config{DevPayments: true, WebhookSecret: "wh", StorageFreeMB: 10, StoragePlans: []StoragePlan{{MB: 300, Price: 5000}}},
		st, auth.NewSigner("test", time.Hour), chat.NewHub(), fs, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return httptest.NewServer(s.Handler()), st
}

func call(t *testing.T, srv *httptest.Server, method, path, token, body string) (int, map[string]any) {
	req, _ := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// callList: мэдэгдлүүд ({"items": [...]}) шиг жагсаалт буцаадаг API.
func callList(t *testing.T, srv *httptest.Server, method, path, token string) (int, []any) {
	code, out := call(t, srv, method, path, token, "")
	items, _ := out["items"].([]any)
	return code, items
}

func TestPurchaseFlow(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	_, tr := call(t, srv, "POST", "/api/auth/register", "", `{"username":"teach","email":"t@x.mn","password":"password1","role":"teacher"}`)
	tt := tr["token"].(string)
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Курс","price":1000,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Төлбөртэй"}`)
	lid := l["id"].(string)

	_, sr := call(t, srv, "POST", "/api/auth/register", "", `{"username":"stud","email":"s@x.mn","password":"password1"}`)
	st := sr["token"].(string)
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, st, ""); code != http.StatusPaymentRequired {
		t.Fatalf("402 хүлээсэн, %d", code)
	}
	_, e := call(t, srv, "POST", "/api/courses/"+cid+"/enroll", st, "")
	oid := e["order"].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/orders/"+oid+"/dev-pay", st, ""); code != 200 {
		t.Fatal("төлбөр")
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, st, ""); code != 200 {
		t.Fatalf("төлсний дараа 200 хүлээсэн, %d", code)
	}
	if code, _ := call(t, srv, "GET", "/t/teach", "", ""); code != 200 {
		t.Fatal("профайл хуудас")
	}
	if code, _ := call(t, srv, "GET", "/t/stud", "", ""); code != 404 {
		t.Fatal("суралцагчид нээлттэй профайл байх ёсгүй")
	}
}

func TestSingleLessonPurchase(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	_, tr := call(t, srv, "POST", "/api/auth/register", "", `{"username":"teach","email":"t@x.mn","password":"password1","role":"teacher"}`)
	tt := tr["token"].(string)
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Хичээлээр","price":0,"published":true}`)
	cid := c["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"x","is_free":false,"price":0}`); code != 400 {
		t.Fatal("багц үнэгүй үед үнэгүй-төлбөртэй хичээл хүлээн авагдлаа")
	}
	_, a := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"A","price":5000}`)
	_, b := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"B","price":7000}`)
	la, lb := a["id"].(string), b["id"].(string)

	_, sr := call(t, srv, "POST", "/api/auth/register", "", `{"username":"stud","email":"s@x.mn","password":"password1"}`)
	st := sr["token"].(string)
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", st, "") // үнэгүй "элсэлт" төлбөртэйг нээхгүй
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+la, st, ""); code != 402 {
		t.Fatalf("402 хүлээсэн, %d", code)
	}
	_, o := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+la+"/buy", st, "")
	order := o["order"].(map[string]any)
	if order["amount"].(float64) != 5000 {
		t.Fatal("дүн буруу")
	}
	call(t, srv, "POST", "/api/orders/"+order["id"].(string)+"/dev-pay", st, "")
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+la, st, ""); code != 200 {
		t.Fatal("худалдаж авсан хичээл нээгдээгүй")
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lb, st, ""); code != 402 {
		t.Fatal("бусад хичээл нээгдэх ёсгүй")
	}
}

func register(t *testing.T, srv *httptest.Server, username, role string) (token, id string) {
	t.Helper()
	code, r := call(t, srv, "POST", "/api/auth/register", "",
		`{"username":"`+username+`","email":"`+username+`@x.mn","password":"password1","role":"`+role+`"}`)
	if code != http.StatusCreated {
		t.Fatalf("бүртгэл %s: %d %v", username, code, r)
	}
	return r["token"].(string), r["user"].(map[string]any)["id"].(string)
}

// Нүүр хуудасны самбар: элссэн сургалт, дангаар авсан хичээл, төлөгдөөгүй захиалга,
// шууд хичээл, багш, чат — бүгд нэг хүсэлтээр, зөвхөн өөрийнх нь.
func TestHome(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, tid := register(t, srv, "teach", "teacher")
	_, free := call(t, srv, "POST", "/api/courses", tt, `{"title":"Үнэгүй","price":0,"published":true}`)
	_, bundle := call(t, srv, "POST", "/api/courses", tt, `{"title":"Багц","price":9000,"published":true}`)
	_, single := call(t, srv, "POST", "/api/courses", tt, `{"title":"Хичээлээр","price":0,"published":true}`)
	fid, bid, sid := free["id"].(string), bundle["id"].(string), single["id"].(string)
	call(t, srv, "POST", "/api/courses/"+fid+"/lessons", tt, `{"title":"F1","is_free":true}`)
	_, sl := call(t, srv, "POST", "/api/courses/"+sid+"/lessons", tt, `{"title":"S1","price":3000}`)

	ctx := t.Context()
	soon := time.Now().Add(time.Hour)
	for _, m := range []store.Meeting{
		{TeacherID: tid, CourseID: fid, Title: "Нээлттэй", StartsAt: soon, DurationMin: 30, MeetURL: "https://meet.google.com/free"},
		{TeacherID: tid, CourseID: bid, Title: "Багцын", StartsAt: soon, DurationMin: 30, MeetURL: "https://meet.google.com/paid"},
		{TeacherID: tid, Title: "Уулзалт: Нууц хүн", StartsAt: soon, DurationMin: 30, MeetURL: "https://meet.google.com/private"},
	} {
		if err := st.CreateMeeting(ctx, &m); err != nil {
			t.Fatal(err)
		}
	}

	st1, uid := register(t, srv, "stud", "student")
	st2, _ := register(t, srv, "other", "student")

	if code, _ := call(t, srv, "GET", "/api/me/home", "", ""); code != http.StatusUnauthorized {
		t.Fatalf("нэвтрээгүй үед 401 хүлээсэн, %d", code)
	}
	call(t, srv, "POST", "/api/courses/"+fid+"/enroll", st1, "")
	call(t, srv, "POST", "/api/courses/"+bid+"/enroll", st1, "") // төлөгдөөгүй захиалга үлдэнэ
	_, o := call(t, srv, "POST", "/api/courses/"+sid+"/lessons/"+sl["id"].(string)+"/buy", st1, "")
	call(t, srv, "POST", "/api/orders/"+o["order"].(map[string]any)["id"].(string)+"/dev-pay", st1, "")
	_, ch := call(t, srv, "POST", "/api/teachers/teach/chat", st1, "")
	call(t, srv, "POST", "/api/chat/"+ch["conversation"].(map[string]any)["id"].(string)+"/messages", st1, `{"body":"Сайн уу"}`)

	code, h := call(t, srv, "GET", "/api/me/home", st1, "")
	if code != 200 {
		t.Fatalf("home: %d %v", code, h)
	}
	list := func(k string) []any { return h[k].([]any) }
	if h["user"].(map[string]any)["id"] != uid {
		t.Fatal("өөр хэрэглэгчийн мэдээлэл")
	}
	access := map[string]string{}
	for _, c := range list("courses") {
		m := c.(map[string]any)
		access[m["course"].(map[string]any)["id"].(string)] = m["access"].(string)
		if m["teacher"].(map[string]any)["username"] != "teach" {
			t.Fatal("сургалтын багш дутуу")
		}
	}
	if len(access) != 2 || access[fid] != "enrolled" || access[sid] != "lessons" {
		t.Fatalf("сургалтууд буруу: %v", access)
	}
	if n := len(list("lessons")); n != 1 {
		t.Fatalf("1 авсан хичээл хүлээсэн, %d", n)
	}
	if p := list("pending"); len(p) != 1 || p[0].(map[string]any)["course_id"] != bid {
		t.Fatalf("төлөгдөөгүй захиалга буруу: %v", p)
	}
	// Зөвхөн өөрийн сургалтуудын шууд хичээл; багцыг төлөөгүй тул тэр нь огт орохгүй.
	ms := list("meetings")
	if len(ms) != 1 || ms[0].(map[string]any)["meet_url"] != "https://meet.google.com/free" {
		t.Fatalf("шууд хичээл буруу: %v", ms)
	}
	if len(list("teachers")) != 1 || len(list("chats")) != 1 {
		t.Fatalf("багш/чат буруу: %v %v", h["teachers"], h["chats"])
	}
	if _, ok := h["teacher"]; ok {
		t.Fatal("суралцагчид багшийн самбар байх ёсгүй")
	}

	// Өөр суралцагч юу ч харахгүй.
	_, h2 := call(t, srv, "GET", "/api/me/home", st2, "")
	for _, k := range []string{"courses", "lessons", "pending", "meetings", "teachers", "chats"} {
		if n := len(h2[k].([]any)); n != 0 {
			t.Fatalf("бусдын %s харагдлаа (%d)", k, n)
		}
	}

	// Багш: самбар + бүрдлийн оноо.
	_, ht := call(t, srv, "GET", "/api/me/home", tt, "")
	tp, ok := ht["teacher"].(map[string]any)
	if !ok || tp["courses"].(float64) != 3 || tp["students"].(float64) != 1 {
		t.Fatalf("багшийн самбар буруу: %v", ht["teacher"])
	}

	// Нээлттэй профайл: Meet холбоос болон ганцаарчилсан уулзалт хэзээ ч гарахгүй.
	req, _ := http.NewRequest("GET", srv.URL+"/api/teachers/teach", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	body := string(raw)
	if strings.Contains(body, "meet.google.com") || strings.Contains(body, "Нууц хүн") {
		t.Fatal("нээлттэй профайлд Meet холбоос эсвэл хувийн уулзалт задарлаа")
	}
	if !strings.Contains(body, "Багцын") {
		t.Fatal("нийтлэгдсэн сургалтын шууд хичээл профайлд алга")
	}
	page, _ := http.Get(srv.URL + "/t/teach")
	raw, _ = io.ReadAll(page.Body)
	page.Body.Close()
	if strings.Contains(string(raw), "meet.google.com") {
		t.Fatal("профайл хуудсанд Meet холбоос задарлаа")
	}
}

func TestSmartProfile(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	st, _ := register(t, srv, "stud", "student")

	if code, _ := call(t, srv, "GET", "/api/me/profile/insights", st, ""); code != http.StatusForbidden {
		t.Fatalf("суралцагчид 403 хүлээсэн, %d", code)
	}
	_, before := call(t, srv, "GET", "/api/me/profile/insights", tt, "")
	if before["score"].(float64) != 0 {
		t.Fatalf("хоосон профайл 0 оноотой байх ёстой: %v", before["score"])
	}

	bad := []string{
		`{"display_name":"A","links":{"facebook":"https://evil.example/facebook.com"}}`,
		`{"display_name":"A","links":{"website":"javascript:alert(1)"}}`,
		`{"display_name":"A","links":{"tiktok":"https://tiktok.com/x"}}`,
		`{"display_name":"A","subjects":["1","2","3","4","5","6","7","8","9"]}`,
		`{"display_name":"A","subjects":["` + strings.Repeat("я", 31) + `"]}`,
		`{"display_name":"A","location":"` + strings.Repeat("я", 61) + `"}`,
		`{"display_name":"A","cover_url":"/files/000000000000000000000000/public/x.webp"}`, // бусдын файл
		`{"display_name":"A","cover_url":"javascript:alert(1)"}`,
	}
	for _, b := range bad {
		if code, r := call(t, srv, "PUT", "/api/me/profile", tt, b); code != http.StatusBadRequest {
			t.Fatalf("400 хүлээсэн (%s): %d %v", b, code, r)
		}
	}

	bio := strings.Repeat("Туршлагатай багш. ", 10)
	code, u := call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Багш Бат","headline":"Математик","bio":"`+bio+`",
		"cover_url":"https://example.mn/cover.webp","subjects":[" Математик ","математик","","Геометр"],"location":"  Улаанбаатар  ",
		"links":{"facebook":"m.facebook.com/bat","website":"","youtube":"https://youtu.be/abc"}}`)
	if code != 200 {
		t.Fatalf("профайл хадгалах: %d %v", code, u)
	}
	if s := u["subjects"].([]any); len(s) != 2 || s[0] != "Математик" || s[1] != "Геометр" {
		t.Fatalf("чиглэл цэгцлэгдээгүй: %v", s)
	}
	links := u["links"].(map[string]any)
	if len(links) != 2 || links["facebook"] != "https://m.facebook.com/bat" || u["location"] != "Улаанбаатар" {
		t.Fatalf("холбоос/байршил буруу: %v %v", links, u["location"])
	}

	_, after := call(t, srv, "GET", "/api/me/profile/insights", tt, "")
	if after["score"].(float64) <= before["score"].(float64) {
		t.Fatal("оноо өсөөгүй")
	}
	tips := after["tips"].([]any)
	if tips[0].(map[string]any)["done"].(bool) || !tips[len(tips)-1].(map[string]any)["done"].(bool) {
		t.Fatal("дуусаагүй зөвлөмж эхэнд байх ёстой")
	}

	// Нээлттэй профайлд шинэ талбарууд шууд (кэш цэвэрлэгдэж) гарна.
	_, p := call(t, srv, "GET", "/api/teachers/teach", "", "")
	pt := p["teacher"].(map[string]any)
	if pt["cover_url"] != "https://example.mn/cover.webp" {
		t.Fatalf("нүүр зураг нээлттэй профайлд алга: %v", pt["cover_url"])
	}
	if len(pt["subjects"].([]any)) != 2 || len(pt["links"].([]any)) != 2 || pt["links"].([]any)[0].(map[string]any)["key"] != "facebook" {
		t.Fatalf("нээлттэй профайл буруу: %v", pt)
	}
	// Суралцагч нэрээ сольж чадна, бусад талбар хэвээр.
	if code, _ := call(t, srv, "PUT", "/api/me/profile", st, `{"display_name":"Шинэ нэр"}`); code != 200 {
		t.Fatalf("суралцагчийн профайл: %d", code)
	}
}

// Бүртгэл зөвхөн имэйл, нууц үгээр: хэрэглэгчийн нэр имэйлээс үүсэж, давхардвал дагавар авна.
func TestRegisterWithoutUsername(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	code, a := call(t, srv, "POST", "/api/auth/register", "", `{"email":"Bat.Erdene@x.mn","password":"password1","role":"teacher"}`)
	if code != http.StatusCreated {
		t.Fatalf("бүртгэл: %d %v", code, a)
	}
	ua := a["user"].(map[string]any)
	if ua["username"] != "bat_erdene" || ua["display_name"] != "bat_erdene" {
		t.Fatalf("нэр имэйлээс үүсээгүй: %v", ua)
	}
	if code, _ := call(t, srv, "GET", "/t/bat_erdene", "", ""); code != 200 {
		t.Fatal("автомат нэртэй багшийн профайл нээгдсэнгүй")
	}
	// Өөр домэйн, ижил нэр -> дагавартай өөр хэрэглэгчийн нэр.
	code, b := call(t, srv, "POST", "/api/auth/register", "", `{"email":"bat.erdene@y.mn","password":"password1"}`)
	ub, _ := b["user"].(map[string]any)
	if code != http.StatusCreated || ub["username"] == "bat_erdene" || !strings.HasPrefix(ub["username"].(string), "bat_erdene_") {
		t.Fatalf("давхардсан нэр шийдэгдээгүй: %d %v", code, b)
	}
	// Ижил имэйл -> ойлгомжтой 409.
	if code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"bat.erdene@x.mn","password":"password1"}`); code != http.StatusConflict || !strings.Contains(r["error"].(string), "имэйл") {
		t.Fatalf("давхар имэйл: %d %v", code, r)
	}
	// Богино/кирилл нэртэй имэйл ч бүртгэгдэнэ.
	if code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"a@x.mn","password":"password1"}`); code != http.StatusCreated {
		t.Fatalf("богино имэйл: %d %v", code, r)
	}
}

// Профайлын холбоос автоматаар үүсээд, багш дараа нь өөрөө солино.
func TestChangeUsername(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	_, a := call(t, srv, "POST", "/api/auth/register", "", `{"email":"bagsh@x.mn","password":"password1","role":"teacher"}`)
	tok := a["token"].(string)
	register(t, srv, "taken", "student")
	_, c := call(t, srv, "POST", "/api/courses", tok, `{"title":"Курс","price":0,"published":true}`)
	cid := c["id"].(string)
	if code, _ := call(t, srv, "GET", "/t/bagsh", "", ""); code != 200 { // кэшийг дулаацуулна
		t.Fatal("автомат холбоос ажиллахгүй байна")
	}
	call(t, srv, "GET", "/c/"+cid, "", "")

	for body, want := range map[string]int{
		`{"username":"ab"}`:            http.StatusBadRequest,
		`{"username":"Буруу нэр"}`:     http.StatusBadRequest,
		`{"username":"../admin"}`:      http.StatusBadRequest,
		`{"username":"taken"}`:         http.StatusConflict,
		`{"username":"  Math_Bagsh "}`: http.StatusOK, // тайрч, жижиг үсэг болгоно
	} {
		if code, r := call(t, srv, "PUT", "/api/me/username", tok, body); code != want {
			t.Fatalf("%s: %d хүлээсэн, %d %v", body, want, code, r)
		}
	}
	if code, _ := call(t, srv, "PUT", "/api/me/username", "", `{"username":"x_y_z"}`); code != http.StatusUnauthorized {
		t.Fatal("нэвтрээгүй хүн холбоос сольж болохгүй")
	}
	if code, _ := call(t, srv, "GET", "/t/math_bagsh", "", ""); code != 200 {
		t.Fatal("шинэ холбоос нээгдсэнгүй")
	}
	if code, _ := call(t, srv, "GET", "/t/bagsh", "", ""); code != 404 {
		t.Fatal("хуучин холбоос кэшээс үйлчилсээр байна")
	}
	res, err := http.Get(srv.URL + "/c/" + cid)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(raw), "/t/math_bagsh") || strings.Contains(string(raw), `"/t/bagsh"`) {
		t.Fatal("сургалтын хуудас багшийн хуучин холбоосыг заасаар байна")
	}
	// Шинэ токен шинэ нэртэй; хуучин нэр чөлөөлөгдөж, өөр хүн авч болно.
	_, r := call(t, srv, "PUT", "/api/me/username", tok, `{"username":"math_bagsh"}`)
	if r["user"].(map[string]any)["username"] != "math_bagsh" || r["token"] == "" {
		t.Fatalf("хариу буруу: %v", r)
	}
	st, _ := register(t, srv, "someone", "student")
	if code, _ := call(t, srv, "PUT", "/api/me/username", st, `{"username":"bagsh"}`); code != 200 {
		t.Fatal("чөлөөлөгдсөн нэрийг авч чадсангүй")
	}
}

func TestPasswordLength(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	if code, _ := call(t, srv, "POST", "/api/auth/register", "", `{"email":"a1@x.mn","password":"12345"}`); code != http.StatusBadRequest {
		t.Fatalf("5 тэмдэгттэй нууц үг татгалзагдах ёстой, %d", code)
	}
	if code, r := call(t, srv, "POST", "/api/auth/register", "", `{"email":"a2@x.mn","password":"123456"}`); code != http.StatusCreated {
		t.Fatalf("6 тэмдэгттэй нууц үг зөвшөөрөгдөх ёстой, %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/auth/login", "", `{"email":"a2@x.mn","password":"123456"}`); code != 200 {
		t.Fatalf("6 тэмдэгттэй нууц үгээр нэвтэрч чадсангүй, %d", code)
	}
}

// Бүлэг чат: багш нээнэ, элссэн суралцагч орно, бусад хүн орохгүй, нэрс харагдана.
func TestGroupChat(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Бүлэгтэй курс","price":0,"published":true}`)
	cid := c["id"].(string)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"L1","is_free":true}`)
	s1, _ := register(t, srv, "stud1", "student")
	s2, _ := register(t, srv, "stud2", "student")
	call(t, srv, "PUT", "/api/me/profile", s1, `{"display_name":"Болд Бат"}`)
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")

	// Элссэн суралцагч түрүүлж орсон ч бүлэг үүснэ; элсээгүй хүнд үүсгэхгүй.
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/chat", s2, ""); code != http.StatusForbidden {
		t.Fatalf("элсээгүй суралцагчид 403 хүлээсэн, %d", code)
	}
	code, g := call(t, srv, "POST", "/api/courses/"+cid+"/chat", s1, "")
	if code != 200 || g["me"] != "visitor" || len(g["messages"].([]any)) != 1 {
		t.Fatalf("суралцагч бүлэг нээж чадсангүй: %d %v", code, g)
	}
	code, g = call(t, srv, "POST", "/api/courses/"+cid+"/chat", tt, "")
	if code != 200 || g["me"] != "teacher" {
		t.Fatalf("багш бүлэг нээж чадсангүй: %d %v", code, g)
	}
	conv := g["conversation"].(map[string]any)
	gid := conv["id"].(string)
	if conv["kind"] != "group" || conv["course_id"] != cid || len(g["messages"].([]any)) != 1 {
		t.Fatalf("бүлгийн мэдээлэл буруу: %v", g)
	}
	// Дахин дуудахад ижил бүлэг, давхар мэндчилгээ үүсэхгүй.
	_, g2 := call(t, srv, "POST", "/api/courses/"+cid+"/chat", tt, "")
	if g2["conversation"].(map[string]any)["id"] != gid || len(g2["messages"].([]any)) != 1 {
		t.Fatal("бүлэг давхардлаа")
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/chat", s2, ""); code != http.StatusForbidden {
		t.Fatalf("элсээгүй суралцагчид 403 хүлээсэн, %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/chat/"+gid+"/messages", s2, ""); code != http.StatusNotFound {
		t.Fatalf("элсээгүй хүн мессеж уншиж болохгүй, %d", code)
	}
	code, m := call(t, srv, "POST", "/api/chat/"+gid+"/messages", s1, `{"body":"Сайн байцгаана уу!"}`)
	if code != http.StatusCreated || m["sender"] != "visitor" || m["sender_name"] != "Болд Бат" || m["sender_id"] == "" {
		t.Fatalf("суралцагчийн мессеж: %d %v", code, m)
	}
	code, j := call(t, srv, "POST", "/api/courses/"+cid+"/chat", s1, "")
	if code != 200 || len(j["messages"].([]any)) != 2 || j["me"] != "visitor" {
		t.Fatalf("суралцагч бүлэгт орж чадсангүй: %d %v", code, j)
	}
	// Нүүр хуудасны чат жагсаалтад бүлэг гарна.
	_, h := call(t, srv, "GET", "/api/me/home", s1, "")
	chats := h["chats"].([]any)
	if len(chats) != 1 || chats[0].(map[string]any)["kind"] != "group" || chats[0].(map[string]any)["title"] != "Бүлэгтэй курс" {
		t.Fatalf("нүүр хуудасны бүлэг чат буруу: %v", chats)
	}
	// Багшийн жагсаалтад бүлэг харагдана.
	_, list := call(t, srv, "GET", "/api/me/conversations", tt, "")
	_ = list
	res, _ := http.NewRequest("GET", srv.URL+"/api/me/conversations", nil)
	res.Header.Set("Authorization", "Bearer "+tt)
	resp, err := http.DefaultClient.Do(res)
	if err != nil {
		t.Fatal(err)
	}
	var convs []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&convs)
	resp.Body.Close()
	if len(convs) != 1 || convs[0]["kind"] != "group" {
		t.Fatalf("багшийн жагсаалт: %v", convs)
	}
}

// Багш суралцагчдаа харж, шууд чат эхлүүлнэ; гаднын хүнтэй эхлүүлж болохгүй.
func TestStudentsAndDirectChat(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Курс А","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Төлбөртэй хичээл","price":2000}`)
	s1, uid1 := register(t, srv, "stud1", "student")
	s2, uid2 := register(t, srv, "stud2", "student")
	_, uid3 := register(t, srv, "stranger", "student")
	call(t, srv, "PUT", "/api/me/profile", s1, `{"display_name":"Болд"}`)
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")
	_, o := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+l["id"].(string)+"/buy", s2, "")
	call(t, srv, "POST", "/api/orders/"+o["order"].(map[string]any)["id"].(string)+"/dev-pay", s2, "")

	req, _ := http.NewRequest("GET", srv.URL+"/api/me/students", nil)
	req.Header.Set("Authorization", "Bearer "+tt)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&rows)
	resp.Body.Close()
	if len(rows) != 2 {
		t.Fatalf("2 суралцагч хүлээсэн: %v", rows)
	}
	byID := map[string]map[string]any{}
	for _, r := range rows {
		byID[r["user"].(map[string]any)["id"].(string)] = r
	}
	if byID[uid1]["user"].(map[string]any)["display_name"] != "Болд" || !byID[uid1]["courses"].([]any)[0].(map[string]any)["enrolled"].(bool) {
		t.Fatalf("элссэн суралцагч буруу: %v", byID[uid1])
	}
	c2 := byID[uid2]["courses"].([]any)[0].(map[string]any)
	if c2["enrolled"].(bool) || len(c2["lessons"].([]any)) != 1 || c2["lessons"].([]any)[0] != "Төлбөртэй хичээл" {
		t.Fatalf("хичээл авсан суралцагч буруу: %v", c2)
	}
	if _, has := byID[uid1]["user"].(map[string]any)["email"]; has {
		t.Fatal("суралцагчийн имэйл задрах ёсгүй")
	}
	if code, _ := call(t, srv, "GET", "/api/me/students", s1, ""); code != http.StatusForbidden {
		t.Fatal("суралцагч жагсаалт харж болохгүй")
	}
	if code, _ := call(t, srv, "POST", "/api/me/students/"+uid3+"/chat", tt, ""); code != http.StatusForbidden {
		t.Fatal("гаднын хүнтэй чат эхлүүлж болохгүй")
	}
	code, d := call(t, srv, "POST", "/api/me/students/"+uid1+"/chat", tt, "")
	if code != 200 || d["me"] != "teacher" {
		t.Fatalf("суралцагчтай чат: %d %v", code, d)
	}
	convID := d["conversation"].(map[string]any)["id"].(string)
	call(t, srv, "POST", "/api/chat/"+convID+"/messages", tt, `{"body":"Сайн уу Болд, хичээлд тавтай морил!"}`)
	// Суралцагч нүүр хуудаснаасаа энэ яриаг харна.
	_, h := call(t, srv, "GET", "/api/me/home", s1, "")
	chats := h["chats"].([]any)
	if len(chats) != 1 || chats[0].(map[string]any)["id"] != convID {
		t.Fatalf("суралцагчид багшийн яриа харагдсангүй: %v", chats)
	}
	// Жагсаалтад яриа холбогдсон.
	req2, _ := http.NewRequest("GET", srv.URL+"/api/me/courses/"+cid+"/students", nil)
	req2.Header.Set("Authorization", "Bearer "+tt)
	resp2, _ := http.DefaultClient.Do(req2)
	rows = nil
	_ = json.NewDecoder(resp2.Body).Decode(&rows)
	resp2.Body.Close()
	found := false
	for _, r := range rows {
		if r["user"].(map[string]any)["id"] == uid1 && r["conv_id"] == convID {
			found = true
		}
	}
	if !found {
		t.Fatalf("conv_id холбогдоогүй: %v", rows)
	}
}

// Дараалсан нээлт: өмнөхөө үзээгүй бол түгжээтэй, хугацаатай хичээл хүлээнэ, AlwaysOpen болон
// багц төлсөн (UnlockAllPaid) суралцагчид бүгд нээлттэй.
func TestDripUnlock(t *testing.T) {
	srv, st := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Дараалсан","price":0,"published":true,"drip":true,"unlock_all_paid":true}`)
	cid := c["id"].(string)
	if c["drip"] != true {
		t.Fatalf("drip хадгалагдаагүй: %v", c)
	}
	_, l1 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true}`)
	_, l2 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Хоёр","price":1000,"unlock_after_h":0}`)
	_, l3 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Гурав","price":1000,"unlock_after_h":168}`)
	_, l4 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Дөрөв","price":1000,"always_open":true}`)
	id := func(m map[string]any) string { return m["id"].(string) }
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"x","price":1,"unlock_after_h":99999}`); code != 400 {
		t.Fatal("хэт урт хугацаа татгалзагдах ёстой")
	}

	s1, uid := register(t, srv, "stud", "student")
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")
	for _, l := range []map[string]any{l2, l3, l4} { // бүх төлбөртэй хичээлийг дангаар авна
		_, o := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+id(l)+"/buy", s1, "")
		call(t, srv, "POST", "/api/orders/"+o["order"].(map[string]any)["id"].(string)+"/dev-pay", s1, "")
	}
	// Эхний хичээлийг үзээгүй тул 2 түгжээтэй; 4 (always_open) нээлттэй.
	if code, r := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+id(l2), s1, ""); code != http.StatusLocked || !strings.Contains(r["error"].(string), "Нэг") {
		t.Fatalf("2-р хичээл түгжээтэй байх ёстой: %d %v", code, r)
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+id(l4), s1, ""); code != 200 {
		t.Fatal("always_open хичээл нээлттэй байх ёстой")
	}
	_, a := call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, "")
	states := a["states"].(map[string]any)
	if states[id(l1)].(map[string]any)["open"] != true || states[id(l2)].(map[string]any)["reason"] != "prev" {
		t.Fatalf("төлөв буруу: %v", states)
	}
	// 1-ийг үзэхэд 2 шууд нээгдэнэ (0 цаг), 3 нь 7 хоногийн таймертай.
	call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+id(l1), s1, "")
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+id(l2), s1, ""); code != 200 {
		t.Fatal("өмнөхөө үзсэний дараа 2 нээгдэх ёстой")
	}
	code, r := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+id(l3), s1, "")
	if code != http.StatusLocked || r["state"].(map[string]any)["reason"] != "timer" {
		t.Fatalf("3 таймертай байх ёстой: %d %v", code, r)
	}
	// Дууссан гэж тэмдэглэх → явц.
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+id(l1)+"/complete", s1, "")
	_, a = call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, "")
	if a["done"].(float64) != 1 {
		t.Fatalf("дууссан тоо буруу: %v", a["done"])
	}
	// Хугацаа өнгөрсөн мэт: 2-ын үзсэн цагийг 8 хоногийн өмнө болгож 3 нээгдэхийг шалгана (санах ойн store).
	if mem, ok := st.(*store.Memory); ok {
		mem.BackdateProgress(uid, id(l2), 8*24*time.Hour)
		if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+id(l3), s1, ""); code != 200 {
			t.Fatal("хугацаа өнгөрсний дараа 3 нээгдэх ёстой")
		}
	}
	// Багцын төлбөр төлсөн суралцагчид (UnlockAllPaid) бүгд шууд нээлттэй.
	_, cb := call(t, srv, "POST", "/api/courses", tt, `{"title":"Багц","price":5000,"published":true,"drip":true,"unlock_all_paid":true}`)
	bid := id(cb)
	call(t, srv, "POST", "/api/courses/"+bid+"/lessons", tt, `{"title":"A"}`)
	_, b2 := call(t, srv, "POST", "/api/courses/"+bid+"/lessons", tt, `{"title":"B","unlock_after_h":720}`)
	s2, _ := register(t, srv, "payer", "student")
	_, e := call(t, srv, "POST", "/api/courses/"+bid+"/enroll", s2, "")
	call(t, srv, "POST", "/api/orders/"+e["order"].(map[string]any)["id"].(string)+"/dev-pay", s2, "")
	if code, _ := call(t, srv, "GET", "/api/courses/"+bid+"/lessons/"+id(b2), s2, ""); code != 200 {
		t.Fatal("багц төлсөн суралцагчид бүх хичээл шууд нээлттэй байх ёстой")
	}
}

func TestLessonSections(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Бүлэгтэй","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l1 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"section":"  1-р   бүлэг "}`)
	if l1["section"] != "1-р бүлэг" {
		t.Fatalf("бүлгийн нэр цэвэрлэгдээгүй: %v", l1["section"])
	}
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Хоёр","is_free":true,"section":"1-р бүлэг"}`)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Гурав","is_free":true,"section":"2-р бүлэг"}`)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"x","is_free":true,"section":"`+strings.Repeat("а", 81)+`"}`); code != 400 {
		t.Fatal("хэт урт бүлгийн нэр татгалзагдах ёстой")
	}
	// Засварт бүлэг солигдоно.
	_, up := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+l1["id"].(string), tt, `{"title":"Нэг","is_free":true,"section":"Шинэ"}`)
	if up["section"] != "Шинэ" {
		t.Fatalf("бүлэг шинэчлэгдээгүй: %v", up)
	}
	_, mine := call(t, srv, "GET", "/api/me/courses/"+cid, tt, "")
	if ls := mine["lessons"].([]any); ls[0].(map[string]any)["section"] != "Шинэ" {
		t.Fatalf("багшийн жагсаалтад бүлэг алга: %v", ls[0])
	}
	// Нээлттэй хуудас бүлгүүдээр (анх таарсан дарааллаар) бүлэглэнэ.
	res, err := http.Get(srv.URL + "/c/" + cid)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	html := string(body)
	for _, want := range []string{`class="lesson-group`, "Шинэ", "1-р бүлэг", "2-р бүлэг"} {
		if !strings.Contains(html, want) {
			t.Fatalf("сургалтын хуудсанд %q алга", want)
		}
	}
	if strings.Index(html, `data-section="Шинэ"`) > strings.Index(html, `data-section="1-р бүлэг"`) || strings.Index(html, `data-section="1-р бүлэг"`) > strings.Index(html, `data-section="2-р бүлэг"`) {
		t.Fatal("бүлгүүдийн дараалал буруу")
	}
}

func TestReorderLessons(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Зөөх","price":0,"published":true,"drip":true}`)
	cid := c["id"].(string)
	var ids []string
	for _, n := range []string{"А", "Б", "В"} {
		_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"`+n+`","is_free":true}`)
		ids = append(ids, l["id"].(string))
	}
	// В-г эхэнд, "1-р бүлэг"-т; А, Б-г "2-р бүлэг"-т.
	body := `{"items":[{"id":"` + ids[2] + `","section":" 1-р  бүлэг"},{"id":"` + ids[0] + `","section":"2-р бүлэг"},{"id":"` + ids[1] + `","section":"2-р бүлэг"}]}`
	code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lesson-order", tt, body)
	if code != 200 {
		t.Fatalf("дараалал хадгалагдсангүй: %d", code)
	}
	_, mine := call(t, srv, "GET", "/api/me/courses/"+cid, tt, "")
	ls := mine["lessons"].([]any)
	got := map[string][2]any{}
	for _, x := range ls {
		m := x.(map[string]any)
		got[m["id"].(string)] = [2]any{m["position"], m["section"]}
	}
	if got[ids[2]] != [2]any{float64(1), "1-р бүлэг"} || got[ids[0]] != [2]any{float64(2), "2-р бүлэг"} || got[ids[1]] != [2]any{float64(3), "2-р бүлэг"} {
		t.Fatalf("буруу дараалал: %v", got)
	}
	// Дутуу жагсаалт ба бусдын хичээлийг татгалзана.
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lesson-order", tt, `{"items":[{"id":"`+ids[0]+`"}]}`); code != 400 {
		t.Fatalf("дутуу жагсаалт 400 байх ёстой: %d", code)
	}
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lesson-order", tt, `{"items":[{"id":"`+ids[0]+`"},{"id":"`+ids[0]+`"},{"id":"`+ids[1]+`"}]}`); code != 400 {
		t.Fatalf("давхардсан id 400 байх ёстой: %d", code)
	}
	other, _ := register(t, srv, "other", "teacher")
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lesson-order", other, body); code == 200 {
		t.Fatal("өөр багш дараалал өөрчилж болохгүй")
	}
}

func TestLessonBlocksQuiz(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, tid := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Блок","price":5000,"published":true}`)
	cid := c["id"].(string)
	blocks := `[{"id":"h001","type":"heading","text":"Оршил"},{"id":"t001","type":"text","text":"**Тод** текст"},
		{"id":"i001","type":"image","url":"/files/` + tid + `/private/abc__a.webp","text":"Зураг"},
		{"id":"q001","type":"quiz","quiz":{"question":"2+2?","options":["3","4","5"],"correct":[1],"explain":"Учир нь 4"}}]`
	code, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":`+blocks+`}`)
	if code != 201 {
		t.Fatalf("блоктой хичээл үүссэнгүй: %d %v", code, l)
	}
	lid := l["id"].(string)
	// Буруу блокууд татгалзагдана.
	for _, bad := range []string{
		`[{"id":"x1","type":"text","text":"богино id"}]`,
		`[{"id":"a001","type":"script","text":"x"}]`,
		`[{"id":"a001","type":"image","url":"/files/other/private/a.webp"}]`,
		`[{"id":"a001","type":"quiz","quiz":{"question":"?","options":["a"],"correct":[0]}}]`,
		`[{"id":"a001","type":"quiz","quiz":{"question":"?","options":["a","b"],"correct":[]}}]`,
		`[{"id":"a001","type":"quiz","quiz":{"question":"?","options":["a","b"],"correct":[0,1]}}]`,
		`[{"id":"a001","type":"text","text":"a"},{"id":"a001","type":"text","text":"b"}]`,
	} {
		if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"x","is_free":true,"blocks":`+bad+`}`); code != 400 {
			t.Fatalf("буруу блок зөвшөөрөгдлөө (%d): %s", code, bad)
		}
	}
	// Суралцагч зөв хариултыг харахгүй, зураг гарын үсэгтэй.
	s1, _ := register(t, srv, "stud", "student")
	_, got := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, s1, "")
	bs := got["blocks"].([]any)
	q := bs[3].(map[string]any)["quiz"].(map[string]any)
	if _, leak := q["correct"]; leak || q["explain"] != nil {
		t.Fatalf("зөв хариулт задарсан: %v", q)
	}
	if u := bs[2].(map[string]any)["url"].(string); !strings.Contains(u, "sig=") {
		t.Fatalf("хувийн зураг гарын үсэггүй: %s", u)
	}
	// Хариулах: буруу → зөв, явцад хадгалагдана.
	_, r1 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/quiz/q001", s1, `{"answer":[0]}`)
	if r1["correct"] != false || r1["explain"] != "Учир нь 4" {
		t.Fatalf("буруу хариулт: %v", r1)
	}
	_, r2 := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/quiz/q001", s1, `{"answer":[1]}`)
	if r2["correct"] != true {
		t.Fatalf("зөв хариулт: %v", r2)
	}
	_, acc := call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, "")
	if p := acc["progress"].(map[string]any)[lid].(map[string]any); p["quiz"].(map[string]any)["q001"] != true {
		t.Fatalf("асуултын үр дүн хадгалагдаагүй: %v", p)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/quiz/nope", s1, `{"answer":[1]}`); code != 404 {
		t.Fatal("байхгүй асуулт 404")
	}
	// Төлбөртэй хичээлийн асуултад эрхгүй хүн хариулахгүй.
	_, pl := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Төлбөртэй","price":1000,"blocks":[{"id":"q002","type":"quiz","quiz":{"question":"?","options":["a","b"],"correct":[0]}}]}`)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+pl["id"].(string)+"/quiz/q002", s1, `{"answer":[0]}`); code != http.StatusPaymentRequired {
		t.Fatalf("төлбөргүй хариулах ёсгүй: %d", code)
	}
	// Багш өөрийн засварын жагсаалтад зөв хариултыг харна.
	_, mine := call(t, srv, "GET", "/api/me/courses/"+cid, tt, "")
	tq := mine["lessons"].([]any)[0].(map[string]any)["blocks"].([]any)[3].(map[string]any)["quiz"].(map[string]any)
	if fmt.Sprint(tq["correct"]) != "[1]" {
		t.Fatalf("багшид зөв хариулт харагдах ёстой: %v", tq)
	}
}

func TestQuizKinds(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, tid := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Төрлүүд","price":0,"published":true}`)
	cid := c["id"].(string)
	blocks := `[{"id":"qtxt","type":"quiz","quiz":{"kind":"text","question":"5x6?","answers":["30","гучин"]}},
		{"id":"qmat","type":"quiz","quiz":{"kind":"match","question":"Холбо","left":["Япон","Франц","Хятад"],"right":["Токио","Парис","Бээжин"]}},
		{"id":"qimg","type":"quiz","quiz":{"kind":"image","question":"Зүрхийг заа","image":"/files/` + tid + `/private/abc__x.webp","spot":{"x":40,"y":50,"r":10}}}]`
	code, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Т","is_free":true,"blocks":`+blocks+`}`)
	if code != 201 {
		t.Fatalf("үүссэнгүй: %d %v", code, l)
	}
	lid := l["id"].(string)
	base := "/api/courses/" + cid + "/lessons/" + lid + "/quiz/"
	s1, _ := register(t, srv, "stud", "student")
	_, got := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, s1, "")
	bs := got["blocks"].([]any)
	if q := bs[0].(map[string]any)["quiz"].(map[string]any); q["answers"] != nil {
		t.Fatal("бичгийн хариулт задарсан")
	}
	right := bs[1].(map[string]any)["quiz"].(map[string]any)["right"].([]any)
	if right[0] == "Токио" && right[1] == "Парис" && right[2] == "Бээжин" {
		t.Fatal("харгалзуулах баруун багана холилдоогүй")
	}
	if bs[2].(map[string]any)["quiz"].(map[string]any)["spot"] != nil {
		t.Fatal("зургийн зөв хэсэг задарсан")
	}
	if _, r := call(t, srv, "POST", base+"qtxt", s1, `{"text":"  Гучин. "}`); r["correct"] != true {
		t.Fatalf("бичгийн хариулт зөв байх ёстой: %v", r)
	}
	if _, r := call(t, srv, "POST", base+"qtxt", s1, `{"text":"31"}`); r["correct"] != false {
		t.Fatal("буруу бичгийн хариулт")
	}
	// Харгалзуулах: харагдаж буй баруун баганаас зөв индексийг олно.
	idx := func(v string) int {
		for i, x := range right {
			if x == v {
				return i
			}
		}
		return -1
	}
	match := fmt.Sprintf(`{"match":[%d,%d,%d]}`, idx("Токио"), idx("Парис"), idx("Бээжин"))
	if _, r := call(t, srv, "POST", base+"qmat", s1, match); r["correct"] != true {
		t.Fatalf("харгалзуулах зөв байх ёстой: %v", r)
	}
	if _, r := call(t, srv, "POST", base+"qimg", s1, `{"point":[45,52]}`); r["correct"] != true {
		t.Fatalf("зургийн цэг зөв байх ёстой: %v", r)
	}
	if _, r := call(t, srv, "POST", base+"qimg", s1, `{"point":[80,20]}`); r["correct"] != false || r["spot"] == nil {
		t.Fatalf("зургийн цэг буруу ба зөв хэсэг ирэх ёстой: %v", r)
	}
	for _, bad := range []string{`{"kind":"text","question":"?","answers":[]}`, `{"kind":"match","question":"?","left":["a","b"],"right":["x"]}`, `{"kind":"image","question":"?","image":"/files/` + tid + `/private/a.webp"}`} {
		if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"x","is_free":true,"blocks":[{"id":"bad1","type":"quiz","quiz":`+bad+`}]}`); code != 400 {
			t.Fatalf("буруу асуулт зөвшөөрөгдлөө: %s", bad)
		}
	}
}

func TestExamFlow(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Шалгалт","price":0,"published":true}`)
	cid := c["id"].(string)
	qs := `[{"id":"intro1","type":"text","text":"Амжилт"},{"id":"q1aa","type":"quiz","quiz":{"question":"1+1","options":["1","2"],"correct":[1],"points":2}},
		{"id":"q2aa","type":"quiz","quiz":{"kind":"text","question":"Нийслэл?","answers":["Улаанбаатар"]}}]`
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Эцсийн шалгалт","price":5000,"exam":{"time_min":30,"attempts":2,"pass_pct":60},"blocks":`+qs+`}`)
	lid := l["id"].(string)
	base := "/api/courses/" + cid + "/lessons/" + lid
	s1, uid := register(t, srv, "stud", "student")
	if code, _ := call(t, srv, "POST", base+"/exam/start", s1, ""); code != http.StatusPaymentRequired {
		t.Fatalf("төлбөргүй шалгалт эхлэх ёсгүй: %d", code)
	}
	_, o := call(t, srv, "POST", base+"/buy", s1, "")
	call(t, srv, "POST", "/api/orders/"+o["order"].(map[string]any)["id"].(string)+"/dev-pay", s1, "")
	// Хичээлийг нээхэд асуултууд харагдахгүй.
	_, got := call(t, srv, "GET", base, s1, "")
	if n := len(got["blocks"].([]any)); n != 1 {
		t.Fatalf("шалгалтын асуулт урьдчилан харагдаж байна: %d", n)
	}
	if code, _ := call(t, srv, "POST", base+"/quiz/q1aa", s1, `{"answer":[1]}`); code != http.StatusForbidden {
		t.Fatal("шалгалтын асуултыг нэг нэгээр шалгаж болохгүй")
	}
	_, st := call(t, srv, "POST", base+"/exam/start", s1, "")
	att := st["attempt"].(map[string]any)["id"].(string)
	if len(st["questions"].([]any)) != 2 {
		t.Fatalf("2 асуулт ирэх ёстой: %v", st)
	}
	// Дахин эхлүүлэхэд мөн оролдлого.
	if _, st2 := call(t, srv, "POST", base+"/exam/start", s1, ""); st2["attempt"].(map[string]any)["id"] != att {
		t.Fatal("идэвхтэй оролдлогыг үргэлжлүүлэх ёстой")
	}
	_, res := call(t, srv, "POST", base+"/exam/submit", s1, `{"attempt_id":"`+att+`","answers":{"q1aa":{"answer":[1]},"q2aa":{"text":"улаанбаатар"}}}`)
	a := res["attempt"].(map[string]any)
	if a["pct"].(float64) != 100 || a["passed"] != true || a["status"] != "submitted" {
		t.Fatalf("дүн буруу: %v", a)
	}
	if code, _ := call(t, srv, "POST", base+"/exam/submit", s1, `{"attempt_id":"`+att+`","answers":{}}`); code != http.StatusConflict {
		t.Fatal("давхар илгээх ёсгүй")
	}
	// Хоёр дахь оролдлого зөрчлөөр хаагдана, гурав дахь нь хориотой.
	_, st3 := call(t, srv, "POST", base+"/exam/start", s1, "")
	att3 := st3["attempt"].(map[string]any)["id"].(string)
	_, res3 := call(t, srv, "POST", base+"/exam/submit", s1, `{"attempt_id":"`+att3+`","answers":{"q1aa":{"answer":[1]}},"terminate":"copy"}`)
	if a3 := res3["attempt"].(map[string]any); a3["status"] != "terminated" || a3["passed"] != false || a3["pct"].(float64) != 67 {
		t.Fatalf("зөрчлөөр хаагдах ёстой: %v", a3)
	}
	if code, _ := call(t, srv, "POST", base+"/exam/start", s1, ""); code != http.StatusConflict {
		t.Fatal("оролдлогын тоо хэтэрсэн")
	}
	// Багшид мэдэгдэл ба самбарт тусна.
	_, notifs := callList(t, srv, "GET", "/api/me/notifications", tt)
	found := false
	for _, n := range notifs {
		if strings.Contains(n.(map[string]any)["title"].(string), "зөрчлөөр хаагдлаа") {
			found = true
		}
	}
	if !found {
		t.Fatalf("багшид шалгалт хаагдсан мэдэгдэл ирээгүй: %v", notifs)
	}
	_, an := call(t, srv, "GET", "/api/me/analytics/students/"+uid+"?course="+cid, tt, "")
	if st := an["student"].(map[string]any); st["exam_best"].(float64) != 100 || st["terminated"].(float64) != 1 {
		t.Fatalf("самбарын шалгалтын дүн буруу: %v", st)
	}
}

func TestActivityAndAnalytics(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Хяналт","price":0,"published":true,"camera":"required"}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"active_min":1}`)
	lid := l["id"].(string)
	s1, uid := register(t, srv, "stud", "student")
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")
	_, st := call(t, srv, "POST", "/api/activity/start", s1, `{"course_id":"`+cid+`","lesson_id":"`+lid+`"}`)
	sid := st["session_id"].(string)
	if st["policy"].(map[string]any)["camera"] != "required" || st["watermark"].(map[string]any)["ip"] == "" {
		t.Fatalf("бодлого/усан тэмдэг буруу: %v", st)
	}
	// Хуурамч их хугацаа хязгаарлагдана (сесс дөнгөж эхэлсэн).
	call(t, srv, "POST", "/api/activity/beat", s1, `{"session_id":"`+sid+`","active":9999,"focus_sum":3,"focus_n":4,"camera":true,
		"events":[{"type":"copy"},{"type":"tab_switch"},{"type":"phone"},{"type":"hack_me"}]}`)
	_, an := call(t, srv, "GET", "/api/me/analytics?course="+cid, tt, "")
	sts := an["data"].(map[string]any)["students"].([]any)
	if len(sts) != 1 {
		t.Fatalf("1 суралцагч байх ёстой: %v", an)
	}
	s := sts[0].(map[string]any)
	if s["active_sec"].(float64) > 10 || s["violations"].(float64) != 3 || s["camera"] != true {
		t.Fatalf("нэгтгэл буруу: %v", s)
	}
	// Идэвхтэй хугацаа хүрээгүй тул дуусгах боломжгүй.
	if code, r := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/complete", s1, ""); code != http.StatusConflict {
		t.Fatalf("идэвхтэй хугацаа шаардах ёстой: %d %v", code, r)
	}
	// Өөр хүн сессийг ашиглаж чадахгүй.
	s2, _ := register(t, srv, "stud2", "student")
	if code, _ := call(t, srv, "POST", "/api/activity/beat", s2, `{"session_id":"`+sid+`","active":5}`); code != 404 {
		t.Fatal("бусдын сесс")
	}
	// Багш сануулга илгээнэ → суралцагчид мэдэгдэл.
	if code, _ := call(t, srv, "POST", "/api/me/students/"+uid+"/remind", tt, `{"course_id":"`+cid+`","message":"Хичээлээ анхааралтай үзээрэй"}`); code != 200 {
		t.Fatalf("сануулга илгээгдсэнгүй: %d", code)
	}
	_, ns := callList(t, srv, "GET", "/api/me/notifications", s1)
	if len(ns) == 0 || !strings.Contains(ns[0].(map[string]any)["body"].(string), "анхааралтай") {
		t.Fatalf("суралцагчид сануулга ирээгүй: %v", ns)
	}
	_, tn := callList(t, srv, "GET", "/api/me/notifications", tt)
	warn := 0
	for _, n := range tn {
		if n.(map[string]any)["type"] == "violation" {
			warn++
		}
	}
	if warn != 2 { // copy, phone (tab_switch мэдэгдэхгүй)
		t.Fatalf("багшид 2 зөрчлийн мэдэгдэл ирэх ёстой: %d", warn)
	}
	// CSV тайлан.
	req, _ := http.NewRequest("GET", srv.URL+"/api/me/analytics/export?course="+cid, nil)
	req.Header.Set("Authorization", "Bearer "+tt)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(body), "Анхаарлын индекс") || !strings.Contains(string(body), "stud") {
		t.Fatalf("CSV буруу: %s", body)
	}
}

func TestQuizImport(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	res, err := http.Get(srv.URL + "/api/quiz-template.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	tpl, _ := io.ReadAll(res.Body)
	res.Body.Close()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "test.xlsx")
	fw.Write(tpl)
	mw.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/me/quiz-import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+tt)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Blocks   []store.Block `json:"blocks"`
		Problems []string      `json:"problems"`
	}
	json.NewDecoder(res.Body).Decode(&out)
	res.Body.Close()
	if len(out.Blocks) != 4 || len(out.Problems) != 0 {
		t.Fatalf("загварын 4 жишээ уншигдах ёстой: %d %v", len(out.Blocks), out.Problems)
	}
	if q := out.Blocks[1].Quiz; q.Kind != "multi" || fmt.Sprint(q.Correct) != "[0 2]" || q.Points != 2 {
		t.Fatalf("олон сонголт буруу: %+v", q)
	}
	if q := out.Blocks[3].Quiz; q.Kind != "match" || q.Right[1] != "Парис" {
		t.Fatalf("харгалзуулах буруу: %+v", q)
	}
}

func TestSearch(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "saraa", "teacher")
	call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Сарангэрэл Батболд"}`)
	_, c1 := call(t, srv, "POST", "/api/courses", tt, `{"title":"ЭЕШ Математик бэлтгэл","price":5000,"published":true,"certificate":true}`)
	call(t, srv, "POST", "/api/courses/"+c1["id"].(string)+"/lessons", tt, `{"title":"Логарифм тэгшитгэл","is_free":true}`)
	call(t, srv, "POST", "/api/courses/"+c1["id"].(string)+"/lessons", tt, `{"title":"Шалгалт","price":1000,"exam":{"pass_pct":50},"blocks":[{"id":"q1aa","type":"quiz","quiz":{"question":"?","options":["a","b"],"correct":[0]}}]}`)
	_, c2 := call(t, srv, "POST", "/api/courses", tt, `{"title":"Англи хэлний үндэс","price":0,"published":true}`)
	call(t, srv, "POST", "/api/courses/"+c2["id"].(string)+"/lessons", tt, `{"title":"Present simple","is_free":true}`)
	call(t, srv, "POST", "/api/courses", tt, `{"title":"Нууц ноорог математик","price":0,"published":false}`)
	time.Sleep(10 * time.Millisecond)

	ids := func(q string) []string {
		_, r := call(t, srv, "GET", "/api/search?"+q, "", "")
		var out []string
		for _, it := range r["items"].([]any) {
			out = append(out, it.(map[string]any)["title"].(string))
		}
		return out
	}
	for q, want := range map[string]string{
		"q=" + url.QueryEscape("математик"): "ЭЕШ Математик бэлтгэл", // кирилл
		"q=matematik":                        "ЭЕШ Математик бэлтгэл", // латин галиг
		"q=" + url.QueryEscape("матем"):      "ЭЕШ Математик бэлтгэл", // угтвар
		"q=" + url.QueryEscape("матиматик"):  "ЭЕШ Математик бэлтгэл", // үсгийн алдаа
		"q=" + url.QueryEscape("логарифм"):   "ЭЕШ Математик бэлтгэл", // хичээлийн гарчгаар
		"q=" + url.QueryEscape("сарангэрэл"): "ЭЕШ Математик бэлтгэл", // багшийн нэрээр
		"q=angli":   "Англи хэлний үндэс",
		"q=present": "Англи хэлний үндэс",
	} {
		got := ids(q)
		if len(got) == 0 || got[0] != want {
			t.Errorf("%s: эхнийх нь %q байх ёстой, гарсан: %v", q, want, got)
		}
	}
	if got := ids("q=" + url.QueryEscape("ноорог")); len(got) != 0 {
		t.Errorf("нийтлээгүй сургалт хайлтад гарах ёсгүй: %v", got)
	}
	if got := ids("tag=free"); len(got) != 1 || got[0] != "Англи хэлний үндэс" {
		t.Errorf("үнэгүй шүүлтүүр: %v", got)
	}
	if got := ids("tag=cert"); len(got) != 1 {
		t.Errorf("сертификат шүүлтүүр: %v", got)
	}
	_, r := call(t, srv, "GET", "/api/search?q="+url.QueryEscape("логарифм"), "", "")
	it := r["items"].([]any)[0].(map[string]any)
	tags := fmt.Sprint(it["tags"])
	if it["match"] != "Логарифм тэгшитгэл" || !strings.Contains(tags, "Сертификаттай") || !strings.Contains(tags, "Шалгалттай") || !strings.Contains(tags, "Төлбөртэй") {
		t.Fatalf("таарсан хичээл/шошго буруу: %v", it)
	}
	if it["teacher"].(map[string]any)["username"] != "saraa" {
		t.Fatal("багшийн холбоос алга")
	}
}

func TestSkeleton(t *testing.T) {
	for _, p := range [][2]string{{"Өнөөдөр", "unuudur"}, {"Хими", "khimi"}, {"Математик", "matematik"}, {"Үзэх", "uzeh"}} {
		if skeleton(p[0]) != skeleton(p[1]) {
			t.Errorf("%s (%s) ≠ %s (%s)", p[0], skeleton(p[0]), p[1], skeleton(p[1]))
		}
	}
}

func testJPEG(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 300, 420))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func rawCall(t *testing.T, srv *httptest.Server, method, path, token string, body []byte) (*http.Response, []byte) {
	req, _ := http.NewRequest(method, srv.URL+path, bytes.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	res.Body.Close()
	return res, b
}

func TestBooks(t *testing.T) {
	bookLim = &bookLimiter{win: map[string]*[2]int64{}, seen: map[string]time.Time{}}
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	code, b := call(t, srv, "POST", "/api/me/books", tt, `{"title":"Бодлогын ном","price":9000,"preview_pages":3}`)
	if code != 201 {
		t.Fatalf("ном үүссэнгүй: %d %v", code, b)
	}
	bid := b["id"].(string)
	if code, _ := call(t, srv, "PUT", "/api/me/books/"+bid, tt, `{"title":"Бодлогын ном","price":9000,"published":true}`); code != http.StatusConflict {
		t.Fatal("файлгүй ном нийтлэгдэх ёсгүй")
	}
	page := testJPEG(t)
	for p := 1; p <= 5; p++ {
		if res, body := rawCall(t, srv, "POST", fmt.Sprintf("/api/me/books/%s/pages/%d", bid, p), tt, page); res.StatusCode != 204 {
			t.Fatalf("хуудас %d: %d %s", p, res.StatusCode, body)
		}
	}
	if res, _ := rawCall(t, srv, "POST", "/api/me/books/"+bid+"/pages/6", tt, []byte("not an image")); res.StatusCode != 400 {
		t.Fatal("зураг биш файл татгалзагдах ёстой")
	}
	if code, _ := call(t, srv, "POST", "/api/me/books/"+bid+"/pages-done", tt, `{"total":6}`); code != 400 {
		t.Fatal("дутуу хуудастай бэлэн болох ёсгүй")
	}
	call(t, srv, "POST", "/api/me/books/"+bid+"/pages-done", tt, `{"total":5}`)
	call(t, srv, "PUT", "/api/me/books/"+bid, tt, `{"title":"Бодлогын ном","price":9000,"published":true}`)

	// Зочин: 1-3 хуудас, 4-р хуудаснаас төлбөр.
	if res, _ := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/1", "", nil); res.StatusCode != 200 || res.Header.Get("Content-Type") != "image/jpeg" || !strings.Contains(res.Header.Get("Cache-Control"), "no-store") {
		t.Fatalf("үнэгүй хуудас: %d %v", res.StatusCode, res.Header)
	}
	if res, body := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/4", "", nil); res.StatusCode != http.StatusPaymentRequired || !strings.Contains(string(body), "need_login") {
		t.Fatalf("4-р хуудас төлбөр нэхэх ёстой: %d %s", res.StatusCode, body)
	}
	s1, _ := register(t, srv, "reader", "student")
	if res, _ := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/4", s1, nil); res.StatusCode != http.StatusPaymentRequired {
		t.Fatal("худалдаж аваагүй уншигч 4-р хуудсыг үзэх ёсгүй")
	}
	_, info := call(t, srv, "GET", "/api/books/"+bid, s1, "")
	if info["access"].(map[string]any)["full"] != false {
		t.Fatal("эрхгүй байх ёстой")
	}
	_, buy := call(t, srv, "POST", "/api/books/"+bid+"/buy", s1, "")
	call(t, srv, "POST", "/api/orders/"+buy["order"].(map[string]any)["id"].(string)+"/dev-pay", s1, "")
	res, full := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/4", s1, nil)
	if res.StatusCode != 200 {
		t.Fatalf("худалдаж авсны дараа нээгдэх ёстой: %d", res.StatusCode)
	}
	_, prev := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/1", "", nil)
	_, mine := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/1", s1, nil)
	if bytes.Equal(prev, mine) || len(full) == 0 {
		t.Fatal("уншигчийн тэмдэг зочныхоос ялгаатай байх ёстой")
	}
	call(t, srv, "POST", "/api/books/"+bid+"/interest", s1, "")
	// Багшийн лог ба тоолуур.
	_, lst := call(t, srv, "GET", "/api/me/books", tt, "")
	_ = lst
	res2, body := rawCall(t, srv, "GET", "/api/me/books", tt, nil)
	var bs []store.Book
	json.Unmarshal(body, &bs)
	if res2.StatusCode != 200 || len(bs) != 1 || bs[0].Sales != 1 || bs[0].Revenue != 9000 || bs[0].Reads != 1 || bs[0].Interest != 1 || bs[0].Previews < 1 {
		t.Fatalf("тоолуур буруу: %+v", bs)
	}
	_, evb := rawCall(t, srv, "GET", "/api/me/book-events", tt, nil)
	for _, want := range []string{`"purchase"`, `"paywall"`, `"read"`, `"preview"`, `"interest"`} {
		if !strings.Contains(string(evb), want) {
			t.Errorf("логт %s алга", want)
		}
	}
	if code, _ := call(t, srv, "DELETE", "/api/me/books/"+bid, tt, ""); code != http.StatusConflict {
		t.Fatal("зарагдсан номыг устгах ёсгүй")
	}
	// Хурдны хязгаар: минутад 40 хуудаснаас их бол 429, багшид мэдэгдэл.
	limited := false
	for i := 0; i < 45; i++ {
		if r, _ := rawCall(t, srv, "GET", "/api/books/"+bid+"/pages/2", s1, nil); r.StatusCode == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("хурдны хязгаар ажиллах ёстой")
	}
	_, ns := callList(t, srv, "GET", "/api/me/notifications", tt)
	got := ""
	for _, n := range ns {
		got += n.(map[string]any)["title"].(string) + "\n"
	}
	if !strings.Contains(got, "Ном зарагдлаа") || !strings.Contains(got, "хэт хурдан") {
		t.Fatalf("багшийн мэдэгдэл дутуу:\n%s", got)
	}
	// Нийтлэгдээгүй ном харагдахгүй, бусад багш засаж чадахгүй.
	t2, _ := register(t, srv, "other", "teacher")
	if code, _ := call(t, srv, "PUT", "/api/me/books/"+bid, t2, `{"title":"x"}`); code != 404 {
		t.Fatal("бусдын номыг засах ёсгүй")
	}
	if res, body := rawCall(t, srv, "GET", "/b/"+bid, "", nil); res.StatusCode != 200 || !strings.Contains(string(body), "Бодлогын ном") {
		t.Fatalf("номын хуудас: %d", res.StatusCode)
	}
}

func TestVideoWatchedAndFileUsage(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, tid := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Видео","price":0,"published":true}`)
	cid := c["id"].(string)
	path := "/files/" + tid + "/private/abcdef0123456789__v.webm"
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[{"id":"vid1","type":"video","url":"`+path+`?exp=1&sig=x"},{"id":"fil1","type":"file","url":"`+path+`","download":true}]}`)
	lid := l["id"].(string)
	bs := l["blocks"].([]any)
	if u := bs[0].(map[string]any)["url"]; u != path {
		t.Fatalf("гарын үсэг хадгалагдах ёсгүй: %v", u)
	}
	if bs[1].(map[string]any)["download"] != true {
		t.Fatal("татах зөвшөөрөл хадгалагдаагүй")
	}
	s1, _ := register(t, srv, "stud", "student")
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/watched/nope", s1, ""); code != 404 {
		t.Fatal("байхгүй видео 404")
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/watched/vid1", s1, ""); code != 200 {
		t.Fatal("видео үзсэн тэмдэглэгдэх ёстой")
	}
	_, acc := call(t, srv, "GET", "/api/courses/"+cid+"/access", s1, "")
	if acc["progress"].(map[string]any)[lid].(map[string]any)["quiz"].(map[string]any)["watch_vid1"] != true {
		t.Fatal("үзсэн төлөв явцад алга")
	}
	_, use := call(t, srv, "GET", "/api/me/files/usage", tt, "")
	if us, ok := use[path].([]any); !ok || len(us) != 2 || us[0].(map[string]any)["lesson"] != "Нэг" {
		t.Fatalf("файлын ашиглалт буруу: %v", use)
	}
}

func TestEngagementEvidence(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "teach", "teacher")
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Идэвх","price":0,"published":true}`)
	cid := c["id"].(string)
	_, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Нэг","is_free":true,"blocks":[
		{"id":"qfast","type":"quiz","quiz":{"question":"Хурдан?","options":["a","b"],"correct":[0]}},
		{"id":"qslow","type":"quiz","quiz":{"question":"Удаан?","options":["a","b"],"correct":[1]}}]}`)
	lid := l["id"].(string)
	s1, uid := register(t, srv, "stud", "student")
	call(t, srv, "POST", "/api/courses/"+cid+"/enroll", s1, "")

	// Хэт хурдан (таамаг) ба хэвийн хурдтай хариулт.
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/quiz/qfast", s1, `{"answer":[0],"ms":300}`)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/quiz/qslow", s1, `{"answer":[1],"ms":8000}`)

	// Дүгнэлт: хэт богино бол татгалзана, хангалттай урт бол хадгална.
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/reflect", s1, `{"text":"богино"}`); code != 400 {
		t.Fatal("3-аас цөөн үгтэй дүгнэлт зөвшөөрөгдлөө")
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/reflect", tt, `{"text":"энэ бол миний л хичээл дээ"}`); code != 400 {
		t.Fatal("багш өөрийн хичээлд дүгнэлт бичих ёсгүй")
	}
	code, ref := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/reflect", s1, `{"text":"Энэ хичээлээр логарифмын дүрмийг сурлаа, маш ойлгомжтой байлаа."}`)
	if code != 200 || ref["words"].(float64) < 5 {
		t.Fatalf("дүгнэлт хадгалагдсангүй: %d %v", code, ref)
	}

	// Видео үзэлтийн зураглал: 0,1,2-р хэсгийг үзсэн, 3, 4-ийг алгассан.
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/progress/qfast", s1, `{"duration":55,"buckets":{"0":1,"1":2,"2":1}}`)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/progress/qfast", s1, `{"duration":55,"buckets":{"0":1}}`)

	_, an := call(t, srv, "GET", "/api/me/analytics?course="+cid, tt, "")
	sts := an["data"].(map[string]any)["students"].([]any)
	if len(sts) != 1 {
		t.Fatalf("1 суралцагч байх ёстой: %v", an)
	}
	st := sts[0].(map[string]any)
	if st["quiz_total"].(float64) != 2 || st["quiz_accuracy"].(float64) != 100 {
		t.Fatalf("асуултын тоолуур буруу: %v", st)
	}
	if st["quiz_guesses"].(float64) != 1 {
		t.Fatalf("таамагласан хариулт тоологдоогүй: %v", st)
	}
	if st["reflections"].(float64) != 1 {
		t.Fatalf("дүгнэлт тоологдоогүй: %v", st)
	}
	if st["video_clips"].(float64) != 1 || st["video_coverage"].(float64) <= 0 {
		t.Fatalf("видео үзэлт тоологдоогүй: %v", st)
	}
	if st["learn_score"].(float64) <= 0 {
		t.Fatalf("суралцсан оноо тооцогдоогүй: %v", st)
	}
	tot := an["data"].(map[string]any)["totals"].(map[string]any)
	if tot["reflections"].(float64) != 1 {
		t.Fatalf("нийт дүгнэлт буруу: %v", tot)
	}

	_, det := call(t, srv, "GET", "/api/me/analytics/students/"+uid+"?course="+cid, tt, "")
	refs := det["reflections"].([]any)
	if len(refs) != 1 || !strings.Contains(refs[0].(map[string]any)["text"].(string), "логарифм") {
		t.Fatalf("дэлгэрэнгүйд дүгнэлт алга: %v", det)
	}
	logs := det["quiz_logs"].([]any)
	if len(logs) != 2 {
		t.Fatalf("асуултын лог 2 байх ёстой: %v", logs)
	}
	watches := det["watches"].([]any)
	if len(watches) != 1 {
		t.Fatalf("видео зураглал 1 байх ёстой: %v", watches)
	}
	buckets := watches[0].(map[string]any)["buckets"].([]any)
	if len(buckets) < 6 || buckets[0].(float64) != 2 || buckets[3].(float64) != 0 {
		t.Fatalf("хэсэг бүрийн тоо буруу: %v", buckets)
	}

	// CSV тайланд шинэ баганууд орсон эсэх.
	req, _ := http.NewRequest("GET", srv.URL+"/api/me/analytics/export?course="+cid, nil)
	req.Header.Set("Authorization", "Bearer "+tt)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	for _, want := range []string{"Асуултад зөв хариулсан", "Дүгнэлт бичсэн тоо", "Суралцсан оноо", "stud"} {
		if !strings.Contains(string(body), want) {
			t.Errorf("CSV-д %q алга", want)
		}
	}

	// Эрхгүй/буруу тохиолдлууд.
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+lid+"/progress/qfast", "", `{"duration":10,"buckets":{"0":1}}`); code != http.StatusUnauthorized {
		t.Fatal("нэвтрээгүй хэрэглэгч видео явц илгээх ёсгүй")
	}
}
