package httpapi

import (
	"context"
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"surgalt/internal/store"
)

// Системийн бэлэг: систем суралцахуйг оноогоор дүгнэж (дүн), түүнийгээ суралцахуйн зоос болгон өгнө.
// Зоосоор бэлгийн хайрцаг нээхэд эрэмбийн шатлалаас (rankLadder) дүр гарч, суралцахуйн зан чанартай холбосон урам
// хэлнэ. Хайрцаг бүр тодорхой эрэмбэд хүрэхэд л нээгдэх эрхтэй — өндөр эрэмбийн хайрцагнаас ховор дүр гарна (оноогоор дүгнэгдэнэ).
//
// Боловсролын ёс зүй: зоос зөвхөн шударга суралцсан онооноос бүрдэнэ (мөнгөөр зарагдахгүй, худалдан авах боломжгүй);
// хайрцаг бүрийн магадлалыг ил харуулна; давхардсан дүрд зоосны хэсгийг буцааж, шинэ дүрийн магадлалыг өсгөнө —
// санамсаргүй шагнал мөрийтэй тоглоом шиг дарамт болохгүй, суралцах сэдэл хэвээр үлдэнэ.

// coinPoints — 1 зоосонд хэрэгтэй оноо (нэг хичээл дээд тал нь 100 оноо ≈ 10 зоос).
const coinPoints = 10

type giftBox struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Cost   int    `json:"cost"`   // нээх үнэ (зоос)
	Unlock int    `json:"unlock"` // нээгдэх эрэмбийн түвшин
	Min    int    `json:"min"`    // гарч болох дүрийн түвшин (rankLadder индекс)
	Max    int    `json:"max"`
	Refund int    `json:"refund"` // давхардсан дүрд буцаах зоос
}

var giftBoxes = []giftBox{
	{ID: "wood", Name: "Модон хайрцаг", Cost: 10, Unlock: 0, Min: 0, Max: 5, Refund: 3},         // 1–2-р үе
	{ID: "silver", Name: "Мөнгөн хайрцаг", Cost: 25, Unlock: 3, Min: 3, Max: 8, Refund: 8},      // 2–3-р үе
	{ID: "gold", Name: "Алтан хайрцаг", Cost: 50, Unlock: 6, Min: 6, Max: 11, Refund: 15},       // 3–4-р үе
	{ID: "legend", Name: "Домогт хайрцаг", Cost: 100, Unlock: 12, Min: 12, Max: 17, Refund: 30}, // 5–6-р үе
}

func giftBoxByID(id string) (giftBox, bool) {
	for _, b := range giftBoxes {
		if b.ID == id {
			return b, true
		}
	}
	return giftBox{}, false
}

// boxWeights — хайрцгийн дүр бүрийн жин: доод түвшин түгээмэл, дээд нь ховор; хараахан цуглуулаагүй дүр 2 дахин
// их магадлалтай (давхардал багасна).
func boxWeights(b giftBox, owned map[int]int) []float64 {
	w := make([]float64, b.Max-b.Min+1)
	for i := range w {
		lvl := b.Min + i
		w[i] = float64(b.Max - lvl + 1)
		if owned[lvl] == 0 {
			w[i] *= 2
		}
	}
	return w
}

// giftState — суралцагчийн зоос, эрэмбэ, цуглуулга (хайрцаг нээлтээс).
type giftState struct {
	Earned, Spent, Refunded, Coins int
	Level                          int
	Owned                          map[int]int
	Opens                          []store.BoxOpen
}

func (s *Server) giftState(ctx context.Context, uid string) (*giftState, error) {
	sum, err := s.store.RankPointsSum(ctx, uid)
	if err != nil {
		return nil, err
	}
	opens, err := s.store.BoxOpens(ctx, uid)
	if err != nil {
		return nil, err
	}
	st := &giftState{Earned: sum / coinPoints, Owned: map[int]int{}, Opens: opens}
	for _, o := range opens {
		st.Spent += o.Cost
		st.Refunded += o.Refund
		st.Owned[o.Animal]++
	}
	st.Coins = max(0, st.Earned-st.Spent+st.Refunded) // оноо буурсан (дахин тооцоолсон) ч сөрөг болохгүй
	// Эрэмбэ: олгосон хамгийн өндөр түвшин эсвэл одоогийн нэгдсэн эрэмбэ (сургалтуудын дундаж оноо) — аль өндөр нь.
	if lvl, err := s.store.UserRankLevel(ctx, uid); err == nil {
		st.Level = lvl
	}
	if ids, err := s.store.RankCourseIDs(ctx, uid); err == nil && len(ids) > 0 {
		st.Level = max(st.Level, RankFor(int(math.Round(float64(sum)/float64(len(ids))))).Level)
	}
	return st, nil
}

type giftAnimal struct {
	Level   int    `json:"level"`
	Name    string `json:"name"`
	En      string `json:"en"`
	Emoji   string `json:"emoji"`
	Sign    string `json:"insignia"`
	Era     int    `json:"era"`
	EraName string `json:"era_name"`
	Meaning string `json:"meaning"`
	Points  int    `json:"points"`
	Count   int    `json:"count"`
	Cheer   string `json:"cheer,omitempty"`
}

func animalAt(lvl, count int) giftAnimal {
	st := rankLadder[lvl]
	return giftAnimal{Level: lvl, Name: st.Name, En: st.En, Emoji: st.Emoji, Sign: st.Sign, Era: lvl/3 + 1, EraName: rankEras[lvl/3], Meaning: st.Meaning, Points: st.Points, Count: count}
}

type giftChance struct {
	Level int     `json:"level"`
	Pct   float64 `json:"pct"`
}

// giftBoxView — хүрээгүй эрэмбийн нэр нууц тул түгжээг оноо, үеэр нь заана.
type giftBoxView struct {
	giftBox
	UnlockPoints int          `json:"unlock_points"`
	UnlockEra    int          `json:"unlock_era"`
	Unlocked     bool         `json:"unlocked"`
	Chances      []giftChance `json:"chances"`
}

func (st *giftState) boxes() []giftBoxView {
	out := make([]giftBoxView, 0, len(giftBoxes))
	for _, b := range giftBoxes {
		v := giftBoxView{giftBox: b, UnlockPoints: rankLadder[b.Unlock].Points, UnlockEra: b.Unlock/3 + 1, Unlocked: st.Level >= b.Unlock}
		w := boxWeights(b, st.Owned)
		var tot float64
		for _, x := range w {
			tot += x
		}
		for i, x := range w {
			v.Chances = append(v.Chances, giftChance{Level: b.Min + i, Pct: math.Round(x/tot*1000) / 10})
		}
		out = append(out, v)
	}
	return out
}

// GET /api/me/gifts — зоос, хайрцагнууд (нээгдсэн эсэх, магадлал), дүрүүдийн цуглуулга, сүүлийн нээлтүүд.
func (s *Server) handleGifts(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	st, err := s.giftState(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	animals := make([]giftAnimal, len(rankLadder))
	for i := range rankLadder {
		animals[i] = animalAt(i, st.Owned[i])
		if i > st.Level && st.Owned[i] == 0 { // хүрээгүй, хайрцагнаас ч олоогүй дүр нууц: зөвхөн түвшин, оноо
			a := &animals[i]
			a.Name, a.En, a.Emoji, a.Sign, a.Meaning = "", "", "", "", ""
			if i/3*3 > st.Level {
				a.EraName = ""
			}
		}
	}
	recent := []store.BoxOpen{}
	for i := len(st.Opens) - 1; i >= 0 && len(recent) < 8; i-- {
		recent = append(recent, st.Opens[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"coins": st.Coins, "earned": st.Earned, "spent": st.Spent, "refunded": st.Refunded, "coin_points": coinPoints,
		"level": st.Level, "rank": animalAt(st.Level, st.Owned[st.Level]),
		"boxes": st.boxes(), "animals": animals, "collected": len(st.Owned), "total": len(rankLadder), "recent": recent,
	})
}

// giftLocks — нэг хэрэглэгчийн зэрэг ирсэн нээх хүсэлтүүд зоосыг давхар зарцуулахгүй.
var giftLocks sync.Map

// POST /api/me/gifts/open {box} — зоосоор хайрцаг нээж, дүр гаргана.
func (s *Server) handleGiftOpen(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var in struct {
		Box string `json:"box"`
	}
	if !decode(w, r, &in) {
		return
	}
	b, ok := giftBoxByID(in.Box)
	if !ok {
		writeErr(w, http.StatusBadRequest, "Ийм хайрцаг байхгүй байна")
		return
	}
	mu, _ := giftLocks.LoadOrStore(c.UID, &sync.Mutex{})
	mu.(*sync.Mutex).Lock()
	defer mu.(*sync.Mutex).Unlock()
	st, err := s.giftState(r.Context(), c.UID)
	if s.storeErr(w, r, err) {
		return
	}
	if st.Level < b.Unlock {
		writeErr(w, http.StatusForbidden, "«"+b.Name+"» нь "+itoa(rankLadder[b.Unlock].Points)+" оноонд ("+itoa(b.Unlock/3+1)+"-р үе) хүрэхэд нээгдэнэ. Хичээлээ шударга судалж оноогоо өсгөөрэй.")
		return
	}
	if st.Coins < b.Cost {
		writeErr(w, http.StatusConflict, "Зоос хүрэлцэхгүй байна: "+itoa(b.Cost)+" зоос хэрэгтэй, танд "+itoa(st.Coins)+" байна. Хичээл бүрийг шударга судалж дуусгавал зоос нэмэгдэнэ.")
		return
	}
	wts := boxWeights(b, st.Owned)
	var tot float64
	for _, x := range wts {
		tot += x
	}
	pick, x := b.Min, rand.Float64()*tot
	for i, v := range wts {
		if x < v {
			pick = b.Min + i
			break
		}
		x -= v
	}
	fresh := st.Owned[pick] == 0
	o := &store.BoxOpen{ID: store.NewID(), UserID: c.UID, Box: b.ID, Animal: pick, Cost: b.Cost, At: time.Now().UTC()}
	if !fresh {
		o.Refund = b.Refund
	}
	if err := s.store.AddBoxOpen(r.Context(), o); s.storeErr(w, r, err) {
		return
	}
	a := animalAt(pick, st.Owned[pick]+1)
	a.Cheer = rankLadder[pick].Cheer
	writeJSON(w, http.StatusOK, map[string]any{
		"animal": a, "new": fresh, "refund": o.Refund, "box": b.ID,
		"coins": st.Coins - b.Cost + o.Refund, "collected": len(st.Owned) + map[bool]int{true: 1, false: 0}[fresh], "total": len(rankLadder),
	})
}
