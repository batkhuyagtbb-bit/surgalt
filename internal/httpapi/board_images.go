package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// Цагаан самбарын зураг хайлт. Google зургийг хуудсанд суулгах (iframe) боломжгүй, зураг хайх API нь
// шинэ хэрэглэгчид хаалттай тул нээлттэй сангуудаас хайна: Wikimedia Commons (түлхүүргүй, сургалтын
// зураг, схем ихтэй) + Pixabay (PIXABAY_KEY байвал, SafeSearch). Openverse шинэ хайлтад 30+ сек болдог тул ашиглахгүй.
// Монгол үгийг англи руу хөрвүүлж (жижиг толь → MyMemory) хайж, үр дүнг 6 цаг кэшлэнэ.
const (
	imgCacheTTL  = 6 * time.Hour
	imgCacheMax  = 3000
	imgPerSource = 20
	imgUA        = "surgalt.mn-board/1 (+https://surgalt.mn)"
)

type imgHit struct {
	Thumb  string `json:"thumb"`
	URL    string `json:"url"` // самбарт оруулах хувилбар (≈1280px хүртэл)
	W      int    `json:"w"`
	H      int    `json:"h"`
	Title  string `json:"title,omitempty"`
	Credit string `json:"credit,omitempty"`
	Page   string `json:"page,omitempty"`
	Src    string `json:"src"`
}

type imgResult struct {
	hits []imgHit
	more bool
	at   time.Time
}

type imgSearch struct {
	mu    sync.Mutex
	cache map[string]imgResult
	tr    map[string]string
}

var imgHTTP = &http.Client{Timeout: 9 * time.Second}

// Түгээмэл хичээлийн хэллэгүүд (орчуулагч андуурдаг).
var mnImgDict = map[string]string{
	"нарны аймаг": "solar system", "дэлхий": "earth planet", "сар": "moon", "нар": "sun", "од": "star", "гариг": "planet",
	"хүний бие": "human body", "араг яс": "skeleton", "зүрх": "heart anatomy", "тархи": "brain", "шүд": "tooth",
	"ургамал": "plant", "мод": "tree", "цэцэг": "flower", "навч": "leaf", "үндэс": "plant root", "эс": "cell biology",
	"амьтан": "animal", "нохой": "dog", "муур": "cat", "морь": "horse", "үхэр": "cow", "хонь": "sheep", "ямаа": "goat", "тэмээ": "camel",
	"шувуу": "bird", "загас": "fish", "эрвээхэй": "butterfly", "зөгий": "bee", "мэлхий": "frog", "үнэг": "fox", "чоно": "wolf", "баавгай": "bear",
	"гэр": "mongolian ger yurt", "монгол гэр": "mongolian ger yurt", "уул": "mountain", "гол": "river", "нуур": "lake", "далай": "sea ocean", "цөл": "desert",
	"бороо": "rain", "цас": "snow", "салхи": "wind", "үүл": "cloud", "солонго": "rainbow", "аянга": "lightning",
	"машин": "car", "автобус": "bus", "онгоц": "airplane", "галт тэрэг": "train", "дугуй": "bicycle", "пуужин": "rocket",
	"ном": "book", "сургууль": "school", "багш": "teacher", "хүүхэд": "children", "гэр бүл": "family",
	"газрын зураг": "map", "монгол улс": "mongolia", "туг": "flag", "монголын туг": "flag of mongolia", "дэлхийн бөмбөрцөг": "globe",
	"гурвалжин": "triangle", "тойрог": "circle", "дөрвөлжин": "square", "тоо": "numbers", "цаг": "clock",
	"алим": "apple", "жимс": "fruit", "хүнсний ногоо": "vegetables", "талх": "bread", "сүү": "milk",
	"галт уул": "volcano", "үлэг гүрвэл": "dinosaur", "соронз": "magnet", "цахилгаан": "electricity", "ус": "water", "гал": "fire",
}

var (
	cyrRe   = regexp.MustCompile(`\p{Cyrillic}`)
	tagRe   = regexp.MustCompile(`<[^>]*>`)
	thumbRe = regexp.MustCompile(`/\d+px-`)
)

// imgQuery — монгол хайлтыг англи болгоно (кэштэй). Орчуулж чадахгүй бол эх хэлбэрээр нь.
func (s *Server) imgQuery(ctx context.Context, q string) string {
	if !cyrRe.MatchString(q) {
		return q
	}
	lq := strings.ToLower(strings.Join(strings.Fields(q), " "))
	if v, ok := mnImgDict[lq]; ok {
		return v
	}
	x := &s.imgs
	x.mu.Lock()
	if v, ok := x.tr[lq]; ok {
		x.mu.Unlock()
		return v
	}
	x.mu.Unlock()
	u := "https://api.mymemory.translated.net/get?langpair=mn%7Cen&q=" + url.QueryEscape(lq)
	var out struct {
		Data struct {
			Text string `json:"translatedText"`
		} `json:"responseData"`
		Status any `json:"responseStatus"`
	}
	if err := imgGet(ctx, u, &out); err != nil {
		return q
	}
	t := strings.ToLower(strings.TrimSpace(html.UnescapeString(out.Data.Text)))
	if t == "" || strings.Contains(t, "mymemory") || cyrRe.MatchString(t) || utf8.RuneCountInString(t) > 120 {
		return q
	}
	x.mu.Lock()
	if x.tr == nil || len(x.tr) > 5000 {
		x.tr = map[string]string{}
	}
	x.tr[lq] = t
	x.mu.Unlock()
	return t
}

func imgGet(ctx context.Context, u string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", imgUA)
	req.Header.Set("Accept", "application/json")
	res, err := imgHTTP.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", res.StatusCode)
	}
	return json.NewDecoder(http.MaxBytesReader(nil, res.Body, 4<<20)).Decode(dst)
}

// GET /api/board/images?q=&kind=(|clip|photo)&page= — самбарт оруулах зураг хайх.
func (s *Server) handleBoardImages(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if !s.searchLim.Allow(c.UID) {
		w.Header().Set("Retry-After", "2")
		writeErr(w, http.StatusTooManyRequests, "хэт олон хайлт, түр хүлээнэ үү")
		return
	}
	q := strings.Join(strings.Fields(r.URL.Query().Get("q")), " ")
	if q == "" || utf8.RuneCountInString(q) > 100 || strings.IndexFunc(q, func(r rune) bool { return unicode.IsControl(r) }) >= 0 {
		writeErr(w, http.StatusBadRequest, "хайх үгээ бичнэ үү (100 тэмдэгт хүртэл)")
		return
	}
	kind := r.URL.Query().Get("kind")
	if kind != "clip" && kind != "photo" {
		kind = ""
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	page = min(max(page, 1), 12)
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	en := s.imgQuery(ctx, q)
	key := kind + "|" + strconv.Itoa(page) + "|" + strings.ToLower(en)
	x := &s.imgs
	x.mu.Lock()
	if v, ok := x.cache[key]; ok && time.Since(v.at) < imgCacheTTL {
		x.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]any{"q": q, "en": en, "items": v.hits, "more": v.more, "page": page})
		return
	}
	x.mu.Unlock()

	type part struct {
		hits []imgHit
		more bool
		err  error
	}
	var srcs []func() part
	if s.cfg.PixabayKey != "" {
		srcs = append(srcs, func() part { h, m, err := s.pixabay(ctx, en, kind, page); return part{h, m, err} })
	}
	srcs = append(srcs, func() part { h, m, err := wikimedia(ctx, en, kind, page); return part{h, m, err} })
	parts := make([]part, len(srcs))
	var wg sync.WaitGroup
	for i, f := range srcs {
		wg.Add(1)
		go func() { defer wg.Done(); parts[i] = f() }()
	}
	wg.Wait()
	var hits []imgHit
	more, failed := false, 0
	for i := 0; ; i++ { // эх сурвалжуудыг ээлжлэн холино
		added := false
		for _, p := range parts {
			if i < len(p.hits) {
				hits, added = append(hits, p.hits[i]), true
			}
		}
		if !added {
			break
		}
	}
	for _, p := range parts {
		more = more || p.more
		if p.err != nil {
			failed++
			s.log.Warn("зураг хайлт", "err", p.err)
		}
	}
	if failed == len(parts) {
		writeErr(w, http.StatusBadGateway, "зургийн сантай холбогдож чадсангүй — дахин оролдоно уу")
		return
	}
	if hits == nil {
		hits = []imgHit{}
	}
	x.mu.Lock()
	if x.cache == nil || len(x.cache) >= imgCacheMax {
		x.cache = map[string]imgResult{}
	}
	x.cache[key] = imgResult{hits, more, time.Now()}
	x.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"q": q, "en": en, "items": hits, "more": more, "page": page})
}

func (s *Server) pixabay(ctx context.Context, q, kind string, page int) ([]imgHit, bool, error) {
	typ := map[string]string{"": "all", "clip": "illustration", "photo": "photo"}[kind]
	u := fmt.Sprintf("https://pixabay.com/api/?key=%s&q=%s&image_type=%s&safesearch=true&per_page=%d&page=%d&lang=en",
		url.QueryEscape(s.cfg.PixabayKey), url.QueryEscape(q), typ, imgPerSource, page)
	var out struct {
		TotalHits int `json:"totalHits"`
		Hits      []struct {
			Preview string `json:"webformatURL"`
			Large   string `json:"largeImageURL"`
			W       int    `json:"imageWidth"`
			H       int    `json:"imageHeight"`
			Tags    string `json:"tags"`
			User    string `json:"user"`
			Page    string `json:"pageURL"`
		} `json:"hits"`
	}
	if err := imgGet(ctx, u, &out); err != nil {
		return nil, false, err
	}
	hits := make([]imgHit, 0, len(out.Hits))
	for _, h := range out.Hits {
		hits = append(hits, imgHit{Thumb: h.Preview, URL: h.Large, W: h.W, H: h.H, Title: h.Tags, Credit: h.User + " · Pixabay", Page: h.Page, Src: "pixabay"})
	}
	return hits, page*imgPerSource < min(out.TotalHits, 500), nil
}

// Хүүхдийн хичээлд тохирохгүй үр дүнг Wikimedia-ийн хайлтаас хасна.
const wikiExclude = " -nude -naked -nudity -erotic -erotica -sex -sexual -porn -pornography -genitalia -genital -penis -vagina -vulva -breasts -topless -lingerie -bdsm -fetish -corpse -gore"

func wikimedia(ctx context.Context, q, kind string, page int) ([]imgHit, bool, error) {
	ft := map[string]string{"": "filetype:bitmap|drawing", "clip": "filetype:drawing", "photo": "filetype:bitmap"}[kind]
	v := url.Values{}
	v.Set("action", "query")
	v.Set("format", "json")
	v.Set("formatversion", "2")
	v.Set("generator", "search")
	v.Set("gsrnamespace", "6")
	v.Set("gsrsearch", q+" "+ft+wikiExclude)
	v.Set("gsrlimit", strconv.Itoa(imgPerSource))
	v.Set("gsroffset", strconv.Itoa((page-1)*imgPerSource))
	v.Set("prop", "imageinfo")
	v.Set("iiprop", "url|size|mime|extmetadata")
	v.Set("iiextmetadatafilter", "Artist|LicenseShortName")
	v.Set("iiurlwidth", "330")
	var out struct {
		Continue *struct {
			Offset int `json:"gsroffset"`
		} `json:"continue"`
		Query struct {
			Pages []struct {
				Index int    `json:"index"`
				Title string `json:"title"`
				Info  []struct {
					URL      string `json:"url"`
					Thumb    string `json:"thumburl"`
					W        int    `json:"width"`
					H        int    `json:"height"`
					Mime     string `json:"mime"`
					Page     string `json:"descriptionurl"`
					Metadata map[string]struct {
						Value string `json:"value"`
					} `json:"extmetadata"`
				} `json:"imageinfo"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := imgGet(ctx, "https://commons.wikimedia.org/w/api.php?"+v.Encode(), &out); err != nil {
		return nil, false, err
	}
	pages := out.Query.Pages
	for i := 1; i < len(pages); i++ { // хайлтын эрэмбээр
		for j := i; j > 0 && pages[j].Index < pages[j-1].Index; j-- {
			pages[j], pages[j-1] = pages[j-1], pages[j]
		}
	}
	hits := make([]imgHit, 0, len(pages))
	for _, p := range pages {
		if len(p.Info) == 0 {
			continue
		}
		in := p.Info[0]
		if in.Thumb == "" || !strings.HasPrefix(in.Mime, "image/") || in.Mime == "image/tiff" || in.Mime == "image/vnd.djvu" {
			continue
		}
		thumb := strings.SplitN(in.Thumb, "?", 2)[0]
		full, w, h := strings.SplitN(in.URL, "?", 2)[0], in.W, in.H
		if in.Mime == "image/svg+xml" || in.W > 1280 { // 1280px-ийн бэлэн хувилбар (SVG → PNG)
			full = thumbRe.ReplaceAllString(thumb, "/1280px-")
			if in.W > 0 {
				w, h = 1280, in.H*1280/in.W
			}
		}
		title := strings.TrimPrefix(p.Title, "File:")
		if i := strings.LastIndexByte(title, '.'); i > 0 {
			title = title[:i]
		}
		artist := strings.Join(strings.Fields(html.UnescapeString(tagRe.ReplaceAllString(in.Metadata["Artist"].Value, " "))), " ")
		if utf8.RuneCountInString(artist) > 60 {
			artist = string([]rune(artist)[:60]) + "…"
		}
		cr := strings.Trim(artist+" · "+in.Metadata["LicenseShortName"].Value+" · Wikimedia", " ·")
		hits = append(hits, imgHit{Thumb: thumb, URL: full, W: w, H: h, Title: title, Credit: cr, Page: in.Page, Src: "wikimedia"})
	}
	return hits, out.Continue != nil, nil
}
