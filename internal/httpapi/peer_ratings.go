package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/store"
)

// ---- Бүлэг чатын харилцан үнэлгээ (peer ratings) ----
// Сургалтын бүлэг чатад суралцагч ангийн найз бүрээ «хамтарч ажилласан» оноо 0–100-аар үнэлнэ — ерөнхийд нь эсвэл
// хичээл бүрээр (дурын үед шинэчилж, сүүлийнх нь үлдэнэ; өөртөө өгөхгүй). Хэн хэдийг өгсөн нь зөвхөн багшид.
//
// Хичээл бүрээр: сургалтын (нийт) оноо хосоор — (үнэлэгч → үнэлэгдэгч) хосын оноо нь тэр үнэлэгчийн өгсөн бүх үнэлгээний
// (ерөнхий + хичээлүүд) дундаж, тоо/босго/Bayes нь давтагдашгүй үнэлэгчээр (олон хичээл үнэлснээр илүү жин авахгүй).
// Хичээлийн оноо — зөвхөн тэр хичээлийн үнэлгээгээр, ижил дүрмээр.
//
// Үнэлэгчийн нууцлал (суралцагчийн хариу): тоо (count) огт өгөхгүй; харагдах оноо нь өнөөдөр 00:00-оос (Улаанбаатар)
// өмнөх тоологдох үнэлгээний Bayes оноо, 5-д бүхэлчилсэн — өдөрт нэг л өөрчлөгдөнө (шууд бус). Босго: үзэгчээс бусад
// peerMinRaters+ тоологдох үнэлэгч. Бусдын оноо зөвхөн босго хангаж, peerShowMin-ээс доошгүй үед («цөөн», «бага»
// ялгагдахгүй); өөрийн оноо босго хангавал хэдэн ч байсан. PUT ч мөн snapshot-ийг буцаана.
//
// Тоологдох үнэлгээ: үнэлэгч ба үнэлэгдэгч хоёулаа одоо тухайн сургалтын суралцагч (элссэн / хичээл авсан, багш биш,
// журналд архивлаагүй — peerClass) БА үнэлэгч сургалтад peerSeasoning-ээс өмнө орсон, сургалтад суралцах сесс эсвэл
// бүлэг чатад мессежтэй (бөөнөөр шинэ бүртгэлээр дайрахаас). Бусад нь хадгалагдаж, багшид counted:false-аар харагдана.
// Бага оноо давтагдвал багш нарт peer_alert мэдэгдэл (өдөрт нэг).
//
// Ачаалал: сургалтын үнэлгээ + үнэлэгчдийн баримт peerCacheTTL кэштэй (PUT/DELETE-д хүчингүй болно; аналитик ч
// ашиглана); суралцагчдыг багшийн кэштэй жагсаалтаас (roster.go), журналын мэдээллийг хүсэлт бүрт нэг удаа уншина.

const (
	peerMinRaters  = 3                // суралцагчдад оноо харагдах хамгийн цөөн (үзэгчээс бусад) тоологдох үнэлэгч
	peerPrior      = 3                // хамтын ажиллагааны рейтинг: ангийн дунджийг хэдэн үнэлгээ гэж тооцох (Bayes)
	peerNoRatings  = 70               // сургалтад тоологдох үнэлгээ огт байхгүй үеийн ангийн дундаж
	peerMaxMembers = 500              // хариунд орох суралцагчийн дээд тоо
	peerShowMin    = 50               // бусдын оноо үүнээс бага бол суралцагчид null
	peerRound      = 5                // суралцагчид харагдах оноог бүхэлчлэх алхам
	peerSeasoning  = 72 * time.Hour   // үнэлэгч сургалтад орсноос хойш тоологдох хүртэлх хугацаа
	peerLow        = 40               // анхааруулгын «бага» оноо (үүнээс бага)
	peerLowRecv    = 2                // нэг хүүхэд ийм олон тоологдох бага оноо авбал багшид мэдэгдэнэ
	peerLowGiven   = 3                // нэг үнэлэгч ийм олон ангийн найздаа бага оноо өгвөл багшид мэдэгдэнэ
	peerCacheTTL   = 25 * time.Second // сургалтын үнэлгээний кэш
	peerAlertType  = "peer_alert"
)

const (
	msgPeerMembers = "бүлэг чатад зөвхөн сургалтад элссэн суралцагчид орно"
	msgPeerClosed  = "харилцан үнэлгээнд оролцох боломжгүй — багштайгаа холбогдоно уу"
	msgPeerTeacher = "багш үнэлгээ өгөхгүй — хэн хэдэн оноо өгснийг харж, зүй бус үнэлгээг устгана"
	msgPeerScore   = "оноо 0-100 хооронд байна"
	msgPeerSelf    = "өөрийгөө үнэлэх боломжгүй"
	msgPeerRatee   = "суралцагч олдсонгүй — сургалтаас гарсан байж магадгүй"
	msgPeerBusy    = "хэт олон удаа үнэлж байна — түр хүлээгээд дахин оролдоно уу"
	msgPeerGone    = "мэдээлэл олдсонгүй — устгагдсан байж магадгүй"
	msgPeerLesson  = "хичээл олдсонгүй"
)

// PeerMember — суралцагчид харагдах ангийн найз: snapshot оноо (босго хангаж, peerShowMin+ үед л), миний өгсөн оноо.
// Хэдэн хүн үнэлсэн, хэн үнэлсэн нь хэзээ ч орохгүй.
type PeerMember struct {
	UserID    string `json:"user_id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Avg       *int   `json:"avg"`
	Mine      *int   `json:"mine"`
}

// PeerMe — суралцагчийн өөрийн snapshot оноо: enough — босго хангасан (тэгвэл avg хэдэн ч байсан харагдана).
type PeerMe struct {
	UserID string `json:"user_id"`
	Avg    *int   `json:"avg"`
	Enough bool   `json:"enough"`
}

// PeerTeacherRow — багшийн хүснэгтийн мөр (бодит, шууд): тоологдох үнэлгээний дундаж/тоо/рейтинг/байр, өгсөн
// (тоологдох эсэхээс үл хамааран) тоо/дундаж, хэн хэдийг өгсөн.
type PeerTeacherRow struct {
	UserID    string      `json:"user_id"`
	Name      string      `json:"name"`
	Username  string      `json:"username"`
	AvatarURL string      `json:"avatar_url"`
	Avg       *int        `json:"avg"`
	Count     int         `json:"count"`
	Given     int         `json:"given"`
	GivenAvg  *int        `json:"given_avg"`
	Collab    *int        `json:"collab"`
	Rank      *int        `json:"rank"`
	Ratings   []PeerGiven `json:"ratings"`
}

// PeerGiven — нэг үнэлгээ (зөвхөн багшийн хариунд); counted — дундаж, рейтингт орсон эсэх; lesson_id "" = ерөнхий.
type PeerGiven struct {
	RaterID     string    `json:"rater_id"`
	RaterName   string    `json:"rater_name"`
	Score       int       `json:"score"`
	At          time.Time `json:"at"`
	Counted     bool      `json:"counted"`
	LessonID    string    `json:"lesson_id"`
	LessonTitle string    `json:"lesson_title"`
}

// PeerLesson — үнэлгээний хичээл сонгогчийн мөр.
type PeerLesson struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ---- кэштэй өгөгдөл ----

// peerLesson — сургалтын хичээлийн товч (хичээл сонгогч, багшийн жагсаалтын нэр).
type peerLesson struct {
	id, title string
	hidden    bool
	pos       int
}

// peerData — сургалтын бүх үнэлгээ, үнэлэгчдийн баримт, хичээлүүд (кэшэд хуваалцана: ЗӨВХӨН УНШИНА).
type peerData struct {
	ratings []store.PeerRating
	facts   map[string]store.RaterFacts
	lessons []peerLesson // дарааллаар
}

// countsAt — t мөчид тоологдох үнэлэгч.
func (d *peerData) countsAt(t time.Time) func(string) bool {
	return func(rater string) bool { return d.facts[rater].CountsAt(t, peerSeasoning) }
}

// with — rating-ийг (тэр хичээлийнх байвал сольж) тусгасан хуулбар; facts, lessons-ийг хуваалцана.
func (d *peerData) with(p store.PeerRating) *peerData {
	out := &peerData{ratings: make([]store.PeerRating, 0, len(d.ratings)+1), facts: d.facts, lessons: d.lessons}
	found := false
	for _, x := range d.ratings {
		if x.RaterID == p.RaterID && x.RateeID == p.RateeID && x.LessonID == p.LessonID {
			x, found = p, true
		}
		out.ratings = append(out.ratings, x)
	}
	if !found {
		out.ratings = append(out.ratings, p)
	}
	return out
}

// titles — хичээлийн нэр ID-гаар (нуусных ч).
func (d *peerData) titles() map[string]string {
	out := make(map[string]string, len(d.lessons))
	for _, l := range d.lessons {
		out[l.id] = l.title
	}
	return out
}

// visibleLessons — дуудагчид харагдах хичээлүүд дарааллаар (багшид нуусан нь ч).
func (d *peerData) visibleLessons(course *store.Course, uid string) []PeerLesson {
	out := make([]PeerLesson, 0, len(d.lessons))
	for _, l := range d.lessons {
		if !l.hidden || teaches(course, uid) {
			out = append(out, PeerLesson{ID: l.id, Title: l.title})
		}
	}
	return out
}

// loadPeers — сургалтын үнэлгээ + үнэлэгчдийн баримт + хичээлүүд (peerCacheTTL кэш; үнэлгээгүй бол баримт уншихгүй).
func (s *Server) loadPeers(ctx context.Context, course *store.Course) (*peerData, error) {
	cid, tid := course.ID, course.TeacherID
	return s.peers.GetOrLoad(ctx, cid, func(ctx context.Context) (*peerData, error) {
		ls, err := s.store.LessonsByCourse(ctx, cid)
		if err != nil {
			return nil, err
		}
		d := &peerData{facts: map[string]store.RaterFacts{}, lessons: make([]peerLesson, len(ls))}
		for i, l := range ls {
			d.lessons[i] = peerLesson{id: l.ID, title: l.Title, hidden: l.Hidden, pos: l.Position}
		}
		sort.SliceStable(d.lessons, func(i, j int) bool { return d.lessons[i].pos < d.lessons[j].pos })
		if d.ratings, err = s.store.PeerRatings(ctx, cid); err != nil {
			return nil, err
		}
		if len(d.ratings) == 0 {
			return d, nil
		}
		seen := map[string]bool{}
		var raters []string
		for _, p := range d.ratings {
			if !seen[p.RaterID] {
				seen[p.RaterID] = true
				raters = append(raters, p.RaterID)
			}
		}
		if d.facts, err = s.store.PeerRaterFacts(ctx, cid, tid, raters); err != nil {
			return nil, err
		}
		return d, nil
	})
}

// ---- тооцоо ----

// peerAgg — нэг суралцагчийн авсан тоологдох хосууд (sum, n) ба өгсөн бүх хосууд (gsum, gn); нийлбэр нь хосын онооны.
type peerAgg struct {
	sum, gsum float64
	n, gn     int
}

type peerEntry struct {
	store.PeerRating
	counted bool
}

// peerPair — нэг үнэлэгчийн нэг хүнд өгсөн үнэлгээнүүд (ерөнхий + хичээлүүд, эсвэл нэг хичээлийнх).
type peerPair struct {
	rater   string
	sum, n  int
	counted bool
}

func (p *peerPair) score() float64 { return float64(p.sum) / float64(p.n) }

// peerBook — нэг сургалтын (эсвэл нэг хичээлийн) үнэлгээний нэгтгэл ангийн гишүүд хооронд.
type peerBook struct {
	class map[string]TeacherBrief // ангийн гишүүд (peerClass)
	agg   map[string]*peerAgg     // uid → нэгтгэл
	pairs map[string][]*peerPair  // үнэлэгдэгч → хосууд
	items map[string][]peerEntry  // үнэлэгдэгч → бүх үнэлгээ (тоологдох эсэхтэй нь; багшийн жагсаалтад)
	total float64                 // тоологдох хосын онооны нийлбэр
	count int                     // тоологдох хосын тоо
}

// newPeerBook — lesson хоосон бол бүх үнэлгээ (сургалтын нийт), эс бөгөөс зөвхөн тэр хичээлийнх; counts(rater) нь
// тоологдох эсэх. before тэг бол шууд (одоогийн, устгаагүй мөрүүд); тэг биш бол тэр хилийн үеийн төлөв (snapshot:
// өнөөдөр засварласан/устгасан үнэлгээний өмнөх утга — store.PeerRating.SnapshotAt).
func newPeerBook(ratings []store.PeerRating, class map[string]TeacherBrief, counts func(string) bool, before time.Time, lesson string) *peerBook {
	b := &peerBook{class: class, agg: map[string]*peerAgg{}, pairs: map[string][]*peerPair{}, items: map[string][]peerEntry{}}
	at := func(uid string) *peerAgg {
		if b.agg[uid] == nil {
			b.agg[uid] = &peerAgg{}
		}
		return b.agg[uid]
	}
	byPair := map[[2]string]*peerPair{}
	for _, p := range ratings {
		if p.RaterID == p.RateeID || (lesson != "" && p.LessonID != lesson) {
			continue
		}
		if before.IsZero() {
			if p.Deleted {
				continue
			}
		} else {
			score, at, ok := p.SnapshotAt(before)
			if !ok {
				continue
			}
			p.Score, p.UpdatedAt = score, at
		}
		if _, ok := class[p.RaterID]; !ok {
			continue
		}
		if _, ok := class[p.RateeID]; !ok {
			continue
		}
		e := peerEntry{PeerRating: p, counted: counts(p.RaterID)}
		b.items[p.RateeID] = append(b.items[p.RateeID], e)
		k := [2]string{p.RaterID, p.RateeID}
		pp := byPair[k]
		if pp == nil {
			pp = &peerPair{rater: p.RaterID, counted: e.counted}
			byPair[k] = pp
			b.pairs[p.RateeID] = append(b.pairs[p.RateeID], pp)
		}
		pp.sum += p.Score
		pp.n++
	}
	for ratee, ps := range b.pairs {
		for _, pp := range ps {
			v := pp.score()
			g := at(pp.rater)
			g.gsum += v
			g.gn++
			if pp.counted {
				a := at(ratee)
				a.sum += v
				a.n++
				b.total += v
				b.count++
			}
		}
	}
	return b
}

func roundInt(v float64) *int {
	x := int(math.Round(v))
	return &x
}

// classAvg — тоологдох бүх хосын онооны дундаж (байхгүй бол peerNoRatings).
func (b *peerBook) classAvg() float64 {
	if b.count == 0 {
		return peerNoRatings
	}
	return b.total / float64(b.count)
}

// avg — тоологдох хосын онооны бүхэл дундаж ба үнэлэгчийн тоо (үнэлгээгүй бол nil, 0).
func (b *peerBook) avg(uid string) (*int, int) {
	a := b.agg[uid]
	if a == nil || a.n == 0 {
		return nil, 0
	}
	return roundInt(a.sum / float64(a.n)), a.n
}

// bayes — round өмнөх рейтинг: (хосын онооны нийлбэр + peerPrior·ангийн дундаж) / (үнэлэгчийн тоо + peerPrior).
func (b *peerBook) bayes(uid string) (float64, bool) {
	a := b.agg[uid]
	if a == nil || a.n == 0 {
		return 0, false
	}
	return (a.sum + peerPrior*b.classAvg()) / float64(a.n+peerPrior), true
}

// collab — хамтын ажиллагааны рейтинг (бүхэл); тоологдох үнэлгээгүй бол nil.
func (b *peerBook) collab(uid string) *int {
	v, ok := b.bayes(uid)
	if !ok {
		return nil
	}
	return roundInt(v)
}

// snap — суралцагчид харагдах оноо: Bayes рейтинг peerRound-д бүхэлчилсэн.
func (b *peerBook) snap(uid string) *int {
	v, ok := b.bayes(uid)
	if !ok {
		return nil
	}
	x := int(math.Round(v/peerRound)) * peerRound
	return &x
}

// enough — uid-ийг viewer-ээс бусад peerMinRaters+ тоологдох (давтагдашгүй) үнэлэгч үнэлсэн.
func (b *peerBook) enough(uid, viewer string) bool {
	n := 0
	for _, pp := range b.pairs[uid] {
		if pp.counted && pp.rater != viewer {
			n++
		}
	}
	return n >= peerMinRaters
}

func briefName(u TeacherBrief) string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}

// member — viewer-т харагдах мөр (snapshot book дээр): бусдын оноо босго хангаж, peerShowMin+ үед л.
func (b *peerBook) member(uid, viewer string, mine map[string]int) PeerMember {
	u := b.class[uid]
	m := PeerMember{UserID: uid, Name: briefName(u), Username: u.Username, AvatarURL: u.AvatarURL}
	if b.enough(uid, viewer) {
		if v := b.snap(uid); v != nil && *v >= peerShowMin {
			m.Avg = v
		}
	}
	if v, ok := mine[uid]; ok {
		m.Mine = &v
	}
	return m
}

// me — суралцагчийн өөрийн snapshot оноо (босго хангавал хэдэн ч байсан).
func (b *peerBook) me(uid string) PeerMe {
	me := PeerMe{UserID: uid, Enough: b.enough(uid, uid)}
	if me.Enough {
		me.Avg = b.snap(uid)
	}
	return me
}

// byName — суралцагчдыг нэрээр (том жижиг үсэг үл харгалзан), дараа нь ID-гаар.
func (b *peerBook) byName(skip string) []string {
	ids := make([]string, 0, len(b.class))
	for id := range b.class {
		if id != skip {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return peerLess(b.class[ids[i]], b.class[ids[j]]) })
	return ids
}

func peerLess(a, c TeacherBrief) bool {
	na, nc := strings.ToLower(briefName(a)), strings.ToLower(briefName(c))
	if na != nc {
		return na < nc
	}
	return a.ID < c.ID
}

// studentView — суралцагчийн хариу (snapshot book): өөрийн оноо, ангийн найзууд (өөрөө орохгүй), миний өгсөн (шууд).
func (b *peerBook) studentView(uid string, mine map[string]int) map[string]any {
	ids := b.byName(uid)
	if len(ids) > peerMaxMembers {
		ids = ids[:peerMaxMembers]
	}
	members := make([]PeerMember, len(ids))
	for i, id := range ids {
		members[i] = b.member(id, uid, mine)
	}
	return map[string]any{"role": "student", "min_raters": peerMinRaters, "me": b.me(uid), "members": members}
}

// teacherView — багшийн хариу (шууд book): рейтингээр буурахаар (үнэлгээгүй нь сүүлд, дараа нь нэрээр), ижил рейтингт
// ижил байр; ratings[] — бүх үнэлгээ хичээлийн нэртэй.
func (b *peerBook) teacherView(titles map[string]string) map[string]any {
	rows := make([]PeerTeacherRow, 0, len(b.class))
	for uid, u := range b.class {
		avg, n := b.avg(uid)
		row := PeerTeacherRow{UserID: uid, Name: briefName(u), Username: u.Username, AvatarURL: u.AvatarURL, Avg: avg, Count: n,
			Collab: b.collab(uid), Ratings: []PeerGiven{}}
		if a := b.agg[uid]; a != nil && a.gn > 0 {
			row.Given, row.GivenAvg = a.gn, roundInt(a.gsum/float64(a.gn))
		}
		for _, e := range b.items[uid] {
			row.Ratings = append(row.Ratings, PeerGiven{RaterID: e.RaterID, RaterName: briefName(b.class[e.RaterID]), Score: e.Score,
				At: e.UpdatedAt, Counted: e.counted, LessonID: e.LessonID, LessonTitle: titles[e.LessonID]})
		}
		sort.Slice(row.Ratings, func(i, j int) bool {
			x, y := row.Ratings[i], row.Ratings[j]
			if !x.At.Equal(y.At) {
				return x.At.After(y.At)
			}
			if x.RaterID != y.RaterID {
				return x.RaterID < y.RaterID
			}
			return x.LessonID < y.LessonID
		})
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		ci, cj := rows[i].Collab, rows[j].Collab
		if (ci == nil) != (cj == nil) {
			return ci != nil
		}
		if ci != nil && *ci != *cj {
			return *ci > *cj
		}
		return peerLess(b.class[rows[i].UserID], b.class[rows[j].UserID])
	})
	for i := range rows {
		if rows[i].Collab == nil {
			break // үнэлгээгүй нь сүүлд
		}
		if i > 0 && *rows[i-1].Collab == *rows[i].Collab {
			r := *rows[i-1].Rank
			rows[i].Rank = &r
		} else {
			r := i + 1
			rows[i].Rank = &r
		}
	}
	if len(rows) > peerMaxMembers {
		rows = rows[:peerMaxMembers]
	}
	var classAvg *int
	if b.count > 0 {
		classAvg = roundInt(b.classAvg())
	}
	return map[string]any{"role": "teacher", "min_raters": peerMinRaters, "class_avg": classAvg, "members": rows}
}

// peerMine — uid-ийн тэр хичээлд ("" = ерөнхий) өгсөн оноо үнэлэгдэгчээр (өөрийн өгөгдөл тул шууд).
func peerMine(ratings []store.PeerRating, uid, lesson string) map[string]int {
	out := map[string]int{}
	for _, p := range ratings {
		if p.RaterID == uid && p.LessonID == lesson && !p.Deleted {
			out[p.RateeID] = p.Score
		}
	}
	return out
}

// peerCutoff — суралцагчийн snapshot-ийн хил: өнөөдөр 00:00 (Улаанбаатар).
func peerCutoff(now time.Time) time.Time {
	l := now.In(mnLoc)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, mnLoc)
}

// studentBook — суралцагчид харагдах snapshot: өнөөдрөөс өмнөх, тэр үед тоологдох байсан үнэлгээ.
func studentBook(d *peerData, class map[string]TeacherBrief, now time.Time, lesson string) *peerBook {
	cut := peerCutoff(now)
	return newPeerBook(d.ratings, class, d.countsAt(cut), cut, lesson)
}

// liveBook — багшид, аналитикт, анхааруулгад: бүх үнэлгээ, одоо тоологдох эсэхээр.
func liveBook(d *peerData, class map[string]TeacherBrief, now time.Time, lesson string) *peerBook {
	return newPeerBook(d.ratings, class, d.countsAt(now), time.Time{}, lesson)
}

// peerClass — сургалтын ангийн гишүүд: багшийн кэштэй жагсаалтаас (roster.go) энэ сургалтынх, багш нар биш, журналд
// архивлаагүй (metas). extra — дуудагч гишүүн гэж шалгасан хүмүүс (жагсаалтын кэш хараахан шинэчлэгдээгүй байж болно).
func (s *Server) peerClass(ctx context.Context, course *store.Course, metas map[string]store.CourseStudentMeta, extra ...string) (map[string]TeacherBrief, error) {
	ro, err := s.teacherRoster(ctx, course.TeacherID)
	if err != nil {
		return nil, err
	}
	ok := func(uid string) bool { return uid != "" && !teaches(course, uid) && !metas[uid].Archived }
	out := map[string]TeacherBrief{}
	for _, uid := range ro.byCourse[course.ID] {
		if i, in := ro.index[uid]; in && ok(uid) {
			out[uid] = ro.rows[i].User
		}
	}
	var missing []string
	for _, uid := range extra {
		if _, in := out[uid]; !in && ok(uid) {
			missing = append(missing, uid)
		}
	}
	if len(missing) > 0 {
		users, err := s.store.UsersByIDs(ctx, missing)
		if err != nil {
			return nil, err
		}
		for _, u := range users {
			out[u.ID] = TeacherBrief{ID: u.ID, Username: u.Username, DisplayName: u.DisplayName, Headline: u.Headline, AvatarURL: u.AvatarURL}
		}
	}
	return out, nil
}

// peerCollab — нэг суралцагчийн хамтын ажиллагааны рейтинг (аналитикт нэртэй нь).
type peerCollab struct {
	Name   string
	Collab int
}

// peerCollabs — сургалтын хамтын ажиллагааны рейтинг (хосоор, бүх хичээлээр) суралцагч бүрээр (тоологдох үнэлгээгүй нь
// орохгүй): аналитикийн «collaboration» хэмжүүрт (analytics.go). Кэштэй өгөгдлөөс; үнэлгээгүй сургалтад жагсаалт,
// журнал уншихгүй.
func (s *Server) peerCollabs(ctx context.Context, course *store.Course) (map[string]peerCollab, error) {
	d, err := s.loadPeers(ctx, course)
	if err != nil || len(d.ratings) == 0 {
		return nil, err
	}
	metas, err := s.store.CourseStudentMetas(ctx, course.ID)
	if err != nil {
		return nil, err
	}
	class, err := s.peerClass(ctx, course, metas)
	if err != nil {
		return nil, err
	}
	b := liveBook(d, class, time.Now(), "")
	out := map[string]peerCollab{}
	for uid := range b.agg {
		if v := b.collab(uid); v != nil {
			out[uid] = peerCollab{Name: briefName(class[uid]), Collab: *v}
		}
	}
	return out, nil
}

// ---- эрх ----

// peerCourse — сургалт, дуудагч багш эсэх, журналын мэдээлэл. Багш (teaches) эсвэл нийтлэгдсэн сургалтын гишүүн
// суралцагч (courseMember); бусад нь 403 (бүлэг чаттай ижил эрх), журналд архивласан суралцагч 403.
func (s *Server) peerCourse(w http.ResponseWriter, r *http.Request) (auth.Claims, *store.Course, bool, map[string]store.CourseStudentMeta, bool) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return c, nil, false, nil, false
	}
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return c, nil, false, nil, false
	}
	teacher, metas, ok := s.peerAccess(w, r, c.UID, course)
	return c, course, teacher, metas, ok
}

// peerAccess — peerCourse-ийн эрхийн хэсэг (сургалт аль хэдийн уншигдсан үед).
func (s *Server) peerAccess(w http.ResponseWriter, r *http.Request, uid string, course *store.Course) (bool, map[string]store.CourseStudentMeta, bool) {
	ctx := r.Context()
	teacher := teaches(course, uid)
	if !teacher {
		member, err := s.courseMember(ctx, uid, course.ID)
		if s.storeErr(w, r, err) {
			return false, nil, false
		}
		if !member || !course.Published {
			writeErr(w, http.StatusForbidden, msgPeerMembers)
			return false, nil, false
		}
	}
	metas, err := s.store.CourseStudentMetas(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return false, nil, false
	}
	if !teacher && metas[uid].Archived {
		writeErr(w, http.StatusForbidden, msgPeerClosed)
		return false, nil, false
	}
	return teacher, metas, true
}

// peerLessonOK — lid хоосон (ерөнхий) эсвэл дуудагчид харагдах энэ сургалтын хичээл.
func peerLessonOK(lessons []PeerLesson, lid string) bool {
	if lid == "" {
		return true
	}
	for _, l := range lessons {
		if l.ID == lid {
			return true
		}
	}
	return false
}

// handlePeerRatings: GET /api/courses/{id}/peer-ratings[?lesson_id=] — суралцагчид өдрийн snapshot оноо л, багшид бүх
// үнэлгээ ба рейтинг (шууд). lesson_id-гүй бол сургалтын нийт (хосоор; суралцагчийн mine — ерөнхий үнэлгээ), өгвөл тэр
// хичээлийнх (mine — тэр хичээлд өгсөн). Хариуд хичээл сонгогчийн жагсаалт (lessons) ба сонгосон lesson_id.
func (s *Server) handlePeerRatings(w http.ResponseWriter, r *http.Request) {
	c, course, teacher, metas, ok := s.peerCourse(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	d, err := s.loadPeers(ctx, course)
	if s.storeErr(w, r, err) {
		return
	}
	lessons, lid := d.visibleLessons(course, c.UID), strings.TrimSpace(r.URL.Query().Get("lesson_id"))
	if !peerLessonOK(lessons, lid) {
		writeErr(w, http.StatusNotFound, msgPeerLesson)
		return
	}
	var extra []string
	if !teacher {
		extra = []string{c.UID}
	}
	class, err := s.peerClass(ctx, course, metas, extra...)
	if s.storeErr(w, r, err) {
		return
	}
	now := time.Now()
	var out map[string]any
	if teacher {
		out = liveBook(d, class, now, lid).teacherView(d.titles())
	} else {
		out = studentBook(d, class, now, lid).studentView(c.UID, peerMine(d.ratings, c.UID, lid))
	}
	out["lessons"], out["lesson_id"] = lessons, lid
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, http.StatusOK, out)
}

// peerScore — бүхэл 0..100 (JSON тоо); бусад бүх утга (байхгүй, null, мөр, бутархай, мужаас гадуур) false.
func peerScore(raw json.RawMessage) (int, bool) {
	var v *float64
	if len(raw) == 0 || json.Unmarshal(raw, &v) != nil || v == nil || *v != math.Trunc(*v) || *v < 0 || *v > 100 {
		return 0, false
	}
	return int(*v), true
}

// peerBlocked — ratee нь rater-ийг хувийн ярианд хориглосон эсэх (moderation.go: хориглосон тал = тэр ярианы үүрэг).
func (s *Server) peerBlocked(ctx context.Context, ratee, rater string) (bool, error) {
	convs, err := s.store.UserDMConversations(ctx, rater, 500)
	if err != nil {
		return false, err
	}
	for _, cv := range convs {
		var side string
		switch {
		case cv.TeacherID == ratee && cv.UserID == rater:
			side = store.SenderTeacher
		case cv.UserID == ratee && cv.TeacherID == rater:
			side = store.SenderVisitor
		default:
			continue
		}
		b, err := s.store.ConversationBlocks(ctx, cv.ID)
		return b[side], err
	}
	return false, nil
}

// handleRatePeer: PUT /api/courses/{id}/peer-ratings/{uid} {"score":0..100,"lesson_id":""|"<lid>"} — гишүүн суралцагч
// ангийн найзаа ерөнхийд нь эсвэл хичээлээр үнэлнэ. Хариу нь тэр түвшний өдрийн snapshot (шинэ үнэлгээ маргаашаас тусна),
// mine нь шууд.
func (s *Server) handleRatePeer(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	course, err := s.store.CourseByID(ctx, r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	if teaches(course, c.UID) {
		writeErr(w, http.StatusForbidden, msgPeerTeacher)
		return
	}
	_, metas, ok := s.peerAccess(w, r, c.UID, course) // гишүүн, нийтлэгдсэн, архивлаагүй
	if !ok {
		return
	}
	rid := r.PathValue("uid")
	if rid == c.UID {
		writeErr(w, http.StatusBadRequest, msgPeerSelf)
		return
	}
	var in struct {
		Score    json.RawMessage `json:"score"`
		LessonID string          `json:"lesson_id"`
	}
	if !decode(w, r, &in) {
		return
	}
	score, ok := peerScore(in.Score)
	if !ok {
		writeErr(w, http.StatusBadRequest, msgPeerScore)
		return
	}
	lid := strings.TrimSpace(in.LessonID)
	if lid != "" {
		l, err := s.store.LessonByID(ctx, course.ID, lid)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			s.storeErr(w, r, err)
			return
		}
		if l == nil || hiddenFrom(l, course, c.UID) {
			writeErr(w, http.StatusNotFound, msgPeerLesson)
			return
		}
	}
	if !s.peerLim.Allow(c.UID) {
		writeErr(w, http.StatusTooManyRequests, msgPeerBusy)
		return
	}
	// Үнэлэгдэгч: энэ сургалтын суралцагч (багш биш, архивлаагүй, бүртгэлтэй), үнэлэгчийг хориглоогүй.
	if teaches(course, rid) || metas[rid].Archived {
		writeErr(w, http.StatusNotFound, msgPeerRatee)
		return
	}
	rm, err := s.courseMember(ctx, rid, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if !rm {
		writeErr(w, http.StatusNotFound, msgPeerRatee)
		return
	}
	blocked, err := s.peerBlocked(ctx, rid, c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	if blocked {
		writeErr(w, http.StatusNotFound, msgPeerRatee)
		return
	}
	class, err := s.peerClass(ctx, course, metas, c.UID, rid)
	if s.storeErr(w, r, err) {
		return
	}
	if _, in := class[rid]; !in {
		writeErr(w, http.StatusNotFound, msgPeerRatee)
		return
	}
	now := time.Now()
	row, err := s.store.SetPeerRating(ctx, course.ID, c.UID, rid, lid, score, peerCutoff(now))
	if s.storeErr(w, r, err) {
		return
	}
	s.peers.Delete(course.ID)
	d, err := s.loadPeers(ctx, course)
	if s.storeErr(w, r, err) {
		return
	}
	// Ачаалал бичихээс өмнө эхэлсэн байж болох тул бичсэн мөрийг (өдрийн snapshot-тай нь) заавал тусгана; анх үнэлж буй
	// хүний баримтыг нэмж уншина.
	d = d.with(row)
	if _, known := d.facts[c.UID]; !known {
		if f, err := s.store.PeerRaterFacts(ctx, course.ID, course.TeacherID, []string{c.UID}); err == nil && len(f) > 0 {
			facts := make(map[string]store.RaterFacts, len(d.facts)+1)
			for k, v := range d.facts {
				facts[k] = v
			}
			facts[c.UID] = f[c.UID]
			d.facts = facts
		}
	}
	if score < peerLow {
		s.peerAlerts(ctx, course, liveBook(d, class, now, ""), c.UID, rid)
	}
	sb := studentBook(d, class, now, lid)
	writeJSON(w, http.StatusOK, map[string]any{"member": sb.member(rid, c.UID, peerMine(d.ratings, c.UID, lid)), "me": sb.me(c.UID), "lesson_id": lid})
}

// handleDeletePeerRating: DELETE /api/courses/{id}/peer-ratings/{uid}[?lesson_id=][&rater=] — суралцагч өөрийн өгсөн
// (тэр хичээлийн, "" = ерөнхий) үнэлгээг, багш ?rater=-ийн тэр хүүхдэд өгсөн (зүй бус) үнэлгээг устгана (аудит лог).
// Үнэлгээ байхгүй бол 404; журналд архивласан суралцагч 403.
func (s *Server) handleDeletePeerRating(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	course, err := s.store.CourseByID(ctx, r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	q := r.URL.Query()
	ratee, rater, lid, teacher := r.PathValue("uid"), c.UID, strings.TrimSpace(q.Get("lesson_id")), teaches(course, c.UID)
	if teacher {
		rater = q.Get("rater")
	} else {
		metas, err := s.store.CourseStudentMetas(ctx, course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		if metas[c.UID].Archived {
			writeErr(w, http.StatusForbidden, msgPeerClosed)
			return
		}
	}
	if rater == "" || rater == ratee {
		writeErr(w, http.StatusNotFound, msgPeerGone)
		return
	}
	ratings, err := s.store.PeerRatings(ctx, course.ID) // кэшээс биш: устгах эсэхийг шинэ өгөгдлөөр
	if s.storeErr(w, r, err) {
		return
	}
	score := -1
	for _, p := range ratings {
		if p.RaterID == rater && p.RateeID == ratee && p.LessonID == lid && !p.Deleted {
			score = p.Score
			break
		}
	}
	if score < 0 {
		writeErr(w, http.StatusNotFound, msgPeerGone)
		return
	}
	if err := s.store.DeletePeerRating(ctx, course.ID, rater, ratee, lid, peerCutoff(time.Now())); s.storeErr(w, r, err) {
		return
	}
	s.peers.Delete(course.ID)
	if teacher {
		s.log.Info("харилцан үнэлгээг багш устгав", "teacher", c.UID, "course", course.ID, "rater", rater, "ratee", ratee,
			"lesson", lid, "score", score)
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- багшид анхааруулга ----

// peerAlertDedup — (сургалт, хүүхэд) бүрт өдөрт нэг peer_alert (процесс дотор; дахин эхлүүлэхэд HasNotification).
type peerAlertDedup struct {
	mu   sync.Mutex
	day  string
	sent map[string]bool
}

// take — өнөөдөр анх удаа бол тэмдэглээд true.
func (d *peerAlertDedup) take(day, key string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.day != day || d.sent == nil {
		d.day, d.sent = day, map[string]bool{}
	}
	if d.sent[key] {
		return false
	}
	d.sent[key] = true
	return true
}

func (d *peerAlertDedup) undo(day, key string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.day == day {
		delete(d.sent, key)
	}
}

// peerAlerts — бага оноо (peerLow-оос бага) бичигдсэний дараа, сургалтын нийт (хосын) оноогоор: (а) ratee peerLowRecv+
// тоологдох үнэлэгчээс бага оноо авсан, (б) rater peerLowGiven+ ангийн найздаа бага оноо өгсөн (тоологдох эсэхээс үл
// хамааран — шинэ бүртгэлээр дайралтыг ч илрүүлнэ) бол сургалтын эзэмшигч ба хамтран заагчдад мэдэгдэнэ.
func (s *Server) peerAlerts(ctx context.Context, course *store.Course, b *peerBook, rater, ratee string) {
	low := 0
	for _, pp := range b.pairs[ratee] { // тоологдох эсэхээс үл хамааран: шинэ бүртгэлүүдийн бөөнөөр дайралтыг ч барина
		if pp.score() < peerLow {
			low++
		}
	}
	if low >= peerLowRecv {
		name := briefName(b.class[ratee])
		s.peerAlert(ctx, course, ratee, "⚠️ Хамтын үнэлгээ: "+name+" олон бага оноо авлаа",
			fmt.Sprintf("«%s» сургалтын бүлэг чатад %s ангийн %d найзаасаа %d-өөс бага оноо авлаа. Хэн өгснийг шалгаж, зүй бус үнэлгээг устгана уу.",
				course.Title, name, low, peerLow))
	}
	given := 0
	for _, ps := range b.pairs {
		for _, pp := range ps {
			if pp.rater == rater && pp.score() < peerLow {
				given++
			}
		}
	}
	if given >= peerLowGiven {
		name := briefName(b.class[rater])
		s.peerAlert(ctx, course, rater, "⚠️ Хамтын үнэлгээ: "+name+" олон хүнд бага оноо өглөө",
			fmt.Sprintf("«%s» сургалтын бүлэг чатад %s ангийн %d найздаа %d-өөс бага оноо өглөө. Шалгаж, зүй бус үнэлгээг устгана уу.",
				course.Title, name, given, peerLow))
	}
}

// peerAlert — (сургалт, uid) бүрт өдөрт нэг мэдэгдэл; холбоос нь бүлэг чатын үнэлгээний самбарт тэр мөрийг нээнэ.
func (s *Server) peerAlert(ctx context.Context, course *store.Course, uid, title, body string) {
	now := time.Now()
	day, key := now.In(mnLoc).Format("2006-01-02"), course.ID+"|"+uid
	if !s.peerAl.take(day, key) {
		return
	}
	recips := []string{course.TeacherID}
	for _, t := range course.Teachers {
		if t != "" && t != course.TeacherID {
			recips = append(recips, t)
		}
	}
	link := "/c/" + course.ID + "#chat=group&peer=" + uid
	had, err := s.store.HasNotification(ctx, recips, peerAlertType, link, peerCutoff(now))
	if err != nil {
		s.log.Warn("peer_alert давхардал шалгах", "err", err)
		s.peerAl.undo(day, key)
		return
	}
	if had {
		return
	}
	ns := make([]*store.Notification, len(recips))
	for i, t := range recips {
		ns[i] = &store.Notification{UserID: t, Type: peerAlertType, Title: title, Body: body, Link: link, Count: 1}
	}
	s.notify(ctx, ns...)
}
