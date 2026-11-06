package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// uploadTo — ?course=-тэй хуулалт (байгууллагын сургалтын хичээлд багш байгууллагын санд хуулна).
func uploadTo(t *testing.T, srv *httptest.Server, token, query string) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "lecture.pdf")
	fw.Write([]byte("%PDF-1.4 test"))
	mw.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/api/me/files?visibility=private"+query, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func findCourse(list []map[string]any, id string) map[string]any {
	for _, c := range list {
		if c["id"] == id {
			return c
		}
	}
	return nil
}

// Байгууллага: хянагч үүсгэж (шимтгэл, файлын сан), байгууллага хөтөлбөр, сургалтаа үүсгээд багш холбоно; холбосон
// багш хичээл оруулж (файл нь байгууллагын санд), зохион байгуулна; сургалтын тохиргоо, устгах, багш холбох нь зөвхөн
// байгууллагад. Сургалт багшийн профайлд гарахгүй, байгууллагын профайлд хөтөлбөрөөр, заадаг багштайгаа харагдана.
func TestOrganization(t *testing.T) {
	srv, _ := newTestServerWith(t, func(c *Config) { c.AdminEmails = []string{"boss@x.mn"} })
	defer srv.Close()
	boss, _ := register(t, srv, "boss", "student")
	student, _ := register(t, srv, "orgstud", "student")
	teacher, teacherID := register(t, srv, "orgteach", "teacher")
	other, _ := register(t, srv, "otherteach", "teacher")

	// Хянагч биш хүн байгууллага үүсгэхгүй.
	if code, _ := call(t, srv, "POST", "/api/admin/orgs", teacher, `{"name":"X","username":"xorg","email":"x@org.mn","password":"secret12"}`); code != http.StatusForbidden {
		t.Fatalf("багш байгууллага үүсгэлээ: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/admin/orgs", boss, `{"name":"МУИС","username":"muis","email":"info@muis.mn","password":"secret12","commission_pct":150}`); code != http.StatusBadRequest {
		t.Fatalf("150%% шимтгэл зөвшөөрөгдлөө: %d", code)
	}
	code, row := call(t, srv, "POST", "/api/admin/orgs", boss, `{"name":"МУИС","username":"muis","email":"info@muis.mn","password":"secret12","commission_pct":10,"storage_mb":500}`)
	if code != http.StatusCreated || row["commission_pct"].(float64) != 10 || row["storage_mb"].(float64) != 500 {
		t.Fatalf("байгууллага үүсгэх: %d %v", code, row)
	}
	orgID := row["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/admin/orgs", boss, `{"name":"Давхар","username":"muis","email":"x2@muis.mn","password":"secret12"}`); code != http.StatusConflict {
		t.Fatalf("давхар нэр: %d", code)
	}
	_, list := call(t, srv, "GET", "/api/admin/orgs", boss, "")
	if orgs := list["orgs"].([]any); len(orgs) != 1 {
		t.Fatalf("байгууллагын жагсаалт: %v", list)
	}

	// Байгууллага өөрөө нэвтэрнэ: /api/me-д гэрээний нөхцөл; шимтгэл анхдагчаас (1%) өөр — гэрээт: хугацаагүй, багц авахгүй.
	code, login := call(t, srv, "POST", "/api/auth/login", "", `{"email":"info@muis.mn","password":"secret12"}`)
	if code != 200 {
		t.Fatalf("байгууллага нэвтрэх: %d %v", code, login)
	}
	org := login["token"].(string)
	_, me := call(t, srv, "GET", "/api/me", org, "")
	if o, _ := me["org"].(map[string]any); o == nil || o["commission_pct"].(float64) != 10 || me["role"] != "teacher" {
		t.Fatalf("/api/me байгууллага: %v", me)
	}
	_, plan := call(t, srv, "GET", "/api/me/plan", org, "")
	if plan["org"] != true || plan["contract"] != true || plan["plan"] != "contract" || plan["active"] != true || plan["commission_pct"].(float64) != 10 || len(plan["plans"].([]any)) != 0 {
		t.Fatalf("гэрээт байгууллагын эрх: %v", plan)
	}
	if code, _ := call(t, srv, "POST", "/api/me/plan/purchase", org, `{"plan":"org_year"}`); code != http.StatusBadRequest {
		t.Fatalf("гэрээт байгууллага багц авлаа: %d", code)
	}
	// Шимтгэлийг анхдагч (1%) болговол багцын журам: үнэгүй хугацаа, байгууллагын багц.
	if code, r := call(t, srv, "PUT", "/api/admin/orgs/"+orgID, boss, `{"commission_pct":1,"storage_mb":500}`); code != 200 || r["plan"] != "trial" {
		t.Fatalf("анхдагч шимтгэл: %d %v", code, r)
	}
	if code, _ := call(t, srv, "POST", "/api/me/plan/purchase", org, `{"plan":"month"}`); code != http.StatusBadRequest {
		t.Fatalf("байгууллагад багшийн багц: %d", code)
	}
	// Байгууллагын багц (жил ₮1,000,000): үнэгүй хугацааны араас үргэлжилнэ.
	code, po := call(t, srv, "POST", "/api/me/plan/purchase", org, `{"plan":"org_year"}`)
	if code != http.StatusCreated || po["order"].(map[string]any)["amount"] != 1000000.0 {
		t.Fatalf("байгууллагын багц: %d %v", code, po)
	}
	if code, _ := call(t, srv, "POST", "/api/orders/"+po["order"].(map[string]any)["id"].(string)+"/dev-pay", org, ""); code != 200 {
		t.Fatalf("байгууллагын багц төлөх: %d", code)
	}
	if _, plan := call(t, srv, "GET", "/api/me/plan", org, ""); plan["plan"] != "trial" || len(plan["upcoming"].([]any)) != 1 || plan["fee_pct"] != 1.0 || plan["contract"] != false ||
		plan["plans"].([]any)[0].(map[string]any)["price"] != 129000.0 {
		t.Fatalf("байгууллагын багцын төлөв: %v", plan)
	}
	// Хянагч нөхцөлийг өөрчилнө.
	if code, r := call(t, srv, "PUT", "/api/admin/orgs/"+orgID, boss, `{"commission_pct":12.5,"storage_mb":1024}`); code != 200 || r["commission_pct"].(float64) != 12.5 {
		t.Fatalf("нөхцөл өөрчлөх: %d %v", code, r)
	}
	if _, st := call(t, srv, "GET", "/api/me/storage", org, ""); st["quota"].(float64) != 1024<<20 {
		t.Fatalf("байгууллагын файлын сан: %v", st)
	}

	// Мэргэжлийн хөтөлбөр.
	if code, _ := call(t, srv, "PUT", "/api/me/org/programs", teacher, `{"programs":[{"name":"Х"}]}`); code != http.StatusNotFound {
		t.Fatalf("багш хөтөлбөр үүсгэлээ: %d", code)
	}
	code, pr := call(t, srv, "PUT", "/api/me/org/programs", org, `{"programs":[{"name":"Програм хангамж","description":"4 жил"},{"name":"Нягтлан бодох бүртгэл"}]}`)
	progs, _ := pr["programs"].([]any)
	if code != 200 || len(progs) != 2 {
		t.Fatalf("хөтөлбөр: %d %v", code, pr)
	}
	progID := progs[0].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/courses", org, `{"title":"Буруу","price":0,"program_id":"nope"}`); code != http.StatusBadRequest {
		t.Fatalf("байхгүй хөтөлбөр: %d", code)
	}
	code, c := call(t, srv, "POST", "/api/courses", org, `{"title":"Алгоритм","price":30000,"published":true,"program_id":"`+progID+`"}`)
	if code != http.StatusCreated || c["program_id"] != progID {
		t.Fatalf("сургалт: %d %v", code, c)
	}
	cid := c["id"].(string)

	// Багш холбох: зөвхөн байгууллага, зөвхөн багшийн бүртгэл.
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/teachers", teacher, `{"who":"@orgteach"}`); code != http.StatusForbidden {
		t.Fatalf("багш өөр багш холболоо: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/teachers", org, `{"who":"orgstud"}`); code != http.StatusBadRequest {
		t.Fatalf("суралцагчийг багшаар холболоо: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/teachers", org, `{"who":"nobody_here"}`); code != http.StatusNotFound {
		t.Fatalf("байхгүй хэрэглэгч: %d", code)
	}
	code, tr := call(t, srv, "POST", "/api/courses/"+cid+"/teachers", org, `{"who":"https://surgalt.mn/t/orgteach"}`)
	if tl, _ := tr["teacher_list"].([]any); code != 200 || len(tl) != 1 || tl[0].(map[string]any)["username"] != "orgteach" {
		t.Fatalf("багш холбох: %d %v", code, tr)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/teachers", org, `{"who":"orgteach@x.mn"}`); code != http.StatusConflict {
		t.Fatalf("давхар холбох: %d", code)
	}

	// Холбосон багшийн студи: ?shared=1-д байгууллагын сургалт, бусад жагсаалтад үгүй.
	_, mine := callArrMap(t, srv, "GET", "/api/me/courses?shared=1", teacher)
	if sc := findCourse(mine, cid); sc == nil || sc["shared"] != true || sc["org"].(map[string]any)["username"] != "muis" {
		t.Fatalf("хуваалцсан сургалт: %v", mine)
	}
	if _, plain := callArrMap(t, srv, "GET", "/api/me/courses", teacher); findCourse(plain, cid) != nil {
		t.Fatal("shared-гүй жагсаалтад байгууллагын сургалт орлоо")
	}
	if code, mc := call(t, srv, "GET", "/api/me/courses/"+cid, teacher, ""); code != 200 || mc["shared"] != true {
		t.Fatalf("хуваалцсан сургалтыг нээх: %d %v", code, mc)
	}
	if code, _ := call(t, srv, "GET", "/api/me/courses/"+cid, other, ""); code != http.StatusNotFound {
		t.Fatalf("холбогдоогүй багш сургалтыг нээлээ: %d", code)
	}

	// Хичээл оруулах: файл байгууллагын санд (?course=), холбогдоогүй багш хуулж чадахгүй.
	if code, _ := uploadTo(t, srv, other, "&course="+cid); code != http.StatusNotFound {
		t.Fatalf("холбогдоогүй багш байгууллагын санд хуулав: %d", code)
	}
	code, up := uploadTo(t, srv, teacher, "&course="+cid)
	path, _ := up["path"].(string)
	if code != http.StatusCreated || !strings.HasPrefix(path, "/files/"+orgID+"/") {
		t.Fatalf("байгууллагын санд хуулах: %d %v", code, up)
	}
	lesson := `{"title":"Эрэмбэлэлт","is_free":false,"price":0,"blocks":[{"id":"b1","type":"file","url":"` + path + `","name":"lecture.pdf"}]}`
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", other, lesson); code != http.StatusNotFound {
		t.Fatalf("холбогдоогүй багш хичээл нэмэв: %d", code)
	}
	code, l := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", teacher, lesson)
	if code != http.StatusCreated {
		t.Fatalf("холбосон багш хичээл нэмэх: %d %v", code, l)
	}
	lid := l["id"].(string)
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid+"/lessons/"+lid, teacher, strings.Replace(lesson, "Эрэмбэлэлт", "Эрэмбэлэлт 2", 1)); code != 200 {
		t.Fatalf("хичээл засах: %d", code)
	}
	// Багш өөрийн хувийн сангийн файлыг байгууллагын хичээлд оруулахгүй (сургалтын эзэмшигчийн сан байх ёстой).
	own := uploadFile(t, srv, teacher, "mine.pdf", []byte("%PDF-1.4 mine"))
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", teacher, strings.Replace(lesson, path, own, 1)); code != http.StatusBadRequest {
		t.Fatalf("багшийн хувийн файл байгууллагын хичээлд орлоо: %d", code)
	}
	// Багш төлбөртэй хичээлийг элсэлтгүй үзэж, суралцагчдыг харна.
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, teacher, ""); code != 200 {
		t.Fatalf("багш хичээлээ үзэх: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/courses/"+cid+"/lessons/"+lid, student, ""); code != http.StatusPaymentRequired {
		t.Fatalf("суралцагч төлбөргүй үзлээ: %d", code)
	}
	if code, _ := callArr(t, srv, "GET", "/api/me/courses/"+cid+"/students", teacher, ""); code != 200 {
		t.Fatalf("суралцагчид: %d", code)
	}

	// Хэлэлцүүлэгт суралцагч бичвэл холбосон багшид ч мэдэгдэл очно.
	_, fl := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", teacher, `{"title":"Танилцуулга","is_free":true,"discussion":true,"blocks":[{"id":"t1","type":"text","text":"Сайн байна уу"}]}`)
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons/"+fl["id"].(string)+"/discussion", student, `{"body":"Асуулт байна"}`); code != http.StatusCreated && code != 200 {
		t.Fatalf("сэтгэгдэл: %d", code)
	}
	_, tn := call(t, srv, "GET", "/api/me/notifications", teacher, "")
	if len(notifTitles(tn, "comment")) != 1 {
		t.Fatalf("холбосон багшид хэлэлцүүлгийн мэдэгдэл очсонгүй: %v", tn)
	}
	_, on := call(t, srv, "GET", "/api/me/notifications", org, "")
	if len(notifTitles(on, "comment")) != 1 {
		t.Fatalf("байгууллагад хэлэлцүүлгийн мэдэгдэл очсонгүй: %v", on)
	}

	// Аналитик (байгууллагын сургалтаар), хориг нээх, сануулга.
	if code, an := call(t, srv, "GET", "/api/me/analytics?course="+cid, teacher, ""); code != 200 || an["data"] == nil {
		t.Fatalf("байгууллагын сургалтын аналитик: %d %v", code, an)
	}
	if code, _ := call(t, srv, "GET", "/api/me/analytics/events?course="+cid, teacher, ""); code != 200 {
		t.Fatalf("аналитикийн үйл явдал: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/me/students/"+student+"/unblock", teacher, `{}`); code != 200 {
		t.Fatalf("хориг нээх: %d", code)
	}
	// Бүлэг чат: холбосон багш багшаар орж, чатын жагсаалтад нь харагдана.
	code, gc := call(t, srv, "POST", "/api/courses/"+cid+"/chat", teacher, "")
	if code != 200 || gc["me"] != "teacher" {
		t.Fatalf("бүлэг чат: %d %v", code, gc)
	}
	gid := gc["conversation"].(map[string]any)["id"].(string)
	if code, _ := call(t, srv, "POST", "/api/chat/"+gid+"/messages", teacher, `{"body":"Сайн байцгаана уу"}`); code != 200 && code != 201 {
		t.Fatalf("бүлэгт бичих: %d", code)
	}
	_, convs := callArrMap(t, srv, "GET", "/api/me/conversations", teacher)
	if findCourse(convs, gid) == nil {
		t.Fatalf("холбосон багшийн чатын жагсаалтад бүлэг алга: %v", convs)
	}
	// Шууд хичээл: үнэгүйг товлоно, төлбөртэйг байгууллага.
	meet := `{"title":"Давтлага","starts_at":"` + time.Now().Add(48*time.Hour).UTC().Format(time.RFC3339) + `","duration_min":60,"course_id":"` + cid + `","meet_url":"https://meet.google.com/abc-defg-hij","price":%d}`
	if code, _ := call(t, srv, "POST", "/api/me/meetings", teacher, fmt.Sprintf(meet, 5000)); code != http.StatusBadRequest {
		t.Fatalf("холбосон багш төлбөртэй шууд хичээл товлолоо: %d", code)
	}
	if code, m := call(t, srv, "POST", "/api/me/meetings", teacher, fmt.Sprintf(meet, 0)); code != http.StatusCreated {
		t.Fatalf("үнэгүй шууд хичээл: %d %v", code, m)
	}
	if code, cm := call(t, srv, "GET", "/api/courses/"+cid+"/meetings", org, ""); code != 200 || len(cm["meetings"].([]any)) != 1 {
		t.Fatalf("сургалтын шууд хичээл: %d %v", code, cm)
	}

	// Сургалтын тохиргоо, устгах — зөвхөн байгууллага.
	if code, _ := call(t, srv, "PUT", "/api/courses/"+cid, teacher, `{"title":"Хакердав","price":0}`); code != http.StatusNotFound {
		t.Fatalf("багш сургалтын тохиргоо өөрчиллөө: %d", code)
	}
	if code, _ := call(t, srv, "DELETE", "/api/courses/"+cid, teacher, ""); code != http.StatusNotFound {
		t.Fatalf("багш байгууллагын сургалтыг устгалаа: %d", code)
	}

	// Нийтийн хуудас: байгууллагын профайлд хөтөлбөр, заадаг багш; багшийн профайлд энэ сургалт гарахгүй.
	_, op := call(t, srv, "GET", "/api/teachers/muis", "", "")
	ps, _ := op["programs"].([]any)
	if op["org"] != true || len(ps) != 1 || ps[0].(map[string]any)["name"] != "Програм хангамж" {
		t.Fatalf("байгууллагын профайл: %v", op)
	}
	if ct := op["course_teachers"].(map[string]any)[cid].([]any); len(ct) != 1 || ct[0].(map[string]any)["username"] != "orgteach" {
		t.Fatalf("сургалтын багш: %v", op["course_teachers"])
	}
	_, tp := call(t, srv, "GET", "/api/teachers/orgteach", "", "")
	if cs, _ := tp["courses"].([]any); len(cs) != 0 {
		t.Fatalf("байгууллагын сургалт багшийн профайлд гарлаа: %v", cs)
	}
	_, pc := call(t, srv, "GET", "/api/courses/"+cid, "", "")
	if pc["org"] != true || pc["program"] != "Програм хангамж" || len(pc["teachers"].([]any)) != 1 {
		t.Fatalf("сургалтын хуудас: %v", pc)
	}
	res, err := http.Get(srv.URL + "/t/muis")
	if err != nil {
		t.Fatal(err)
	}
	html, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(html), `href="/t/orgteach"`) || !strings.Contains(string(html), "Програм хангамж") {
		t.Fatalf("байгууллагын профайл HTML: %d", res.StatusCode)
	}
	res, _ = http.Get(srv.URL + "/c/" + cid)
	html, _ = io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(html), `href="/t/orgteach"`) {
		t.Fatalf("сургалтын HTML-д багшийн холбоос алга: %d", res.StatusCode)
	}
	if !strings.Contains(string(html), `data-teacher="orgteach"`) {
		t.Fatal("«Багшаас асуух» чат заадаг багш руу очих ёстой")
	}

	// Багш сургалтаас гарна → эрх хаагдана.
	if code, _ := call(t, srv, "DELETE", "/api/courses/"+cid+"/teachers/"+teacherID, teacher, ""); code != 200 {
		t.Fatalf("багш гарах: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/courses/"+cid+"/lessons", teacher, lesson); code != http.StatusNotFound {
		t.Fatalf("гарсан багш хичээл нэмэв: %d", code)
	}

}

// Удирдагч: хянагч байгаа хэрэглэгчийг хайж байгууллагын удирдагчаар холбоно; тэр хүн өөрийн бүртгэлээр нэвтэрч
// /api/me-ийн orgs-оос байгууллага руу шилжиж (байгууллагын токен) удирдана. Хянагч нэр, холбоос, нөхцөлийг засаж,
// удирдагч хасахад шилжсэн эрх хүчингүй болно; сургалтгүй байгууллагыг устгана, сургалттайг устгахгүй.
func TestOrganizationManagers(t *testing.T) {
	srv, _ := newTestServerWith(t, func(c *Config) { c.AdminEmails = []string{"boss@x.mn"} })
	defer srv.Close()
	boss, _ := register(t, srv, "boss", "student")
	mgr, mgrID := register(t, srv, "dorj", "student")
	other, _ := register(t, srv, "bold", "student")

	if code, _ := callArr(t, srv, "GET", "/api/admin/users?q=dorj", mgr, ""); code != http.StatusForbidden {
		t.Fatalf("хянагч биш хэрэглэгч хайлаа: %d", code)
	}
	_, found := callArrMap(t, srv, "GET", "/api/admin/users?q=dor", boss)
	if len(found) != 1 || found[0]["id"] != mgrID {
		t.Fatalf("хэрэглэгч хайх: %v", found)
	}
	if code, _ := call(t, srv, "POST", "/api/admin/orgs", boss, `{"name":"Сургууль","username":"school1","commission_pct":5,"storage_mb":100}`); code != http.StatusBadRequest {
		t.Fatalf("удирдагч, нэвтрэлтгүй байгууллага: %d", code)
	}
	code, row := call(t, srv, "POST", "/api/admin/orgs", boss, `{"name":"Сургууль","username":"school1","commission_pct":5,"storage_mb":100,"manager":"`+mgrID+`"}`)
	if code != http.StatusCreated || row["login"] != false || len(row["managers"].([]any)) != 1 {
		t.Fatalf("удирдагчтай байгууллага: %d %v", code, row)
	}
	orgID := row["id"].(string)

	// Удирдагчийн цэсэнд байгууллага гарч, шилжинэ; бусад хүн шилжихгүй.
	_, me := call(t, srv, "GET", "/api/me", mgr, "")
	if os, _ := me["orgs"].([]any); len(os) != 1 || os[0].(map[string]any)["username"] != "school1" {
		t.Fatalf("удирдагчийн /api/me orgs: %v", me["orgs"])
	}
	if code, _ := call(t, srv, "POST", "/api/me/orgs/"+orgID+"/switch", other, ""); code != http.StatusNotFound {
		t.Fatalf("удирдагч биш хүн шилжлээ: %d", code)
	}
	code, sw := call(t, srv, "POST", "/api/me/orgs/"+orgID+"/switch", mgr, "")
	if code != 200 || sw["token"] == nil {
		t.Fatalf("шилжих: %d %v", code, sw)
	}
	orgTok := sw["token"].(string)
	_, om := call(t, srv, "GET", "/api/me", orgTok, "")
	if om["username"] != "school1" || om["org"] == nil || om["consent_needed"] != false || om["phone_required"] != false {
		t.Fatalf("байгууллагын нэрийн өмнөөс: %v", om)
	}
	if code, _ := call(t, srv, "POST", "/api/courses", orgTok, `{"title":"Анхны сургалт","price":0}`); code != http.StatusCreated {
		t.Fatalf("удирдагч байгууллагын сургалт үүсгэх: %d", code)
	}

	// Засах: нэр, профайлын холбоос, нөхцөл.
	code, ed := call(t, srv, "PUT", "/api/admin/orgs/"+orgID, boss, `{"name":"Шинэ сургууль","username":"school2","commission_pct":7,"storage_mb":300}`)
	if code != 200 || ed["name"] != "Шинэ сургууль" || ed["username"] != "school2" || ed["commission_pct"].(float64) != 7 {
		t.Fatalf("байгууллага засах: %d %v", code, ed)
	}
	if code, _ := call(t, srv, "GET", "/api/teachers/school2", "", ""); code != 200 {
		t.Fatalf("шинэ холбоос: %d", code)
	}

	// Удирдагч нэмэх, хасах: хассан хүний байгууллагын токен хүчингүй.
	_, otherMe := call(t, srv, "GET", "/api/me", other, "")
	if code, _ := call(t, srv, "POST", "/api/admin/orgs/"+orgID+"/managers", boss, `{"user_id":"`+otherMe["id"].(string)+`"}`); code != 200 {
		t.Fatalf("удирдагч нэмэх: %d", code)
	}
	if code, _ := call(t, srv, "POST", "/api/admin/orgs/"+orgID+"/managers", boss, `{"user_id":"`+orgID+`"}`); code != http.StatusBadRequest {
		t.Fatalf("байгууллагыг өөрийг нь удирдагч болголоо: %d", code)
	}
	time.Sleep(5 * time.Millisecond) // токены олгосон цагаас хойш хүчингүй болгоно
	if code, _ := call(t, srv, "DELETE", "/api/admin/orgs/"+orgID+"/managers/"+mgrID, boss, ""); code != 200 {
		t.Fatalf("удирдагч хасах: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/me", orgTok, ""); code != http.StatusUnauthorized {
		t.Fatalf("хассан удирдагчийн байгууллагын токен хүчинтэй хэвээр: %d", code)
	}
	if _, me := call(t, srv, "GET", "/api/me", mgr, ""); me["orgs"] != nil {
		t.Fatalf("хассан удирдагчид байгууллага харагдсаар: %v", me["orgs"])
	}

	// Устгах: сургалттай бол устгахгүй; сургалтгүйг устгана.
	if code, _ := call(t, srv, "DELETE", "/api/admin/orgs/"+orgID, boss, ""); code != http.StatusConflict {
		t.Fatalf("сургалттай байгууллагыг устгалаа: %d", code)
	}
	_, empty := call(t, srv, "POST", "/api/admin/orgs", boss, `{"name":"Түр","username":"tmporg","commission_pct":5,"storage_mb":100,"email":"tmp@org.mn","password":"secret12"}`)
	if empty["login"] != true {
		t.Fatalf("нэвтрэлттэй байгууллага: %v", empty)
	}
	if code, _ := call(t, srv, "DELETE", "/api/admin/orgs/"+empty["id"].(string), boss, ""); code != 200 {
		t.Fatalf("сургалтгүй байгууллага устгах: %d", code)
	}
	if code, _ := call(t, srv, "GET", "/api/teachers/tmporg", "", ""); code != http.StatusNotFound {
		t.Fatalf("устгасан байгууллагын профайл: %d", code)
	}
}
