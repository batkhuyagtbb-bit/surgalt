package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	"surgalt/internal/store"
)

// ---- Сургалтын журнал: багшийн өгсөн нэр, бүлэг, архив, суралцахуйн өгөгдөл устгах ----
// Нэр (alias) нь зөвхөн тухайн сургалтын журналд; суралцагчийн бүртгэлийн нэр хэзээ ч өөрчлөгдөхгүй. Бүлэг нь сургалт
// бүрт далд (архивлаагүй дор хаяж нэг суралцагчид байвал бий). Архив нь журналаас нууж, өгөгдлийг хадгална. Устгал нь
// архивласан суралцагчийн энэ сургалт дахь суралцахуйн өгөгдлийг бүрмөсөн арилгана (элсэлт, худалдан авалт хэвээр) —
// зөвхөн сургалтын эзэмшигч. Нэр, бүлэг, архив хэзээ ч суралцагчид харагдахгүй.

const (
	journalAliasMax  = 80
	journalGroupMax  = 40
	journalGroupsMax = 100
	// NotifCourseReset — багш суралцагчийн сургалтын явцыг шинэчилсэн (өгөгдлийг устгасан).
	NotifCourseReset = "course_reset"
)

// JournalStudentMeta — засварын хариуны суралцагч (журналын мөрийн нэр, бүлэг, архивын хэсэг).
type JournalStudentMeta struct {
	UserID      string     `json:"user_id"`
	Name        string     `json:"name"`
	AccountName string     `json:"account_name"`
	Alias       string     `json:"alias,omitempty"`
	Group       string     `json:"group,omitempty"`
	Archived    bool       `json:"archived,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

// journalMsgs — нэр/бүлгийн нэрийн алдааны мессежүүд.
type journalMsgs struct{ bad, long, letter string }

var (
	aliasMsgs = journalMsgs{"нэрэнд зөвшөөрөгдөөгүй тэмдэгт байна", "нэр 80 тэмдэгтээс хэтрэхгүй байх ёстой", "нэрэнд дор хаяж нэг үсэг эсвэл тоо байх ёстой"}
	groupMsgs = journalMsgs{"бүлгийн нэрэнд зөвшөөрөгдөөгүй тэмдэгт байна", "бүлгийн нэр 40 тэмдэгтээс хэтрэхгүй байх ёстой", "бүлгийн нэрэнд дор хаяж нэг үсэг эсвэл тоо байх ёстой"}
)

// mongolScript — уламжлалт монгол бичгийн блок (U+1800–U+18AF).
func mongolScript(r rune) bool { return r >= 0x1800 && r <= 0x18AF }

// journalBadRune — нэрэнд хориглох тэмдэгт: удирдах, формат (Cf), үл харагдах дүүргэгч. Монгол бичгийн эгшиг тусгаарлагч
// (U+180E), ZWNJ/ZWJ (U+200C/D) нь хажуудаа монгол бичгийн үсэгтэй үед л зөвшөөрөгдөнө (хувилбар сонгогч U+180B–180D,
// U+180F нь Cf биш тул үргэлж зөвшөөрөгдөнө); бусад бичигт тэдгээр нь нэрийг үл ялгагдах болгодог.
func journalBadRune(rs []rune, i int) bool {
	r := rs[i]
	switch {
	case r == 0x180E || r == 0x200C || r == 0x200D:
		return !((i > 0 && mongolScript(rs[i-1])) || (i+1 < len(rs) && mongolScript(rs[i+1])))
	case r >= 0x180B && r <= 0x180F:
		return false
	case r == 0x3164 || r == 0x115F || r == 0x1160 || r == 0xFFA0 || r == 0x2800 || r == 0x034F ||
		(r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF): // хангыл дүүргэгч, хоосон брайль, CGJ, хувилбар сонгогч
		return true
	}
	return unicode.IsControl(r) || unicode.Is(unicode.Cf, r)
}

// journalText — нэр/бүлгийн нэрийг шалгаж цэвэрлэнэ: NFC хэлбэрт оруулж, хориглосон тэмдэгт (journalBadRune) байвал
// m.bad; хоосон зайг нэгтгэсний дараа maxLen тэмдэгтээс урт бол m.long; хоосон биш боловч үсэг, тоо огт үгүй бол m.letter.
func journalText(raw string, maxLen int, m journalMsgs) (string, string) {
	rs := []rune(norm.NFC.String(raw))
	for i := range rs {
		if journalBadRune(rs, i) {
			return "", m.bad
		}
	}
	v := strings.Join(strings.Fields(string(rs)), " ")
	if utf8.RuneCountInString(v) > maxLen {
		return "", m.long
	}
	if v != "" && !strings.ContainsFunc(v, func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }) {
		return "", m.letter
	}
	return v, ""
}

// journalName — шүүлт, бүлгийн хайлтад: NFC, хоосон зай нэгтгэсэн (хадгалсан нэртэй ижил хэлбэр).
func journalName(raw string) string { return strings.Join(strings.Fields(norm.NFC.String(raw)), " ") }

// journalGroupName — бүлгийн шинэ нэр: өөр суралцагчийн байгаа бүлэгтэй (том жижиг үсэг үл харгалзан) таарвал тэр
// бүлгийн бичлэг; шинэ бүлэг нь сургалтын 101 дэх болох бол алдаа.
func journalGroupName(name, uid string, all map[string]store.CourseStudentMeta) (string, error) {
	if name == "" {
		return "", nil
	}
	seen := map[string]bool{}
	for id, m := range all {
		if id == uid || m.Group == "" {
			continue
		}
		if strings.EqualFold(m.Group, name) {
			return m.Group, nil
		}
		seen[strings.ToLower(m.Group)] = true
	}
	if len(seen) >= journalGroupsMax {
		return "", apiError(http.StatusBadRequest, "нэг сургалтад 100 хүртэл бүлэг үүсгэх боломжтой", nil)
	}
	return name, nil
}

// journalMember — журналд харагдах суралцагч: элссэн, хичээл худалдаж авсан, явцтай эсвэл хичээл үзэж байсан.
func (s *Server) journalMember(ctx context.Context, uid, courseID string) (bool, error) {
	if ok, err := s.courseMember(ctx, uid, courseID); err != nil || ok {
		return ok, err
	}
	prog, err := s.store.LessonProgress(ctx, uid, courseID)
	if err != nil || len(prog) > 0 {
		return len(prog) > 0, err
	}
	ss, err := s.store.Sessions(ctx, store.ActivityFilter{UserID: uid, CourseID: courseID}, 1)
	return len(ss) > 0, err
}

// journalCounts — бүх суралцагчаар тоолсон бүлэг, бүлэггүй, архивын тоог хариунд нэмнэ. jr nil бол кэшээс (эсвэл тооцож)
// авна. Тоог тооцож чадахгүй бол лог бичээд хоосон утга өгнө (засвар аль хэдийн хадгалагдсан).
func (s *Server) journalCounts(ctx context.Context, course *store.Course, jr *journalAgg, metas map[string]store.CourseStudentMeta, out map[string]any) {
	if jr == nil {
		var err error
		if jr, err = s.journal(ctx, course); err != nil {
			s.log.Warn("журналын тоо", "err", err, "course", course.ID)
			out["groups"], out["ungrouped"], out["archived"] = []JournalGroup{}, 0, 0
			return
		}
	}
	v := journalPage(jr, metas, journalFilter{}, 0)
	out["groups"], out["ungrouped"], out["archived"] = v.groups, v.ungrouped, v.archived
}

// handleJournalStudent: PUT /api/me/courses/{id}/journal/students/{uid} — {"alias"?, "group"?, "archived"?} (хэсэгчилсэн).
func (s *Server) handleJournalStudent(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		Alias    *string `json:"alias"`
		Group    *string `json:"group"`
		Archived *bool   `json:"archived"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Alias == nil && in.Group == nil && in.Archived == nil {
		writeErr(w, http.StatusBadRequest, "өөрчлөх талбар алга")
		return
	}
	ctx, uid := r.Context(), r.PathValue("uid")
	if uid == course.TeacherID || slices.Contains(course.Teachers, uid) {
		writeErr(w, http.StatusBadRequest, "сургалтын багшийг журналд засах боломжгүй")
		return
	}
	member, err := s.journalMember(ctx, uid, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if !member {
		writeErr(w, http.StatusNotFound, "энэ суралцагч таны сургалтад байхгүй байна")
		return
	}
	var alias, group, msg string
	if in.Alias != nil {
		if alias, msg = journalText(*in.Alias, journalAliasMax, aliasMsgs); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
	}
	if in.Group != nil {
		if group, msg = journalText(*in.Group, journalGroupMax, groupMsgs); msg != "" {
			writeErr(w, http.StatusBadRequest, msg)
			return
		}
	}
	account := ""
	if u, err := s.store.UserByID(ctx, uid); err == nil {
		account = u.DisplayName
	}
	if alias == journalName(account) { // бүртгэлийн нэртэй ижил бол тусдаа нэр хадгалахгүй
		alias = ""
	}
	var metas map[string]store.CourseStudentMeta
	cur, err := s.store.UpdateCourseStudentMeta(ctx, course.ID, uid, c.UID, func(cur *store.CourseStudentMeta, all map[string]store.CourseStudentMeta) error {
		metas = all
		if in.Alias != nil {
			cur.Alias = alias
		}
		if in.Group != nil {
			g, err := journalGroupName(group, uid, all)
			if err != nil {
				return err
			}
			cur.Group = g
		}
		if in.Archived != nil {
			switch {
			case *in.Archived && !cur.Archived:
				now := time.Now().UTC()
				cur.Archived, cur.ArchivedAt = true, &now
			case !*in.Archived:
				cur.Archived, cur.ArchivedAt = false, nil
			}
		}
		return nil
	})
	if s.apiErr(w, r, err) {
		return
	}
	if metas == nil {
		metas = map[string]store.CourseStudentMeta{}
	}
	metas[uid] = *cur
	if account == "" {
		account = "Суралцагч"
	}
	st := JournalStudentMeta{UserID: uid, Name: account, AccountName: account, Alias: cur.Alias, Group: cur.Group, Archived: cur.Archived, ArchivedAt: cur.ArchivedAt}
	if cur.Alias != "" {
		st.Name = cur.Alias
	}
	out := map[string]any{"student": st}
	s.journalCounts(ctx, course, nil, metas, out)
	writeJSON(w, http.StatusOK, out)
}

// handleJournalGroups: PUT /api/me/courses/{id}/journal/groups — {"from":"5А","to":"5Б"} бүлгийн нэр солих; to="" бол тараана.
func (s *Server) handleJournalGroups(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	var in struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if !decode(w, r, &in) {
		return
	}
	to, msg := journalText(in.To, journalGroupMax, groupMsgs)
	if msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	from := journalName(in.From)
	if from == "" {
		writeErr(w, http.StatusBadRequest, "бүлгийн нэрээ оруулна уу")
		return
	}
	ctx := r.Context()
	changed, err := s.store.RenameCourseGroup(ctx, course.ID, from, to, c.UID)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "ийм бүлэг олдсонгүй — хуудсаа сэргээнэ үү")
		return
	}
	if s.storeErr(w, r, err) {
		return
	}
	metas, err := s.store.CourseStudentMetas(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	out := map[string]any{"changed": changed}
	s.journalCounts(ctx, course, nil, metas, out)
	writeJSON(w, http.StatusOK, out)
}

// handleJournalReset: POST /api/me/courses/{id}/journal/students/{uid}/reset — {"confirm":true}. Архивласан суралцагчийн
// энэ сургалт дахь суралцахуйн өгөгдлийг бүрмөсөн устгана (зөвхөн эзэмшигч). Элсэлт, худалдан авалт хэвээр; суралцагч
// журналд «эхлээгүй» байдлаар буцаж харагдана. Ачааллаас хамгаалалт: эзэмшигч тус бүрт минутад 1 (3 хүртэл дараалан),
// нэг (сургалт, суралцагч)-д 10 минутад 1, процесс дотор нэг удаад нэг устгал.
func (s *Server) handleJournalReset(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requireTeacherAny(w, r)
	if !ok {
		return
	}
	course, ok := s.teachCourse(w, r, c.UID)
	if !ok {
		return
	}
	if c.UID != course.TeacherID { // байгууллагын удирдагч байгууллагын нэрээр (токен нь байгууллагынх) орвол эзэмшигч
		writeErr(w, http.StatusForbidden, "зөвхөн сургалтын эзэмшигч өгөгдөл устгах эрхтэй")
		return
	}
	var in struct {
		Confirm bool `json:"confirm"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !in.Confirm {
		writeErr(w, http.StatusBadRequest, "устгахыг баталгаажуулна уу")
		return
	}
	uid := r.PathValue("uid")
	if uid == course.TeacherID || slices.Contains(course.Teachers, uid) {
		writeErr(w, http.StatusBadRequest, "сургалтын багшийг журналд засах боломжгүй")
		return
	}
	ctx := r.Context()
	// Урьдчилсан шалгалт (store түгжээн дор дахин шалгана): ажил хийгдэхгүй 409/429 нь давтамжийн хязгаарт тооцогдохгүй,
	// архивлаагүй суралцагчийн санах ойн самбарыг хөндөхгүй.
	metas, err := s.store.CourseStudentMetas(ctx, course.ID)
	if s.storeErr(w, r, err) {
		return
	}
	if m, ok := metas[uid]; !ok || !m.Archived {
		writeErr(w, http.StatusConflict, msgResetNotArchived)
		return
	} else if m.ResetAt != nil && time.Since(*m.ResetAt) < store.ResetCooldown {
		writeErr(w, http.StatusTooManyRequests, msgResetTooSoon)
		return
	}
	if !s.wipeLim.Allow(c.UID) {
		writeErr(w, http.StatusTooManyRequests, "хэт олон хүсэлт ирлээ — түр хүлээгээд дахин оролдоно уу")
		return
	}
	// Хүсэлт тасарсан ч устгал дуусна; HTTP WriteTimeout (30 сек)-оос өмнө хариулна. Хугацаа хэтэрвэл суралцагч
	// архивтаа үлдэж, дахин оролдоход аюулгүй (устгал давтагдахад ижил үр дүн).
	sctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 25*time.Second)
	defer cancel()
	// Процесс дотор нэг удаад нэг устгал (хүнд mutation, сессийн буферийн зогсолт). Хүлээлт ≤ wipeWait: өөр устгалын
	// ард удаан хүлээвэл үлдсэн хугацаа устгалд хүрэлцэхгүй болж хагас устгал үүснэ — тэгснээс юу ч хийлгүй 503.
	wait := time.NewTimer(wipeWait)
	defer wait.Stop()
	select {
	case s.wipeSem <- struct{}{}:
	case <-wait.C:
		writeErr(w, http.StatusServiceUnavailable, msgResetSlow)
		return
	case <-ctx.Done():
		return
	}
	held := true
	defer func() { // panic-ийн үед ч дараагийн устгалыг хаахгүй
		if held {
			<-s.wipeSem
		}
	}()
	ip := s.clientIP(r)
	lessons, lerr := s.store.LessonsByCourse(sctx, course.ID)
	if lerr != nil {
		s.log.Warn("журнал: устгалын хичээлүүд", "err", lerr, "course", course.ID)
	}
	s.boardForget(lessons, uid) // DELETE-ээс өмнө: flushBoards санах ойн хуучин хувийн самбарыг дахин бичихгүй
	res, err := s.store.ResetCourseStudent(sctx, course.ID, uid, c.UID)
	timedOut := sctx.Err() != nil
	<-s.wipeSem
	held = false
	switch {
	case errors.Is(err, store.ErrConflict):
		writeErr(w, http.StatusConflict, msgResetNotArchived)
		return
	case errors.Is(err, store.ErrResetTooSoon):
		writeErr(w, http.StatusTooManyRequests, msgResetTooSoon)
		return
	case errors.Is(err, store.ErrResetNoTime): // юу ч устгаагүй
		s.log.Warn("журнал: устгалд хугацаа хүрэлцсэнгүй", "course", course.ID, "user", uid, "actor", c.UID, "ip", ip)
		writeErr(w, http.StatusServiceUnavailable, msgResetSlow)
		return
	}
	actx, acancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer acancel()
	// Амжилттай эсвэл хагас дууссан: кэшүүдийг үргэлж цэвэрлэнэ. Тоог хуучин нэгтгэлээр (устгасан мөрийг built-ээр
	// танина) тооцоод хаяна; нэгтгэл кэшэд байхгүй бол шинээр тооцсон нь цэвэр.
	jr, _ := s.journals.Peek(course.ID)
	if lerr != nil {
		lessons, _ = s.store.LessonsByCourse(actx, course.ID)
	}
	s.resetForget(course, lessons, uid)
	s.boardMetricsForget(actx, lessons, uid) // устгах өгөгдөлгүй (Wiped=false) үед ч
	if err != nil {
		s.log.Warn("журнал: суралцагчийн өгөгдлийн устгал амжилтгүй", "course", course.ID, "user", uid, "actor", c.UID, "ip", ip, "err", err)
		switch {
		case timedOut || errors.Is(err, context.DeadlineExceeded):
			writeErr(w, http.StatusServiceUnavailable, msgResetSlow)
		case errors.Is(err, store.ErrResetPartial):
			writeErr(w, http.StatusServiceUnavailable, "устгал бүрэн дуусаагүй — дахин оролдоно уу")
		default:
			s.storeErr(w, r, err)
		}
		return
	}
	s.log.Info("журнал: суралцагчийн өгөгдлийг устгав", "course", course.ID, "user", uid, "actor", c.UID, "ip", ip, "wiped", res.Wiped)

	if res.Wiped { // устгах юм байгаагүй бол (зөвхөн архив, нэр арилсан) мэдэгдэхгүй
		s.notify(actx, &store.Notification{UserID: uid, Type: NotifCourseReset, ActorID: c.UID, Title: "🔄 Сургалтын явц шинээр эхэллээ",
			Body: short("«" + course.Title + "» сургалтын явцыг багш тань шинэчиллээ. Хичээлээ эхнээс нь үзээрэй."), Link: "/c/" + course.ID})
	}
	out := map[string]any{"ok": true, "reset_at": res.At}
	metas, err = s.store.CourseStudentMetas(actx, course.ID)
	if err != nil {
		s.log.Warn("журналын мэдээлэл", "err", err, "course", course.ID)
		metas = map[string]store.CourseStudentMeta{}
	}
	s.journalCounts(actx, course, jr, metas, out)
	writeJSON(w, http.StatusOK, out)
}

// wipeWait — өөр устгал дуусахыг хүлээх дээд хугацаа (үлдсэн ≥22 сек нь устгалд).
const wipeWait = 3 * time.Second

// boardMetricsForget — сангийн "_m" (самбарын оролцооны хэмжүүр: байсан, идэвхтэй хугацаа, засвар, чат, ярьсан) мөрүүдээс
// суралцагчийг хасна — эс бөгөөс addBoardStats багшийн хяналтад түүнийг харуулсаар байна. Санах ойд ачаалсан самбарыг
// алгасна (boardForget хассан, flushBoards хадгална). h.mu-г барьж байхад самбар ачаалагдахгүй (boardSession сангаас
// h.mu дор уншдаг), ачаалаагүй самбарыг flushBoards бичдэггүй тул түгжээн дор дахин уншаад бичихэд уралдаан үүсэхгүй,
// бусдын шинэ хэмжүүрийг хуучин хуулбараар дарахгүй.
func (s *Server) boardMetricsForget(ctx context.Context, lessons []store.Lesson, uid string) {
	var ids []string
	for i := range lessons {
		if slices.ContainsFunc(lessons[i].Blocks, func(b store.Block) bool { return b.Type == "board" }) {
			ids = append(ids, lessons[i].ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	rows, err := s.store.BoardRoomsNamed(ctx, ids, boardMetricsRoom)
	if err != nil {
		s.log.Warn("журнал: самбарын хэмжүүр уншиж чадсангүй", "err", err)
		return
	}
	var cand []string // суралцагчийн ID агуулсан мөрүүдийн хичээл (түгжээгүй урьдчилсан шүүлт)
	for _, row := range rows {
		if bytes.Contains(row.Data, []byte(`"`+uid+`"`)) && !slices.Contains(cand, row.LessonID) {
			cand = append(cand, row.LessonID)
		}
	}
	if len(cand) == 0 {
		return
	}
	h := &s.boards
	h.mu.Lock()
	defer h.mu.Unlock()
	if rows, err = s.store.BoardRoomsNamed(ctx, cand, boardMetricsRoom); err != nil {
		s.log.Warn("журнал: самбарын хэмжүүр уншиж чадсангүй", "err", err)
		return
	}
	var save []store.BoardRoomData
	for _, row := range rows {
		if h.m[row.LessonID+"/"+row.BlockID] != nil {
			continue
		}
		var m boardMetrics
		if json.Unmarshal(row.Data, &m) != nil {
			continue
		}
		if _, ok := m.U[uid]; !ok {
			continue
		}
		delete(m.U, uid)
		if data, err := json.Marshal(m); err == nil {
			save = append(save, store.BoardRoomData{LessonID: row.LessonID, BlockID: row.BlockID, Room: boardMetricsRoom, Data: data})
		}
	}
	if len(save) > 0 {
		if err := s.store.SaveBoardRooms(ctx, save); err != nil {
			s.log.Warn("журнал: самбарын хэмжүүрээс хасаж чадсангүй", "err", err, "rows", len(save))
		}
	}
}

const (
	msgResetNotArchived = "эхлээд суралцагчийг архивлана уу — зөвхөн архивласан суралцагчийн өгөгдлийг устгана"
	msgResetTooSoon     = "энэ суралцагчийн өгөгдлийг саяхан устгасан — 10 минутын дараа дахин оролдоно уу"
	msgResetSlow        = "устгал удааширч байна — хэдэн секундийн дараа дахин оролдоно уу"
)

// resetForget — устгалын дараа (амжилттай ч, хагас ч) суралцагчийн энэ сургалтын кэшүүд: журнал, сургалтын хуудас,
// эзэмшигчийн хяналтын нэгтгэл, хичээл бүрийн хориг, дараалсан нээлт, үүлэн үгийн нэгтгэл, санах ойн самбар.
func (s *Server) resetForget(course *store.Course, lessons []store.Lesson, uid string) {
	s.journals.Delete(course.ID)
	s.courses.Delete(course.ID)
	s.analyticsForget(course)
	for i := range lessons {
		l := &lessons[i]
		s.blocks.Delete(uid + "|" + l.ID)
		s.dripOpen.Delete(uid + "|" + l.ID)
		for _, b := range l.Blocks {
			if b.Type == "cloud" { // 2 секундын нэгтгэлд суралцагчийн хуучин үгс (mine, тоо) үлдэхгүй; бусад үзэгч шинэчилнэ
				s.clouds.Delete(cloudKey(l.ID, b.ID))
				s.cloudChanged(l.ID, b.ID)
			}
		}
	}
	s.boardForget(lessons, uid)
}

// analyticsForget — зөвхөн эзэмшигчийн хяналтын нэгтгэлүүд: түлхүүр нь эзэмшигч|сургалт|хоног (analytics.go), сургалт
// нь хоосон (бүх сургалт) эсвэл энэ сургалт, хоног 1–365. Хамтран заагчийн «бүх сургалт» хүрээнд байгууллагын сургалтын
// үйл явдал ордоггүй (эзэмшигчийн ID-гаар бичигддэг) тул тэднийх хөндөгдөхгүй.
func (s *Server) analyticsForget(course *store.Course) {
	for _, cid := range []string{"", course.ID} {
		for d := 1; d <= 365; d++ {
			s.analyticsAgg.Delete(course.TeacherID + "|" + cid + "|" + strconv.Itoa(d))
		}
	}
}

// boardForget — суралцагчийн хувийн самбар, оролцооны хэмжүүрийг САНАХ ОЙД АЧААЛСАН самбарын сессүүдээс хасна (эс бөгөөс
// flushBoards санах ойн хуулбарыг дахин хадгална). Ачаалаагүй самбарыг хөндөхгүй: сангийн "u:<uid>" өрөөг
// ResetCourseStudent зөвхөн тухайн өрөөтэй хичээлүүдээр устгана.
func (s *Server) boardForget(lessons []store.Lesson, uid string) {
	h := &s.boards
	for i := range lessons {
		l := &lessons[i]
		for _, b := range l.Blocks {
			if b.Type != "board" {
				continue
			}
			h.mu.Lock()
			bs := h.m[l.ID+"/"+b.ID]
			h.mu.Unlock()
			if bs == nil {
				continue
			}
			bs.mu.Lock()
			if !bs.gone { // буулгасан бол бичигдэх өөрчлөлтгүй байсан; дараа нь сангаас (устгасны дараа) ачаална
				delete(bs.rooms, "u:"+uid)
				if _, ok := bs.met.U[uid]; ok {
					delete(bs.met.U, uid)
					bs.metDirty = true
				}
				delete(bs.peers, uid)
				delete(bs.act, uid)
				delete(bs.talk, uid)
			}
			bs.mu.Unlock()
		}
	}
}
