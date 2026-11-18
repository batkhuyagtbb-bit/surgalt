package store

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"golang.org/x/sync/errgroup"
)

// ---- Суралцагчийн нэг сургалт дахь суралцахуйн өгөгдлийг бүрмөсөн устгах (багшийн журнал, journal_meta.go) ----
// Хүснэгт бүрт нэг lightweight DELETE (синхрон). ReplacingMergeTree-д DELETE-ийн нөхцөл нь түлхүүрийн бүх хувилбарт ижил
// баганаар л байх ёстой (эс бөгөөс хуучин хувилбар FINAL-д дахин гарч ирнэ) тул хичээлээр түлхүүрлэсэн хүснэгтэд
// course_id-оос гадна has(lids, lesson_id)-г нэмнэ. Устгах: явц, сесс, үйл явдал, асуултын лог, дүгнэлт, видео үзэлт,
// шалгалтын оролдлого, даалгаврын хариу, тэмдэглэл, үүлэн үг, хувийн цагаан самбар, давтах сануулга, сургалтын оноо,
// бүлэг чатын харилцан үнэлгээ (өгсөн ба авсан).
// Үлдэх: элсэлт, хичээлийн эрх, захиалга, нэхэмжлэх, нууц үгээр нээсэн, шууд хичээлийн эрх, эрэмбийн түвшин, бэлгийн
// хайрцаг, хэлэлцүүлэг, чат, мэдэгдэл.
//
// Ачаалал: эхлээд нэг асуулгаар (resetProbe) аль хүснэгтэд мөр байгааг шалгаж, байгаад нь л DELETE ажиллуулна; юу ч
// байхгүй бол устгал, хаалт, сессийн буферийн зогсолт огт болохгүй. activity_events, quiz_logs нь (teacher_id, at, id)-
// аар эрэмбэлэгдсэн тул эзэмшигчийн teacher_id-г нөхцөлд нэмж бүтэн хүснэгт уншихаас сэргийлнэ: эдгээрийг бичдэг бүх
// газар (activity.go, blocks.go, exam.go, service.go, notes_stats.go, handlers_blocks.go) сургалтын эзэмшигчийн ID-г
// бичдэг, сургалтын teacher_id хэзээ ч өөрчлөгддөггүй (UpdateCourse, SetCourseOrg, DeleteCourse нь өөр эзэмшигчийг
// ErrNotFound-оор татгалздаг; хуваалцсан сургалтыг хуулахад шинэ ID үүснэ).

// resetRestoreTimeout — устгал эхэлсний дараа төлсөн эрхийг сэргээх, мөрийг тэмдэглэх салангид хугацаа (хүсэлтийн
// хугацаа дууссан ч).
const resetRestoreTimeout = 15 * time.Second

type resetStmt struct {
	table string
	where string
	args  []any
}

func (s resetStmt) sql() string { return "DELETE FROM " + s.table + " WHERE " + s.where }

// progressViewedCond — lesson_progress-ийн «үзсэн» мөр (LessonProgress.Viewed-тэй ижил): зөвхөн төлсөн эрхтэй мөр нь
// viewed_at = 1970.
const progressViewedCond = "(toUnixTimestamp(viewed_at) > 0 OR completed_at IS NOT NULL)"

// resetLessonCond — хичээлээр түлхүүрлэсэн хүснэгтийн нөхцөл: энэ сургалтын эсвэл (хуучин хувилбарт сургалт нь өөр
// байсан ч) энэ сургалтын хичээл.
func resetLessonCond(courseID string, lids []string) (string, []any) {
	if len(lids) == 0 {
		return "course_id = ?", []any{courseID}
	}
	return "(course_id = ? OR has(?, lesson_id))", []any{courseID, lids}
}

// resetPhase1 — rank_points-оос бусад устгалууд (зэрэг ажиллана). boardLids — суралцагчийн "u:<uid>" самбартай хичээлүүд.
func resetPhase1(courseID, ownerID, userID string, lids, boardLids []string) []resetStmt {
	lc, la := resetLessonCond(courseID, lids)
	uc := func(extra ...any) []any { return append([]any{userID}, extra...) }
	out := []resetStmt{
		{"lesson_progress", "user_id = ? AND " + lc, uc(la...)},
		{"study_sessions", "user_id = ? AND course_id = ?", uc(courseID)},
		{"activity_events", "teacher_id = ? AND user_id = ? AND course_id = ?", []any{ownerID, userID, courseID}},
		{"quiz_logs", "teacher_id = ? AND user_id = ? AND course_id = ?", []any{ownerID, userID, courseID}},
		{"reflections", "user_id = ? AND course_id = ?", uc(courseID)},
		{"video_watches", "user_id = ? AND course_id = ?", uc(courseID)},
		{"exam_attempts", "user_id = ? AND course_id = ?", uc(courseID)},
		{"submissions", "user_id = ? AND " + lc, uc(la...)},
		// Бүлэг чатын харилцан үнэлгээ: тэр суралцагчийн өгсөн ба авсан (түлхүүрийн баганаар — бүх хувилбарт ижил).
		{"peer_ratings", "course_id = ? AND (rater_id = ? OR ratee_id = ?)", []any{courseID, userID, userID}},
	}
	if len(lids) > 0 { // зөвхөн хичээлээр түлхүүрлэсэн (course_id-гүй) хүснэгтүүд
		out = append(out,
			resetStmt{"highlights", "user_id = ? AND has(?, lesson_id)", []any{userID, lids}},
			resetStmt{"cloud_answers", "has(?, lesson_id) AND user_id = ?", []any{lids, userID}},
			resetStmt{"review_reminders", "user_id = ? AND has(?, lesson_id)", []any{userID, lids}},
		)
	}
	if len(boardLids) > 0 {
		out = append(out, resetStmt{"board_rooms", "has(?, lesson_id) AND room = ?", []any{boardLids, "u:" + userID}})
	}
	return out
}

// resetRankStmt — сүүлийн шат: сургалтын оноо.
func resetRankStmt(courseID, userID string) resetStmt {
	return resetStmt{"rank_points", "user_id = ? AND course_id = ?", []any{userID, courseID}}
}

// resetProbe — хүснэгт бүрт «мөр байна уу» (LIMIT 1) нэг UNION ALL асуулгаар; самбарын хувьд "u:<uid>" өрөөтэй хичээлүүд.
// Мөр бүр (хүснэгт, хичээл).
func resetProbe(courseID, ownerID, userID string, lids []string) (string, []any) {
	var parts []string
	var args []any
	for _, st := range append(resetPhase1(courseID, ownerID, userID, lids, nil), resetRankStmt(courseID, userID)) {
		where := st.where
		if st.table == "lesson_progress" { // зөвхөн төлсөн эрхийн (үзээгүй) мөр нь өгөгдөлд тооцогдохгүй (LessonProgress.Viewed)
			where += " AND " + progressViewedCond
		}
		parts = append(parts, "(SELECT '"+st.table+"' AS t, '' AS l FROM "+st.table+" WHERE "+where+" LIMIT 1)")
		args = append(args, st.args...)
	}
	if len(lids) > 0 {
		parts = append(parts, "(SELECT DISTINCT 'board_rooms' AS t, lesson_id AS l FROM board_rooms WHERE has(?, lesson_id) AND room = ?)")
		args = append(args, lids, "u:"+userID)
	}
	return strings.Join(parts, " UNION ALL "), args
}

// resetLockKeys — суралцагчийн одоо байгаа мөрүүдийг «уншаад бичдэг» бичигчдийн түгжээний түлхүүрүүд (нэг асуулгаар).
// Эдгээрийг устгалын турш барьснаар DELETE-ээс өмнө уншсан хуучин мөрийг дараа нь дахин бичих боломжгүй болно.
func (c *ClickHouse) resetLockKeys(ctx context.Context, courseID, userID string, lids []string) ([]string, error) {
	lc, la := resetLessonCond(courseID, lids)
	parts := []string{
		"SELECT DISTINCT concat('prog:', user_id, ':', lesson_id) AS k FROM lesson_progress WHERE user_id = ? AND " + lc,
		"SELECT DISTINCT concat('sub:', user_id, ':', lesson_id) AS k FROM submissions WHERE user_id = ? AND " + lc,
		"SELECT concat('attempt:', id) AS k FROM exam_attempts FINAL WHERE user_id = ? AND course_id = ? AND status = ?",
		"SELECT DISTINCT concat('watch:', id) AS k FROM video_watches WHERE user_id = ? AND course_id = ?",
	}
	args := append([]any{userID}, la...)
	args = append(args, userID)
	args = append(args, la...)
	args = append(args, userID, courseID, AttemptActive, userID, courseID)
	if c.sess == nil { // олон хуулбар: beat бүр session:<id> түгжээгээр уншаад бичнэ (нэг хуулбарт буферийг hold/drop хийнэ)
		parts = append(parts, "SELECT concat('session:', id) AS k FROM study_sessions FINAL WHERE user_id = ? AND course_id = ? AND ended = false AND last_at >= ?")
		args = append(args, userID, courseID, time.Now().Add(-time.Hour).UTC())
	}
	var keys []string
	err := c.query(ctx, strings.Join(parts, " UNION ALL "), args, func(r driver.Rows) error {
		var k string
		if err := r.Scan(&k); err != nil {
			return err
		}
		keys = append(keys, k)
		return nil
	})
	return keys, err
}

// lockAll — түлхүүрүүдийг эрэмбэлж (давхардлыг хасаж) дарааллаар түгжинэ; алдаа гарвал авснаа тавина.
// Буцаасан функцийг хэд дуудсан ч нэг л удаа тавина.
func (c *ClickHouse) lockAll(ctx context.Context, keys []string) (func(), error) {
	keys = slices.Compact(slices.Sorted(slices.Values(keys)))
	unlocks := make([]func(), 0, len(keys))
	releaseAll := func() {
		for i := len(unlocks) - 1; i >= 0; i-- {
			unlocks[i]()
		}
	}
	for _, k := range keys {
		u, err := c.lock(ctx, k)
		if err != nil {
			releaseAll()
			return nil, err
		}
		unlocks = append(unlocks, u)
	}
	var once sync.Once
	return func() { once.Do(releaseAll) }, nil
}

// paidPass — төлсөн хоцролт/оролцооны захиалга (эрх нь lesson_progress.quiz дотор хадгалагддаг).
type paidPass struct{ lesson, kind string }

func (p paidPass) keys() []string {
	if p.kind == OrderKindLate {
		return []string{FeePassKey, LatePassKey}
	}
	return []string{FeePassKey}
}

// resetMinBudget — устгал эхлүүлэхэд хэрэгтэй хамгийн бага үлдсэн хугацаа: үүнээс бага бол юу ч устгахгүйгээр
// ErrResetNoTime (дундуур тасарсан хагас устгалаас сэргийлнэ).
const resetMinBudget = 10 * time.Second

// resetSteps — нэг устгалын алхмууд (ClickHouse, санах ой). runReset дарааллыг удирдана.
type resetSteps struct {
	wipe    func(ctx context.Context) error // бүх устгал; nil бол устгах өгөгдөл алга
	release func()                          // хаалтыг тавих (restore-оос өмнө; давтан дуудахад аюулгүй)
	// restore — төлсөн эрхийг сэргээх. force: устгал амжилтгүй (цуцлагдсан DELETE сервер дээр үргэлжилж болно) — мөр
	// эрхтэй хэвээр харагдсан ч шинэ ver-тэй дахин бичнэ (дараа нь үүссэн мөрийг тэр mutation хөндөхгүй).
	restore func(ctx context.Context, force bool) error
	finish  func(ctx context.Context) error // мөрийг тэмдэглэх (архив, нэр, бүлэг арилгаж reset_at)
}

// runReset — устгал эхэлмэгц (алдаа, хугацаа дуусах, panic) хаалтыг тавьж, төлсөн эрхийг ХҮСЭЛТЭЭС САЛАНГИД хугацаанд
// заавал сэргээнэ — эс бөгөөс хоцролтын эрх алга болж суралцагчаас дахин төлбөр авна. Устгал эсвэл сэргээлт амжилтгүй
// бол мөрийг хөндөхгүй (архивтаа үлдэнэ) бөгөөд алдаа ErrResetPartial-ийг агуулна. Үлдсэн хугацаа resetMinBudget-ээс
// бага бол устгал эхлэхгүй (ErrResetNoTime).
func runReset(ctx context.Context, st resetSteps) (err error) {
	detached := func() (context.Context, context.CancelFunc) {
		return context.WithTimeout(context.WithoutCancel(ctx), resetRestoreTimeout)
	}
	restored := false
	restore := func(force bool) error {
		restored = true
		st.release()
		rctx, cancel := detached()
		defer cancel()
		if err := st.restore(rctx, force); err != nil {
			return fmt.Errorf("төлсөн эрх сэргээх: %w", err)
		}
		return nil
	}
	if st.wipe != nil {
		if dl, ok := ctx.Deadline(); ok && time.Until(dl) < resetMinBudget {
			return ErrResetNoTime
		}
		defer func() {
			if !restored { // panic: устгалын дундуур тасарсан
				_ = restore(true)
			}
			if err != nil && !errors.Is(err, ErrResetPartial) {
				err = fmt.Errorf("%w: %w", ErrResetPartial, err)
			}
		}()
		if werr := st.wipe(ctx); werr != nil {
			return errors.Join(werr, restore(true))
		}
	}
	if err := restore(false); err != nil {
		return err
	}
	fctx, cancel := detached()
	defer cancel()
	return st.finish(fctx)
}

// resetFound — resetProbe-ийн үр дүн: мөртэй хүснэгтүүд, "u:<uid>" самбартай хичээлүүд.
func (c *ClickHouse) resetFound(ctx context.Context, courseID, ownerID, userID string, lids []string) (map[string]bool, []string, error) {
	q, args := resetProbe(courseID, ownerID, userID, lids)
	found := map[string]bool{}
	var boards []string
	err := c.query(ctx, q, args, func(r driver.Rows) error {
		var t, l string
		if err := r.Scan(&t, &l); err != nil {
			return err
		}
		found[t] = true
		if t == "board_rooms" && l != "" {
			boards = append(boards, l)
		}
		return nil
	})
	return found, boards, err
}

// paidPasses — суралцагчийн энэ сургалтад төлсөн хоцролт/оролцооны захиалгууд.
func (c *ClickHouse) paidPasses(ctx context.Context, courseID, userID string) ([]paidPass, error) {
	var out []paidPass
	err := c.query(ctx, "SELECT lesson_id, kind FROM orders FINAL WHERE user_id = ? AND course_id = ? AND status = ? AND has(?, kind)",
		[]any{userID, courseID, string(OrderPaid), []string{OrderKindLate, OrderKindFee}}, func(r driver.Rows) error {
			var p paidPass
			if err := r.Scan(&p.lesson, &p.kind); err != nil {
				return err
			}
			if p.lesson != "" {
				out = append(out, p)
			}
			return nil
		})
	return out, err
}

// unionPasses — давхардалгүй нэгтгэл (урьдчилж уншсан + сэргээх үед дахин уншсан).
func unionPasses(a, b []paidPass) []paidPass {
	seen := map[paidPass]bool{}
	var out []paidPass
	for _, p := range append(append([]paidPass{}, a...), b...) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// restorePasses — төлсөн эрхийг хичээлийг «үзээгүй» хэвээр тавина (force биш бол аль хэдийн байвал бичихгүй). Нэгд нь
// алдаа гарсан ч бусдыг үргэлжлүүлнэ.
func restorePasses(ctx context.Context, grant func(ctx context.Context, lessonID string, force bool, keys []string) error, passes []paidPass, force bool) error {
	var errs []error
	for _, p := range passes {
		if err := grant(ctx, p.lesson, force, p.keys()); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// resetWipe — мөртэй хүснэгтүүдийг устгана: эхний шат зэрэг, rank_points сүүлд. Нэг хуулбарт сессийн буферийг зогсоож,
// суралцагчийн (бичигдээгүй) сессүүдийг DELETE-ийн өмнө ба дараа хаяна.
func (c *ClickHouse) resetWipe(ctx context.Context, courseID, userID string, stmts []resetStmt, rank bool) error {
	// Синхрон устгал: шинэ ClickHouse-д lightweight DELETE анхдагчаар хүлээдэг (lightweight_deletes_sync=2);
	// тэр тохиргоо байхгүй хуучин хувилбарт mutations_sync удирддаг. Тиймээс бүх хувилбар таньдаг mutations_sync-ийг
	// өгнө — lightweight_deletes_sync-ийг нэрлэвэл хуучин сервер «Unknown setting» гэж устгалыг бүхэлд нь унагаана.
	dctx := clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{"mutations_sync": 2}))
	mine := func(s *StudySession) bool { return s.UserID == userID && s.CourseID == courseID }
	buf := c.sess != nil && slices.ContainsFunc(stmts, func(s resetStmt) bool { return s.table == "study_sessions" })
	resume := func() {}
	if buf {
		resume = c.sess.hold()
		defer resume()
		c.sess.drop(mine)
	}
	g, gctx := errgroup.WithContext(dctx)
	for _, st := range stmts {
		g.Go(func() error {
			if err := c.conn.Exec(gctx, st.sql(), st.args...); err != nil {
				return fmt.Errorf("reset %s: %w", st.table, err)
			}
			return nil
		})
	}
	err := g.Wait()
	if buf { // устгалын хооронд beat-ээр дахин ачаалагдсаныг ч хаяад flush-ийг сэргээнэ
		c.sess.drop(mine)
		resume()
	}
	if err != nil || !rank {
		return err
	}
	// Зэрэг ачаалсан сургалтын хуудас оноог үлдсэн нотолгооноос дахин тооцдог тул хамгийн сүүлд.
	rk := resetRankStmt(courseID, userID)
	if err := c.conn.Exec(dctx, rk.sql(), rk.args...); err != nil {
		return fmt.Errorf("reset rank_points: %w", err)
	}
	return nil
}

func (c *ClickHouse) ResetCourseStudent(ctx context.Context, courseID, userID, by string) (ResetResult, error) {
	var res ResetResult
	// Сургалтын журналын түгжээ; архивлаагүй эсвэл саяхан устгасан бол юу ч хийхгүй.
	unlock, err := c.lock(ctx, "cmeta:"+courseID)
	if err != nil {
		return res, err
	}
	defer unlock()
	all, err := c.CourseStudentMetas(ctx, courseID)
	if err != nil {
		return res, err
	}
	cur, ok := all[userID]
	if err := resetAllowed(cur, ok, time.Now()); err != nil {
		return res, err
	}
	co, err := c.CourseByID(ctx, courseID) // activity_events, quiz_logs-ийн teacher_id
	if err != nil {
		return res, err
	}
	// Сургалтын бүх хичээл (устгагдсаныг ч — FINAL-гүй).
	lids := []string{}
	if err := c.query(ctx, "SELECT DISTINCT id FROM lessons WHERE course_id = ?", []any{courseID}, func(r driver.Rows) error {
		var id string
		if err := r.Scan(&id); err != nil {
			return err
		}
		lids = append(lids, id)
		return nil
	}); err != nil {
		return res, err
	}
	found, boardLids, err := c.resetFound(ctx, courseID, co.TeacherID, userID, lids)
	if err != nil {
		return res, err
	}
	// Төлсөн эрхийг устгалаас ӨМНӨ уншина: устгал дундуур тасарсан ч сэргээж чадна.
	passes, err := c.paidPasses(ctx, courseID, userID)
	if err != nil {
		return res, err
	}
	var stmts []resetStmt
	for _, st := range resetPhase1(courseID, co.TeacherID, userID, lids, boardLids) {
		if found[st.table] {
			stmts = append(stmts, st)
		}
	}
	rank := found["rank_points"]
	res.Wiped = len(stmts) > 0 || rank
	release := func() {}
	steps := resetSteps{
		release: func() { release() },
		restore: func(ctx context.Context, force bool) error {
			// Устгалын хооронд төлөгдсөн захиалга (prog: хаалтгүй, мөргүй хичээл) — дахин уншиж нэгтгэнэ.
			all := passes
			if fresh, err := c.paidPasses(ctx, courseID, userID); err == nil {
				all = unionPasses(passes, fresh)
			}
			return restorePasses(ctx, func(ctx context.Context, lid string, force bool, keys []string) error {
				return c.grantLessonPass(ctx, userID, courseID, lid, force, keys)
			}, all, force)
		},
		finish: func(ctx context.Context) error {
			m := resetMeta(userID, by, time.Now())
			res.At = *m.ResetAt
			return c.writeMetas(ctx, courseID, by, []CourseStudentMeta{m})
		},
	}
	if res.Wiped {
		// Хаалт: явц, даалгавар, оролдлого, видео, (олон хуулбарт) сессийн бичигчдийг устгал дуустал хүлээлгэнэ.
		// Бичигч бүр эдгээрээс зөвхөн НЭГИЙГ барьж, "cmeta:"-г хэзээ ч авдаггүй тул түгжрэл үүсэхгүй.
		keys, err := c.resetLockKeys(ctx, courseID, userID, lids)
		if err != nil {
			return res, err
		}
		if release, err = c.lockAll(ctx, keys); err != nil {
			return res, err
		}
		defer release()
		steps.wipe = func(ctx context.Context) error { return c.resetWipe(ctx, courseID, userID, stmts, rank) }
	}
	if err := runReset(ctx, steps); err != nil {
		return ResetResult{}, err
	}
	return res, nil
}
