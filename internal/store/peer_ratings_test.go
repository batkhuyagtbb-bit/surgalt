package store

import (
	"context"
	"testing"
	"time"
)

// Харилцан үнэлгээ (санах ой): (хичээл тус бүр) шинэчлэхэд сүүлийнх үлдэнэ, хичээлийн үнэлгээ ерөнхийгөөс тусдаа, устгал
// зөвхөн тэр хичээлийнх, (ratee, rater, lesson)-оор эрэмбэлсэн; бүртгэл устгахад өгсөн ба авсан нь, сургалт устгахад тэр
// сургалтынх, журналын устгалд тэр суралцагчийн тэр сургалтад өгсөн ба авсан (бүх хичээлийнх) л устана.
func TestPeerRatingsMemory(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	cut := time.Now().Add(-12 * time.Hour) // өдрийн хил: эндээс хойшхи бичлэг «өнөөдрийнх»
	pairs := func(cid string) []string {
		ps, err := m.PeerRatings(ctx, cid)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, len(ps))
		for i, p := range ps {
			out[i] = p.RaterID + ">" + p.RateeID
			if p.LessonID != "" {
				out[i] += "@" + p.LessonID
			}
			if p.CourseID != cid || p.UpdatedAt.IsZero() {
				t.Fatalf("мөр: %+v", p)
			}
		}
		return out
	}
	set := func(cid, a, b, lid string, score int) {
		t.Helper()
		if _, err := m.SetPeerRating(ctx, cid, a, b, lid, score, cut); err != nil {
			t.Fatal(err)
		}
	}
	set("c1", "u1", "u2", "", 80)
	set("c1", "u1", "u2", "", 40) // шинэчлэл
	set("c1", "u1", "u2", "l1", 90)
	set("c1", "u3", "u2", "", 90)
	set("c1", "u2", "u1", "", 70)
	set("c2", "u1", "u2", "", 10)
	if got := pairs("c1"); len(got) != 4 || got[0] != "u2>u1" || got[1] != "u1>u2" || got[2] != "u1>u2@l1" || got[3] != "u3>u2" {
		t.Fatalf("жагсаалт: %v", got)
	}
	ps, _ := m.PeerRatings(ctx, "c1")
	if ps[1].Score != 40 || ps[2].Score != 90 {
		t.Fatalf("сүүлийн оноо үлдэх ёстой, хичээлийнх тусдаа: %+v", ps)
	}
	if err := m.DeletePeerRating(ctx, "c1", "u1", "u2", "l1", cut); err != nil {
		t.Fatal(err)
	}
	if got := pairs("c1"); len(got) != 3 || got[1] != "u1>u2" {
		t.Fatalf("хичээлийн үнэлгээ устгахад ерөнхий нь үлдэх ёстой: %v", got)
	}
	if err := m.DeletePeerRating(ctx, "c1", "u3", "u2", "", cut); err != nil {
		t.Fatal(err)
	}
	if err := m.DeletePeerRating(ctx, "c1", "u3", "u2", "", cut); err != nil { // байхгүйг устгах — алдаагүй
		t.Fatal(err)
	}
	if got := pairs("c1"); len(got) != 2 {
		t.Fatalf("устгасны дараа: %v", got)
	}

	// Журналын устгал: u1-ийн c1 дэх өгсөн ба авсан (хичээлийнх ч); u2→u3 (өөр хүмүүс), c2 хэвээр.
	m.lessons["c1"] = []*Lesson{{ID: "l1", CourseID: "c1"}}
	set("c1", "u2", "u3", "", 55)
	set("c1", "u2", "u1", "l1", 65)
	now := time.Now()
	setMeta(t, m, "c1", "u1", func(c *CourseStudentMeta) { c.Archived, c.ArchivedAt = true, &now })
	res, err := m.ResetCourseStudent(ctx, "c1", "u1", "t")
	if err != nil || !res.Wiped {
		t.Fatalf("журналын устгал: %+v %v", res, err)
	}
	if got := pairs("c1"); len(got) != 1 || got[0] != "u2>u3" {
		t.Fatalf("журналын устгалын дараа c1: %v", got)
	}
	if got := pairs("c2"); len(got) != 1 {
		t.Fatalf("өөр сургалт хөндөгдлөө: %v", got)
	}

	// Бүртгэл устгах: u2-ийн өгсөн (c1) ба авсан (c2).
	u := &User{Username: "peer2", Email: "p2@x.mn", Role: RoleStudent}
	if err := m.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	set("c1", u.ID, "u3", "l1", 60)
	set("c2", "u9", u.ID, "", 60)
	set("c2", "u8", "u9", "", 50)
	if err := m.DeleteUserData(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	for _, cid := range []string{"c1", "c2"} {
		for _, p := range pairs(cid) {
			if p == u.ID+">u3@l1" || p == "u9>"+u.ID {
				t.Fatalf("устгасан хэрэглэгчийн үнэлгээ үлдлээ (%s): %v", cid, p)
			}
		}
	}
	if got := pairs("c2"); len(got) != 2 { // u1>u2, u8>u9
		t.Fatalf("бусдын үнэлгээ хөндөгдлөө: %v", got)
	}

	// Сургалт устгах: тэр сургалтынх бүгд.
	m.courses["c2"] = &Course{ID: "c2", TeacherID: "t1"}
	if err := m.DeleteCourse(ctx, "t1", "c2"); err != nil {
		t.Fatal(err)
	}
	if got := pairs("c2"); len(got) != 0 {
		t.Fatalf("сургалт устгасны дараа: %v", got)
	}
	if got := pairs("c1"); len(got) != 1 {
		t.Fatalf("өөр сургалт хөндөгдлөө: %v", got)
	}

	// Тестийн setter: updated_at өөрчлөгдөнө.
	old := time.Now().Add(-48 * time.Hour)
	if !m.SetPeerRatingAt("c1", "u2", "u3", "", old) {
		t.Fatal("SetPeerRatingAt")
	}
	if ps, _ := m.PeerRatings(ctx, "c1"); !ps[0].UpdatedAt.Equal(old) {
		t.Fatalf("updated_at: %+v", ps)
	}
}

// Өдрийн snapshot-ийг авч явах: хилээс өмнөх үнэлгээг өнөөдөр засах/устгахад хилийн үеийн утга (SnapshotAt) өөрчлөгдөхгүй;
// өнөөдөр дахин засахад анхны snapshot хэвээр; маргаашийн хилд шинэ төлөв (устгасан бол байхгүй); өнөөдөр анх өгсөн
// үнэлгээ хилд байхгүй; snapshot хэрэггүй устгал бүр мөсөн хасагдана.
func TestPeerRatingSnapshotMemory(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	cut := time.Now().Add(-12 * time.Hour)
	yesterday := cut.Add(-time.Hour)
	if _, err := m.SetPeerRating(ctx, "c1", "a", "b", "", 80, cut); err != nil {
		t.Fatal(err)
	}
	m.SetPeerRatingAt("c1", "a", "b", "", yesterday)
	row := func() PeerRating {
		t.Helper()
		ps, _ := m.PeerRatings(ctx, "c1")
		for _, p := range ps {
			if p.RaterID == "a" && p.RateeID == "b" && p.LessonID == "" {
				return p
			}
		}
		t.Fatalf("мөр алга: %+v", ps)
		return PeerRating{}
	}
	snap := func(at time.Time) (int, bool) {
		s, _, ok := row().SnapshotAt(at)
		return s, ok
	}
	w, err := m.SetPeerRating(ctx, "c1", "a", "b", "", 20, cut) // өнөөдөр засав
	if err != nil || !w.SnapOK || w.SnapScore != 80 || !w.SnapAt.Equal(yesterday) || w.Score != 20 {
		t.Fatalf("засварын мөр: %+v %v", w, err)
	}
	if s, ok := snap(cut); !ok || s != 80 {
		t.Fatalf("өнөөдрийн хилд 80 хэвээр: %d %v", s, ok)
	}
	m.SetPeerRating(ctx, "c1", "a", "b", "", 40, cut) // өнөөдөр дахин
	if s, ok := snap(cut); !ok || s != 80 {
		t.Fatalf("дахин засахад анхны snapshot хэвээр: %d %v", s, ok)
	}
	if err := m.DeletePeerRating(ctx, "c1", "a", "b", "", cut); err != nil {
		t.Fatal(err)
	}
	if p := row(); !p.Deleted || !p.SnapOK {
		t.Fatalf("tombstone: %+v", p)
	}
	if s, ok := snap(cut); !ok || s != 80 {
		t.Fatalf("өнөөдөр устгасан ч хилд 80: %d %v", s, ok)
	}
	if _, ok := snap(time.Now().Add(time.Hour)); ok {
		t.Fatal("маргаашийн хилд устгасан үнэлгээ үлдлээ")
	}
	// Өнөөдөр анх өгсөн үнэлгээ хилд байхгүй; устгахад бүр мөсөн хасагдана.
	w, _ = m.SetPeerRating(ctx, "c1", "c", "b", "l1", 90, cut)
	if w.SnapOK {
		t.Fatalf("шинэ үнэлгээнд snapshot: %+v", w)
	}
	if _, _, ok := w.SnapshotAt(cut); ok {
		t.Fatal("өнөөдрийн шинэ үнэлгээ хилд орлоо")
	}
	if _, _, ok := w.SnapshotAt(time.Now().Add(time.Hour)); !ok {
		t.Fatal("маргаашийн хилд шинэ үнэлгээ орох ёстой")
	}
	_ = m.DeletePeerRating(ctx, "c1", "c", "b", "l1", cut)
	if ps, _ := m.PeerRatings(ctx, "c1"); len(ps) != 1 {
		t.Fatalf("snapshot-гүй устгал үлдлээ: %+v", ps)
	}
	// Устгасны дараа (өнөөдөр) дахин өгвөл snapshot нь устгалаас өмнөх 80 хэвээр.
	w, _ = m.SetPeerRating(ctx, "c1", "a", "b", "", 60, cut)
	if !w.SnapOK || w.SnapScore != 80 || w.Deleted {
		t.Fatalf("устгасны дараах дахин үнэлгээ: %+v", w)
	}
	// Хуучин (хилээс өмнө) устгасан tombstone-ийн дараах үнэлгээ snapshot-гүй.
	m.SetPeerRating(ctx, "c1", "x", "y", "", 50, cut)
	m.SetPeerRatingAt("c1", "x", "y", "", yesterday.Add(-time.Hour))
	m.SetPeerRating(ctx, "c1", "x", "y", "", 55, cut) // snapshot: 50
	m.ShiftPeerRatings("c1", 24*time.Hour)            // нэг хоног өнгөрөв
	_ = m.DeletePeerRating(ctx, "c1", "x", "y", "", cut)
	w, _ = m.SetPeerRating(ctx, "c1", "x", "y", "", 70, cut)
	if !w.SnapOK || w.SnapScore != 55 {
		t.Fatalf("хоног өнгөрсний дараах устгал+дахин үнэлгээ (snapshot = өчигдрийн 55): %+v", w)
	}
}

// Үнэлэгчийн баримт: элссэн (эсвэл төлсөн хичээлийн эрх) мөч, сургалтын сесс эсвэл бүлэг чатын мессежийн хамгийн эрт
// мөч; өөр сургалтынх тооцогдохгүй; CountsAt — 3 хоног + идэвх.
func TestPeerRaterFactsMemory(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	co := &Course{ID: "c1", TeacherID: "t1"}
	m.courses["c1"] = co
	m.courses["c2"] = &Course{ID: "c2", TeacherID: "t1"}
	day := 24 * time.Hour
	_ = m.Enroll(ctx, "u1", co)
	m.SetEnrolledAt("u1", "c1", time.Now().Add(-5*day))
	_ = m.Enroll(ctx, "u2", co)
	_ = m.Enroll(ctx, "u3", co)
	m.SetEnrolledAt("u3", "c1", time.Now().Add(-5*day))
	_ = m.CreateSession(ctx, &StudySession{UserID: "u1", CourseID: "c1", StartedAt: time.Now().Add(-2 * day), LastAt: time.Now()})
	_ = m.CreateSession(ctx, &StudySession{UserID: "u1", CourseID: "c1", StartedAt: time.Now().Add(-4 * day), LastAt: time.Now()})
	_ = m.CreateSession(ctx, &StudySession{UserID: "u3", CourseID: "c2", StartedAt: time.Now().Add(-4 * day), LastAt: time.Now()}) // өөр сургалт
	conv, err := m.GetOrCreateGroupConversation(ctx, "t1", "c1", "Анги")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.AddMessage(ctx, &Message{ConversationID: conv.ID, Sender: SenderVisitor, SenderID: "u2", Body: "сайн уу"}); err != nil {
		t.Fatal(err)
	}
	f, err := m.PeerRaterFacts(ctx, "c1", "t1", []string{"u1", "u2", "u3", "u4"})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if f["u1"].JoinedAt.IsZero() || f["u1"].ActiveAt.After(now.Add(-4*day+time.Minute)) || !f["u1"].CountsAt(now, 72*time.Hour) {
		t.Fatalf("u1 (5 хоног, сесс): %+v", f["u1"])
	}
	if f["u2"].ActiveAt.IsZero() || f["u2"].CountsAt(now, 72*time.Hour) {
		t.Fatalf("u2 (шинэ, мессежтэй): %+v", f["u2"])
	}
	if !f["u3"].ActiveAt.IsZero() || f["u3"].CountsAt(now, 72*time.Hour) {
		t.Fatalf("u3 (идэвхгүй, өөр сургалтын сесс): %+v", f["u3"])
	}
	if _, ok := f["u4"]; ok {
		t.Fatalf("мэдээлэлгүй хүн: %+v", f)
	}
	// u1 нь 3 хоногийн өмнө (элсснээс 2 хоног) тоологдохгүй.
	if f["u1"].CountsAt(now.Add(-3*day), 72*time.Hour) {
		t.Fatalf("u1 3 хоногийн өмнө: %+v", f["u1"])
	}
}

// Илгээсэн сануулга: SentAt тавигдаж, нэг хэрэглэгчийн since-ээс хойшхийг л уншина; давтан claim хийхгүй.
func TestSentReviewsMemory(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	got, _ := m.ClaimReviews(ctx, []ReviewItem{{UserID: "u1", CourseID: "c1", LessonID: "l1", Stage: 1}, {UserID: "u1", CourseID: "c1", LessonID: "l2", Stage: 3},
		{UserID: "u2", CourseID: "c1", LessonID: "l1", Stage: 1}})
	if len(got) != 3 || got[0].SentAt.IsZero() {
		t.Fatalf("claim: %+v", got)
	}
	if again, _ := m.ClaimReviews(ctx, []ReviewItem{{UserID: "u1", CourseID: "c1", LessonID: "l1", Stage: 1}}); len(again) != 0 {
		t.Fatalf("давтан claim: %+v", again)
	}
	if !m.SetReviewSentAt("u1", "l2", 3, time.Now().Add(-8*24*time.Hour)) {
		t.Fatal("SetReviewSentAt")
	}
	rs, err := m.SentReviews(ctx, "u1", time.Now().Add(-7*24*time.Hour))
	if err != nil || len(rs) != 1 || rs[0].LessonID != "l1" || rs[0].CourseID != "c1" || rs[0].Stage != 1 || rs[0].SentAt.IsZero() {
		t.Fatalf("SentReviews: %+v %v", rs, err)
	}
}
