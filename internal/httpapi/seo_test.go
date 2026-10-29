package httpapi

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func getPage(t *testing.T, srv *httptest.Server, path string) (int, string) {
	t.Helper()
	res, err := http.Get(srv.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

var ldRe = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)

// ldGraph — хуудасны JSON-LD-г задалж (хүчинтэй JSON эсэхийг шалгана), бүх @type-ийг буцаана.
func ldTypes(t *testing.T, page string) (map[string]map[string]any, string) {
	t.Helper()
	m := ldRe.FindStringSubmatch(page)
	if m == nil {
		t.Fatal("JSON-LD алга")
	}
	var doc struct {
		Graph []map[string]any `json:"@graph"`
	}
	if err := json.Unmarshal([]byte(m[1]), &doc); err != nil {
		t.Fatalf("JSON-LD хүчингүй: %v\n%s", err, m[1])
	}
	out := map[string]map[string]any{}
	for _, n := range doc.Graph {
		if ty, _ := n["@type"].(string); ty != "" {
			out[ty] = n
		}
	}
	return out, m[1]
}

func TestSEO(t *testing.T) {
	srv, _ := newTestServer(t)
	defer srv.Close()
	tt, _ := register(t, srv, "seoteach", "teacher")
	call(t, srv, "PUT", "/api/me/profile", tt, `{"display_name":"Болд Багш","headline":"Математикийн багш","bio":"20 жил математик заасан.","subjects":["Математик"],"links":{"website":"https://example.com"}}`)
	_, c := call(t, srv, "POST", "/api/courses", tt, `{"title":"Алгебр </script><script>alert(1)</script>","description":"ЭЕШ-д бэлтгэх алгебрийн бүрэн сургалт: тэгшитгэл, функц, график.","price":25000,"published":true}`)
	cid := c["id"].(string)
	call(t, srv, "POST", "/api/courses/"+cid+"/lessons", tt, `{"title":"Шугаман тэгшитгэл","is_free":true}`)
	_, lk := call(t, srv, "POST", "/api/courses", tt, `{"title":"10А анги","price":0,"published":true,"access_code":"Анги-10А"}`)
	_, dr := call(t, srv, "POST", "/api/courses", tt, `{"title":"Ноорог","price":0,"published":false}`)

	// robots.txt
	if code, body := getPage(t, srv, "/robots.txt"); code != 200 || !strings.Contains(body, "Sitemap: ") || !strings.Contains(body, "/sitemap.xml") || !strings.Contains(body, "Disallow: /api/") {
		t.Fatalf("robots.txt: %d %q", code, body)
	}
	// sitemap.xml: хүчинтэй XML, профайл + нийтлэгдсэн сургалт; нууц үгтэй, ноорог орохгүй
	code, sm := getPage(t, srv, "/sitemap.xml")
	if code != 200 {
		t.Fatalf("sitemap: %d", code)
	}
	var us struct {
		URLs []struct {
			Loc string `xml:"loc"`
		} `xml:"url"`
	}
	if err := xml.Unmarshal([]byte(sm), &us); err != nil {
		t.Fatalf("sitemap XML хүчингүй: %v", err)
	}
	locs := map[string]bool{}
	for _, u := range us.URLs {
		locs[u.Loc] = true
	}
	has := func(suffix string) bool {
		for l := range locs {
			if strings.HasSuffix(l, suffix) {
				return true
			}
		}
		return false
	}
	if !has("/t/seoteach") || !has("/c/"+cid) {
		t.Fatalf("sitemap-д профайл, сургалт байх ёстой: %v", locs)
	}
	if has("/c/"+lk["id"].(string)) || has("/c/"+dr["id"].(string)) {
		t.Fatalf("нууц үгтэй / ноорог сургалт sitemap-д орох ёсгүй: %v", locs)
	}

	// Сургалтын хуудас: canonical, Course + offers + hasCourseInstance + BreadcrumbList, XSS-гүй
	_, page := getPage(t, srv, "/c/"+cid)
	if !strings.Contains(page, `rel="canonical" href="`) || !strings.Contains(page, "/c/"+cid+`"`) {
		t.Fatal("сургалтын хуудсанд canonical алга")
	}
	if strings.Contains(page, "<script>alert(1)") {
		t.Fatal("гарчиг script болж гарсан (XSS)")
	}
	types, raw := ldTypes(t, page)
	course := types["Course"]
	if course == nil || types["BreadcrumbList"] == nil {
		t.Fatalf("Course, BreadcrumbList байх ёстой: %s", raw)
	}
	offers, _ := course["offers"].([]any)
	if len(offers) != 1 || offers[0].(map[string]any)["priceCurrency"] != "MNT" || offers[0].(map[string]any)["price"].(float64) != 25000 {
		t.Fatalf("offers: %v", course["offers"])
	}
	if inst, _ := course["hasCourseInstance"].([]any); len(inst) != 1 || inst[0].(map[string]any)["courseMode"] != "Online" {
		t.Fatalf("hasCourseInstance: %v", course["hasCourseInstance"])
	}
	if !strings.Contains(page, `content="index, follow, max-image-preview:large`) {
		t.Fatal("robots meta (index) алга")
	}
	if !strings.Contains(page, "| surgalt.mn</title>") {
		t.Fatal("гарчигт брэнд алга")
	}

	// Профайл: ProfilePage + Person (sameAs, knowsAbout) + сургалтын жагсаалт (нууц үгтэйг оруулахгүй)
	_, page = getPage(t, srv, "/t/seoteach")
	types, raw = ldTypes(t, page)
	if types["ProfilePage"] == nil || types["Person"] == nil || types["ItemList"] == nil {
		t.Fatalf("ProfilePage, Person, ItemList байх ёстой: %s", raw)
	}
	if p := types["Person"]; p["name"] != "Болд Багш" || p["jobTitle"] != "Математикийн багш" {
		t.Fatalf("Person: %v", p)
	}
	if items := types["ItemList"]["itemListElement"].([]any); len(items) != 1 {
		t.Fatalf("жагсаалтад зөвхөн нийтийн сургалт: %v", items)
	}

	// Нүүр: WebSite (сайтын нэр); хувийн хуудсууд noindex
	_, page = getPage(t, srv, "/")
	if types, _ := ldTypes(t, page); types["WebSite"] == nil || types["Organization"] == nil {
		t.Fatal("нүүрэнд WebSite, Organization байх ёстой")
	}
	for _, p := range []string{"/login", "/me"} {
		if _, body := getPage(t, srv, p); !strings.Contains(body, `content="noindex, nofollow"`) {
			t.Fatalf("%s noindex байх ёстой", p)
		}
	}
}

func TestMetaDesc(t *testing.T) {
	long := strings.Repeat("математик ", 40)
	d := metaDesc(long)
	if n := len([]rune(d)); n > 156 || !strings.HasSuffix(d, "…") {
		t.Fatalf("урт тайлбар таслагдах ёстой: %d %q", n, d)
	}
	if metaDesc("  богино\n\nтайлбар ") != "богино тайлбар" {
		t.Fatal("зай нэгтгэх")
	}
}
