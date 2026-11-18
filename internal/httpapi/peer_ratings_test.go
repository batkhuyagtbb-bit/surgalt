package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"surgalt/internal/store"
)

// ---- Бүлэг чатын харилцан үнэлгээ (peer_ratings.go) — HTTP түвшний тест (санах ойн store) ----

// prClass — багш, үнэгүй нийтлэгдсэн сургалт (nLessons хичээл), элссэн n суралцагч (нэр нь prefix+a, b, …).
type prClass struct {
	srv       *httptest.Server
	s         *Server
	st        store.Store
	mem       *store.Memory
	tt, tid   string
	cid       string
	lids      []string
	tok, uids []string
}

func newPRClass(t *testing.T, prefix string, n, nLessons int) *prClass {
	t.Helper()
	srv, s, st := newPlanServer(t)
	mem, ok := st.(*store.Memory)
	if !ok {
		t.Skip("цаг өөрчлөх нь санах ойн store-д л")
	}
	p := &prClass{srv: srv, s: s, st: st, mem: mem}
	p.tt, p.tid = register(t, srv, prefix+"t", "teacher")
	p.cid = jmCourse(t, srv, p.tt, `{"title":"Хамтын анги","price":0,"published":true}`)
	for i := range max(nLessons, 1) {
		p.lids = append(p.lids, jmLesson(t, srv, p.tt, p.cid, `{"title":"Хичээл `+itoa(i+1)+`","is_free":true}`))
	}
	for i := range n {
		tok, uid := register(t, srv, prefix+string(rune('a'+i)), "student")
		jmEnroll(t, srv, tok, p.cid)
		p.tok, p.uids = append(p.tok, tok), append(p.uids, uid)
	}
	return p
}

// season — суралцагчийг тоологдох үнэлэгч болгоно: 10 хоногийн өмнө элссэн, 5 хоногийн өмнө сесстэй.
func (p *prClass) season(t *testing.T, idx ...int) {
	t.Helper()
	now := time.Now()
	for _, i := range idx {
		if !p.mem.SetEnrolledAt(p.uids[i], p.cid, now.Add(-10*24*time.Hour)) {
			t.Fatalf("элсэлт %d", i)
		}
		if err := p.st.CreateSession(context.Background(), &store.StudySession{UserID: p.uids[i], CourseID: p.cid, LessonID: p.lids[0],
			TeacherID: p.tid, Kind: "lesson", StartedAt: now.Add(-5 * 24 * time.Hour), LastAt: now.Add(-5 * 24 * time.Hour)}); err != nil {
			t.Fatal(err)
		}
	}
	p.s.peers.Delete(p.cid)
}

// age — үнэлгээг өчигдрийнх болгоно (суралцагчийн snapshot-д орно).
func (p *prClass) age(t *testing.T, from, to int, lid string) {
	t.Helper()
	if !p.mem.SetPeerRatingAt(p.cid, p.uids[from], p.uids[to], lid, peerCutoff(time.Now()).Add(-time.Hour)) {
		t.Fatalf("үнэлгээ олдсонгүй %d→%d %s", from, to, lid)
	}
	p.s.peers.Delete(p.cid)
}

func (p *prClass) base() string { return "/api/courses/" + p.cid + "/peer-ratings" }

func (p *prClass) rate(t *testing.T, tok, uid, body string) (int, map[string]any) {
	return call(t, p.srv, "PUT", p.base()+"/"+uid, tok, body)
}

func (p *prClass) rateL(t *testing.T, from, to int, lid string, score int) map[string]any {
	t.Helper()
	code, r := p.rate(t, p.tok[from], p.uids[to], `{"score":`+itoa(score)+`,"lesson_id":"`+lid+`"}`)
	if code != http.StatusOK {
		t.Fatalf("үнэлгээ %d→%d (%s): %d %v", from, to, lid, code, r)
	}
	return r
}

func (p *prClass) rateOK(t *testing.T, from, to, score int) map[string]any {
	t.Helper()
	return p.rateL(t, from, to, "", score)
}

// rateAged — үнэлээд өчигдрийнх болгоно.
func (p *prClass) rateAged(t *testing.T, from, to int, lid string, score int) {
	t.Helper()
	p.rateL(t, from, to, lid, score)
	p.age(t, from, to, lid)
}

func (p *prClass) get(t *testing.T, tok, query string) map[string]any {
	t.Helper()
	path := p.base()
	if query != "" {
		path += "?" + query
	}
	code, r := call(t, p.srv, "GET", path, tok, "")
	if code != http.StatusOK {
		t.Fatalf("GET peer-ratings %s: %d %v", query, code, r)
	}
	return r
}

func (p *prClass) del(t *testing.T, tok, uid, query string) int {
	t.Helper()
	path := p.base() + "/" + uid
	if query != "" {
		path += "?" + query
	}
	code, _ := call(t, p.srv, "DELETE", path, tok, "")
	return code
}

func prMember(r map[string]any, uid string) map[string]any {
	for _, x := range r["members"].([]any) {
		if m := x.(map[string]any); m["user_id"] == uid {
			return m
		}
	}
	return nil
}

func prKeys(m map[string]any) string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return strings.Join(ks, ",")
}

// prNum — JSON тоо эсвэл null (-1).
func prNum(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return -1
}

// Эрх: гишүүн биш нь унших/үнэлэх 403; нэвтрээгүй 401; багш үнэлэхгүй (403); өөрийгөө 400; оноо бүхэл 0..100 биш бол 400;
// гишүүн биш, багш, архивласан, байхгүй, үнэлэгчийг хориглосон хүнийг үнэлэх 404; архивласан суралцагчид GET/PUT/DELETE
// 403; өөр сургалтын эсвэл нуусан хичээл 404; хэт олон үнэлгээ 429.
func TestPeerRatingsAuth(t *testing.T) {
	p := newPRClass(t, "pra_", 5, 2)
	ctx := context.Background()
	stranger, sid := register(t, p.srv, "pra_x", "student")
	otherCourse := jmCourse(t, p.srv, p.tt, `{"title":"Өөр","price":0,"published":true}`)
	foreign := jmLesson(t, p.srv, p.tt, otherCourse, `{"title":"Өөр хичээл","is_free":true}`)
	hidden := p.lids[1]
	if code, _ := call(t, p.srv, "PUT", "/api/courses/"+p.cid+"/lessons/"+hidden+"/visibility", p.tt, `{"hidden":true}`); code != http.StatusOK {
		t.Fatalf("хичээл нуух: %d", code)
	}

	code, r := call(t, p.srv, "GET", p.base(), "", "")
	jmFail(t, "нэвтрээгүй GET", code, r, http.StatusUnauthorized, "")
	code, r = call(t, p.srv, "GET", p.base(), stranger, "")
	jmFail(t, "гишүүн биш GET", code, r, http.StatusForbidden, msgPeerMembers)
	code, r = p.rate(t, stranger, p.uids[0], `{"score":50}`)
	jmFail(t, "гишүүн биш PUT", code, r, http.StatusForbidden, msgPeerMembers)
	code, r = p.rate(t, p.tt, p.uids[0], `{"score":50}`)
	jmFail(t, "багш PUT", code, r, http.StatusForbidden, msgPeerTeacher)
	code, r = p.rate(t, p.tok[0], p.uids[0], `{"score":50}`)
	jmFail(t, "өөрийгөө", code, r, http.StatusBadRequest, msgPeerSelf)
	for _, body := range []string{`{"score":101}`, `{"score":-1}`, `{"score":50.5}`, `{"score":"50"}`, `{"score":null}`, `{}`} {
		code, r = p.rate(t, p.tok[0], p.uids[1], body)
		jmFail(t, "оноо "+body, code, r, http.StatusBadRequest, msgPeerScore)
	}
	for name, lid := range map[string]string{"өөр сургалтын хичээл": foreign, "нуусан хичээл": hidden, "байхгүй хичээл": "nolesson"} {
		code, r = p.rate(t, p.tok[0], p.uids[1], `{"score":50,"lesson_id":"`+lid+`"}`)
		jmFail(t, name, code, r, http.StatusNotFound, msgPeerLesson)
		code, r = call(t, p.srv, "GET", p.base()+"?lesson_id="+lid, p.tok[0], "")
		jmFail(t, "GET "+name, code, r, http.StatusNotFound, msgPeerLesson)
	}
	if g := p.get(t, p.tt, "lesson_id="+hidden); g["lesson_id"] != hidden { // багшид нуусан хичээл ч
		t.Fatalf("багшийн нуусан хичээлийн GET: %v", g)
	}
	code, r = p.rate(t, p.tok[0], sid, `{"score":50}`)
	jmFail(t, "гишүүн бишийг үнэлэх", code, r, http.StatusNotFound, msgPeerRatee)
	code, r = p.rate(t, p.tok[0], p.tid, `{"score":50}`)
	jmFail(t, "багшийг үнэлэх", code, r, http.StatusNotFound, msgPeerRatee)
	code, r = p.rate(t, p.tok[0], "nobody", `{"score":50}`)
	jmFail(t, "байхгүй хүн", code, r, http.StatusNotFound, msgPeerRatee)

	// Хориг: d (3) нь a (0)-г хувийн ярианд хориглосон → a нь d-г үнэлэх 404; d нь a-г үнэлж чадна.
	dm, err := p.st.CreateDM(ctx, p.uids[3], p.uids[0], "pra_a")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.st.SetConversationBlock(ctx, dm.ID, store.SenderTeacher, true); err != nil { // d = TeacherID тал
		t.Fatal(err)
	}
	code, r = p.rate(t, p.tok[0], p.uids[3], `{"score":50}`)
	jmFail(t, "хориглосон хүнийг үнэлэх", code, r, http.StatusNotFound, msgPeerRatee)
	if code, r = p.rate(t, p.tok[3], p.uids[0], `{"score":50}`); code != http.StatusOK {
		t.Fatalf("хориглогч нь үнэлж чадна: %d %v", code, r)
	}

	// Архив: e (4) — үнэлэгдэхгүй (404), өөрөө GET/PUT/DELETE 403, жагсаалтад орохгүй.
	p.rateOK(t, 4, 1, 70)
	jmArchive(t, p.srv, p.tt, p.cid, p.uids[4])
	code, r = p.rate(t, p.tok[0], p.uids[4], `{"score":50}`)
	jmFail(t, "архивласныг үнэлэх", code, r, http.StatusNotFound, msgPeerRatee)
	code, r = call(t, p.srv, "GET", p.base(), p.tok[4], "")
	jmFail(t, "архивласан GET", code, r, http.StatusForbidden, msgPeerClosed)
	code, r = p.rate(t, p.tok[4], p.uids[1], `{"score":50}`)
	jmFail(t, "архивласан PUT", code, r, http.StatusForbidden, msgPeerClosed)
	if code := p.del(t, p.tok[4], p.uids[1], ""); code != http.StatusForbidden {
		t.Fatalf("архивласан DELETE: %d", code)
	}
	if m := prMember(p.get(t, p.tok[0], ""), p.uids[4]); m != nil {
		t.Fatalf("архивласан суралцагч жагсаалтад орлоо: %v", m)
	}
	tv := p.get(t, p.tt, "")
	if prMember(tv, p.uids[4]) != nil || prMember(tv, p.tid) != nil {
		t.Fatalf("архивласан суралцагч эсвэл багш багшийн жагсаалтад орлоо: %v", tv)
	}
	if code, r = call(t, p.srv, "GET", "/api/courses/nope/peer-ratings", p.tok[0], ""); code != http.StatusNotFound {
		t.Fatalf("байхгүй сургалт: %d %v", code, r)
	}
	// Rate limit: үнэлэгч бүрт burst 30.
	limited := false
	for range 40 {
		if code, _ = p.rate(t, p.tok[1], p.uids[0], `{"score":60}`); code == http.StatusTooManyRequests {
			limited = true
			break
		}
		if code != http.StatusOK {
			t.Fatalf("үнэлгээ: %d", code)
		}
	}
	if !limited {
		t.Fatal("хэт олон үнэлгээ 429 өгсөнгүй")
	}
	if code, _ = p.rate(t, p.tok[2], p.uids[1], `{"score":60}`); code != http.StatusOK { // өөр үнэлэгч хязгаарлагдахгүй
		t.Fatalf("өөр үнэлэгч: %d", code)
	}
}

// Нууцлал (S1): суралцагчийн хариунд тоо (count) огт байхгүй, me-д enough; харагдах оноо — өчигдрийн snapshot-ийн
// Bayes, 5-д бүхэлчилсэн; босго үзэгчийг тоолохгүй; бусдын оноо 50-аас бага бол null, өөрийн бага оноо харагдана;
// өнөөдрийн үнэлгээ snapshot-д орохгүй (PUT ч snapshot буцаана); хэн өгсөн нь зөвхөн багшид.
func TestPeerRatingsPrivacy(t *testing.T) {
	p := newPRClass(t, "prp_", 5, 1)
	A, B, C, D, E := 0, 1, 2, 3, 4
	p.season(t, A, B, C, D, E)
	p.rateAged(t, A, B, "", 83)
	p.rateAged(t, C, B, "", 60)
	p.rateAged(t, D, B, "", 100)
	p.rateAged(t, A, C, "", 10)
	p.rateAged(t, B, C, "", 20)
	p.rateAged(t, D, C, "", 30)
	// Нийт 303/6 = 50.5. B: (243 + 151.5)/6 = 65.75 → snapshot 65 (5-д), рейтинг 66; C: (60 + 151.5)/6 = 35.25 → 35.

	for _, who := range []int{A, B, C, D, E} {
		res, body := rawCall(t, p.srv, "GET", p.base(), p.tok[who], nil)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("GET: %d %s", res.StatusCode, body)
		}
		s := string(body)
		for _, bad := range []string{`"count"`, `"rater_id"`, `"rater_name"`, `"ratings"`, `"given`, `"collab"`, `"rank"`, `"counted"`} {
			if strings.Contains(s, bad) {
				t.Fatalf("суралцагчид %s очлоо: %s", bad, s)
			}
		}
		g := p.get(t, p.tok[who], "")
		if prKeys(g) != "lesson_id,lessons,me,members,min_raters,role" || g["role"] != "student" || prNum(g["min_raters"]) != 3 || g["lesson_id"] != "" {
			t.Fatalf("суралцагчийн хариу: %v", g)
		}
		if k := prKeys(g["me"].(map[string]any)); k != "avg,enough,user_id" {
			t.Fatalf("me талбарууд: %s", k)
		}
		ms := g["members"].([]any)
		if len(ms) != 4 {
			t.Fatalf("гишүүд (өөрөөс бусад 4): %v", ms)
		}
		for i, x := range ms {
			m := x.(map[string]any)
			if k := prKeys(m); k != "avatar_url,avg,mine,name,user_id,username" {
				t.Fatalf("гишүүний талбарууд: %s", k)
			}
			if m["user_id"] == p.uids[who] {
				t.Fatal("өөрөө гишүүдийн жагсаалтад орлоо")
			}
			if i > 0 && strings.ToLower(ms[i-1].(map[string]any)["name"].(string)) > strings.ToLower(m["name"].(string)) {
				t.Fatalf("нэрээр эрэмбэлээгүй: %v", ms)
			}
		}
	}
	// Босго үзэгчийг тоолохгүй: E (үнэлээгүй) B-г 3 хүнээр харна → 65; A, C, D-д бусад нь 2 → null.
	if m := prMember(p.get(t, p.tok[E], ""), p.uids[B]); prNum(m["avg"]) != 65 || m["mine"] != nil {
		t.Fatalf("E-ийн харах B: %v", m)
	}
	for _, who := range []int{A, C, D} {
		if m := prMember(p.get(t, p.tok[who], ""), p.uids[B]); m["avg"] != nil {
			t.Fatalf("%d: үзэгчийг тоолсон босго: %v", who, m)
		}
	}
	if m := prMember(p.get(t, p.tok[A], ""), p.uids[B]); prNum(m["mine"]) != 83 {
		t.Fatalf("A-ийн mine: %v", m)
	}
	// Бусдын 50-аас бага оноо null; өөрийн бага оноо харагдана.
	if m := prMember(p.get(t, p.tok[E], ""), p.uids[C]); m["avg"] != nil {
		t.Fatalf("бусдын бага оноо харагдлаа: %v", m)
	}
	if me := p.get(t, p.tok[C], "")["me"].(map[string]any); me["enough"] != true || prNum(me["avg"]) != 35 {
		t.Fatalf("өөрийн бага оноо: %v", me)
	}
	if me := p.get(t, p.tok[B], "")["me"].(map[string]any); me["enough"] != true || prNum(me["avg"]) != 65 {
		t.Fatalf("B-ийн өөрийн оноо: %v", me)
	}
	if me := p.get(t, p.tok[A], "")["me"].(map[string]any); me["enough"] != false || me["avg"] != nil {
		t.Fatalf("үнэлэгдээгүй A: %v", me)
	}

	// Өнөөдрийн үнэлгээ snapshot-д орохгүй: E→B 0; PUT ч, GET ч 65 хэвээр; багшид шууд (4 хүн, 61).
	r := p.rateOK(t, E, B, 0)
	if m := r["member"].(map[string]any); prNum(m["avg"]) != 65 || prNum(m["mine"]) != 0 || m["count"] != nil {
		t.Fatalf("PUT snapshot: %v", r)
	}
	if k := prKeys(r); k != "lesson_id,me,member" {
		t.Fatalf("PUT талбарууд: %s", k)
	}
	if m := prMember(p.get(t, p.tok[E], ""), p.uids[B]); prNum(m["avg"]) != 65 {
		t.Fatalf("өнөөдрийн үнэлгээ snapshot-д орлоо: %v", m)
	}
	tv := p.get(t, p.tt, "")
	mb := prMember(tv, p.uids[B])
	if tv["role"] != "teacher" || prNum(mb["count"]) != 4 || prNum(mb["avg"]) != 61 {
		t.Fatalf("багшид шууд: %v", mb)
	}
	got := map[string]int{}
	for _, x := range mb["ratings"].([]any) {
		rt := x.(map[string]any)
		if k := prKeys(rt); k != "at,counted,lesson_id,lesson_title,rater_id,rater_name,score" {
			t.Fatalf("үнэлгээний талбарууд: %s", k)
		}
		if rt["counted"] != true || rt["lesson_id"] != "" || rt["lesson_title"] != "" {
			t.Fatalf("тоологдох ерөнхий үнэлгээ: %v", rt)
		}
		got[rt["rater_name"].(string)] = prNum(rt["score"])
	}
	if got["prp_a"] != 83 || got["prp_c"] != 60 || got["prp_d"] != 100 || got["prp_e"] != 0 {
		t.Fatalf("үнэлэгчид: %v", got)
	}
}

// Тоологдох үнэлэгч (S2): шинэ (3 хоноогүй) эсвэл идэвхгүй (сесс, бүлэг чатын мессежгүй) үнэлэгчийн үнэлгээ хадгалагдаж,
// багшид counted:false, дундаж/тоо/рейтинг/ангийн дунджид орохгүй; бүлэг чатад бичсэн нь идэвхтэйд тооцогдоно.
func TestPeerRatingsEligibility(t *testing.T) {
	p := newPRClass(t, "pre_", 5, 1)
	ctx := context.Background()
	A, B, C, D, E := 0, 1, 2, 3, 4
	p.season(t, A)
	// C: 10 хоногийн өмнө элссэн ч идэвхгүй; D: 10 хоног + бүлэг чатад бичсэн; E: шинэ.
	p.mem.SetEnrolledAt(p.uids[C], p.cid, time.Now().Add(-10*24*time.Hour))
	p.mem.SetEnrolledAt(p.uids[D], p.cid, time.Now().Add(-10*24*time.Hour))
	conv, err := p.st.GetOrCreateGroupConversation(ctx, p.tid, p.cid, "Хамтын анги")
	if err != nil {
		t.Fatal(err)
	}
	if err := p.st.AddMessage(ctx, &store.Message{ConversationID: conv.ID, Sender: store.SenderVisitor, SenderID: p.uids[D], Body: "сайн уу"}); err != nil {
		t.Fatal(err)
	}
	p.rateOK(t, A, B, 80)
	p.rateOK(t, C, B, 10)
	p.rateOK(t, D, B, 60)
	p.rateOK(t, E, B, 0)
	tv := p.get(t, p.tt, "")
	m := prMember(tv, p.uids[B])
	if prNum(m["count"]) != 2 || prNum(m["avg"]) != 70 || prNum(tv["class_avg"]) != 70 {
		t.Fatalf("зөвхөн A, D тоологдоно: %v %v", m, tv["class_avg"])
	}
	// Рейтинг: (140 + 3·70)/5 = 70.
	if prNum(m["collab"]) != 70 {
		t.Fatalf("рейтинг: %v", m)
	}
	counted := map[string]bool{}
	for _, x := range m["ratings"].([]any) {
		rt := x.(map[string]any)
		counted[rt["rater_id"].(string)] = rt["counted"] == true
	}
	if len(counted) != 4 || !counted[p.uids[A]] || counted[p.uids[C]] || !counted[p.uids[D]] || counted[p.uids[E]] {
		t.Fatalf("counted тэмдэг: %v", counted)
	}
	// Өгсөн нь тоологдох эсэхээс үл хамааран (багш шалгана).
	if me := prMember(tv, p.uids[E]); prNum(me["given"]) != 1 || prNum(me["given_avg"]) != 0 {
		t.Fatalf("шинэ үнэлэгчийн өгсөн: %v", me)
	}
	// E-г 3 хоноод сесстэй болгоход тоологдоно (кэш 25 сек — тестэд цэвэрлэнэ).
	p.season(t, E)
	if m := prMember(p.get(t, p.tt, ""), p.uids[B]); prNum(m["count"]) != 3 {
		t.Fatalf("хугацаа өнгөрсний дараа: %v", m)
	}
}

// Багшид анхааруулга (peer_alert): хүүхэд 2+ бага (<40) хосын оноо авбал (тоологдох эсэхээс үл хамааран — шинэ
// бүртгэлүүдийн бөөнөөр дайралтыг ч), үнэлэгч 3+ найздаа бага оноо өгвөл — эзэмшигчид, (сургалт, хүүхэд) бүрт өдөрт нэг
// (процесс дахин эхэлсэн ч HasNotification-оор).
func TestPeerAlerts(t *testing.T) {
	p := newPRClass(t, "prl_", 8, 1)
	A, B, C, D, E, F, G, H := 0, 1, 2, 3, 4, 5, 6, 7
	p.season(t, A, B, C, D, E) // F, G, H — шинэ бүртгэл (тоологдохгүй)
	alerts := func(uid string) []map[string]any {
		_, ns := callList(t, p.srv, "GET", "/api/me/notifications", p.tt)
		var out []map[string]any
		for _, x := range ns {
			n := x.(map[string]any)
			if n["type"] == peerAlertType && n["link"] == "/c/"+p.cid+"#chat=group&peer="+uid {
				out = append(out, n)
			}
		}
		return out
	}
	p.rateOK(t, A, E, 10)
	if n := len(alerts(p.uids[E])); n != 0 {
		t.Fatalf("1 бага оноонд анхааруулга: %d", n)
	}
	p.rateOK(t, B, E, 20)
	as := alerts(p.uids[E])
	if len(as) != 1 || !strings.HasPrefix(as[0]["title"].(string), "⚠️ Хамтын үнэлгээ: ") || !strings.Contains(as[0]["title"].(string), "prl_e") {
		t.Fatalf("(а) анхааруулга: %v", as)
	}
	p.rateOK(t, C, E, 5)
	if n := len(alerts(p.uids[E])); n != 1 {
		t.Fatalf("өдөрт нэг удаа: %d", n)
	}
	// Бөөнөөр дайралт: шинэ бүртгэл G, H (тоологдохгүй) D-д 0 өгвөл ч (а) анхааруулна.
	p.rateOK(t, G, D, 0)
	if n := len(alerts(p.uids[D])); n != 0 {
		t.Fatalf("1 шинэ бүртгэлийн бага оноонд: %d", n)
	}
	p.rateOK(t, H, D, 0)
	if as := alerts(p.uids[D]); len(as) != 1 || !strings.Contains(as[0]["body"].(string), "2 найзаасаа") {
		t.Fatalf("шинэ бүртгэлүүдийн бөөнөөр дайралт: %v", as)
	}
	// (б) Шинэ бүртгэл F 3 найздаа 0 өгвөл.
	p.rateOK(t, F, A, 0)
	p.rateOK(t, F, B, 0)
	if n := len(alerts(p.uids[F])); n != 0 {
		t.Fatalf("2 бага оноонд (б): %d", n)
	}
	p.rateOK(t, F, C, 0)
	if as := alerts(p.uids[F]); len(as) != 1 || !strings.Contains(as[0]["body"].(string), "3 найздаа") {
		t.Fatalf("(б) анхааруулга: %v", as)
	}
	// Процесс дахин эхэлсэн мэт (санах ойн давхардал цэвэрлэгдэнэ) — HasNotification давтахгүй.
	p.s.peerAl.mu.Lock()
	p.s.peerAl.sent = nil
	p.s.peerAl.mu.Unlock()
	p.rateOK(t, F, D, 0)
	if n, m := len(alerts(p.uids[F])), len(alerts(p.uids[D])); n != 1 || m != 1 {
		t.Fatalf("дахин эхэлсний дараа давхардлаа: F %d, D %d", n, m)
	}
	// Нэг л бага оноо авсан A-д анхааруулга үгүй.
	if n := len(alerts(p.uids[A])); n != 0 {
		t.Fatalf("1 бага оноонд (а): %d", n)
	}
}

// Өдрийн snapshot тогтвортой: өмнөх өдрийн үнэлгээг өнөөдөр засах, устгах, мөн өнөөдөр шинээр өгөх нь суралцагчийн GET ба
// PUT-ийн хариуны оноог маргааш хүртэл өөрчлөхгүй (нэг нэгээр нь хөдөлгөж бусдын оноог тооцоолох халдлагаас); багшид шууд;
// маргааш шинэ утга тусна.
func TestPeerRatingsSnapshotStable(t *testing.T) {
	p := newPRClass(t, "prs_", 6, 1)
	A, B, C, D, E, F := 0, 1, 2, 3, 4, 5
	L := p.lids[0]
	p.season(t, A, B, C, D, E, F)
	p.rateAged(t, A, B, "", 80)
	p.rateAged(t, A, B, L, 40) // A-ийн хос: 60
	p.rateAged(t, C, B, "", 60)
	p.rateAged(t, D, B, "", 100)
	p.rateAged(t, E, B, "", 75)
	// Хосууд 60, 60, 100, 75 → 295/4 = 73.75 → Bayes (295 + 221.25)/7 = 73.75 → snapshot 75; багшид 74.
	view := func(who int) int {
		t.Helper()
		return prNum(prMember(p.get(t, p.tok[who], ""), p.uids[B])["avg"])
	}
	live := func() map[string]any {
		t.Helper()
		return prMember(p.get(t, p.tt, ""), p.uids[B])
	}
	if view(A) != 75 || view(F) != 75 || prNum(live()["avg"]) != 74 {
		t.Fatalf("эхлэл: A %d, F %d, багш %v", view(A), view(F), live())
	}
	// Өнөөдөр A ерөнхийгөө 0 болгов → PUT-ийн хариу, GET 75 хэвээр; багшид шууд: (20+60+100+75)/4 = 63.75 → 64.
	if r := p.rateOK(t, A, B, 0); prNum(r["member"].(map[string]any)["avg"]) != 75 {
		t.Fatalf("засварын PUT snapshot: %v", r)
	}
	if view(A) != 75 || view(F) != 75 || prNum(live()["avg"]) != 64 {
		t.Fatalf("засварын дараа: A %d, F %d, багш %v", view(A), view(F), live())
	}
	// Өнөөдөр A хичээлийн үнэлгээгээ устгав → 75 хэвээр; багшид: (0+60+100+75)/4 = 58.75 → 59.
	if code := p.del(t, p.tok[A], p.uids[B], "lesson_id="+L); code != http.StatusNoContent {
		t.Fatalf("устгах: %d", code)
	}
	if view(A) != 75 || view(F) != 75 || prNum(live()["avg"]) != 59 {
		t.Fatalf("устгасны дараа: A %d, F %d, багш %v", view(A), view(F), live())
	}
	// Дахин засах ч (өнөөдөр), өнөөдөр шинээр өгсөн F-ийн үнэлгээ ч тусахгүй.
	if r := p.rateOK(t, A, B, 10); prNum(r["member"].(map[string]any)["avg"]) != 75 {
		t.Fatalf("дахин засварын PUT snapshot: %v", r)
	}
	p.rateOK(t, F, B, 0)
	if view(A) != 75 || view(C) != 75 {
		t.Fatalf("өнөөдрийн шинэ үнэлгээ тусчээ: A %d, C %d", view(A), view(C))
	}
	if m := live(); prNum(m["count"]) != 5 || prNum(m["avg"]) != 49 { // (10+60+100+75+0)/5
		t.Fatalf("багшид шууд: %v", m)
	}
	// Маргааш: хосууд 10, 60, 100, 75, 0 → 245/5 = 49 → Bayes (245 + 147)/8 = 49 → 50.
	p.mem.ShiftPeerRatings(p.cid, 24*time.Hour)
	p.s.peers.Delete(p.cid)
	if view(A) != 50 || view(F) != 50 {
		t.Fatalf("маргааш: A %d, F %d", view(A), view(F))
	}
	if r := p.rateOK(t, C, B, 60); prNum(r["member"].(map[string]any)["avg"]) != 50 { // ижил оноогоор дахин — 50 хэвээр
		t.Fatalf("маргаашийн PUT: %v", r)
	}
}

// Тооцоо: Bayes рейтинг, байр (ижил рейтингт ижил байр, үнэлгээгүй нь сүүлд), багшийн бодит дундаж (3-аас цөөнд ч),
// өгсөн тоо/дундаж, ангийн дундаж; шинэчлэл хуучныг солино; суралцагч өөрийнхөө, багш ?rater=-ээр устгана.
func TestPeerRatingsCollab(t *testing.T) {
	p := newPRClass(t, "prc_", 5, 1)
	A, B, C, D, E := 0, 1, 2, 3, 4
	p.season(t, A, B, C, D, E)
	p.rateOK(t, A, B, 90)
	p.rateOK(t, C, B, 80)
	p.rateOK(t, D, B, 70)
	p.rateOK(t, B, A, 50)
	p.rateOK(t, A, C, 100)
	p.rateOK(t, B, E, 100)
	// Нийт 490 / 6 = 81.67 → B: (240+245)/6 = 80.8 → 81; A: (50+245)/4 = 73.75 → 74; C, E: (100+245)/4 = 86.25 → 86.
	tv := p.get(t, p.tt, "")
	if prNum(tv["class_avg"]) != 82 {
		t.Fatalf("ангийн дундаж: %v", tv["class_avg"])
	}
	type row struct{ collab, rank, avg, count, given, givenAvg int }
	want := map[int]row{
		C: {86, 1, 100, 1, 1, 80},
		E: {86, 1, 100, 1, 0, -1},
		B: {81, 3, 80, 3, 2, 75},
		A: {74, 4, 50, 1, 2, 95},
		D: {-1, -1, -1, 0, 1, 70},
	}
	ms := tv["members"].([]any)
	order := []int{C, E, B, A, D}
	if len(ms) != len(order) {
		t.Fatalf("гишүүд: %v", ms)
	}
	for i, idx := range order {
		m := ms[i].(map[string]any)
		if m["user_id"] != p.uids[idx] {
			t.Fatalf("%d-р мөр %v (хүлээсэн %s): %v", i, m["user_id"], p.uids[idx], ms)
		}
		w := want[idx]
		g := row{prNum(m["collab"]), prNum(m["rank"]), prNum(m["avg"]), prNum(m["count"]), prNum(m["given"]), prNum(m["given_avg"])}
		if g != w {
			t.Fatalf("%s: %+v, хүлээсэн %+v", m["name"], g, w)
		}
	}
	// Шинэчлэл: A→B 90 → 30 (тоо хэвээр 3, дундаж 60).
	p.rateOK(t, A, B, 30)
	if ps, _ := p.st.PeerRatings(context.Background(), p.cid); len(ps) != 6 {
		t.Fatalf("шинэчлэл шинэ мөр нэмэх ёсгүй: %v", ps)
	}
	// Нийт 430 / 6 = 71.67 → B: (180+215)/6 = 65.8 → 66.
	if m := prMember(p.get(t, p.tt, ""), p.uids[B]); prNum(m["collab"]) != 66 || prNum(m["avg"]) != 60 || prNum(m["count"]) != 3 {
		t.Fatalf("шинэчилсний дараах рейтинг: %v", m)
	}
	// Суралцагч өөрийн өгсөн үнэлгээгээ устгана; дахин устгавал 404; бусдын үнэлгээг ?rater=-ээр устгаж чадахгүй.
	if code := p.del(t, p.tok[A], p.uids[C], ""); code != http.StatusNoContent {
		t.Fatalf("өөрийн үнэлгээ устгах: %d", code)
	}
	if code := p.del(t, p.tok[A], p.uids[C], ""); code != http.StatusNotFound {
		t.Fatalf("дахин устгах: %d", code)
	}
	if code := p.del(t, p.tok[E], p.uids[B], "rater="+p.uids[D]); code != http.StatusNotFound {
		t.Fatalf("суралцагч бусдын үнэлгээ устгав: %d", code)
	}
	if m := prMember(p.get(t, p.tt, ""), p.uids[C]); prNum(m["count"]) != 0 || m["collab"] != nil || m["rank"] != nil {
		t.Fatalf("устгасны дараа C: %v", m)
	}
	// Багш: ?rater= заавал.
	if code := p.del(t, p.tt, p.uids[B], ""); code != http.StatusNotFound {
		t.Fatalf("багш rater-гүй: %d", code)
	}
	if code := p.del(t, p.tt, p.uids[B], "rater="+p.uids[E]); code != http.StatusNotFound { // E нь B-г үнэлээгүй
		t.Fatalf("байхгүй үнэлгээ: %d", code)
	}
	if code := p.del(t, p.tt, p.uids[B], "rater="+p.uids[D]); code != http.StatusNoContent {
		t.Fatalf("багш устгах: %d", code)
	}
	m := prMember(p.get(t, p.tt, ""), p.uids[B])
	if prNum(m["count"]) != 2 {
		t.Fatalf("багш устгасны дараа B: %v", m)
	}
	for _, x := range m["ratings"].([]any) {
		if x.(map[string]any)["rater_id"] == p.uids[D] {
			t.Fatalf("устгасан үнэлгээ үлдлээ: %v", m)
		}
	}
	if code := p.del(t, p.tok[D], p.uids[B], ""); code != http.StatusNotFound { // багш устгасан тул D-д ч алга
		t.Fatalf("D-ийн устгагдсан үнэлгээ: %d", code)
	}
}

// Хичээл бүрээр (L1–L3): хичээлийн PUT/GET; сургалтын нийт оноо хосоор (нэг үнэлэгч 5 хичээл үнэлсэн ч нэг л тоологдоно);
// хичээлийн босго тусдаа; хичээлийн жагсаалт (суралцагчид нуусан нь үгүй, дарааллаар); DELETE ?lesson_id=.
func TestPeerRatingsLessons(t *testing.T) {
	p := newPRClass(t, "prq_", 5, 6)
	A, B, C, D, E := 0, 1, 2, 3, 4
	p.season(t, A, B, C, D, E)
	L := p.lids
	if code, _ := call(t, p.srv, "PUT", "/api/courses/"+p.cid+"/lessons/"+L[5]+"/visibility", p.tt, `{"hidden":true}`); code != http.StatusOK {
		t.Fatalf("хичээл нуух: %d", code)
	}
	ids := func(v any) []string {
		var out []string
		for _, x := range v.([]any) {
			l := x.(map[string]any)
			if prKeys(l) != "id,title" {
				t.Fatalf("хичээлийн талбарууд: %v", l)
			}
			out = append(out, l["id"].(string))
		}
		return out
	}
	if got := ids(p.get(t, p.tok[A], "")["lessons"]); strings.Join(got, ",") != strings.Join(L[:5], ",") {
		t.Fatalf("суралцагчийн хичээлүүд: %v", got)
	}
	if got := ids(p.get(t, p.tt, "")["lessons"]); len(got) != 6 {
		t.Fatalf("багшийн хичээлүүд (нуусан нь ч): %v", got)
	}

	// Хичээлийн үнэлгээ: PUT → GET ?lesson_id; ерөнхий mine-д орохгүй.
	r := p.rateL(t, A, B, L[0], 90)
	if r["lesson_id"] != L[0] || prNum(r["member"].(map[string]any)["mine"]) != 90 {
		t.Fatalf("хичээлийн PUT: %v", r)
	}
	g := p.get(t, p.tok[A], "lesson_id="+L[0])
	if g["lesson_id"] != L[0] || prNum(prMember(g, p.uids[B])["mine"]) != 90 {
		t.Fatalf("хичээлийн GET: %v", g)
	}
	if m := prMember(p.get(t, p.tok[A], ""), p.uids[B]); m["mine"] != nil {
		t.Fatalf("хичээлийн үнэлгээ ерөнхий mine-д орлоо: %v", m)
	}

	// Хос: A нь B-г ерөнхий 100 + 4 хичээлээр (90 дээр нэмж 80, 70, 60) → хос 80; C ерөнхий 20 → B: 2 үнэлэгч, дундаж 50.
	p.rateOK(t, A, B, 100)
	p.rateL(t, A, B, L[1], 80)
	p.rateL(t, A, B, L[2], 70)
	p.rateL(t, A, B, L[3], 60)
	p.rateOK(t, C, B, 20)
	tv := p.get(t, p.tt, "")
	mb := prMember(tv, p.uids[B])
	// Нийт хос: 80 + 20 = 100 / 2 = 50 → рейтинг (100 + 150)/5 = 50.
	if prNum(mb["count"]) != 2 || prNum(mb["avg"]) != 50 || prNum(mb["collab"]) != 50 || len(mb["ratings"].([]any)) != 6 {
		t.Fatalf("хосын нэгтгэл: %v", mb)
	}
	if ma := prMember(tv, p.uids[A]); prNum(ma["given"]) != 1 || prNum(ma["given_avg"]) != 80 {
		t.Fatalf("A-ийн өгсөн (хосоор): %v", ma)
	}
	titled := false
	for _, x := range mb["ratings"].([]any) {
		rt := x.(map[string]any)
		if rt["lesson_id"] == L[2] && rt["lesson_title"] == "Хичээл 3" && prNum(rt["score"]) == 70 {
			titled = true
		}
	}
	if !titled {
		t.Fatalf("хичээлийн нэртэй үнэлгээ: %v", mb["ratings"])
	}
	// Багшийн хичээлийн шүүлт: зөвхөн тэр хичээлийнх.
	tl := p.get(t, p.tt, "lesson_id="+L[0])
	if ml := prMember(tl, p.uids[B]); prNum(ml["count"]) != 1 || prNum(ml["avg"]) != 90 || len(ml["ratings"].([]any)) != 1 {
		t.Fatalf("багшийн хичээлийн хариу: %v", ml)
	}

	// Хичээлийн босго: L[4]-т A, C, D (өчигдөр) → E харна, A харахгүй; L[3]-т зөвхөн A (+ өнөөдрийнх) → E-д null.
	p.rateAged(t, A, D, L[4], 70)
	p.rateAged(t, B, D, L[4], 80)
	p.rateAged(t, C, D, L[4], 90)
	if m := prMember(p.get(t, p.tok[E], "lesson_id="+L[4]), p.uids[D]); m["avg"] == nil {
		t.Fatalf("хичээлийн босго хангасан: %v", m)
	}
	if m := prMember(p.get(t, p.tok[A], "lesson_id="+L[4]), p.uids[D]); m["avg"] != nil || prNum(m["mine"]) != 70 {
		t.Fatalf("үзэгчийг тоолсон хичээлийн босго: %v", m)
	}
	if m := prMember(p.get(t, p.tok[E], "lesson_id="+L[3]), p.uids[D]); m["avg"] != nil {
		t.Fatalf("үнэлгээгүй хичээл: %v", m)
	}

	// DELETE ?lesson_id=: зөвхөн тэр хичээлийнх; ерөнхий хэвээр.
	if code := p.del(t, p.tok[A], p.uids[B], "lesson_id="+L[0]); code != http.StatusNoContent {
		t.Fatalf("хичээлийн үнэлгээ устгах: %d", code)
	}
	if code := p.del(t, p.tok[A], p.uids[B], "lesson_id="+L[0]); code != http.StatusNotFound {
		t.Fatalf("дахин устгах: %d", code)
	}
	if code := p.del(t, p.tt, p.uids[B], "rater="+p.uids[A]+"&lesson_id="+L[1]); code != http.StatusNoContent {
		t.Fatalf("багш хичээлийн үнэлгээ устгах: %d", code)
	}
	if m := prMember(p.get(t, p.tok[A], ""), p.uids[B]); prNum(m["mine"]) != 100 {
		t.Fatalf("ерөнхий үнэлгээ устлаа: %v", m)
	}
	if m := prMember(p.get(t, p.tt, ""), p.uids[B]); len(m["ratings"].([]any)) != 4 {
		t.Fatalf("устгасны дараах үнэлгээнүүд: %v", m["ratings"])
	}
}

// Кэш (R): багшийн хариу кэшээс (store-д шууд бичсэн нь кэш дуустал харагдахгүй), харин PUT/DELETE кэшийг шууд хүчингүй
// болгоно.
func TestPeerRatingsCache(t *testing.T) {
	p := newPRClass(t, "prk_", 3, 1)
	A, B, C := 0, 1, 2
	p.season(t, A, B, C)
	p.rateOK(t, A, B, 80)
	if m := prMember(p.get(t, p.tt, ""), p.uids[B]); prNum(m["count"]) != 1 {
		t.Fatalf("эхний: %v", m)
	}
	if _, err := p.st.SetPeerRating(context.Background(), p.cid, p.uids[C], p.uids[B], "", 40, peerCutoff(time.Now())); err != nil {
		t.Fatal(err)
	}
	if m := prMember(p.get(t, p.tt, ""), p.uids[B]); prNum(m["count"]) != 1 {
		t.Fatalf("кэш ашиглагдаагүй: %v", m)
	}
	p.rateOK(t, A, C, 70) // PUT → хүчингүй
	if m := prMember(p.get(t, p.tt, ""), p.uids[B]); prNum(m["count"]) != 2 {
		t.Fatalf("PUT-ийн дараа кэш шинэчлэгдээгүй: %v", m)
	}
	if code := p.del(t, p.tok[A], p.uids[B], ""); code != http.StatusNoContent {
		t.Fatalf("устгах: %d", code)
	}
	if m := prMember(p.get(t, p.tt, ""), p.uids[B]); prNum(m["count"]) != 1 {
		t.Fatalf("DELETE-ийн дараа кэш шинэчлэгдээгүй: %v", m)
	}
}

// Цэвэрлэгээ: бүртгэл устгах (өгсөн ба авсан), журналын «бүрмөсөн устгах» (тэр сургалтад өгсөн ба авсан, хичээлийнх ч),
// сургалт устгах (тэр сургалтынх) — бусад хэвээр.
func TestPeerRatingsCleanup(t *testing.T) {
	p := newPRClass(t, "prx_", 4, 1)
	ctx := context.Background()
	A, B, C, D := 0, 1, 2, 3
	p.rateOK(t, A, B, 70)
	p.rateL(t, A, B, p.lids[0], 75)
	p.rateOK(t, B, A, 60)
	p.rateOK(t, C, D, 50)
	p.rateOK(t, D, C, 40)
	p.rateL(t, C, A, p.lids[0], 90)
	ratings := func() map[string]int {
		ps, err := p.st.PeerRatings(ctx, p.cid)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]int{}
		for _, r := range ps {
			out[r.RaterID+">"+r.RateeID+"@"+r.LessonID] = r.Score
		}
		return out
	}
	key := func(a, b int) string { return p.uids[a] + ">" + p.uids[b] + "@" }

	// Журналын устгал: A архивлаад устгана → A-ийн өгсөн (A>B, A>B@l) ба авсан (B>A, C>A@l) устна.
	jmArchive(t, p.srv, p.tt, p.cid, p.uids[A])
	jmResetOK(t, p.srv, "устгах", p.tt, p.cid, p.uids[A], `{"confirm":true}`)
	got := ratings()
	if len(got) != 2 || got[key(C, D)] != 50 || got[key(D, C)] != 40 {
		t.Fatalf("журналын устгалын дараа: %v", got)
	}

	// Бүртгэл устгах: D-ийн өгсөн ба авсан.
	if code, r := call(t, p.srv, "POST", "/api/me/delete", p.tok[D], `{"confirm":"УСТГАХ"}`); code != http.StatusOK {
		t.Fatalf("бүртгэл устгах: %d %v", code, r)
	}
	if got := ratings(); len(got) != 0 {
		t.Fatalf("бүртгэл устгасны дараа: %v", got)
	}

	// Сургалт устгах: тэр сургалтын бүх үнэлгээ.
	p.rateOK(t, B, C, 80)
	if _, err := p.st.SetPeerRating(ctx, "other-course", p.uids[B], p.uids[C], "", 10, peerCutoff(time.Now())); err != nil {
		t.Fatal(err)
	}
	if err := p.st.DeleteCourse(ctx, p.tid, p.cid); err != nil {
		t.Fatal(err)
	}
	if got := ratings(); len(got) != 0 {
		t.Fatalf("сургалт устгасны дараа: %v", got)
	}
	if ps, _ := p.st.PeerRatings(ctx, "other-course"); len(ps) != 1 {
		t.Fatalf("өөр сургалтын үнэлгээ устлаа: %v", ps)
	}
}

// Аналитикийн «collaboration» хэмжүүр: самбаргүй суралцагчид харилцан үнэлгээний рейтинг; самбартай бол хоёрын дундаж.
func TestPeerCollabDim(t *testing.T) {
	st := &StudentStat{peerSum: 80, peerN: 1}
	if d := st.dimScores(); d[7] != 80 {
		t.Fatalf("зөвхөн peer: %v", d)
	}
	st = &StudentStat{BoardDays: 1, CollabScore: 60, peerSum: 160, peerN: 2}
	if d := st.dimScores(); d[7] != 70 {
		t.Fatalf("самбар 60 + peer 80: %v", d)
	}
	st = &StudentStat{BoardDays: 1, CollabScore: 60}
	if d := st.dimScores(); d[7] != 60 {
		t.Fatalf("зөвхөн самбар: %v", d)
	}
	if d := (&StudentStat{}).dimScores(); d[7] != -1 {
		t.Fatalf("өгөгдөлгүй: %v", d)
	}

	// Бодит аналитик: үнэлэгдсэн суралцагчийн collaboration = рейтинг (тоологдох үнэлэгчээр).
	p := newPRClass(t, "prd_", 3, 1)
	p.season(t, 0, 1, 2)
	p.rateOK(t, 0, 1, 90)
	p.rateOK(t, 2, 1, 90)
	p.rateOK(t, 1, 0, 60)
	// Нийт 240/3 = 80 → B: (180+240)/5 = 84.
	code, r := call(t, p.srv, "GET", "/api/me/analytics/students/"+p.uids[1]+"?course="+p.cid, p.tt, "")
	if code != http.StatusOK {
		t.Fatalf("аналитик: %d %v", code, r)
	}
	stu, _ := r["student"].(map[string]any)
	if stu == nil {
		t.Fatalf("суралцагчийн нэгтгэл алга: %v", r)
	}
	if dims := stu["dims"].([]any); prNum(dims[7]) != 84 {
		t.Fatalf("collaboration хэмжүүр: %v", dims)
	}
	if tv := prMember(p.get(t, p.tt, ""), p.uids[1]); prNum(tv["collab"]) != 84 {
		t.Fatalf("багшийн хүснэгтийн рейтинг: %v", tv)
	}
}
