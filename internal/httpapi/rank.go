package httpapi

import (
	"context"
	"math"
	"sort"

	"surgalt/internal/store"
)

// Цэргийн цол — идэвхтэй, шударга суралцсаны урамшуулал. Хичээл бүрт нотолгоонд суурилсан оноо (0-100)
// → тухайн хичээлийн цол; бүх хичээлийн нийлбэр оноо → суралцагчийн нэгдсэн цол. Хуулах оролдлого
// (copy, screenshot, print, devtools, download, утас) эсвэл автоматаар зогсоосон хичээлд цол олгохгүй.

// LessonRank — нэг хичээл дэх үнэлгээ.
type LessonRank struct {
	LessonID     string   `json:"lesson_id"`
	Title        string   `json:"title,omitempty"`
	Points       int      `json:"points"` // 0-100
	Rank         string   `json:"rank"`
	Disqualified bool     `json:"disqualified,omitempty"` // хуулах оролдлоготой — цолгүй
	Reasons      []string `json:"reasons,omitempty"`      // юунаас бүрдсэн / юу дутуу
}

// RankInfo — нэгдсэн цол.
type RankInfo struct {
	Points   int    `json:"points"`
	Level    int    `json:"level"` // 0..len(rankLadder)-1
	Name     string `json:"name"`
	Next     int    `json:"next,omitempty"` // дараагийн цолд хүрэх оноо (0 = дээд цол)
	NextName string `json:"next_name,omitempty"`
	Lessons  int    `json:"lessons"`  // үнэлэгдсэн хичээл
	Honest   int    `json:"honest"`   // хуулах оролдлогогүй хичээл
	Cheated  int    `json:"cheated"`  // цол олгогдоогүй хичээл
	Insignia string `json:"insignia"` // ★ тэмдэг (UI-д)
	Progress int    `json:"progress"` // дараагийн цол хүртэлх хувь
}

type rankStep struct {
	Points int
	Name   string
	Sign   string
}

// rankLadder — Монгол Улсын Зэвсэгт хүчний цолын дараалал (нийлбэр оноогоор). Нэг хичээл дээд тал нь 100 оноо.
var rankLadder = []rankStep{
	{0, "Шинэ цэрэг", "·"},
	{60, "Байлдагч", "▪"},
	{150, "Ахлах байлдагч", "▪▪"},
	{260, "Дэд түрүүч", "▾"},
	{400, "Түрүүч", "▾▾"},
	{560, "Ахлах түрүүч", "▾▾▾"},
	{750, "Сургагч ахлагч", "◆"},
	{980, "Дэд дэслэгч", "★"},
	{1250, "Дэслэгч", "★★"},
	{1560, "Ахлах дэслэгч", "★★★"},
	{1900, "Ахмад", "★★★★"},
	{2300, "Хошууч", "✦"},
	{2750, "Дэд хурандаа", "✦✦"},
	{3300, "Хурандаа", "✦✦✦"},
	{4000, "Бригадын генерал", "✪"},
	{4800, "Хошууч генерал", "✪✪"},
	{5700, "Дэслэгч генерал", "✪✪✪"},
	{6800, "Генерал", "✪✪✪✪"},
}

// lessonRankName — нэг хичээлийн оноогоор олгох цол.
func lessonRankName(points int, disqualified bool) string {
	switch {
	case disqualified:
		return "Цолгүй"
	case points >= 90:
		return "Ахлах түрүүч"
	case points >= 75:
		return "Түрүүч"
	case points >= 55:
		return "Дэд түрүүч"
	case points >= 35:
		return "Ахлах байлдагч"
	case points > 0:
		return "Байлдагч"
	}
	return "Шинэ цэрэг"
}

// cheatEvents — цол хасах зөрчлүүд (хуулах, зураг авах, татах, автоматаар зогсоосон).
var cheatEvents = []string{"copy", "screenshot", "print", "save", "devtools", "download", "auto_block", "context_menu"}

// RankFor — нийлбэр оноогоор нэгдсэн цол.
func RankFor(points int) RankInfo {
	ri := RankInfo{Points: points}
	for i, st := range rankLadder {
		if points >= st.Points {
			ri.Level, ri.Name, ri.Insignia = i, st.Name, st.Sign
		}
	}
	if ri.Level+1 < len(rankLadder) {
		nx := rankLadder[ri.Level+1]
		ri.Next, ri.NextName = nx.Points, nx.Name
		cur := rankLadder[ri.Level].Points
		ri.Progress = int(math.Round(float64(points-cur) / float64(nx.Points-cur) * 100))
	} else {
		ri.Progress = 100
	}
	return ri
}

// lessonPoints — нэг хичээлийн нотолгоог нэгтгэж 0-100 оноо гаргана.
// Үзсэн 5, дууссан 10, идэвхтэй хугацаа 30, дүгнэлт 15 — үргэлж; асуулга 30, видео 10 — байвал.
// Таб солилт бүр −5 (дээд тал нь −30). Хуулах оролдлого → 0, цолгүй.
func lessonPoints(l *store.Lesson, p *store.LessonProgress, sessions []*store.StudySession, reflected bool, videoCoverage int, hasVideo bool) LessonRank {
	lr := LessonRank{LessonID: l.ID, Title: l.Title}
	if p == nil && len(sessions) == 0 {
		lr.Rank = lessonRankName(0, false)
		return lr
	}
	active, total, tabs, cheats := 0, 0, 0, 0
	for _, s := range sessions {
		active += s.ActiveSec
		total += s.ActiveSec + s.IdleSec + s.AwaySec
		tabs += s.Counts["tab_switch"]
		for _, k := range cheatEvents {
			cheats += s.Counts[k]
		}
	}
	if cheats > 0 {
		lr.Disqualified = true
		lr.Reasons = append(lr.Reasons, "хуулах оролдлого / хориг")
		lr.Rank = lessonRankName(0, true)
		return lr
	}
	earned, possible := 0.0, 0.0
	add := func(w, frac float64, ok string, miss string) {
		possible += w
		earned += w * math.Max(0, math.Min(1, frac))
		if frac >= 1 && ok != "" {
			lr.Reasons = append(lr.Reasons, ok)
		} else if frac < 1 && miss != "" {
			lr.Reasons = append(lr.Reasons, miss)
		}
	}
	add(5, 1, "", "")
	completed := p != nil && p.CompletedAt != nil
	add(10, b2f(completed), "дууссан", "дуусгаагүй")
	switch {
	case l.ActiveMin > 0:
		add(30, float64(active)/float64(l.ActiveMin*60), "идэвхтэй хугацаа хүрсэн", "идэвхтэй хугацаа дутуу")
	case total > 0:
		add(30, float64(active)/float64(total)/0.8, "анхаарлаа төвлөрүүлсэн", "идэвхгүй хугацаа их")
	default:
		add(30, 0, "", "хичээл үзээгүй")
	}
	add(15, b2f(reflected), "дүгнэлт бичсэн", "дүгнэлт бичээгүй")
	if got, qt, done, _ := quizMastery(l, p); qt > 0 {
		f := float64(got) / float64(qt)
		if done {
			f = 1
		}
		add(30, f, "бүх асуултад зөв", "асуулга дутуу")
	}
	if hasVideo {
		add(10, float64(videoCoverage)/100, "видеог бүрэн үзсэн", "видео дутуу")
	}
	pts := earned / possible * 100
	if pen := math.Min(30, float64(tabs)*5); pen > 0 {
		pts -= pen
		lr.Reasons = append(lr.Reasons, "таб солилт −"+itoa(int(pen)))
	}
	lr.Points = max(0, min(100, int(math.Round(pts))))
	lr.Rank = lessonRankName(lr.Points, false)
	return lr
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

func hasVideoBlock(l *store.Lesson) bool {
	for _, b := range l.Blocks {
		if b.Type == "video" {
			return true
		}
	}
	return l.VideoURL != ""
}

// rankEvidence — нэг суралцагчийн нэг сургалт дахь нотолгоо (сесс, дүгнэлт, видео) хичээлээр нь.
type rankEvidence struct {
	sessions map[string][]*store.StudySession // lesson → сесс
	reflect  map[string]bool
	video    map[string][2]int // lesson → [үзсэн bucket, нийт bucket] нийлбэр
}

func newRankEvidence() *rankEvidence {
	return &rankEvidence{sessions: map[string][]*store.StudySession{}, reflect: map[string]bool{}, video: map[string][2]int{}}
}

func (e *rankEvidence) addSession(s *store.StudySession) {
	if s.Kind == "lesson" || s.Kind == "" {
		e.sessions[s.LessonID] = append(e.sessions[s.LessonID], s)
	}
}

func (e *rankEvidence) addWatch(v *store.VideoWatch) {
	if v.Duration <= 0 {
		return
	}
	total := (v.Duration + store.VideoBucketSec - 1) / store.VideoBucketSec
	cur := e.video[v.LessonID]
	e.video[v.LessonID] = [2]int{cur[0] + min(len(v.Buckets), total), cur[1] + total}
}

func (e *rankEvidence) coverage(lessonID string) int {
	v := e.video[lessonID]
	if v[1] == 0 {
		return 0
	}
	return v[0] * 100 / v[1]
}

// computeRanks — сургалтын хичээл бүрийн цол ба нийлбэр оноо (зөвхөн ямар нэг идэвх бүртгэгдсэн хичээл).
func computeRanks(lessons []store.Lesson, progress map[string]store.LessonProgress, e *rankEvidence) ([]LessonRank, int) {
	var out []LessonRank
	sum := 0
	for i := range lessons {
		l := &lessons[i]
		var p *store.LessonProgress
		if pp, ok := progress[l.ID]; ok {
			p = &pp
		}
		if p == nil && len(e.sessions[l.ID]) == 0 {
			continue
		}
		lr := lessonPoints(l, p, e.sessions[l.ID], e.reflect[l.ID], e.coverage(l.ID), hasVideoBlock(l))
		sum += lr.Points
		out = append(out, lr)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Points > out[j].Points })
	return out, sum
}

// summarize — хичээлийн цолуудаас нэгдсэн цол.
func summarizeRank(ranks []LessonRank, sum int) RankInfo {
	ri := RankFor(sum)
	for _, r := range ranks {
		ri.Lessons++
		if r.Disqualified {
			ri.Cheated++
		} else {
			ri.Honest++
		}
	}
	return ri
}

// courseRanks — нэг суралцагчийн нэг сургалт дахь цолууд (сесс, дүгнэлт, видеог сангаас ачаална).
func (s *Server) courseRanks(ctx context.Context, userID, courseID string, lessons []store.Lesson, progress map[string]store.LessonProgress) ([]LessonRank, RankInfo) {
	f := store.ActivityFilter{UserID: userID, CourseID: courseID}
	e := newRankEvidence()
	if ss, err := s.store.Sessions(ctx, f, 5000); err == nil {
		for i := range ss {
			e.addSession(&ss[i])
		}
	}
	if refs, err := s.store.Reflections(ctx, f, 5000); err == nil {
		for _, r := range refs {
			e.reflect[r.LessonID] = true
		}
	}
	if ws, err := s.store.VideoWatches(ctx, f); err == nil {
		for i := range ws {
			e.addWatch(&ws[i])
		}
	}
	ranks, sum := computeRanks(lessons, progress, e)
	return ranks, summarizeRank(ranks, sum)
}
