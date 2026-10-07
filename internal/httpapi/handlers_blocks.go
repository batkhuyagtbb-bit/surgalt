package httpapi

import (
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Хичээлийн блокуудын хязгаар.
const (
	maxBlocks       = 120
	maxTextRunes    = 20_000
	maxQuizOptions  = 8
	maxBlockURL     = 1000
	maxBlocksTotalB = 400_000 // нийт текстийн хэмжээ (байт)
)

var blockIDRe = regexp.MustCompile(`^[a-z0-9]{4,16}$`)

// ValidBlockID — агуулгын хэсгийн ID зөв эсэх (seed, импорт шалгахад).
func ValidBlockID(id string) bool { return blockIDRe.MatchString(id) }

// fixBlockID — хуучин өгөгдөл, импортоос ирсэн буруу (богино, том үсэг, тэмдэгттэй) эсвэл давхардсан ID-г
// зөв ID болгоно. Тогтвортой: ижил дараалалд ижил ID гарна, тиймээс дахин хадгалахад өөрчлөгдөхгүй.
func fixBlockID(id string, seen map[string]bool) string {
	if blockIDRe.MatchString(id) && !seen[id] {
		return id
	}
	c := strings.Map(func(r rune) rune {
		r = unicode.ToLower(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return -1
	}, id)
	for len(c) < 4 {
		c += "0"
	}
	if len(c) > 16 {
		c = c[:16]
	}
	base := c
	for i := 1; seen[c]; i++ {
		suf := strconv.Itoa(i)
		c = base[:min(len(base), 16-len(suf))] + suf
	}
	return c
}

// validateBlocks нь блокуудыг шалгаж, хоосон зайг цэвэрлэнэ. Алдаа бол монгол мессеж буцаана.
func validateBlocks(bs []store.Block, teacherID string) string {
	if len(bs) > maxBlocks {
		return "агуулга хэт олон хэсэгтэй (дээд тал нь 120)"
	}
	seen := map[string]bool{}
	total := 0
	for i := range bs {
		b := &bs[i]
		b.Parts = nil                 // сервер тооцоолдог талбар
		b.ID = fixBlockID(b.ID, seen) // буруу ID-тай хуучин хичээлийг ч хадгалж болно
		seen[b.ID] = true
		if !store.BlockTypes[b.Type] {
			return "агуулгын төрөл буруу: " + b.Type
		}
		b.Name = strings.TrimSpace(b.Name)
		if utf8.RuneCountInString(b.Name) > 200 || b.Size < 0 {
			return "файлын нэр хэт урт"
		}
		switch b.Type {
		case "text":
			b.URL, b.Quiz = "", nil
			if strings.TrimSpace(b.Text) == "" {
				return "хоосон текстийн хэсэг байна — бичих эсвэл устгана уу"
			}
			if utf8.RuneCountInString(b.Text) > maxTextRunes {
				return "нэг текстийн хэсэг 20,000 тэмдэгтээс хэтрэхгүй"
			}
		case "embed": // HTML embed: суралцагчид sandbox iframe-д (эх хуудастай холбоогүй) харагдана
			b.URL, b.Quiz = "", nil
			b.Text = strings.TrimSpace(b.Text)
			if b.Text == "" || len(b.Text) > 60_000 {
				return "HTML embed 1-60,000 тэмдэгт"
			}
			if b.Height == 0 {
				b.Height = 420
			}
			if b.Height < 80 || b.Height > 2000 {
				return "embed-ийн өндөр 80-2000 px"
			}
		case "heading":
			b.URL, b.Quiz = "", nil
			b.Text = strings.TrimSpace(b.Text)
			if b.Text == "" || utf8.RuneCountInString(b.Text) > 200 {
				return "гарчиг 1-200 тэмдэгт"
			}
		case "image", "audio", "video", "file":
			b.Quiz = nil
			b.URL = stripSig(strings.TrimSpace(b.URL))
			if b.URL == "" {
				return "файл хавсаргаагүй хэсэг байна — файлаа оруулах эсвэл устгана уу"
			}
			if len(b.URL) > maxBlockURL || (!validURL(b.URL) && !files.OwnedBy(b.URL, teacherID)) {
				return "медиа: http(s) холбоос эсвэл өөрийн файлын сангийн файл"
			}
			if utf8.RuneCountInString(b.Text) > 500 {
				return "тайлбар 500 тэмдэгтээс хэтрэхгүй"
			}
		case "quiz":
			b.URL, b.Text = "", ""
			if b.Quiz == nil {
				return "асуулт хоосон байна"
			}
			if msg := validateQuiz(b.Quiz, teacherID); msg != "" {
				return msg
			}
		}
		total += len(b.Text) + len(b.Name) + len(b.URL)
		if b.Quiz != nil {
			total += len(b.Quiz.Question) + len(b.Quiz.Explain)
			for _, o := range slices.Concat(b.Quiz.Options, b.Quiz.Answers, b.Quiz.Left, b.Quiz.Right) {
				total += len(o)
			}
		}
	}
	if total > maxBlocksTotalB {
		return "хичээлийн агуулга хэт их — хоёр хичээл болгон хуваана уу"
	}
	return ""
}

// viewerBlocks: суралцагчид харуулах хуулбар — хувийн файлын холбоосыг гарын үсэгтэй болгож,
// асуултын зөв хариулт ба тайлбарыг хасна (хариулсны дараа /quiz-ээс ирнэ).
func (s *Server) viewerBlocks(bs []store.Block) []store.Block {
	out := make([]store.Block, len(bs))
	for i, b := range bs {
		if b.URL != "" {
			if parts := s.files.Parts(b.URL); b.Type == "video" && len(parts) > 1 { // 6 минутын хэсгүүд дараалан
				b.Parts = make([]string, len(parts))
				for k, p := range parts {
					b.Parts[k] = s.viewerMedia(p, "")
				}
			}
			if b.Type == "file" && b.Download {
				b.URL = s.media(b.URL) // багш татахыг зөвшөөрсөн: гарын үсэгтэй URL (татаж болно)
			} else {
				b.URL = s.viewerMedia(b.URL, "") // видео/аудио/баримт → нуусан тасалбар (зөвхөн хичээл дотроос)
			}
		}
		if b.Quiz != nil {
			b.Quiz = s.viewerQuiz(b.ID, b.Quiz)
		}
		out[i] = b
	}
	return out
}

// handleAnswerQuiz: POST /api/courses/{id}/lessons/{lid}/quiz/{bid} {"answer":[0,2]}
// Хичээлийг үзэх эрхтэй хүн хариулна. Нэвтэрсэн бол үр дүн явцад хадгалагдана.
func (s *Server) handleAnswerQuiz(w http.ResponseWriter, r *http.Request) {
	course, err := s.store.CourseByID(r.Context(), r.PathValue("id"))
	if s.storeErr(w, r, err) {
		return
	}
	l, err := s.store.LessonByID(r.Context(), course.ID, r.PathValue("lid"))
	if s.storeErr(w, r, err) {
		return
	}
	p, logged := s.principal(r)
	uid := ""
	if logged && !p.IsGuest() {
		uid = p.UID
	}
	if hiddenFrom(l, course, uid) {
		s.apiErr(w, r, errLessonHidden)
		return
	}
	if !course.Published && course.TeacherID != uid {
		writeErr(w, http.StatusNotFound, "олдсонгүй")
		return
	}
	if !l.IsFree {
		if uid == "" {
			writeErr(w, http.StatusUnauthorized, "нэвтэрнэ үү")
			return
		}
		has, err := s.lessonAccess(r.Context(), uid, course, l)
		if s.storeErr(w, r, err) {
			return
		}
		if !has {
			writeErr(w, http.StatusPaymentRequired, "энэ хичээл төлбөртэй")
			return
		}
	}
	var q *store.Quiz
	bid := r.PathValue("bid")
	for _, b := range l.Blocks {
		if b.ID == bid && b.Quiz != nil {
			q = b.Quiz
		}
	}
	if q == nil {
		writeErr(w, http.StatusNotFound, "асуулт олдсонгүй")
		return
	}
	if l.Exam != nil {
		writeErr(w, http.StatusForbidden, "шалгалтын асуултыг дуусгаад нэг дор шалгана")
		return
	}
	var in struct {
		QuizAnswer
		Ms int `json:"ms"` // асуулт харагдсанаас хариулах хүртэл (таамаглалыг илрүүлэхэд)
	}
	if !decode(w, r, &in) {
		return
	}
	if in.empty() {
		writeErr(w, http.StatusBadRequest, "хариултаа сонгоно уу")
		return
	}
	correct := gradeQuiz(bid, q, in.QuizAnswer)
	if uid != "" && uid != course.TeacherID {
		if err := s.store.SaveQuizResult(r.Context(), uid, course.ID, l.ID, bid, correct); s.storeErr(w, r, err) {
			return
		}
		s.store.AddQuizLog(r.Context(), store.QuizLog{UserID: uid, UserName: s.displayName(r.Context(), uid, ""), CourseID: course.ID, LessonID: l.ID, TeacherID: course.TeacherID, BlockID: bid,
			Question: short(q.Question), Correct: correct, Ms: min(max(in.Ms, 0), 3600_000)})
	}
	out := revealQuiz(bid, q)
	out["correct"] = correct
	// Хичээлийн бүх асуултад зөв хариулсан эсэх — дараагийн хичээл үүнээс хамаарч нээгдэнэ.
	if uid != "" && uid != course.TeacherID {
		prog, err := s.store.LessonProgress(r.Context(), uid, course.ID)
		if s.storeErr(w, r, err) {
			return
		}
		var pp *store.LessonProgress
		if p, ok := prog[l.ID]; ok {
			pp = &p
		}
		got, total, done, _ := quizMastery(l, pp)
		if done && (pp == nil || pp.QuizDoneAt == nil) {
			if err := s.store.MarkQuizDone(r.Context(), uid, course.ID, l.ID); s.storeErr(w, r, err) {
				return
			}
		}
		out["quiz_total"], out["quiz_correct"], out["mastered"] = total, got, done
	}
	writeJSON(w, http.StatusOK, out)
}

// examIntro: шалгалт бол асуултуудыг хасна (зөвхөн "Шалгалт эхлүүлэх"-ээр ирнэ).
func examIntro(l *store.Lesson) []store.Block {
	if l.Exam == nil {
		return l.Blocks
	}
	out := []store.Block{}
	for _, b := range l.Blocks {
		if b.Type != "quiz" {
			out = append(out, b)
		}
	}
	return out
}

// handleVideoWatched: POST /api/courses/{id}/lessons/{lid}/watched/{bid} — видеог нэг удаа бүрэн үзсэн.
// Дараа нь гүйлгэх боломж нээгдэнэ (явцад "watch_<bid>" түлхүүрээр хадгална).
func (s *Server) handleVideoWatched(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	bid := r.PathValue("bid")
	if !blockIDRe.MatchString(bid) {
		writeErr(w, http.StatusBadRequest, "буруу хэсэг")
		return
	}
	course, l, ok := s.lessonForUser(w, r, c.UID, r.PathValue("id"), r.PathValue("lid"))
	if !ok {
		return
	}
	found := bid == "main" && l.VideoURL != ""
	for _, b := range l.Blocks {
		if b.ID == bid && b.Type == "video" {
			found = true
		}
	}
	if !found {
		writeErr(w, http.StatusNotFound, "видео олдсонгүй")
		return
	}
	if err := s.store.SaveQuizResult(r.Context(), c.UID, course.ID, l.ID, "watch_"+bid, true); s.storeErr(w, r, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"watched": true})
}

// handleFileUsage: GET /api/me/files/usage — файл бүр аль хичээлд ашиглагдаж буй (файлын санг цэгцлэхэд).
func (s *Server) handleFileUsage(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacher(w, r)
	if !ok {
		return
	}
	courses, err := s.store.CoursesByTeacher(r.Context(), c.UID, false)
	if s.storeErr(w, r, err) {
		return
	}
	type use struct {
		CourseID string `json:"course_id"`
		Course   string `json:"course"`
		LessonID string `json:"lesson_id"`
		Lesson   string `json:"lesson"`
		Section  string `json:"section,omitempty"`
		Position int    `json:"position,omitempty"`
	}
	out := map[string][]use{}
	add := func(path string, u use) {
		if i := strings.IndexByte(path, '?'); i >= 0 {
			path = path[:i]
		}
		if strings.HasPrefix(path, "/files/") {
			out[path] = append(out[path], u)
		}
	}
	for _, co := range courses {
		ls, err := s.store.LessonsByCourse(r.Context(), co.ID)
		if s.storeErr(w, r, err) {
			return
		}
		for _, l := range ls {
			u := use{co.ID, co.Title, l.ID, l.Title, l.Section, l.Position}
			add(l.VideoURL, u)
			for _, b := range l.Blocks {
				add(b.URL, u)
				if b.Quiz != nil {
					add(b.Quiz.Image, u)
				}
			}
		}
	}
	if bs, err := s.store.BooksByTeacher(r.Context(), c.UID, false); err == nil {
		for _, b := range bs {
			add(b.CoverURL, use{Course: "Ном", Lesson: b.Title})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// stripSig: өөрийн файлын замаас гарын үсэг (?exp=&sig=)-ийг хасна — DB-д зөвхөн зам хадгална.
func stripSig(u string) string {
	if strings.HasPrefix(u, "/files/") {
		if i := strings.IndexByte(u, '?'); i >= 0 {
			return u[:i]
		}
	}
	return u
}
