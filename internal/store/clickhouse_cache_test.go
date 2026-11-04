package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Уншилтын кэш, сесс/үзэлтийн буфер бодит ClickHouse дээр: бичилт бүр кэшийг шинэчилж, буцаасан утгыг өөрчлөхөд
// кэш эвдрэхгүй, буферийн өөрчлөлт уншихад харагдаж, flush-ийн дараа санд бичигдэнэ.
// SURGALT_TEST_CLICKHOUSE_DSN тохируулсан үед л ажиллана (түр t_xxxx сан үүсгээд устгана).
func TestClickHouseReadCacheAndBuffers(t *testing.T) {
	ch := testClickHouse(t)
	ctx := context.Background()
	if ch.rc == nil || ch.sess == nil || ch.visits == nil {
		t.Fatal("нэг хуулбарт кэш, буфер асах ёстой")
	}

	// Хэрэглэгч: кэш, гүн хуулбар, шинэчлэл.
	u := &User{Username: "cu1", Email: "cu1@x.mn", Role: RoleStudent, DisplayName: "Анх"}
	if err := ch.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	got, err := ch.UserByID(ctx, u.ID)
	if err != nil || got.DisplayName != "Анх" {
		t.Fatalf("хэрэглэгч: %v %v", got, err)
	}
	got.DisplayName = "эвдэв"
	if again, _ := ch.UserByID(ctx, u.ID); again.DisplayName != "Анх" {
		t.Fatal("буцаасан объектыг өөрчлөхөд кэш эвдэрлээ")
	}
	if _, err := ch.UpdateUser(ctx, u.ID, func(x *User) error { x.DisplayName = "Шинэ"; return nil }); err != nil {
		t.Fatal(err)
	}
	if again, _ := ch.UserByID(ctx, u.ID); again.DisplayName != "Шинэ" {
		t.Fatalf("шинэчлэлийн дараа хуучин нэр: %s", again.DisplayName)
	}

	// Сургалт, хичээл.
	tch := &User{Username: "ct1", Email: "ct1@x.mn", Role: RoleTeacher, DisplayName: "Багш"}
	if err := ch.CreateUser(ctx, tch); err != nil {
		t.Fatal(err)
	}
	co := &Course{TeacherID: tch.ID, Title: "Кэш", Published: true}
	if err := ch.CreateCourse(ctx, co); err != nil {
		t.Fatal(err)
	}
	if c1, _ := ch.CourseByID(ctx, co.ID); c1.LessonCount != 0 {
		t.Fatalf("хичээлийн тоо: %d", c1.LessonCount)
	}
	l := &Lesson{CourseID: co.ID, Title: "Нэг", IsFree: true}
	if err := ch.CreateLesson(ctx, l); err != nil {
		t.Fatal(err)
	}
	if c1, _ := ch.CourseByID(ctx, co.ID); c1.LessonCount != 1 {
		t.Fatalf("хичээл нэмсний дараа тоо: %d", c1.LessonCount)
	}
	l.Title = "Нэг (зассан)"
	if err := ch.UpdateLesson(ctx, l); err != nil {
		t.Fatal(err)
	}
	if gl, _ := ch.LessonByID(ctx, co.ID, l.ID); gl.Title != "Нэг (зассан)" {
		t.Fatalf("засварын дараа хуучин гарчиг: %s", gl.Title)
	}
	if ls, _ := ch.LessonsByCourse(ctx, co.ID); len(ls) != 1 || ls[0].Title != "Нэг (зассан)" {
		t.Fatalf("хичээлийн жагсаалт: %v", ls)
	}
	l2 := &Lesson{CourseID: co.ID, Title: "Хоёр", IsFree: true}
	if err := ch.CreateLesson(ctx, l2); err != nil {
		t.Fatal(err)
	}
	if err := ch.DeleteLesson(ctx, co.ID, l.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := ch.LessonByID(ctx, co.ID, l.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("устгасан хичээл олдлоо: %v", err)
	}

	// Элсэлт: элсээгүйг кэшлэхгүй — элсмэгц true.
	if ok, _ := ch.IsEnrolled(ctx, u.ID, co.ID); ok {
		t.Fatal("элсээгүй")
	}
	if err := ch.Enroll(ctx, u.ID, co); err != nil {
		t.Fatal(err)
	}
	if ok, _ := ch.IsEnrolled(ctx, u.ID, co.ID); !ok {
		t.Fatal("элссэний дараа хуучин «элсээгүй» буцлаа")
	}

	// Сессийн буфер: beat санд биш санах ойд; уншихад харагдана; flush-ийн дараа санд.
	s := &StudySession{UserID: u.ID, UserName: "Шинэ", CourseID: co.ID, LessonID: l2.ID, TeacherID: tch.ID, Kind: "lesson", Title: l2.Title, StartedAt: time.Now(), LastAt: time.Now()}
	if err := ch.CreateSession(ctx, s); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if err := ch.AddSessionBeat(ctx, s.ID, SessionBeat{ActiveSec: 10, Counts: map[string]int{"tab_switch": 1}}); err != nil {
			t.Fatal(err)
		}
	}
	if gs, _ := ch.SessionByID(ctx, s.ID); gs.ActiveSec != 30 || gs.Counts["tab_switch"] != 3 {
		t.Fatalf("санах ойн сесс: %+v", gs)
	}
	if ss, _ := ch.Sessions(ctx, ActivityFilter{UserID: u.ID}, 10); len(ss) != 1 || ss[0].ActiveSec != 30 {
		t.Fatalf("Sessions overlay: %+v", ss)
	}
	if db, _ := ch.sessionByIDDB(ctx, s.ID); db.ActiveSec != 0 {
		t.Fatalf("flush-ээс өмнө санд бичигдсэн: %d", db.ActiveSec)
	}
	ch.sess.flush(ctx)
	if db, _ := ch.sessionByIDDB(ctx, s.ID); db.ActiveSec != 30 || db.Counts["tab_switch"] != 3 {
		t.Fatalf("flush-ийн дараа санд: %+v", db)
	}

	// Үзэлтийн буфер.
	v := &PageVisit{TeacherID: tch.ID, Kind: "course", TargetID: co.ID, VisitorID: "a:test", At: time.Now()}
	if err := ch.SaveVisit(ctx, v); err != nil {
		t.Fatal(err)
	}
	v.Seconds = 42
	if err := ch.SaveVisit(ctx, v); err != nil {
		t.Fatal(err)
	}
	if gv, _ := ch.VisitByID(ctx, v.ID); gv.Seconds != 42 {
		t.Fatalf("үзэлт: %d", gv.Seconds)
	}
	ch.visits.flush(ctx)
	if dv, _ := ch.visitByIDDB(ctx, v.ID); dv.Seconds != 42 {
		t.Fatalf("flush-ийн дараа санд: %d", dv.Seconds)
	}

	// Сургалт устгах.
	if err := ch.DeleteCourse(ctx, tch.ID, co.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := ch.CourseByID(ctx, co.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("устгасан сургалт олдлоо: %v", err)
	}
}
