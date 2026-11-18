package store

import (
	"context"
	"testing"
)

// Багшийн нээсэн хичээл: (суралцагч, сургалт)-аар уншина; бүртгэл устгахад суралцагчийнх, сургалт устгахад
// тэр сургалтынх устна (журналын «бүрмөсөн устгах» хөндөхгүйг TestAccessRequestManualLesson шалгана).
func TestLessonGrantsMemory(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	teacher := &User{Username: "lgt", Email: "lgt@x.mn", Role: RoleTeacher}
	stu := &User{Username: "lgs", Email: "lgs@x.mn", Role: RoleStudent}
	other := &User{Username: "lgo", Email: "lgo@x.mn", Role: RoleStudent}
	for _, u := range []*User{teacher, stu, other} {
		if err := m.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	c1 := &Course{TeacherID: teacher.ID, Title: "Нэг"}
	c2 := &Course{TeacherID: teacher.ID, Title: "Хоёр"}
	_ = m.CreateCourse(ctx, c1)
	_ = m.CreateCourse(ctx, c2)
	if g, err := m.LessonGrants(ctx, stu.ID, c1.ID); err != nil || g == nil || len(g) != 0 {
		t.Fatalf("хоосон: %v %v", g, err)
	}
	for _, x := range []struct{ u, c, l string }{{stu.ID, c1.ID, "l1"}, {stu.ID, c1.ID, "l2"}, {stu.ID, c2.ID, "l9"}, {other.ID, c1.ID, "l1"}} {
		if err := m.GrantLesson(ctx, x.u, x.c, x.l, teacher.ID); err != nil {
			t.Fatal(err)
		}
	}
	_ = m.GrantLesson(ctx, stu.ID, c1.ID, "l1", teacher.ID) // давтан — хэвээр
	if g, _ := m.LessonGrants(ctx, stu.ID, c1.ID); len(g) != 2 || !g["l1"] || !g["l2"] {
		t.Fatalf("c1: %v", g)
	}
	if err := m.DeleteCourse(ctx, teacher.ID, c1.ID); err != nil {
		t.Fatal(err)
	}
	if g, _ := m.LessonGrants(ctx, other.ID, c1.ID); len(g) != 0 {
		t.Fatalf("устгасан сургалтын эрх үлдлээ: %v", g)
	}
	if g, _ := m.LessonGrants(ctx, stu.ID, c2.ID); !g["l9"] {
		t.Fatalf("өөр сургалтын эрх устах ёсгүй: %v", g)
	}
	if err := m.DeleteUserData(ctx, stu.ID); err != nil {
		t.Fatal(err)
	}
	if g, _ := m.LessonGrants(ctx, stu.ID, c2.ID); len(g) != 0 {
		t.Fatalf("устгасан бүртгэлийн эрх үлдлээ: %v", g)
	}
}
