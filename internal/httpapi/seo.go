package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// SEO: хайлтын систем (Google, Bing) багшийн профайл, сургалт, номыг бүрэн олж, зөв ойлгож, баялаг үр дүнгээр
// (rich results) харуулахад зориулсан техникийн хэсэг. Байр суурийг ямар ч код баталгаажуулахгүй — энэ нь
// Google-ийн шаарддаг бүх дохиог (мөлхөх боломж, бүтэцтэй өгөгдөл, canonical, гарчиг, тайлбар) бүрэн өгнө:
//   - /robots.txt — мөлхөх дүрэм + sitemap-ийн байршил (API, хувийн медиаг хаана)
//   - /sitemap.xml — нийтлэгдсэн сургалт, тэдгээрийн багшийн профайл, ном (lastmod, зурагтай; 1 цаг кэш)
//   - schema.org JSON-LD: ProfilePage + Person, Course (+ Offer, CourseInstance, хөтөлбөр), Book, BreadcrumbList,
//     WebSite + Organization (сайтын нэр)
//   - Google Search Console / Bing баталгаажуулах meta (GOOGLE_SITE_VERIFICATION, BING_SITE_VERIFICATION)

// Хайлтын системийн баталгаажуулах кодууд (New-д Config-оос онооно; template-д gsv/bsv).
var siteVerifyGoogle, siteVerifyBing string

// siteURL — сайтын эх (зураггүй хуудасны хуваалцах зураг /og.jpg); siteFBApp — fb:app_id. New-д онооно.
var siteURL, siteFBApp string

const sitemapTTL = time.Hour

type sitemapCache struct {
	mu   sync.Mutex
	at   time.Time
	body []byte
}

// handleRobots: GET /robots.txt
func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	// Номын хавтас нь хуваалцах зураг (og:image) — Facebook, хайлтын систем татаж чаддаг байх ёстой.
	_, _ = w.Write([]byte("User-agent: *\nAllow: /\nAllow: /api/books/*/cover\nDisallow: /api/\nDisallow: /board-img/\n\nSitemap: " + s.cfg.PublicURL + "/sitemap.xml\n"))
}

// handleSitemap: GET /sitemap.xml
func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	body, err := s.sitemap(r.Context())
	if err != nil {
		s.log.Warn("sitemap", "err", err)
		http.Error(w, "sitemap unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(body)
}

type smURL struct {
	loc, lastmod, freq, prio string
	images                   []smImage
}

type smImage struct{ loc, title string }

func (s *Server) sitemap(ctx context.Context) ([]byte, error) {
	sc := &s.smap
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if sc.body != nil && time.Since(sc.at) < sitemapTTL {
		return sc.body, nil
	}
	docs, err := s.searchDocs(ctx) // нийтлэгдсэн сургалтууд + багш (хайлтын индексээс — DB-д дахин хүрэхгүй)
	if err != nil {
		return nil, err
	}
	base := s.cfg.PublicURL
	type tinfo struct {
		t    PublicTeacher
		last time.Time
	}
	teachers := map[string]*tinfo{}
	var courses []smURL
	newest := time.Time{}
	for _, d := range docs {
		c := d.course
		ti := teachers[c.TeacherID]
		if ti == nil {
			ti = &tinfo{t: d.teacher}
			teachers[c.TeacherID] = ti
		}
		if c.UpdatedAt.After(ti.last) {
			ti.last = c.UpdatedAt
		}
		if c.UpdatedAt.After(newest) {
			newest = c.UpdatedAt
		}
		if c.Locked { // нууц үгтэй (ангийн дотоод) сургалтыг хайлтад санал болгохгүй
			continue
		}
		courses = append(courses, smURL{loc: base + "/c/" + c.ID, lastmod: smDate(c.UpdatedAt), freq: "weekly", prio: "0.8"})
	}
	urls := []smURL{{loc: base + "/", lastmod: smDate(newest), freq: "daily", prio: "1.0"}}
	tids := make([]string, 0, len(teachers))
	for id := range teachers {
		tids = append(tids, id)
	}
	sort.Strings(tids)
	var books []smURL
	for _, id := range tids {
		ti := teachers[id]
		u := smURL{loc: base + "/t/" + ti.t.Username, lastmod: smDate(ti.last), freq: "weekly", prio: "0.9"}
		if img := absURL(ti.t.AvatarURL, base); img != "" {
			u.images = append(u.images, smImage{img, ti.t.DisplayName})
		}
		urls = append(urls, u)
		if bs, err := s.store.BooksByTeacher(ctx, id, true); err == nil {
			for i := range bs {
				b := publicBook(&bs[i], false)
				bu := smURL{loc: base + "/b/" + b.ID, lastmod: smDate(bs[i].UpdatedAt), freq: "monthly", prio: "0.6"}
				if img := absURL(b.Cover, base); img != "" {
					bu.images = append(bu.images, smImage{img, b.Title})
				}
				books = append(books, bu)
			}
		}
	}
	urls = append(append(urls, courses...), books...)

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	buf.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:image="http://www.google.com/schemas/sitemap-image/1.1">` + "\n")
	for _, u := range urls {
		buf.WriteString("<url><loc>" + xmlEsc(u.loc) + "</loc>")
		if u.lastmod != "" {
			buf.WriteString("<lastmod>" + u.lastmod + "</lastmod>")
		}
		buf.WriteString("<changefreq>" + u.freq + "</changefreq><priority>" + u.prio + "</priority>")
		for _, im := range u.images {
			buf.WriteString("<image:image><image:loc>" + xmlEsc(im.loc) + "</image:loc><image:title>" + xmlEsc(im.title) + "</image:title></image:image>")
		}
		buf.WriteString("</url>\n")
	}
	buf.WriteString("</urlset>\n")
	sc.body, sc.at = buf.Bytes(), time.Now()
	return sc.body, nil
}

func smDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

func xmlEsc(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// absURL — өөрийн файл (/files/…) бол бүтэн хаяг болгоно; http(s) бол хэвээр; бусад тохиолдолд хоосон.
func absURL(u, base string) string {
	switch {
	case strings.HasPrefix(u, "https://"), strings.HasPrefix(u, "http://"):
		return u
	case strings.HasPrefix(u, "/") && !strings.HasPrefix(u, "//"):
		return base + u
	}
	return ""
}

// metaDesc — meta description: мөр, илүүдэл зайг нэгтгэж ~155 тэмдэгтэд үгийн заагаар таслана.
func metaDesc(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const max = 155
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)[:max]
	if i := strings.LastIndex(string(r), " "); i > max/2 {
		return strings.TrimRight(string(r)[:i], " ,.;:—-") + "…"
	}
	return string(r) + "…"
}

// courseDesc — сургалтын хуудасны тайлбар (багш тайлбар бичээгүй бол гарчиг, багш, хичээлийн тооноос).
func courseDesc(pc PublicCourse) string {
	c, t := pc.Course, pc.Teacher
	if d := metaDesc(c.Description); utf8.RuneCountInString(d) >= 50 {
		return d
	}
	parts := []string{c.Title + " — " + t.DisplayName + " багшийн онлайн сургалт", strconv.Itoa(c.LessonCount) + " хичээл"}
	if c.FreeLessonCount > 0 && !c.Locked {
		parts = append(parts, strconv.Itoa(c.FreeLessonCount)+" үнэгүй хичээл")
	}
	if d := strings.TrimSpace(c.Description); d != "" {
		parts = append([]string{d}, parts...)
	}
	return metaDesc(strings.Join(parts, " · ") + ". surgalt.mn")
}

// ldJS — JSON-LD-г <script type="application/ld+json">-д аюулгүй (</script>, & … escape) оруулна.
func ldJS(v any) template.JS {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		return template.JS("{}")
	}
	return template.JS(strings.TrimSpace(buf.String()))
}

func ldCrumbs(items ...[2]string) map[string]any {
	list := make([]map[string]any, len(items))
	for i, it := range items {
		list[i] = map[string]any{"@type": "ListItem", "position": i + 1, "name": it[0], "item": it[1]}
	}
	return map[string]any{"@type": "BreadcrumbList", "itemListElement": list}
}

func ldPerson(t PublicTeacher, base string) map[string]any {
	url := base + "/t/" + t.Username
	p := map[string]any{"@type": "Person", "@id": url + "#person", "name": t.DisplayName, "alternateName": "@" + t.Username,
		"url": url, "jobTitle": firstNonEmpty(t.Headline, "Багш")}
	if d := metaDesc(t.Bio); d != "" {
		p["description"] = d
	}
	if img := absURL(t.AvatarURL, base); img != "" {
		p["image"] = img
	}
	if len(t.Subjects) > 0 {
		p["knowsAbout"] = t.Subjects
	}
	if t.Location != "" {
		p["homeLocation"] = map[string]any{"@type": "Place", "name": t.Location}
	}
	var same []string
	for _, l := range t.Links {
		if strings.HasPrefix(l.URL, "https://") || strings.HasPrefix(l.URL, "http://") {
			same = append(same, l.URL)
		}
	}
	if len(same) > 0 {
		p["sameAs"] = same
	}
	return p
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func ldWebsiteRef(base string) map[string]any { return map[string]any{"@id": base + "/#website"} }

// ldProfile — багшийн профайл: ProfilePage (mainEntity = Person), сургалтуудын жагсаалт, замын мөр.
func ldProfile(p PublicProfile, base string) template.JS {
	t := p.Teacher
	url := base + "/t/" + t.Username
	person := ldPerson(t, base)
	if t.StudentCount > 0 {
		person["interactionStatistic"] = map[string]any{"@type": "InteractionCounter",
			"interactionType": "https://schema.org/FollowAction", "userInteractionCount": t.StudentCount}
	}
	page := map[string]any{"@type": "ProfilePage", "@id": url, "url": url, "name": t.DisplayName + " — " + firstNonEmpty(t.Headline, "Багш"),
		"inLanguage": "mn", "mainEntity": map[string]any{"@id": url + "#person"}, "isPartOf": ldWebsiteRef(base)}
	if p.OGImage != "" {
		page["primaryImageOfPage"] = p.OGImage
	}
	graph := []any{page, person}
	var items []map[string]any
	for _, c := range p.Courses {
		if c.Locked {
			continue
		}
		items = append(items, map[string]any{"@type": "ListItem", "position": len(items) + 1, "url": base + "/c/" + c.ID, "name": c.Title})
	}
	if len(items) > 0 {
		graph = append(graph, map[string]any{"@type": "ItemList", "name": t.DisplayName + " — сургалтууд", "itemListElement": items})
	}
	graph = append(graph, ldCrumbs([2]string{"surgalt.mn", base + "/"}, [2]string{t.DisplayName, url}))
	return ldJS(map[string]any{"@context": "https://schema.org", "@graph": graph})
}

// ldCourse — сургалт: Course (Google «Course info»: offers + hasCourseInstance), хөтөлбөр, замын мөр.
func ldCourse(pc PublicCourse, base string) template.JS {
	c, t := pc.Course, pc.Teacher
	url := base + "/c/" + c.ID
	instructor := map[string]any{"@type": "Person", "name": t.DisplayName, "url": base + "/t/" + t.Username}
	course := map[string]any{"@type": "Course", "@id": url + "#course", "name": c.Title, "description": courseDesc(pc), "url": url,
		"inLanguage": "mn", "provider": map[string]any{"@type": "Organization", "name": "surgalt.mn", "sameAs": base + "/"},
		"creator": instructor, "image": base + "/t/" + t.Username + "/og.jpg"}
	if !c.CreatedAt.IsZero() {
		course["dateCreated"] = c.CreatedAt.UTC().Format(time.RFC3339)
	}
	if !c.UpdatedAt.IsZero() {
		course["dateModified"] = c.UpdatedAt.UTC().Format(time.RFC3339)
	}
	if !c.Locked {
		paid := c.LessonCount - c.FreeLessonCount
		cat := "Free"
		switch {
		case c.Price > 0 && c.FreeLessonCount > 0, c.Price == 0 && paid > 0 && c.FreeLessonCount > 0:
			cat = "Partially Free"
		case c.Price > 0 || paid > 0:
			cat = "Paid"
		}
		course["offers"] = []any{map[string]any{"@type": "Offer", "category": cat, "price": c.Price, "priceCurrency": "MNT",
			"availability": "https://schema.org/InStock", "url": url}}
		course["isAccessibleForFree"] = cat == "Free"
		inst := map[string]any{"@type": "CourseInstance", "courseMode": "Online", "instructor": []any{instructor}}
		mins := 0
		for _, l := range pc.Lessons {
			mins += l.ActiveMin
		}
		if mins > 0 {
			inst["courseWorkload"] = "PT" + strconv.Itoa(mins) + "M"
		}
		course["hasCourseInstance"] = []any{inst}
	}
	// Хөтөлбөр: бүлэг (эсвэл хичээл) бүрийн нэр — хайлтад «юу заадаг» дохио.
	var syl []map[string]any
	if pc.Grouped {
		for _, sec := range pc.Sections {
			if sec.Title == "" || len(syl) >= 30 {
				continue
			}
			names := make([]string, 0, len(sec.Lessons))
			for _, l := range sec.Lessons {
				names = append(names, l.Title)
			}
			syl = append(syl, map[string]any{"@type": "Syllabus", "name": sec.Title, "description": metaDesc(strings.Join(names, " · "))})
		}
	} else {
		for _, l := range pc.Lessons {
			if len(syl) >= 30 {
				break
			}
			syl = append(syl, map[string]any{"@type": "Syllabus", "name": l.Title})
		}
	}
	if len(syl) > 0 {
		course["syllabusSections"] = syl
	}
	crumbs := ldCrumbs([2]string{"surgalt.mn", base + "/"}, [2]string{t.DisplayName, base + "/t/" + t.Username}, [2]string{c.Title, url})
	return ldJS(map[string]any{"@context": "https://schema.org", "@graph": []any{course, crumbs}})
}

// ldBook — ном, өгүүлэл: Book (+ Offer), замын мөр.
func ldBook(d map[string]any, base string) template.JS {
	b, ok1 := d["Book"].(PublicBook)
	t, ok2 := d["Teacher"].(PublicTeacher)
	if !ok1 || !ok2 {
		return template.JS("{}")
	}
	url := base + "/b/" + b.ID
	book := map[string]any{"@type": "Book", "@id": url + "#book", "name": b.Title, "url": url, "inLanguage": "mn", "bookFormat": "https://schema.org/EBook",
		"author":    map[string]any{"@type": "Person", "name": firstNonEmpty(b.Author, t.DisplayName)},
		"publisher": map[string]any{"@type": "Person", "name": t.DisplayName, "url": base + "/t/" + t.Username},
		"offers":    map[string]any{"@type": "Offer", "price": b.Price, "priceCurrency": "MNT", "availability": "https://schema.org/InStock", "url": url}}
	if dsc := metaDesc(b.Description); dsc != "" {
		book["description"] = dsc
	}
	if img := absURL(b.Cover, base); img != "" {
		book["image"] = img
	}
	if b.Pages > 0 {
		book["numberOfPages"] = b.Pages
	}
	crumbs := ldCrumbs([2]string{"surgalt.mn", base + "/"}, [2]string{t.DisplayName, base + "/t/" + t.Username}, [2]string{b.Title, url})
	return ldJS(map[string]any{"@context": "https://schema.org", "@graph": []any{book, crumbs}})
}

// ldHome — нүүр: WebSite (хайлтын үр дүнд сайтын нэр) + Organization.
func ldHome(base string) template.JS {
	site := map[string]any{"@type": "WebSite", "@id": base + "/#website", "url": base + "/", "name": "surgalt.mn", "alternateName": []string{"Сургалт.мн", "Surgalt"},
		"inLanguage": "mn", "publisher": map[string]any{"@id": base + "/#org"},
		"potentialAction": map[string]any{"@type": "SearchAction", "target": map[string]any{"@type": "EntryPoint", "urlTemplate": base + "/?q={search_term_string}"},
			"query-input": "required name=search_term_string"}}
	org := map[string]any{"@type": "Organization", "@id": base + "/#org", "name": "surgalt.mn", "url": base + "/",
		"description": "Багш нар онлайн сургалт, ном, шууд хичээлээ зардаг, суралцагчид нэг дороос суралцдаг монгол платформ."}
	return ldJS(map[string]any{"@context": "https://schema.org", "@graph": []any{site, org}})
}

// pageTitle — хайлтын үр дүнгийн гарчиг: «… | surgalt.mn» (брэнд нь CTR-т тусална).
func pageTitle(parts ...string) string {
	var keep []string
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, " — ") + " | surgalt.mn"
}
