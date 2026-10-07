package httpapi

import (
	"context"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"surgalt/internal/store"
)

// Сургалтын хайлт: санах ой дахь хөнгөн индекс.
//   - Кирилл ↔ латин: "matematik", "математик" хоёул олдоно (галиг + урт эгшиг, о/у, х/kh нэгтгэнэ).
//   - Угтвар ("мат" → "математик"), дэд мөр, 1-2 үсгийн алдаа (Левенштейн) тэсвэрлэнэ.
//   - Талбарын жин: гарчиг > багшийн нэр > хичээл, бүлэг, шошго > тайлбар; бага зэрэг алдартай нь дээр.
//   - Индекс 2 минут тутам (өөрчлөлт орвол 5 секундийн дараа) шинэчлэгдэнэ; хайлт нь DB-д хүрэхгүй.

const (
	searchPageSize   = 30
	searchMaxCourses = 50_000
	searchTTL        = 2 * time.Minute
	searchDirtyAfter = 5 * time.Second
)

type searchField struct {
	toks   []string
	weight float64
}

type searchDoc struct {
	course  store.Course
	teacher PublicTeacher
	tags    []string // шүүлтүүрийн түлхүүр: free, paid, cert, exam, freelessons
	labels  []string // харагдах шошго
	fields  []searchField
	lessons []string   // хичээл, бүлгийн гарчиг (таарсныг харуулах)
	lessTok [][]string // хичээл тус бүрийн токен
	boost   float64
}

type searchIndex struct {
	mu    sync.Mutex
	docs  []*searchDoc
	built time.Time
	dirty time.Time // хамгийн сүүлд өөрчлөгдсөн
}

// markDirty — сургалт/хичээл өөрчлөгдөхөд дуудна (хөнгөн).
func (x *searchIndex) markDirty() {
	x.mu.Lock()
	x.dirty = time.Now()
	x.mu.Unlock()
}

func (s *Server) searchDocs(ctx context.Context) ([]*searchDoc, error) {
	x := s.search
	x.mu.Lock()
	defer x.mu.Unlock()
	stale := x.docs == nil || time.Since(x.built) > searchTTL || (x.dirty.After(x.built) && time.Since(x.built) > searchDirtyAfter)
	if !stale {
		return x.docs, nil
	}
	docs, err := s.buildSearch(ctx)
	if err != nil {
		if x.docs != nil { // хуучнаараа үргэлжлүүлнэ
			s.log.Warn("search rebuild", "err", err)
			return x.docs, nil
		}
		return nil, err
	}
	x.docs, x.built = docs, time.Now()
	return docs, nil
}

func (s *Server) buildSearch(ctx context.Context) ([]*searchDoc, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	courses, err := s.store.PublishedCourses(ctx, searchMaxCourses)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(courses))
	tids := map[string]bool{}
	for i, c := range courses {
		ids[i] = c.ID
		tids[c.TeacherID] = true
	}
	lessons := map[string][]store.Lesson{}
	for start := 0; start < len(ids); start += 500 { // том $in-ээс зайлсхийнэ
		ls, err := s.store.LessonOutlines(ctx, ids[start:min(start+500, len(ids))])
		if err != nil {
			return nil, err
		}
		for _, l := range ls {
			if l.Hidden { // багш хаасан (бэлтгэж дуусаагүй) хичээл хайлтад гарахгүй
				continue
			}
			lessons[l.CourseID] = append(lessons[l.CourseID], l)
		}
	}
	tlist := make([]string, 0, len(tids))
	for id := range tids {
		tlist = append(tlist, id)
	}
	users, err := s.store.UsersByIDs(ctx, tlist)
	if err != nil {
		return nil, err
	}
	teachers := map[string]*store.User{}
	for i := range users {
		teachers[users[i].ID] = &users[i]
	}
	docs := make([]*searchDoc, 0, len(courses))
	for _, c := range courses {
		u := teachers[c.TeacherID]
		if u == nil {
			continue
		}
		d := &searchDoc{course: c, teacher: publicTeacher(u)}
		d.course.Description = ""
		ls := lessons[c.ID]
		free, paid, exam := 0, 0, false
		kinds := map[string]bool{}
		var lessonText, sections []string
		seenSec := map[string]bool{}
		for _, l := range ls {
			if l.IsFree {
				free++
			} else {
				paid++
			}
			if l.Exam != nil {
				exam = true
			}
			for _, k := range []string{store.LessonFormats[l.Format], store.LessonModes[l.Mode]} {
				if k != "" {
					kinds[k] = true
				}
			}
			d.lessons = append(d.lessons, l.Title)
			d.lessTok = append(d.lessTok, tokens(l.Title))
			lessonText = append(lessonText, l.Title)
			if l.Section != "" && !seenSec[l.Section] {
				seenSec[l.Section] = true
				sections = append(sections, l.Section)
			}
		}
		d.course.LessonCount, d.course.FreeLessonCount = len(ls), free // тоолуурыг бодит хичээлээс
		switch {
		case c.Price == 0 && paid == 0:
			d.tags, d.labels = append(d.tags, "free"), append(d.labels, "Үнэгүй")
		default:
			d.tags, d.labels = append(d.tags, "paid"), append(d.labels, "Төлбөртэй")
			if free > 0 {
				d.tags, d.labels = append(d.tags, "freelessons"), append(d.labels, strconv.Itoa(free)+" үнэгүй хичээл")
			}
		}
		if c.Certificate {
			d.tags, d.labels = append(d.tags, "cert"), append(d.labels, "Сертификаттай")
		}
		if exam {
			d.tags, d.labels = append(d.tags, "exam"), append(d.labels, "Шалгалттай")
		}
		kl := make([]string, 0, len(kinds))
		for k := range kinds {
			kl = append(kl, k)
		}
		slices.Sort(kl)
		kindText := strings.Join(kl, " ") // хайлтад бүгд, харагдах шошгонд 3 хүртэл
		d.labels = append(d.labels, kl[:min(len(kl), 3)]...)
		desc := c.Description
		if utf8.RuneCountInString(desc) > 600 {
			desc = string([]rune(desc)[:600])
		}
		d.fields = []searchField{
			{tokens(c.Title), 5},
			{tokens(u.DisplayName + " " + u.Username), 4},
			{tokens(strings.Join(sections, " ") + " " + kindText + " " + strings.Join(d.labels, " ")), 2.5},
			{tokens(strings.Join(lessonText, " ")), 2},
			{tokens(desc), 1}, // багшийн ерөнхий сэдэв бүх сургалтад нь таарахгүйн тулд оруулахгүй
		}
		d.boost = math.Log1p(float64(c.Views))*0.15 + float64(min(free, 5))*0.05
		docs = append(docs, d)
	}
	return docs, nil
}

// ---- хэвийн болгох ----

var cyrLat = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e", 'ж': "j", 'з': "z", 'и': "i", 'й': "i", 'к': "k", 'л': "l",
	'м': "m", 'н': "n", 'о': "o", 'ө': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "o", 'ү': "o", 'ф': "f", 'х': "h", 'ц': "c",
	'ч': "ch", 'ш': "sh", 'щ': "sh", 'ъ': "", 'ы': "i", 'ь': "i", 'э': "e", 'ю': "io", 'я': "ia",
}

// skeleton: кирилл/латин хоёулаа нэг "араг" болно — "Өнөөдөр", "onoodor", "unuudur" → "onodor".
func skeleton(w string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(w) {
		if l, ok := cyrLat[r]; ok {
			b.WriteString(l)
			continue
		}
		switch r {
		case 'u', 'ü', 'ö':
			r = 'o'
		case 'y':
			r = 'i'
		case 'w':
			r = 'v'
		case 'q':
			r = 'k'
		case 'x':
			r = 'h'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	s := strings.NewReplacer("kh", "h", "ts", "c", "yu", "io", "ya", "ia", "ye", "e").Replace(b.String())
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ { // давхар үсгийг нэгтгэнэ (урт эгшиг)
		if i > 0 && s[i] == s[i-1] {
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
}

func tokens(text string) []string {
	f := strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	out := make([]string, 0, len(f))
	seen := map[string]bool{}
	for _, w := range f {
		k := skeleton(w)
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

// lev — Левенштейний зай (max-аас хэтэрвэл эрт зогсоно).
func lev(a, b string, max int) int {
	if d := len(a) - len(b); d > max || -d > max {
		return max + 1
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		best := cur[0]
		for j := 1; j <= len(b); j++ {
			c := 1
			if a[i-1] == b[j-1] {
				c = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+c)
			best = min(best, cur[j])
		}
		if best > max {
			return max + 1
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// tokenScore — асуулгын нэг үг нэг баримтын үгтэй хэр таарч байна (0 = таараагүй).
func tokenScore(q, t string) float64 {
	switch {
	case q == t:
		return 1
	case len(q) >= 2 && strings.HasPrefix(t, q):
		return 0.8
	case len(q) >= 4 && strings.Contains(t, q):
		return 0.5
	}
	if len(q) >= 4 {
		max := 1
		if len(q) >= 8 {
			max = 2
		}
		// угтвараар нь (бичиж байгаа үг) эсвэл бүтэн үгээр алдаа тэсвэрлэнэ
		if len(t) > len(q) && lev(q, t[:len(q)], max) <= max {
			return 0.45
		}
		if lev(q, t, max) <= max {
			return 0.55
		}
	}
	return 0
}

func bestIn(q string, toks []string) float64 {
	best := 0.0
	for _, t := range toks {
		if s := tokenScore(q, t); s > best {
			best = s
			if s == 1 {
				break
			}
		}
	}
	return best
}

// SearchHit — хайлтын нэг үр дүн.
type SearchHit struct {
	CourseID    string        `json:"course_id"`
	Title       string        `json:"title"`
	Teacher     PublicTeacher `json:"teacher"`
	Price       int64         `json:"price"`
	Lessons     int           `json:"lessons"`
	FreeLessons int           `json:"free_lessons"`
	Views       int64         `json:"views"`
	Tags        []string      `json:"tags"`
	Match       string        `json:"match,omitempty"` // таарсан хичээлийн гарчиг
	score       float64
}

// handleSearch: GET /api/search?q=&tag=free|paid|cert|exam&page=1
// SearchResult — хайлтын нэг хуудас.
type SearchResult struct {
	Items   []SearchHit `json:"items"`
	Total   int         `json:"total"`
	Page    int         `json:"page"`
	Pages   int         `json:"pages"`
	PerPage int         `json:"per_page"`
}

// SearchCourses нь нийтлэгдсэн сургалтуудаас хайна (HTTP ба gRPC хоёулаа үүнийг дуудна).
func (s *Server) SearchCourses(ctx context.Context, query, tag string, page int) (*SearchResult, error) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > 100 {
		query = string([]rune(query)[:100])
	}
	page = max(page, 1)
	docs, err := s.searchDocs(ctx)
	if err != nil {
		return nil, err
	}
	qt := tokens(query)
	if len(qt) > 8 {
		qt = qt[:8]
	}
	var hits []SearchHit
	for _, d := range docs {
		if tag != "" && !slices.Contains(d.tags, tag) {
			continue
		}
		score, matched := 0.0, 0
		for _, t := range qt {
			best := 0.0
			for _, f := range d.fields {
				if v := bestIn(t, f.toks) * f.weight; v > best {
					best = v
				}
			}
			if best > 0 {
				matched++
				score += best
			}
		}
		if len(qt) > 0 {
			if matched == 0 {
				continue
			}
			// Бүх үг таарвал илүү: хагас таарвал оноо огцом буурна.
			frac := float64(matched) / float64(len(qt))
			if len(qt) > 1 && frac < 0.5 {
				continue
			}
			score *= frac * frac
		}
		score += d.boost
		h := SearchHit{CourseID: d.course.ID, Title: d.course.Title, Teacher: d.teacher, Price: d.course.Price, Lessons: d.course.LessonCount,
			FreeLessons: d.course.FreeLessonCount, Views: d.course.Views, Tags: d.labels, score: score}
		if len(qt) > 0 { // аль хичээл хамгийн их таарсныг харуулна (гарчигт таараагүй үед)
			if bestIn(qt[0], d.fields[0].toks) == 0 || len(qt) > 1 {
				bi, bs := -1, 0.0
				for i, lt := range d.lessTok {
					sc := 0.0
					for _, t := range qt {
						sc += bestIn(t, lt)
					}
					if sc > bs {
						bi, bs = i, sc
					}
				}
				if bi >= 0 && bs >= 0.8 {
					h.Match = d.lessons[bi]
				}
			}
		}
		hits = append(hits, h)
	}
	slices.SortFunc(hits, func(a, b SearchHit) int {
		if a.score != b.score {
			if a.score > b.score {
				return -1
			}
			return 1
		}
		return strings.Compare(a.CourseID, b.CourseID)
	})
	total := len(hits)
	pages := max(1, (total+searchPageSize-1)/searchPageSize)
	page = min(page, pages)
	from := (page - 1) * searchPageSize
	items := hits[from:min(from+searchPageSize, total)]
	if items == nil {
		items = []SearchHit{}
	}
	return &SearchResult{Items: items, Total: total, Page: page, Pages: pages, PerPage: searchPageSize}, nil
}

// handleSearch: GET /api/search?q=&tag=free|paid|cert|exam&page=1
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	res, err := s.SearchCourses(r.Context(), q.Get("q"), q.Get("tag"), page)
	if s.storeErr(w, r, err) {
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=15")
	writeJSON(w, http.StatusOK, res)
}
