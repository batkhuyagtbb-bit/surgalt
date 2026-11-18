package httpapi

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"surgalt/internal/store"
)

// «Үүлэн үг» (word cloud): хичээл дэх асуултад суралцагч бүр 1-3 үг илгээж, бүгдийнх нь хариулт
// үүл болж харагдана (олон хүн бичсэн үг томорно). Багш үр дүнг харна, өөрөө хариулахгүй.
const (
	cloudWordMax = 30  // нэг үгийн урт (тэмдэгт)
	cloudTop     = 100 // үүлэнд харуулах үгийн дээд тоо
)

type cloudWord struct {
	W string `json:"w"`
	N int    `json:"n"`
}

// cloudBlock — хичээлийн тухайн үүлэн үгийн хэсэг.
func cloudBlock(l *store.Lesson, bid string) *store.Block {
	for i := range l.Blocks {
		if l.Blocks[i].ID == bid && l.Blocks[i].Type == "cloud" {
			return &l.Blocks[i]
		}
	}
	return nil
}

// normCloudWord — нэг үгийг нэгтгэнэ: жижиг үсэг, илүү зай ба захын тэмдэгтгүй.
func normCloudWord(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	s = strings.TrimFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	if utf8.RuneCountInString(s) > cloudWordMax {
		return ""
	}
	return s
}

// cloudAgg — нэг үүлэн үгийн нэгтгэл. Хичээл дээр байгаа бүх суралцагч 5 секунд тутам асуудаг тул 2 секунд
// кэшилж хуваалцана (хүн бүрт бүх хариултыг сангаас уншихгүй); шинэ хариулт ирэхэд шууд шинэчлэгдэнэ.
type cloudAgg struct {
	words []cloudWord
	all   map[string][]string // хэрэглэгч → үгс («миний хариулт»-д)
}

func cloudKey(lessonID, bid string) string { return lessonID + "|" + bid }

func (s *Server) cloudResult(w http.ResponseWriter, r *http.Request, lessonID, bid, uid string, max int) {
	agg, err := s.cloudAgg(r.Context(), lessonID, bid)
	if s.storeErr(w, r, err) {
		return
	}
	mine := agg.all[uid]
	if mine == nil {
		mine = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"words": agg.words, "mine": mine, "people": len(agg.all), "max": max})
}

// cloudAgg — үүлэн үгийн нэгтгэл (2 секунд кэштэй; хариулт ирэхэд шинэчлэгдэнэ).
func (s *Server) cloudAgg(ctx context.Context, lessonID, bid string) (*cloudAgg, error) {
	return s.clouds.GetOrLoad(ctx, cloudKey(lessonID, bid), func(ctx context.Context) (*cloudAgg, error) {
		all, err := s.store.CloudWords(ctx, lessonID, bid)
		if err != nil {
			return nil, err
		}
		counts := map[string]int{}
		for _, ws := range all {
			for _, x := range ws {
				counts[x]++
			}
		}
		words := make([]cloudWord, 0, len(counts))
		for k, n := range counts {
			words = append(words, cloudWord{k, n})
		}
		sort.Slice(words, func(i, j int) bool {
			if words[i].N != words[j].N {
				return words[i].N > words[j].N
			}
			return words[i].W < words[j].W
		})
		if len(words) > cloudTop {
			words = words[:cloudTop]
		}
		return &cloudAgg{words: words, all: all}, nil
	})
}

func (s *Server) handleGetCloud(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	_, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	b := cloudBlock(l, r.PathValue("bid"))
	if b == nil {
		writeErr(w, http.StatusNotFound, "үүлэн үгийн хэсэг олдсонгүй")
		return
	}
	s.cloudResult(w, r, l.ID, b.ID, c.UID, max(b.Max, 1))
}

func (s *Server) handleSendCloud(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	b := cloudBlock(l, r.PathValue("bid"))
	if b == nil {
		writeErr(w, http.StatusNotFound, "үүлэн үгийн хэсэг олдсонгүй")
		return
	}
	if teaches(course, c.UID) {
		writeErr(w, http.StatusBadRequest, "багш өөрийн хичээлийн үүлэн үгэнд хариулахгүй, зөвхөн үр дүнг нь харна")
		return
	}
	var in struct {
		Words []string `json:"words"`
	}
	if !decode(w, r, &in) {
		return
	}
	limit := max(b.Max, 1)
	words, seen := []string{}, map[string]bool{}
	for _, raw := range in.Words {
		if x := normCloudWord(raw); x != "" && !seen[x] {
			seen[x] = true
			words = append(words, x)
		}
	}
	if len(words) == 0 {
		writeErr(w, http.StatusBadRequest, "үгээ бичнэ үү (30 хүртэл тэмдэгт)")
		return
	}
	if len(words) > limit {
		words = words[:limit]
	}
	if err := s.store.SetCloudWords(r.Context(), l.ID, b.ID, c.UID, words); s.storeErr(w, r, err) {
		return
	}
	s.clouds.Delete(cloudKey(l.ID, b.ID))
	s.cloudResult(w, r, l.ID, b.ID, c.UID, limit)
	s.cloudChanged(l.ID, b.ID) // хичээлийг харж буй бусдад WebSocket-оор
}
