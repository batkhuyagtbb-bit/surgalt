package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func setMeta(t *testing.T, m *Memory, course, uid string, fn func(*CourseStudentMeta)) {
	t.Helper()
	if _, err := m.UpdateCourseStudentMeta(context.Background(), course, uid, "teacher", func(cur *CourseStudentMeta, _ map[string]CourseStudentMeta) error {
		fn(cur)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// Журналын мэдээллийг хэсэгчлэн засах: fn-ийн алдаа өөрчлөлтгүй буцаж, юу ч бичигдэхгүй; хоосон мөр уншигдахгүй;
// буцаасан утгыг өөрчилсөн ч store хөндөгдөхгүй.
func TestCourseStudentMetaPatch(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	setMeta(t, m, "c1", "u2", func(c *CourseStudentMeta) { c.Group = "5А" })

	var seen map[string]CourseStudentMeta
	got, err := m.UpdateCourseStudentMeta(ctx, "c1", "u1", "t1", func(cur *CourseStudentMeta, all map[string]CourseStudentMeta) error {
		seen = all
		if cur.UserID != "u1" || cur.Alias != "" {
			t.Fatalf("шинэ мөрийн cur: %+v", cur)
		}
		cur.Alias = "Бат"
		return nil
	})
	if err != nil || got.UserID != "u1" || got.Alias != "Бат" {
		t.Fatalf("засвар: %+v %v", got, err)
	}
	if seen["u2"].Group != "5А" {
		t.Fatalf("fn бусад мөрийг харах ёстой: %+v", seen)
	}
	got.Alias = "өөрчилсөн" // буцаасан хуулбар
	metas, _ := m.CourseStudentMetas(ctx, "c1")
	if metas["u1"].Alias != "Бат" || metas["u2"].Group != "5А" || len(metas) != 2 {
		t.Fatalf("уншилт: %+v", metas)
	}

	// fn-ийн алдаа яг тэр хэвээр буцаж, бичихгүй.
	boom := errors.New("boom")
	_, err = m.UpdateCourseStudentMeta(ctx, "c1", "u1", "t1", func(cur *CourseStudentMeta, _ map[string]CourseStudentMeta) error {
		cur.Alias = "Дорж"
		return boom
	})
	if err != boom {
		t.Fatalf("fn-ийн алдаа: %v", err)
	}
	if metas, _ := m.CourseStudentMetas(ctx, "c1"); metas["u1"].Alias != "Бат" {
		t.Fatalf("алдаатай засвар бичигдлээ: %+v", metas["u1"])
	}

	// Архив: огноо хадгалагдаж, хуулбар нь тусдаа.
	at := time.Now().UTC()
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Archived, c.ArchivedAt = true, &at })
	metas, _ = m.CourseStudentMetas(ctx, "c1")
	*metas["u1"].ArchivedAt = time.Time{}
	if metas2, _ := m.CourseStudentMetas(ctx, "c1"); metas2["u1"].ArchivedAt == nil || !metas2["u1"].ArchivedAt.Equal(at) || !metas2["u1"].Archived {
		t.Fatalf("архивын огноо: %+v", metas2["u1"])
	}

	// Бүх талбарыг арилгавал «мэдээлэлгүй» — уншихад гарахгүй. Өөр сургалт хөндөгдөхгүй.
	setMeta(t, m, "c2", "u1", func(c *CourseStudentMeta) { c.Alias = "Өөр" })
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Alias, c.Group, c.Archived, c.ArchivedAt = "", "", false, nil })
	metas, _ = m.CourseStudentMetas(ctx, "c1")
	if _, ok := metas["u1"]; ok || len(metas) != 1 {
		t.Fatalf("хоосон мөр уншигдлаа: %+v", metas)
	}
	if metas, _ := m.CourseStudentMetas(ctx, "c2"); metas["u1"].Alias != "Өөр" {
		t.Fatalf("өөр сургалтын мөр: %+v", metas)
	}
}

func TestRenameCourseGroup(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Group = "5А" })
	setMeta(t, m, "c1", "u2", func(c *CourseStudentMeta) { c.Group = "5а" })
	setMeta(t, m, "c1", "u3", func(c *CourseStudentMeta) { c.Group = "5Б" })
	setMeta(t, m, "c1", "u4", func(c *CourseStudentMeta) { c.Group, c.Archived = "5А", true }) // архивласан ч хамт
	setMeta(t, m, "c2", "u1", func(c *CourseStudentMeta) { c.Group = "5А" })                   // өөр сургалт
	groups := func(course string) map[string]string {
		out := map[string]string{}
		ms, _ := m.CourseStudentMetas(ctx, course)
		for uid, x := range ms {
			out[uid] = x.Group
		}
		return out
	}

	if n, err := m.RenameCourseGroup(ctx, "c1", "5а", "5В", "t"); err != nil || n != 3 {
		t.Fatalf("нэр солих: %d %v", n, err)
	}
	if g := groups("c1"); g["u1"] != "5В" || g["u2"] != "5В" || g["u4"] != "5В" || g["u3"] != "5Б" {
		t.Fatalf("нэр солисны дараа: %v", g)
	}
	if g := groups("c2"); g["u1"] != "5А" {
		t.Fatalf("өөр сургалт өөрчлөгдлөө: %v", g)
	}
	// Өөр байгаа бүлэгтэй таарвал нэгтгэж, тэр бүлгийн бичлэг үлдэнэ.
	if n, err := m.RenameCourseGroup(ctx, "c1", "5в", "5б", "t"); err != nil || n != 3 {
		t.Fatalf("нэгтгэх: %d %v", n, err)
	}
	if g := groups("c1"); g["u1"] != "5Б" || g["u2"] != "5Б" || g["u3"] != "5Б" || g["u4"] != "5Б" {
		t.Fatalf("нэгтгэсний дараа: %v", g)
	}
	// Зөвхөн том жижиг үсэг солих — бичсэнээр нь.
	if n, err := m.RenameCourseGroup(ctx, "c1", "5Б", "5б", "t"); err != nil || n != 4 {
		t.Fatalf("үсэг солих: %d %v", n, err)
	}
	if g := groups("c1"); g["u3"] != "5б" {
		t.Fatalf("үсэг солисны дараа: %v", g)
	}
	// Ижил нэр — таарсан ч өөрчлөлтгүй.
	if n, err := m.RenameCourseGroup(ctx, "c1", "5б", "5б", "t"); err != nil || n != 0 {
		t.Fatalf("ижил нэр: %d %v", n, err)
	}
	// Олдоогүй бүлэг.
	if _, err := m.RenameCourseGroup(ctx, "c1", "9Я", "1", "t"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("олдоогүй бүлэг: %v", err)
	}
	// Тараах: бүлэггүй болж, архивласан мөр архивтаа үлдэнэ; нэр, бүлэггүй архивгүй мөр уншигдахгүй.
	if n, err := m.RenameCourseGroup(ctx, "c1", "5Б", "", "t"); err != nil || n != 4 {
		t.Fatalf("тараах: %d %v", n, err)
	}
	ms, _ := m.CourseStudentMetas(ctx, "c1")
	if len(ms) != 1 || !ms["u4"].Archived || ms["u4"].Group != "" {
		t.Fatalf("тараасны дараа: %+v", ms)
	}
	if _, err := m.RenameCourseGroup(ctx, "c1", "5Б", "x", "t"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("тараасан бүлэг: %v", err)
	}
}

// Бүртгэл устгахад журналын нэр, бүлэг бүх сургалтаас устана.
func TestDeleteUserDataRemovesMeta(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	u := &User{Username: "child", Email: "c@x.mn", Role: RoleStudent, DisplayName: "Хүүхэд"}
	other := &User{Username: "other", Email: "o@x.mn", Role: RoleStudent, DisplayName: "Бусад"}
	if err := m.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	if err := m.CreateUser(ctx, other); err != nil {
		t.Fatal(err)
	}
	setMeta(t, m, "c1", u.ID, func(c *CourseStudentMeta) { c.Alias, c.Group = "Бат", "5А" })
	setMeta(t, m, "c2", u.ID, func(c *CourseStudentMeta) { c.Archived = true })
	setMeta(t, m, "c1", other.ID, func(c *CourseStudentMeta) { c.Group = "5А" })
	if err := m.DeleteUserData(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"c1", "c2"} {
		ms, _ := m.CourseStudentMetas(ctx, c)
		if _, ok := ms[u.ID]; ok {
			t.Fatalf("%s: устгасан хэрэглэгчийн журналын мөр үлдлээ: %+v", c, ms)
		}
	}
	if ms, _ := m.CourseStudentMetas(ctx, "c1"); ms[other.ID].Group != "5А" {
		t.Fatalf("бусдын мөр устлаа: %+v", ms)
	}
}

// Санах ойн устгал: архивлаагүй бол ErrConflict; зөвхөн (сургалт, суралцагч)-ийн суралцахуйн өгөгдөл устаж, элсэлт,
// хичээлийн эрх, өөр сургалт, бусад суралцагч хэвээр; төлсөн хоцролтын эрх сэргэнэ; мөр нь шинэ эхлэлийн тэмдэгтэй.
func TestResetCourseStudentMemory(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	m.lessons["c1"] = []*Lesson{{ID: "l1", CourseID: "c1"}, {ID: "l2", CourseID: "c1"}}
	m.lessons["c2"] = []*Lesson{{ID: "l9", CourseID: "c2"}}
	m.enrollments[[2]string{"u1", "c1"}] = time.Now()
	m.lessonAcc[[2]string{"u1", "l2"}] = "c1"
	m.orders["o1"] = &Order{ID: "o1", Kind: OrderKindLate, UserID: "u1", CourseID: "c1", LessonID: "l2", Status: OrderPaid}
	m.orders["o2"] = &Order{ID: "o2", Kind: OrderKindFee, UserID: "u1", CourseID: "c1", LessonID: "l1", Status: OrderPending} // төлөгдөөгүй

	fill := func(uid, cid, lid string) {
		_ = m.MarkLessonCompleted(ctx, uid, cid, lid)
		_ = m.SaveQuizResult(ctx, uid, cid, lid, "q1", true)
		_ = m.CreateSession(ctx, &StudySession{UserID: uid, CourseID: cid, LessonID: lid, ActiveSec: 60, LastAt: time.Now()})
		_ = m.AddActivityEvents(ctx, []ActivityEvent{{UserID: uid, CourseID: cid, LessonID: lid, Type: "auto_block"}})
		_ = m.AddQuizLog(ctx, QuizLog{UserID: uid, CourseID: cid, LessonID: lid, BlockID: "q1", Correct: true})
		_ = m.SaveReflection(ctx, &Reflection{UserID: uid, CourseID: cid, LessonID: lid, Text: "бодол"})
		_ = m.AddVideoWatch(ctx, VideoWatch{UserID: uid, CourseID: cid, LessonID: lid, BlockID: "v1", Duration: 10, Buckets: map[int]int{0: 1}})
		_ = m.CreateExamAttempt(ctx, &ExamAttempt{UserID: uid, CourseID: cid, LessonID: lid, Status: AttemptSubmitted, StartedAt: time.Now()})
		_ = m.SaveSubmission(ctx, &Submission{UserID: uid, CourseID: cid, LessonID: lid, Text: "хариу"})
		_ = m.SetHighlights(ctx, uid, lid, []Highlight{{ID: "h1", Block: "content", Start: 0, End: 3, Color: "y"}})
		_ = m.SetCloudWords(ctx, lid, "cl1", uid, []string{"үг"})
		_ = m.SaveBoardRooms(ctx, []BoardRoomData{{LessonID: lid, BlockID: "b1", Room: "u:" + uid, Data: []byte(`{}`)}, {LessonID: lid, BlockID: "b1", Room: "main", Data: []byte(`{}`)}})
		_, _ = m.ClaimReviews(ctx, []ReviewItem{{UserID: uid, CourseID: cid, LessonID: lid, Stage: 1}})
		_, _ = m.SaveRankPoints(ctx, uid, cid, 50)
	}
	fill("u1", "c1", "l1")
	fill("u1", "c2", "l9")
	fill("u2", "c1", "l1")

	if _, err := m.ResetCourseStudent(ctx, "c1", "u1", "t"); !errors.Is(err, ErrConflict) {
		t.Fatalf("мөргүй үед: %v", err)
	}
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Alias, c.Group = "Бат", "5А" })
	if _, err := m.ResetCourseStudent(ctx, "c1", "u1", "t"); !errors.Is(err, ErrConflict) {
		t.Fatalf("архивлаагүй үед: %v", err)
	}
	now := time.Now()
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Archived, c.ArchivedAt = true, &now })
	res, err := m.ResetCourseStudent(ctx, "c1", "u1", "t")
	if err != nil || !res.Wiped || res.At.IsZero() {
		t.Fatalf("устгал: %+v %v", res, err)
	}

	type counts struct{ prog, sess, evs, quiz, refl, watch, att, sub, hl, cloud, board, rank int }
	count := func(uid, cid, lid string) counts {
		var c counts
		pr, _ := m.LessonProgress(ctx, uid, cid)
		for _, p := range pr {
			if p.CompletedAt != nil || p.Quiz["q1"] {
				c.prog++
			}
		}
		f := ActivityFilter{UserID: uid, CourseID: cid}
		ss, _ := m.Sessions(ctx, f, 0)
		evs, _ := m.ActivityEvents(ctx, f, 0)
		qs, _ := m.QuizLogs(ctx, f, 0)
		rs, _ := m.Reflections(ctx, f, 0)
		ws, _ := m.VideoWatches(ctx, f)
		as, _ := m.ExamAttempts(ctx, f)
		c.sess, c.evs, c.quiz, c.refl, c.watch, c.att = len(ss), len(evs), len(qs), len(rs), len(ws), len(as)
		if _, err := m.SubmissionFor(ctx, uid, lid); err == nil {
			c.sub = 1
		}
		hs, _ := m.Highlights(ctx, uid, lid)
		c.hl = len(hs)
		if cw, _ := m.CloudWords(ctx, lid, "cl1"); len(cw[uid]) > 0 {
			c.cloud = 1
		}
		if br, _ := m.BoardRooms(ctx, lid, "b1"); br["u:"+uid] != nil {
			c.board = 1
		}
		c.rank = m.learn.rankPts[[2]string{uid, cid}]
		return c
	}
	if got := count("u1", "c1", "l1"); got != (counts{}) {
		t.Fatalf("устгагдаагүй өгөгдөл үлдлээ: %+v", got)
	}
	full := counts{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 50}
	if got := count("u1", "c2", "l9"); got != full {
		t.Fatalf("өөр сургалтын өгөгдөл хөндөгдлөө: %+v", got)
	}
	if got := count("u2", "c1", "l1"); got != full {
		t.Fatalf("өөр суралцагчийн өгөгдөл хөндөгдлөө: %+v", got)
	}
	if br, _ := m.BoardRooms(ctx, "l1", "b1"); br["main"] == nil {
		t.Fatal("багшийн самбар устлаа")
	}
	has := func(k string) bool { _, ok := m.reviews[k]; return ok }
	if has("u1|l1|1") || !has("u1|l9|1") || !has("u2|l1|1") {
		t.Fatalf("давтах сануулга: %v", m.reviews)
	}
	// Элсэлт, худалдсан хичээл хэвээр; төлсөн хоцролтын эрх сэргэсэн, төлөгдөөгүйнх үгүй.
	if ok, _ := m.IsEnrolled(ctx, "u1", "c1"); !ok {
		t.Fatal("элсэлт устлаа")
	}
	if b, _ := m.PurchasedLessons(ctx, "u1", "c1"); len(b) != 1 {
		t.Fatalf("худалдсан хичээл: %v", b)
	}
	// Төлсөн хоцролтын эрх сэргэсэн боловч хичээл «үзээгүй» (явцад харагдахгүй); төлөгдөөгүйнх үгүй.
	if pr, _ := m.LessonProgress(ctx, "u1", "c1"); len(pr) != 0 {
		t.Fatalf("сэргээсэн эрх хичээлийг үзсэн болгов: %+v", pr)
	}
	ps, _ := m.LessonPasses(ctx, "u1", "c1")
	if !ps["l2"][LatePassKey] || !ps["l2"][FeePassKey] || len(ps["l1"]) != 0 {
		t.Fatalf("төлсөн эрх: %+v", ps)
	}
	if items, _ := m.ReviewDue(ctx, time.Now().Add(-time.Hour), time.Now().Add(time.Hour)); len(items) != 2 { // u1/l9, u2/l1
		t.Fatalf("эрхийн мөр давтах сануулгад орлоо: %+v", items)
	}
	// Мөр: нэр, бүлэг, архив арилж, устгасан тэмдэг үлдэнэ.
	ms, _ := m.CourseStudentMetas(ctx, "c1")
	if x := ms["u1"]; x.Alias != "" || x.Group != "" || x.Archived || x.ArchivedAt != nil || x.ResetAt == nil || x.ResetBy != "t" || !x.ResetAt.Equal(res.At) {
		t.Fatalf("устгасны дараах мөр: %+v", x)
	}
	// Дахин устгахад архивлах шаардлагатай; архивласан ч 10 минутын дотор ErrResetTooSoon.
	if _, err := m.ResetCourseStudent(ctx, "c1", "u1", "t"); !errors.Is(err, ErrConflict) {
		t.Fatalf("архивгүй дахин устгал: %v", err)
	}
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Archived = true })
	if _, err := m.ResetCourseStudent(ctx, "c1", "u1", "t"); !errors.Is(err, ErrResetTooSoon) {
		t.Fatalf("саяхан устгасан: %v", err)
	}
	old := time.Now().Add(-ResetCooldown - time.Second)
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.ResetAt = &old })
	// Устгах өгөгдөл алга (зөвхөн эрхийн мөр — өгөгдөлд тооцогдохгүй, R5): мөр шинэчлэгдэх боловч Wiped=false.
	if res, err := m.ResetCourseStudent(ctx, "c1", "u1", "t"); err != nil || res.Wiped {
		t.Fatalf("10 минутын дараа: %+v %v", res, err)
	}
	if ps, _ := m.LessonPasses(ctx, "u1", "c1"); !ps["l2"][LatePassKey] {
		t.Fatalf("дахин устгалын дараа эрх: %+v", ps)
	}
	m2 := NewMemory()
	m2.lessons["c1"] = []*Lesson{{ID: "l1", CourseID: "c1"}}
	setMeta(t, m2, "c1", "u1", func(c *CourseStudentMeta) { c.Archived, c.Alias = true, "Бат" })
	if res, err := m2.ResetCourseStudent(ctx, "c1", "u1", "t"); err != nil || res.Wiped {
		t.Fatalf("өгөгдөлгүй суралцагч: %+v %v", res, err)
	}
	if x := mustMetas(t, m2, "c1")["u1"]; x.Archived || x.Alias != "" || x.ResetAt == nil {
		t.Fatalf("өгөгдөлгүй үед ч мөр шинэчлэгдэнэ: %+v", x)
	}
}

func mustMetas(t *testing.T, m *Memory, course string) map[string]CourseStudentMeta {
	t.Helper()
	ms, err := m.CourseStudentMetas(context.Background(), course)
	if err != nil {
		t.Fatal(err)
	}
	return ms
}

// B1: явцыг устгасны дараах алдаанд ч төлсөн эрх заавал сэргэнэ; алдаа ErrResetPartial, суралцагч архивтаа үлдэж дахин
// оролдоход бүрэн дуусна.
func TestResetRestoresPassesOnFailure(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	m.lessons["c1"] = []*Lesson{{ID: "l1", CourseID: "c1"}, {ID: "l2", CourseID: "c1"}}
	m.orders["o1"] = &Order{ID: "o1", Kind: OrderKindLate, UserID: "u1", CourseID: "c1", LessonID: "l1", Status: OrderPaid}
	m.orders["o2"] = &Order{ID: "o2", Kind: OrderKindFee, UserID: "u1", CourseID: "c1", LessonID: "l2", Status: OrderPaid}
	for _, lid := range []string{"l1", "l2"} {
		_ = m.MarkLessonViewed(ctx, "u1", "c1", lid)
		_ = m.SaveQuizResult(ctx, "u1", "c1", lid, "q1", true)
	}
	_ = m.GrantLessonPass(ctx, "u1", "c1", "l1", LatePassKey, FeePassKey)
	_ = m.GrantLessonPass(ctx, "u1", "c1", "l2", FeePassKey)
	_ = m.SaveSubmission(ctx, &Submission{UserID: "u1", CourseID: "c1", LessonID: "l1", Text: "хариу"})
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Archived = true })

	boom := errors.New("сан унасан")
	m.resetFaults = func(step string) error {
		if step == "lesson_progress" { // явц аль хэдийн устсан
			return boom
		}
		return nil
	}
	_, err := m.ResetCourseStudent(ctx, "c1", "u1", "t")
	if !errors.Is(err, ErrResetPartial) || !errors.Is(err, boom) {
		t.Fatalf("хагас устгалын алдаа: %v", err)
	}
	ps, _ := m.LessonPasses(ctx, "u1", "c1")
	if !ps["l1"][LatePassKey] || !ps["l1"][FeePassKey] || !ps["l2"][FeePassKey] {
		t.Fatalf("алдааны дараа төлсөн эрх сэргээгүй: %+v", ps)
	}
	if pr, _ := m.LessonProgress(ctx, "u1", "c1"); len(pr) != 0 {
		t.Fatalf("явц устсан, эрхийн мөр үзээгүй байх ёстой: %+v", pr)
	}
	if x := mustMetas(t, m, "c1")["u1"]; !x.Archived || x.ResetAt != nil {
		t.Fatalf("алдааны дараа архивтаа үлдэх ёстой: %+v", x)
	}
	if _, err := m.SubmissionFor(ctx, "u1", "l1"); err != nil {
		t.Fatal("алдааны дараах алхмын өгөгдөл (даалгавар) хараахан устаагүй байх ёстой")
	}

	m.resetFaults = nil
	if _, err := m.ResetCourseStudent(ctx, "c1", "u1", "t"); err != nil {
		t.Fatalf("дахин оролдлого: %v", err)
	}
	if _, err := m.SubmissionFor(ctx, "u1", "l1"); err == nil {
		t.Fatal("дахин оролдоход даалгавар устах ёстой")
	}
	if ps, _ := m.LessonPasses(ctx, "u1", "c1"); !ps["l1"][LatePassKey] || !ps["l2"][FeePassKey] {
		t.Fatalf("дахин оролдлогын дараа эрх: %+v", ps)
	}
}

// runReset: хугацаа дууссан (эсвэл алдаатай) устгалын дараа ч сэргээлт салангид, амьд context-оор ажиллаж, хаалтыг түрүүлж
// тавина; сэргээлт ба тэмдэглэл амжилттай дууссан үед л мөр бичигдэнэ.
func TestRunResetAlwaysRestores(t *testing.T) {
	var log []string
	steps := func(wipeErr, restoreErr error) resetSteps {
		log = nil
		return resetSteps{
			wipe:    func(ctx context.Context) error { log = append(log, "wipe"); return wipeErr },
			release: func() { log = append(log, "release") },
			restore: func(ctx context.Context, force bool) error {
				s := "restore"
				if force { // устгал амжилтгүй: хойшилсон DELETE-ээс хамгаалж заавал дахин бичнэ (R1)
					s = "restore!"
				}
				if ctx.Err() != nil {
					s += "-dead-ctx"
				}
				log = append(log, s)
				return restoreErr
			},
			finish: func(ctx context.Context) error { log = append(log, "finish"); return nil },
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // хүсэлт цуцлагдсан
	err := runReset(ctx, steps(context.DeadlineExceeded, nil))
	if !errors.Is(err, ErrResetPartial) || !errors.Is(err, context.DeadlineExceeded) || strings.Join(log, ",") != "wipe,release,restore!" {
		t.Fatalf("хугацаа дууссан: %v %v", err, log)
	}
	if err := runReset(context.Background(), steps(nil, errors.New("x"))); !errors.Is(err, ErrResetPartial) || strings.Join(log, ",") != "wipe,release,restore" {
		t.Fatalf("сэргээлтийн алдаа: %v %v", err, log)
	}
	if err := runReset(context.Background(), steps(nil, nil)); err != nil || strings.Join(log, ",") != "wipe,release,restore,finish" {
		t.Fatalf("амжилттай: %v %v", err, log)
	}
	st := steps(nil, nil)
	st.wipe = nil // устгах юм алга
	if err := runReset(context.Background(), st); err != nil || strings.Join(log, ",") != "release,restore,finish" {
		t.Fatalf("өгөгдөлгүй: %v %v", err, log)
	}
	st = steps(nil, nil)
	st.wipe = func(context.Context) error { panic("гэнэт") }
	func() {
		defer func() { _ = recover() }()
		_ = runReset(context.Background(), st)
	}()
	if strings.Join(log, ",") != "release,restore!" {
		t.Fatalf("panic-ийн үед ч сэргээх ёстой: %v", log)
	}
	// R4: үлдсэн хугацаа хүрэлцэхгүй бол юу ч хийхгүй (хагас устгал үүсэхгүй); өгөгдөлгүй үед хугацаа хамаарахгүй.
	short, cancel2 := context.WithTimeout(context.Background(), resetMinBudget-time.Second)
	defer cancel2()
	if err := runReset(short, steps(nil, nil)); !errors.Is(err, ErrResetNoTime) || errors.Is(err, ErrResetPartial) || len(log) != 0 {
		t.Fatalf("хугацаа хүрэлцэхгүй: %v %v", err, log)
	}
	st = steps(nil, nil)
	st.wipe = nil
	if err := runReset(short, st); err != nil || strings.Join(log, ",") != "release,restore,finish" {
		t.Fatalf("өгөгдөлгүй, богино хугацаа: %v %v", err, log)
	}
	long, cancel3 := context.WithTimeout(context.Background(), resetMinBudget+5*time.Second)
	defer cancel3()
	if err := runReset(long, steps(nil, nil)); err != nil {
		t.Fatalf("хангалттай хугацаа: %v", err)
	}
}

// R1: эрх тавих мөр — force үед өөрчлөлтгүй ч бичнэ (үзсэн мөр үзсэн хэвээр), мөргүй бол үзээгүй мөр.
func TestPassRow(t *testing.T) {
	now := time.Now()
	viewed := &LessonProgress{LessonID: "l1", ViewedAt: now, Quiz: map[string]bool{"q1": true, FeePassKey: true, LatePassKey: true}}
	if _, write := passRow(viewed, "l1", []string{FeePassKey, LatePassKey}, false); write {
		t.Fatal("өөрчлөлтгүй, force биш — бичих ёсгүй")
	}
	p, write := passRow(viewed, "l1", []string{FeePassKey, LatePassKey}, true)
	if !write || !p.Viewed() || !p.ViewedAt.Equal(now) || !p.Quiz["q1"] || !p.Quiz[LatePassKey] {
		t.Fatalf("force: үзсэн мөрийг хэвээр, шинэ ver-тэй бичих ёстой: %v %+v", write, p)
	}
	p, write = passRow(nil, "l2", []string{FeePassKey}, false)
	if !write || p.Viewed() || p.LessonID != "l2" || !p.Quiz[FeePassKey] || p.Quiz[LatePassKey] {
		t.Fatalf("мөргүй: үзээгүй эрхийн мөр: %v %+v", write, p)
	}
	if got := unionPasses([]paidPass{{"l1", "late"}, {"l2", "fee"}}, []paidPass{{"l2", "fee"}, {"l3", "fee"}}); len(got) != 3 {
		t.Fatalf("нэгтгэл: %v", got)
	}
}

// R1 (санах ой, алдааны hook): юу ч устгаагүй үеийн алдаанд ч эрх хэвээр, хагас алдаа; R3: устгалын хооронд төлөгдсөн
// захиалгын эрх сэргээх үед дахин уншигдаж сэргэнэ.
func TestResetRestoreForcedAndLatePayment(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	m.lessons["c1"] = []*Lesson{{ID: "l1", CourseID: "c1"}, {ID: "l3", CourseID: "c1"}}
	m.orders["o1"] = &Order{ID: "o1", Kind: OrderKindLate, UserID: "u1", CourseID: "c1", LessonID: "l1", Status: OrderPaid}
	_ = m.MarkLessonViewed(ctx, "u1", "c1", "l1")
	_ = m.GrantLessonPass(ctx, "u1", "c1", "l1", LatePassKey, FeePassKey)
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Archived = true })
	m.resetFaults = func(step string) error {
		if step == "start" {
			return context.DeadlineExceeded
		}
		return nil
	}
	if _, err := m.ResetCourseStudent(ctx, "c1", "u1", "t"); !errors.Is(err, ErrResetPartial) {
		t.Fatalf("эхэнд тасарсан устгал: %v", err)
	}
	if ps, _ := m.LessonPasses(ctx, "u1", "c1"); !ps["l1"][LatePassKey] {
		t.Fatalf("эрх алдагдлаа: %+v", ps)
	}
	// Устгал эхэлсний дараа (урьдчилж уншсаны дараа) l3-ийн оролцооны төлбөр төлөгдөнө (MarkOrderPaid-тэй адил).
	m.resetFaults = func(step string) error {
		if step == "start" {
			m.mu.Lock()
			m.orders["o3"] = &Order{ID: "o3", Kind: OrderKindFee, UserID: "u1", CourseID: "c1", LessonID: "l3", Status: OrderPaid}
			m.mu.Unlock()
			_ = m.SaveQuizResult(ctx, "u1", "c1", "l3", FeePassKey, true)
		}
		return nil
	}
	if _, err := m.ResetCourseStudent(ctx, "c1", "u1", "t"); err != nil {
		t.Fatal(err)
	}
	ps, _ := m.LessonPasses(ctx, "u1", "c1")
	if !ps["l1"][LatePassKey] || !ps["l3"][FeePassKey] {
		t.Fatalf("устгалын хооронд төлсөн эрх сэргэх ёстой: %+v", ps)
	}
	if pr, _ := m.LessonProgress(ctx, "u1", "c1"); len(pr) != 0 {
		t.Fatalf("эрхийн мөрүүд үзээгүй байх ёстой: %+v", pr)
	}
}

// R5: зөвхөн эрхийн мөр нүүрний эрэмбэд «суралцсан сургалт»-д тооцогдохгүй.
func TestStudentRanksIgnorePassRows(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	teacher := &User{Username: "srt", Email: "srt@x.mn", Role: RoleTeacher}
	_ = m.CreateUser(ctx, teacher)
	co := &Course{TeacherID: teacher.ID, Title: "Нэг"}
	_ = m.CreateCourse(ctx, co)
	m.lessons[co.ID] = []*Lesson{{ID: "l1", CourseID: co.ID}}
	_ = m.GrantLessonPass(ctx, "u1", co.ID, "l1", FeePassKey)
	rs, err := m.StudentRanks(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if r := rs["u1"]; r != nil && r.Courses != 0 {
		t.Fatalf("эрхийн мөр сургалтад тооцогдлоо: %+v", r)
	}
	_ = m.MarkLessonViewed(ctx, "u1", co.ID, "l1")
	if rs, _ := m.StudentRanks(ctx, nil); rs["u1"] == nil || rs["u1"].Courses != 1 {
		t.Fatalf("үзсэний дараа: %+v", rs["u1"])
	}
}

// B9: DELETE ба шалгах асуулгын параметрийн тоо, хичээлээр түлхүүрлэсэн хүснэгтийн нөхцөл, хичээлгүй сургалт.
func TestResetStatements(t *testing.T) {
	count := func(t *testing.T, what, sql string, args []any) {
		t.Helper()
		if n := strings.Count(sql, "?"); n != len(args) {
			t.Fatalf("%s: %d «?», %d утга: %s", what, n, len(args), sql)
		}
	}
	lessonKeyed := map[string]bool{"lesson_progress": true, "submissions": true, "highlights": true, "cloud_answers": true, "review_reminders": true, "board_rooms": true}
	for _, lids := range [][]string{{"l1", "l2"}, {}} {
		stmts := resetPhase1("c1", "own", "u1", lids, lids)
		tables := map[string]bool{}
		for _, st := range stmts {
			tables[st.table] = true
			count(t, st.table, st.sql(), st.args)
			if !strings.HasPrefix(st.sql(), "DELETE FROM "+st.table+" WHERE ") {
				t.Fatalf("%s: %s", st.table, st.sql())
			}
			if lessonKeyed[st.table] && len(lids) > 0 && !strings.Contains(st.where, "has(?, lesson_id)") {
				t.Fatalf("%s: хичээлийн нөхцөлгүй: %s", st.table, st.where)
			}
			if (st.table == "activity_events" || st.table == "quiz_logs") && (!strings.HasPrefix(st.where, "teacher_id = ?") || st.args[0] != "own") {
				t.Fatalf("%s: эзэмшигчийн teacher_id: %s %v", st.table, st.where, st.args)
			}
			for _, a := range st.args {
				if s, ok := a.([]string); ok && len(s) == 0 {
					t.Fatalf("%s: хоосон массив дамжуулав", st.table)
				}
			}
		}
		for _, tb := range []string{"lesson_progress", "study_sessions", "activity_events", "quiz_logs", "reflections", "video_watches", "exam_attempts", "submissions"} {
			if !tables[tb] {
				t.Fatalf("%v: %s алга", lids, tb)
			}
		}
		if len(lids) == 0 && (tables["highlights"] || tables["cloud_answers"] || tables["review_reminders"] || tables["board_rooms"]) {
			t.Fatalf("хичээлгүй үед зөвхөн хичээлээр түлхүүрлэсэн хүснэгт: %v", tables)
		}
		if len(lids) > 0 && len(tables) != 13 {
			t.Fatalf("хүснэгтийн тоо: %v", tables)
		}
		rk := resetRankStmt("c1", "u1")
		count(t, "rank_points", rk.sql(), rk.args)
		q, args := resetProbe("c1", "own", "u1", lids)
		count(t, "probe", q, args)
		want := len(resetPhase1("c1", "own", "u1", lids, nil)) + 1 // + rank_points
		if len(lids) > 0 {
			want++ // board_rooms
		}
		if n := strings.Count(q, "(SELECT "); n != want || strings.Count(q, " UNION ALL ") != want-1 {
			t.Fatalf("probe: %d хэсэг (%d хүлээсэн): %s", n, want, q)
		}
		if len(lids) > 0 != strings.Contains(q, "board_rooms") {
			t.Fatalf("probe самбар: %s", q)
		}
		// R5: probe нь зөвхөн эрхийн (үзээгүй) явцын мөрийг өгөгдөлд тооцохгүй; DELETE нь бүх мөрийг устгана.
		if !strings.Contains(q, "FROM lesson_progress WHERE user_id = ? AND ") || !strings.Contains(q, progressViewedCond+" LIMIT 1)") {
			t.Fatalf("probe явц: %s", q)
		}
		for _, st := range stmts {
			if strings.Contains(st.sql(), "viewed_at") {
				t.Fatalf("DELETE нь эрхийн мөрийг ч устгах ёстой: %s", st.sql())
			}
		}
	}
	// Самбарын устгал зөвхөн "u:<uid>" өрөөтэй хичээлүүдээр.
	for _, st := range resetPhase1("c1", "own", "u1", []string{"l1", "l2"}, []string{"l2"}) {
		if st.table == "board_rooms" && (len(st.args[0].([]string)) != 1 || st.args[1] != "u:u1") {
			t.Fatalf("самбар: %v", st.args)
		}
	}
}

// B11: эрхийн мөр хичээлийг үзсэн болгохгүй; жинхэнэ үзэлт, хариулт, дуусгалт ирэхэд viewed болно, эрх хэвээр.
func TestGrantLessonPassNotViewed(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	m.lessons["c1"] = []*Lesson{{ID: "l1", CourseID: "c1"}, {ID: "l2", CourseID: "c1"}, {ID: "l3", CourseID: "c1"}}
	for _, lid := range []string{"l1", "l2", "l3"} {
		if err := m.GrantLessonPass(ctx, "u1", "c1", lid, LatePassKey, FeePassKey, "q1"); err != nil {
			t.Fatal(err)
		}
	}
	if pr, _ := m.LessonProgress(ctx, "u1", "c1"); len(pr) != 0 {
		t.Fatalf("эрхийн мөр явцад харагдлаа: %+v", pr)
	}
	if cp, _ := m.CourseProgress(ctx, "c1"); len(cp) != 0 {
		t.Fatalf("эрхийн мөр сургалтын явцад харагдлаа: %+v", cp)
	}
	ps, _ := m.LessonPasses(ctx, "u1", "c1")
	if len(ps) != 3 || !ps["l1"][LatePassKey] || ps["l1"]["q1"] {
		t.Fatalf("эрх (зөвхөн эрхийн түлхүүр): %+v", ps)
	}
	_ = m.MarkLessonViewed(ctx, "u1", "c1", "l1")
	_ = m.SaveQuizResult(ctx, "u1", "c1", "l2", "q1", false)
	_ = m.SaveQuizResult(ctx, "u1", "c1", "l3", FeePassKey, true) // эрхийн түлхүүр — үзсэнд тооцохгүй
	pr, _ := m.LessonProgress(ctx, "u1", "c1")
	if len(pr) != 2 || pr["l1"].ViewedAt.Unix() <= 0 || pr["l2"].ViewedAt.Unix() <= 0 || !pr["l1"].Quiz[LatePassKey] {
		t.Fatalf("үзсэний дараа: %+v", pr)
	}
	_ = m.MarkLessonCompleted(ctx, "u1", "c1", "l3")
	if pr, _ := m.LessonProgress(ctx, "u1", "c1"); pr["l3"].CompletedAt == nil || pr["l3"].ViewedAt.Unix() <= 0 {
		t.Fatalf("дуусгасны дараа: %+v", pr["l3"])
	}
	// Ердийн төлбөр (MarkOrderPaid) өмнөх шигээ үзсэн мөр үүсгэнэ — өөрчлөгдөөгүй.
	_ = m.SaveQuizResult(ctx, "u2", "c1", "l1", LatePassKey, true)
	if pr, _ := m.LessonProgress(ctx, "u2", "c1"); pr["l1"].ViewedAt.Unix() <= 0 {
		t.Fatalf("SaveQuizResult-ийн шинэ мөр: %+v", pr)
	}
}

// B8: сургалт устгах, багш бүртгэлээ устгахад журналын мөрүүд устана.
func TestCourseMetaRemovedWithCourseAndOwner(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	owner := &User{Username: "own", Email: "own@x.mn", Role: RoleTeacher, DisplayName: "Багш"}
	t2 := &User{Username: "t2", Email: "t2@x.mn", Role: RoleTeacher, DisplayName: "Өөр багш"}
	_ = m.CreateUser(ctx, owner)
	_ = m.CreateUser(ctx, t2)
	c1 := &Course{TeacherID: owner.ID, Title: "Нэг"}
	c2 := &Course{TeacherID: owner.ID, Title: "Хоёр"}
	other := &Course{TeacherID: t2.ID, Title: "Өөр"}
	for _, c := range []*Course{c1, c2, other} {
		if err := m.CreateCourse(ctx, c); err != nil {
			t.Fatal(err)
		}
		setMeta(t, m, c.ID, "s1", func(x *CourseStudentMeta) { x.Alias = "Бат" })
	}
	if err := m.DeleteCourse(ctx, owner.ID, c1.ID); err != nil {
		t.Fatal(err)
	}
	if len(mustMetas(t, m, c1.ID)) != 0 || len(mustMetas(t, m, c2.ID)) != 1 {
		t.Fatal("сургалт устгахад журналын мөр")
	}
	if err := m.DeleteUserData(ctx, owner.ID); err != nil {
		t.Fatal(err)
	}
	if len(mustMetas(t, m, c2.ID)) != 0 || len(mustMetas(t, m, other.ID)) != 1 {
		t.Fatal("эзэмшигч бүртгэлээ устгахад журналын мөр")
	}
}
