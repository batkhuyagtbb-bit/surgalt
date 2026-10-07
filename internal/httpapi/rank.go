package httpapi

import (
	"context"
	"math"
	"sort"
	"time"

	"surgalt/internal/store"
)

// Цэргийн цол — идэвхтэй, шударга суралцсаны урамшуулал. Цолыг хичээл бүрт олгохгүй: хичээлүүдийг
// нэгтгэж (интеграц) нэг цол олгоно. Хичээл бүр нотолгоонд суурилсан оноо (0-100) НЭМНЭ; бүлгийн бүх
// хичээлийг дуусгавал бүлгийн, сургалтын бүх хичээлийг дуусгавал сургалтын интеграцын нэмэлт оноо
// авна. Сургалтын оноо = хичээлүүдийн оноо + интеграц → сургалтын цол; бүх сургалтын нийлбэр → нэгдсэн цол.
// Хуулах оролдлого (copy, screenshot, print, devtools, download, утас) эсвэл автоматаар зогсоосон
// хичээлийн оноо тооцогдохгүй бөгөөд тухайн бүлэг, сургалтын интеграцын нэмэлтийг хаана.

// LessonRank — нэг хичээлийн нэгдсэн цолд нэмэх оноо (хичээлд цол олгохгүй).
type LessonRank struct {
	LessonID     string   `json:"lesson_id"`
	Title        string   `json:"title,omitempty"`
	Points       int      `json:"points"`                 // 0-100: нэгдсэн цолд нэмэх оноо
	Disqualified bool     `json:"disqualified,omitempty"` // хуулах оролдлоготой — оноо тооцогдохгүй
	Reasons      []string `json:"reasons,omitempty"`      // юунаас бүрдсэн / юу дутуу
	// Суралцагчийн өөрийн самбарт: яаж судалсан бэ.
	CourseID    string     `json:"course_id,omitempty"`
	CourseTitle string     `json:"course_title,omitempty"`
	ActiveSec   int        `json:"active_sec"`
	TotalSec    int        `json:"total_sec"`
	Sessions    int        `json:"sessions"`
	Completed   bool       `json:"completed"`
	QuizCorrect int        `json:"quiz_correct"`
	QuizTotal   int        `json:"quiz_total"`
	VideoPct    int        `json:"video_pct"`
	HasVideo    bool       `json:"has_video"`
	Reflected   bool       `json:"reflected"`
	TabSwitches int        `json:"tab_switches"`
	LastAt      *time.Time `json:"last_at,omitempty"`
}

// RankInfo — нэгдсэн цол.
type RankInfo struct {
	Points   int    `json:"points"`
	Level    int    `json:"level"` // 0..len(rankLadder)-1
	Name     string `json:"name"`
	Next     int    `json:"next,omitempty"` // дараагийн цолд хүрэх оноо (0 = дээд цол)
	NextName string `json:"next_name,omitempty"`
	Lessons  int    `json:"lessons"` // үнэлэгдсэн хичээл
	Honest   int    `json:"honest"`  // хуулах оролдлогогүй хичээл
	Cheated  int    `json:"cheated"` // оноо нь тооцогдоогүй хичээл
	// Integration — оноо юунаас бүрдсэн: хичээлүүдийн оноо + бүтэн бүлэг/сургалтын нэмэлт.
	Integration Integration `json:"integration"`
	Insignia    string      `json:"insignia"` // ★ тэмдэг (UI-д)
	Progress    int         `json:"progress"` // дараагийн цол хүртэлх хувь
	// Систем өөрөө оношилж өгөх зөвлөмж: юу дутуу байгаа, дараагийн цолд юу хэрэгтэй.
	Tips []string `json:"tips,omitempty"`
	// itips — интеграцын зөвлөмж (бүх сургалтын цолд сургалтын нэрээр нэгтгэнэ).
	itips []string
}

// Integration — хичээлүүдийг нэгтгэсэн үнэлгээний задаргаа.
type Integration struct {
	LessonPoints int `json:"lesson_points"` // хичээлүүдийн оноо (нийлбэр)
	Bonus        int `json:"bonus"`         // бүтэн бүлэг, бүтэн сургалтын нэмэлт
	Sections     int `json:"sections"`      // бүх хичээлийг нь дуусгасан бүлэг
	SectionsAll  int `json:"sections_all"`  // нийт бүлэг
	Courses      int `json:"courses"`       // бүх хичээлийг нь дуусгасан сургалт
	CoursesAll   int `json:"courses_all"`   // үнэлэгдсэн сургалт
}

// Интеграцын нэмэлт: бүлэг/сургалтын бүх хичээлийг шударгаар дуусгавал хичээл тутамд.
const (
	sectionBonusPerLesson = 10
	courseBonusPerLesson  = 20
)

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

// RankStep — шатлалын нэг цол (клиентэд: шатлалын зураас).
type RankStep struct {
	Points   int    `json:"points"`
	Name     string `json:"name"`
	Insignia string `json:"insignia"`
}

// RankLadder — бүх цолын шатлал (суралцагчид зорилгоо харуулахад).
func RankLadder() []RankStep {
	out := make([]RankStep, len(rankLadder))
	for i, s := range rankLadder {
		out[i] = RankStep{Points: s.Points, Name: s.Name, Insignia: s.Sign}
	}
	return out
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

// lessonPoints — нэг хичээлийн нотолгоог нэгтгэж нэгдсэн цолд нэмэх 0-100 оноо гаргана (цол биш).
// Үзсэн 5, дууссан 10, идэвхтэй хугацаа 30, дүгнэлт 15 — үргэлж; асуулга 30, видео 10 — байвал.
// Таб солилт бүр −5 (дээд тал нь −30). Хуулах оролдлого → 0, оноо тооцогдохгүй.
func lessonPoints(l *store.Lesson, p *store.LessonProgress, sessions []*store.StudySession, reflected bool, videoCoverage int, hasVideo bool) LessonRank {
	lr := LessonRank{LessonID: l.ID, Title: l.Title, HasVideo: hasVideo, VideoPct: videoCoverage, Reflected: reflected}
	if p == nil && len(sessions) == 0 {
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
		if lr.LastAt == nil || s.LastAt.After(*lr.LastAt) {
			t := s.LastAt
			lr.LastAt = &t
		}
	}
	lr.ActiveSec, lr.TotalSec, lr.Sessions, lr.TabSwitches = active, total, len(sessions), tabs
	if p != nil {
		lr.Completed = p.CompletedAt != nil
		lr.QuizCorrect, lr.QuizTotal, _, _ = quizMastery(l, p)
		if lr.LastAt == nil {
			t := p.ViewedAt
			lr.LastAt = &t
		}
	}
	if cheats > 0 {
		lr.Disqualified = true
		lr.Reasons = append(lr.Reasons, "хуулах оролдлого / хориг")
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

// computeRanks — нэг сургалтын хичээлүүдийг нэгтгэж (интеграц) нэг цол гаргана: хичээл бүрийн оноо
// (зөвхөн идэвх бүртгэгдсэн хичээл) + бүтэн дуусгасан бүлэг, сургалтын нэмэлт. Хичээлд цол олгохгүй.
func computeRanks(lessons []store.Lesson, progress map[string]store.LessonProgress, e *rankEvidence) ([]LessonRank, RankInfo) {
	var out []LessonRank
	sum := 0
	disq := map[string]bool{}
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
		disq[l.ID] = lr.Disqualified
		out = append(out, lr)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Points > out[j].Points })
	in, tips := integrate(lessons, progress, disq)
	in.LessonPoints = sum
	ri := summarizeRank(out, sum+in.Bonus)
	ri.Integration, ri.itips = in, tips
	ri.Tips = append(append([]string{}, tips...), ri.Tips...)
	return out, ri
}

// integrate — бүлэг, сургалтын бүх хичээлийг шударгаар (хуулах оролдлогогүй) дуусгасан эсэхээр интеграцын
// нэмэлт оноо тооцож, дуусахад хамгийн ойр бүлгийг зөвлөнө. Дууссан = MarkLessonCompleted (хичээл,
// тэнцсэн шалгалт, илгээсэн даалгавар бүгд).
func integrate(lessons []store.Lesson, progress map[string]store.LessonProgress, disq map[string]bool) (Integration, []string) {
	var in Integration
	var tips []string
	if len(lessons) == 0 {
		return in, nil
	}
	done := func(id string) bool { p, ok := progress[id]; return ok && p.CompletedAt != nil && !disq[id] }
	type section struct {
		title    string
		n, ok    int
		disabled bool
	}
	var order []*section
	byTitle := map[string]*section{}
	allOK, anyBad := 0, false
	for i := range lessons {
		id := lessons[i].ID
		if done(id) {
			allOK++
		}
		anyBad = anyBad || disq[id]
		t := lessons[i].Section
		if t == "" {
			continue
		}
		sc := byTitle[t]
		if sc == nil {
			sc = &section{title: t}
			byTitle[t] = sc
			order = append(order, sc)
		}
		sc.n++
		if done(id) {
			sc.ok++
		}
		sc.disabled = sc.disabled || disq[id]
	}
	in.SectionsAll, in.CoursesAll = len(order), 1
	var near *section // дуусахад хамгийн ойр бүлэг
	for _, sc := range order {
		switch {
		case sc.ok == sc.n && !sc.disabled:
			in.Sections++
			in.Bonus += sc.n * sectionBonusPerLesson
		case !sc.disabled && sc.ok > 0 && (near == nil || sc.n-sc.ok < near.n-near.ok):
			near = sc
		}
	}
	if allOK == len(lessons) && !anyBad {
		in.Courses = 1
		in.Bonus += len(lessons) * courseBonusPerLesson
	}
	if near != nil {
		tips = append(tips, "«"+near.title+"» бүлгийн үлдсэн "+itoa(near.n-near.ok)+" хичээлийг дуусгавал бүлгийн интеграц (+"+itoa(near.n*sectionBonusPerLesson)+" оноо)")
	}
	if in.Courses == 0 && !anyBad && allOK > 0 {
		tips = append(tips, "сургалтын бүх "+itoa(len(lessons))+" хичээлийг дуусгавал сургалтын интеграц (+"+itoa(len(lessons)*courseBonusPerLesson)+" оноо)")
	}
	return in, tips
}

// mergeRanks — сургалт бүрийн нэгдсэн цолыг нэгтгэж бүх сургалтын нэг цол гаргана.
// titles[i] — courses[i]-ийн нэр (интеграцын зөвлөмжид).
func mergeRanks(rows []LessonRank, courses []RankInfo, titles []string) RankInfo {
	var in Integration
	total := 0
	var itips []string
	for i, c := range courses {
		total += c.Points
		in.LessonPoints += c.Integration.LessonPoints
		in.Bonus += c.Integration.Bonus
		in.Sections += c.Integration.Sections
		in.SectionsAll += c.Integration.SectionsAll
		in.Courses += c.Integration.Courses
		in.CoursesAll += c.Integration.CoursesAll
		if len(c.itips) > 0 && len(itips) < 2 {
			t := c.itips[0]
			if i < len(titles) && titles[i] != "" {
				t = titles[i] + ": " + t
			}
			itips = append(itips, t)
		}
	}
	ri := summarizeRank(rows, total)
	ri.Integration, ri.itips = in, itips
	ri.Tips = append(append([]string{}, itips...), ri.Tips...)
	return ri
}

// summarize — хичээлийн цолуудаас нэгдсэн цол ба автомат оношилгоо.
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
	ri.Tips = diagnose(ranks, ri)
	return ri
}

// diagnose — хичээлүүдийн дутуу зүйлсийг тоолж, хамгийн их оноо алдуулж буйгаас эхлэн зөвлөнө.
func diagnose(ranks []LessonRank, ri RankInfo) []string {
	if len(ranks) == 0 {
		return nil
	}
	type tip struct{ key, text string }
	tips := []tip{
		{"хуулах оролдлого / хориг", "хуулах, зураг авах оролдлого бүү хий — тийм хичээлийн оноо тооцогдохгүй, интеграцын нэмэлтийг хаана"},
		{"асуулга дутуу", "хичээл доторх асуултуудад бүгдэд нь зөв хариул (+30 оноо/хичээл)"},
		{"идэвхтэй хугацаа дутуу", "хичээлээ идэвхтэй, дуустал үз (+30 оноо/хичээл)"},
		{"идэвхгүй хугацаа их", "хичээл үзэхдээ өөр цонх руу бүү шилж, анхаарлаа төвлөрүүл (+30 хүртэл)"},
		{"дүгнэлт бичээгүй", "хичээл бүрийн дараа «юу сурсан бэ?» дүгнэлтээ бич (+15 оноо/хичээл)"},
		{"дуусгаагүй", "хичээлээ «дууслаа» гэж тэмдэглэ (+10 оноо/хичээл)"},
		{"видео дутуу", "видеог дуустал үз (+10 оноо/хичээл)"},
	}
	count := map[string]int{}
	for _, r := range ranks {
		for _, reason := range r.Reasons {
			count[reason]++
		}
	}
	var out []string
	for _, t := range tips {
		if n := count[t.key]; n > 0 {
			out = append(out, itoa(n)+" хичээлд: "+t.text)
		}
		if len(out) == 3 {
			break
		}
	}
	if ri.Next > 0 {
		out = append(out, "дараагийн «"+ri.NextName+"» цолд "+itoa(ri.Next-ri.Points)+" оноо дутуу")
	}
	return out
}

// autoAward — систем өөрөө цол олгоно: сургалт бүрийн оноог хадгалж, нийлбэр түвшин дээшилбэл
// суралцагчид мэдэгдэл илгээж, олгосон түвшинг хадгална. Нэгдсэн (бүх сургалтын) цолыг буцаана.
// awarded=true бол яг энэ хүсэлтээр шинэ цол олгогдсон (хөтөч баяр хүргэж салют буудуулна).
func (s *Server) autoAward(ctx context.Context, uid, courseID, link string, coursePts int) (info RankInfo, awarded bool) {
	total, err := s.store.SaveRankPoints(ctx, uid, courseID, coursePts)
	if err != nil {
		return RankFor(coursePts), false
	}
	info = RankFor(total)
	cur, err := s.store.UserRankLevel(ctx, uid)
	if err == nil && info.Level > cur {
		if err := s.store.SetUserRankLevel(ctx, uid, info.Level); err == nil {
			awarded = true
			s.notify(ctx, &store.Notification{UserID: uid, Type: "rank", Title: "🎖 Шинэ цол: " + info.Name,
				Body: "Баяр хүргэе! Систем таны бүх хичээлийг нэгтгэн идэвхтэй, шударга суралцсан байдлыг үнэлж «" + info.Name + "» цол олголоо (" + itoa(total) + " оноо).", Link: link})
		}
	}
	return info, awarded
}

// courseRanks — нэг суралцагчийн нэг сургалт дахь хичээлүүдийн оноо ба нэгдсэн цол (сесс, дүгнэлт, видеог сангаас ачаална).
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
	return computeRanks(lessons, progress, e)
}
