package httpapi

import (
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"net/http"
	"slices"
	"strconv"
	"time"

	"surgalt/internal/store"
)

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// StudentStat — нэг суралцагчийн нэгтгэл.
type StudentStat struct {
	UserID     string         `json:"user_id"`
	Name       string         `json:"name"`
	Sessions   int            `json:"sessions"`
	TotalSec   int            `json:"total_sec"`
	ActiveSec  int            `json:"active_sec"`
	IdleSec    int            `json:"idle_sec"`
	AwaySec    int            `json:"away_sec"`
	ActivePct  int            `json:"active_pct"` // идэвхтэй / нийт
	Attention  int            `json:"attention"`  // анхаарал төвлөрлийн индекс 0-100
	Camera     bool           `json:"camera"`
	Violations int            `json:"violations"`
	Counts     map[string]int `json:"counts"`
	Lessons    int            `json:"lessons"`
	ExamBest   int            `json:"exam_best"` // -1 = шалгалт өгөөгүй
	ExamCount  int            `json:"exam_count"`
	Terminated int            `json:"terminated"`
	LastAt     time.Time      `json:"last_at"`
	Live       bool           `json:"live"`
	Risk       string         `json:"risk"` // ok | watch | risk

	// Идэвхтэй суралцсаны бодит нотолгоо (хугацаа, камераас гадна).
	QuizTotal       int `json:"quiz_total"`
	QuizCorrect     int `json:"quiz_correct"`
	QuizAccuracy    int `json:"quiz_accuracy"`    // зөв хариултын хувь (0 = асуулт хараахан авч үзээгүй)
	QuizAvgMs       int `json:"quiz_avg_ms"`      // хариулахад дундаж зарцуулсан хугацаа
	QuizGuesses     int `json:"quiz_guesses"`     // 1.5 секундээс богино хугацаанд хариулсан (таамаглал)
	Reflections     int `json:"reflections"`      // "юу сурсан бэ?" гэж бичсэн тоо
	ReflectionWords int `json:"reflection_words"` // дундаж үгийн тоо
	VideoClips      int `json:"video_clips"`      // хэдэн видео эхэлсэн
	VideoCoverage   int `json:"video_coverage"`   // видеоны дундаж хэдэн хувийг нь үзсэн (0-100)
	ActiveDays      int `json:"active_days"`      // энэ хугацаанд суралцсан өдрийн тоо
	StreakDays      int `json:"streak_days"`      // дараалсан өдрийн тоо (өнөөдөр/өчигдрөөс)
	LearnScore      int `json:"learn_score"`      // идэвхтэй суралцсан нэгдсэн оноо 0-100
}

func (st *StudentStat) add(x *store.StudySession) {
	st.Sessions++
	st.ActiveSec += x.ActiveSec
	st.IdleSec += x.IdleSec
	st.AwaySec += x.AwaySec
	st.Camera = st.Camera || x.Camera
	for k, v := range x.Counts {
		st.Counts[k] += v
		if eventInfo[k].Violation {
			st.Violations += v
		}
	}
	if x.LastAt.After(st.LastAt) {
		st.LastAt = x.LastAt
	}
	if !x.Ended && time.Since(x.LastAt) < 45*time.Second {
		st.Live = true
	}
}

// finish — хувь, индекс, эрсдэлийн түвшин. Анхаарлын индекс = идэвхтэй хувь × дундаж төвлөрөл
// (камергүй бол төвлөрөл = 1) − зөрчлийн торгууль.
// learnScore нь хугацаанаас гадуур бодит сурсан эсэхийг нэгтгэнэ: зөв хариулт, видеог
// дуустал үзсэн, дүгнэлт бичсэн, тогтмол ирсэн эсэх — байхгүй дохио дундажид орохгүй.
func (st *StudentStat) learnScore() int {
	var sum float64
	n := 0
	if st.QuizTotal > 0 {
		sum += float64(st.QuizAccuracy)
		n++
	}
	if st.VideoClips > 0 {
		sum += float64(st.VideoCoverage)
		n++
	}
	if st.Lessons > 0 {
		sum += math.Min(100, float64(st.Reflections)/float64(st.Lessons)*100)
		n++
	}
	if st.ActiveDays > 0 {
		sum += math.Min(100, float64(st.StreakDays)*20)
		n++
	}
	if n == 0 {
		return 0
	}
	score := sum / float64(n)
	if st.QuizTotal > 0 { // таамаглалаар зөв хариулсан мэт харагдахаас хамгаална
		score -= math.Min(25, float64(st.QuizGuesses)/float64(st.QuizTotal)*50)
	}
	return max(0, min(100, int(math.Round(score))))
}

func (st *StudentStat) finish(focusSum float64, focusN int) {
	st.TotalSec = st.ActiveSec + st.IdleSec + st.AwaySec
	if st.TotalSec > 0 {
		st.ActivePct = int(math.Round(float64(st.ActiveSec) / float64(st.TotalSec) * 100))
	}
	focus := 1.0
	if focusN > 0 {
		focus = focusSum / float64(focusN)
	}
	hours := math.Max(float64(st.TotalSec)/3600, 0.25)
	penalty := math.Min(30, float64(st.Violations)/hours*2)
	st.Attention = max(0, int(math.Round(float64(st.ActivePct)*focus-penalty)))
	if st.QuizTotal > 0 {
		st.QuizAccuracy = int(math.Round(float64(st.QuizCorrect) / float64(st.QuizTotal) * 100))
		st.QuizAvgMs /= st.QuizTotal
	}
	if st.VideoClips > 0 {
		st.VideoCoverage /= st.VideoClips
	}
	if st.Reflections > 0 {
		st.ReflectionWords /= st.Reflections
	}
	st.LearnScore = st.learnScore()
	switch {
	case st.TotalSec == 0:
		st.Risk = "ok"
	case st.Attention < 40 || st.Terminated > 0 || st.Counts["copy"]+st.Counts["screenshot"]+st.Counts["phone"] > 2:
		st.Risk = "risk"
	case st.Attention < 70 || st.Violations > 5:
		st.Risk = "watch"
	default:
		st.Risk = "ok"
	}
}

type analyticsData struct {
	Students []*StudentStat        `json:"students"`
	Daily    []map[string]any      `json:"daily"`
	Events   []store.ActivityEvent `json:"events"`
	Totals   map[string]any        `json:"totals"`
}

func (s *Server) buildAnalytics(ctx context.Context, teacherID, courseID, userID string, days int) (*analyticsData, []store.StudySession, []store.ExamAttempt, error) {
	since := time.Now().AddDate(0, 0, -days)
	f := store.ActivityFilter{TeacherID: teacherID, CourseID: courseID, UserID: userID, Since: since}
	ss, err := s.store.Sessions(ctx, f, 20000)
	if err != nil {
		return nil, nil, nil, err
	}
	atts, err := s.store.ExamAttempts(ctx, store.ActivityFilter{TeacherID: teacherID, CourseID: courseID, UserID: userID, Since: since})
	if err != nil {
		return nil, nil, nil, err
	}
	evLimit := 100
	if userID != "" {
		evLimit = 500
	}
	evs, err := s.store.ActivityEvents(ctx, f, evLimit)
	if err != nil {
		return nil, nil, nil, err
	}
	quizLogs, err := s.store.QuizLogs(ctx, f, 20000)
	if err != nil {
		return nil, nil, nil, err
	}
	refs, err := s.store.Reflections(ctx, f, 20000)
	if err != nil {
		return nil, nil, nil, err
	}
	watches, err := s.store.VideoWatches(ctx, f)
	if err != nil {
		return nil, nil, nil, err
	}
	by := map[string]*StudentStat{}
	focus := map[string][2]float64{}
	lessons := map[string]map[string]bool{}
	get := func(uid, name string) *StudentStat {
		st := by[uid]
		if st == nil {
			st = &StudentStat{UserID: uid, Name: name, Counts: map[string]int{}, ExamBest: -1}
			by[uid] = st
			lessons[uid] = map[string]bool{}
		}
		return st
	}
	dayActive := map[string][2]int{}
	userDays := map[string]map[string]bool{} // uid -> "2006-01-02" -> идэвхтэй байсан эсэх
	for i := range ss {
		x := &ss[i]
		if x.UserID == teacherID {
			continue // багш өөрөө үзсэнийг тоолохгүй
		}
		st := get(x.UserID, x.UserName)
		st.add(x)
		fs := focus[x.UserID]
		focus[x.UserID] = [2]float64{fs[0] + x.FocusSum, fs[1] + float64(x.FocusN)}
		lessons[x.UserID][x.LessonID] = true
		d := x.StartedAt.Local().Format("2006-01-02")
		da := dayActive[d]
		dayActive[d] = [2]int{da[0] + x.ActiveSec, da[1] + x.IdleSec + x.AwaySec}
		if x.ActiveSec > 0 {
			if userDays[x.UserID] == nil {
				userDays[x.UserID] = map[string]bool{}
			}
			userDays[x.UserID][d] = true
		}
	}
	for _, l := range quizLogs {
		if l.UserID == teacherID {
			continue
		}
		st := get(l.UserID, l.UserName)
		st.QuizTotal++
		st.QuizAvgMs += l.Ms
		if l.Correct {
			st.QuizCorrect++
		}
		if l.Ms > 0 && l.Ms < 1500 {
			st.QuizGuesses++
		}
	}
	for _, rf := range refs {
		if rf.UserID == teacherID {
			continue
		}
		st := get(rf.UserID, rf.UserName)
		st.Reflections++
		st.ReflectionWords += rf.Words
	}
	for _, v := range watches {
		if v.UserID == teacherID || v.Duration <= 0 {
			continue
		}
		st := get(v.UserID, v.UserName)
		total := (v.Duration + store.VideoBucketSec - 1) / store.VideoBucketSec
		if total == 0 {
			continue
		}
		st.VideoClips++
		st.VideoCoverage += min(100, len(v.Buckets)*100/total)
	}
	for uid, days := range userDays {
		st := get(uid, "")
		st.ActiveDays = len(days)
		streak, d := 0, time.Now()
		for {
			if !days[d.Format("2006-01-02")] {
				if d.Format("2006-01-02") == time.Now().Format("2006-01-02") {
					d = d.AddDate(0, 0, -1) // өнөөдөр хараахан ороогүй ч өчигдрөөс тоолно
					continue
				}
				break
			}
			streak++
			d = d.AddDate(0, 0, -1)
		}
		st.StreakDays = streak
	}
	for _, a := range atts {
		if a.UserID == teacherID {
			continue
		}
		st := get(a.UserID, a.UserName)
		st.ExamCount++
		if a.Status == store.AttemptTerminated {
			st.Terminated++
		}
		if a.Status != store.AttemptActive && a.Pct > st.ExamBest {
			st.ExamBest = a.Pct
		}
	}
	out := &analyticsData{Students: []*StudentStat{}, Daily: []map[string]any{}, Events: []store.ActivityEvent{}}
	var tot StudentStat
	tot.Counts = map[string]int{}
	live := 0
	for uid, st := range by {
		st.Lessons = len(lessons[uid])
		st.finish(focus[uid][0], int(focus[uid][1]))
		out.Students = append(out.Students, st)
		tot.ActiveSec += st.ActiveSec
		tot.IdleSec += st.IdleSec
		tot.AwaySec += st.AwaySec
		tot.Violations += st.Violations
		if st.Live {
			live++
		}
	}
	slices.SortFunc(out.Students, func(a, b *StudentStat) int { return b.LastAt.Compare(a.LastAt) })
	for i := days - 1; i >= 0; i-- {
		d := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
		v := dayActive[d]
		out.Daily = append(out.Daily, map[string]any{"day": d, "active_min": v[0] / 60, "inactive_min": v[1] / 60, "active_sec": v[0], "inactive_sec": v[1]})
	}
	for _, e := range evs {
		if e.UserID != teacherID || e.Type == "teacher_remind" {
			out.Events = append(out.Events, e)
		}
	}
	total := tot.ActiveSec + tot.IdleSec + tot.AwaySec
	activePct, attention, learn, reflections := 0, 0, 0, 0
	for _, st := range out.Students {
		attention += st.Attention
		learn += st.LearnScore
		reflections += st.Reflections
	}
	if total > 0 {
		activePct = int(math.Round(float64(tot.ActiveSec) / float64(total) * 100))
	}
	if n := len(out.Students); n > 0 {
		attention /= n
		learn /= n
	}
	out.Totals = map[string]any{"students": len(out.Students), "live": live, "active_sec": tot.ActiveSec, "total_sec": total, "active_pct": activePct,
		"attention": attention, "violations": tot.Violations, "exams": len(atts), "learn_score": learn, "reflections": reflections}
	return out, ss, atts, nil
}

func analyticsDays(r *http.Request) int {
	d, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if d <= 0 || d > 365 {
		return 30
	}
	return d
}

// handleAnalytics: GET /api/me/analytics?course=&days=
func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	data, _, _, err := s.buildAnalytics(r.Context(), c.UID, r.URL.Query().Get("course"), "", analyticsDays(r))
	if s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "labels": eventLabels()})
}

// handleStudentAnalytics: GET /api/me/analytics/students/{uid}?course=&days=
func (s *Server) handleStudentAnalytics(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	uid := r.PathValue("uid")
	days := analyticsDays(r)
	courseID := r.URL.Query().Get("course")
	data, ss, atts, err := s.buildAnalytics(r.Context(), c.UID, courseID, uid, days)
	if s.storeErr(w, r, err) {
		return
	}
	f := store.ActivityFilter{TeacherID: c.UID, CourseID: courseID, UserID: uid, Since: time.Now().AddDate(0, 0, -days)}
	quizLogs, err := s.store.QuizLogs(r.Context(), f, 300)
	if s.storeErr(w, r, err) {
		return
	}
	refs, err := s.store.Reflections(r.Context(), f, 100)
	if s.storeErr(w, r, err) {
		return
	}
	watches, err := s.store.VideoWatches(r.Context(), f)
	if s.storeErr(w, r, err) {
		return
	}
	titleByLesson := map[string]string{}
	for _, x := range ss {
		titleByLesson[x.LessonID] = x.Title
	}
	// Видео хэсэг бүрийг хэдэн удаа үзсэн — алгассан ба давтаж үзсэн хэсгийг харуулна.
	type watchView struct {
		LessonID string `json:"lesson_id"`
		Lesson   string `json:"lesson"`
		Duration int    `json:"duration"`
		Buckets  []int  `json:"buckets"`
	}
	watchRows := make([]watchView, len(watches))
	for i, v := range watches {
		total := (v.Duration + store.VideoBucketSec - 1) / store.VideoBucketSec
		row := make([]int, total)
		for k, n := range v.Buckets {
			if k < total {
				row[k] = n
			}
		}
		watchRows[i] = watchView{LessonID: v.LessonID, Lesson: titleByLesson[v.LessonID], Duration: v.Duration, Buckets: row}
	}
	// Хичээл тус бүрийн нэгтгэл.
	type lessonRow struct {
		LessonID  string `json:"lesson_id"`
		Title     string `json:"title"`
		CourseID  string `json:"course_id"`
		ActiveSec int    `json:"active_sec"`
		TotalSec  int    `json:"total_sec"`
		Sessions  int    `json:"sessions"`
		Violation int    `json:"violations"`
	}
	rows := map[string]*lessonRow{}
	var order []string
	for _, x := range ss {
		lr := rows[x.LessonID]
		if lr == nil {
			lr = &lessonRow{LessonID: x.LessonID, Title: x.Title, CourseID: x.CourseID}
			rows[x.LessonID] = lr
			order = append(order, x.LessonID)
		}
		lr.Sessions++
		lr.ActiveSec += x.ActiveSec
		lr.TotalSec += x.ActiveSec + x.IdleSec + x.AwaySec
		for k, v := range x.Counts {
			if eventInfo[k].Violation {
				lr.Violation += v
			}
		}
	}
	perLesson := make([]*lessonRow, 0, len(order))
	for _, id := range order {
		perLesson = append(perLesson, rows[id])
	}
	if len(ss) > 100 {
		ss = ss[:100]
	}
	var st *StudentStat
	if len(data.Students) > 0 {
		st = data.Students[0]
	}
	writeJSON(w, http.StatusOK, map[string]any{"student": st, "sessions": ss, "lessons": perLesson, "exams": atts, "events": data.Events, "daily": data.Daily, "labels": eventLabels(),
		"quiz_logs": quizLogs, "reflections": refs, "watches": watchRows})
}

func eventLabels() map[string]string {
	out := make(map[string]string, len(eventInfo))
	for k, v := range eventInfo {
		out[k] = v.Label
	}
	return out
}

// handleAnalyticsExport: GET /api/me/analytics/export?course=&student=&days= — Excel-д нээгдэх CSV (UTF-8 BOM).
func (s *Server) handleAnalyticsExport(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	data, ss, atts, err := s.buildAnalytics(r.Context(), c.UID, q.Get("course"), q.Get("student"), analyticsDays(r))
	if s.storeErr(w, r, err) {
		return
	}
	name := "angi-tailan"
	if q.Get("student") != "" {
		name = "suragch-tailan"
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.csv"`, name, time.Now().Format("2006-01-02")))
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM: Excel монгол үсгийг зөв уншина
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Суралцагч", "Нийт хугацаа (мин)", "Идэвхтэй (мин)", "Идэвхгүй (мин)", "Өөр цонхонд (мин)", "Идэвхтэй хувь %", "Анхаарлын индекс %",
		"Зөрчил", "Таб шилжсэн", "Хуулах", "Автоматаар зогссон", "Хичээл", "Шалгалтын шилдэг %", "Хаагдсан шалгалт",
		"Асуултад зөв хариулсан %", "Дундаж хариулах хугацаа (сек)", "Таамагласан хариулт", "Дүгнэлт бичсэн тоо", "Видео үзэлтийн бүрэн байдал %",
		"Тогтмол ирсэн өдөр", "Суралцсан оноо", "Эрсдэл", "Сүүлд"})
	risk := map[string]string{"ok": "Хэвийн", "watch": "Анхаарах", "risk": "Эрсдэлтэй"}
	for _, st := range data.Students {
		best := ""
		if st.ExamBest >= 0 {
			best = strconv.Itoa(st.ExamBest)
		}
		accuracy := ""
		if st.QuizTotal > 0 {
			accuracy = strconv.Itoa(st.QuizAccuracy)
		}
		_ = cw.Write([]string{st.Name, strconv.Itoa(st.TotalSec / 60), strconv.Itoa(st.ActiveSec / 60), strconv.Itoa(st.IdleSec / 60), strconv.Itoa(st.AwaySec / 60),
			strconv.Itoa(st.ActivePct), strconv.Itoa(st.Attention), strconv.Itoa(st.Violations), strconv.Itoa(st.Counts["tab_switch"]), strconv.Itoa(st.Counts["copy"]),
			strconv.Itoa(st.Counts["auto_block"]), strconv.Itoa(st.Lessons), best, strconv.Itoa(st.Terminated),
			accuracy, strconv.Itoa(st.QuizAvgMs / 1000), strconv.Itoa(st.QuizGuesses), strconv.Itoa(st.Reflections), strconv.Itoa(st.VideoCoverage),
			strconv.Itoa(st.StreakDays), strconv.Itoa(st.LearnScore), risk[st.Risk], st.LastAt.Local().Format("2006-01-02 15:04")})
	}
	if q.Get("student") != "" { // суралцагчийн дэлгэрэнгүй: сесс, шалгалт, лог
		_ = cw.Write(nil)
		_ = cw.Write([]string{"Хичээл", "Эхэлсэн", "Идэвхтэй (мин)", "Нийт (мин)", "Төлөв"})
		for _, x := range ss {
			end := "Үргэлжилж байна"
			if x.Ended {
				end = eventLabel(x.EndReason)
			}
			_ = cw.Write([]string{x.Title, x.StartedAt.Local().Format("2006-01-02 15:04"), strconv.Itoa(x.ActiveSec / 60), strconv.Itoa((x.ActiveSec + x.IdleSec + x.AwaySec) / 60), end})
		}
		_ = cw.Write(nil)
		_ = cw.Write([]string{"Шалгалт", "Огноо", "Оноо %", "Төлөв", "Зөрчил"})
		for _, a := range atts {
			_ = cw.Write([]string{a.LessonID, a.StartedAt.Local().Format("2006-01-02 15:04"), strconv.Itoa(a.Pct), a.Status + " " + a.Reason, strconv.Itoa(a.Violations)})
		}
		_ = cw.Write(nil)
		_ = cw.Write([]string{"Цаг", "Үйл явдал", "Дэлгэрэнгүй"})
		for _, e := range data.Events {
			_ = cw.Write([]string{e.At.Local().Format("2006-01-02 15:04:05"), eventLabel(e.Type), e.Detail})
		}
	}
	cw.Flush()
}
