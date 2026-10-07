package main

import (
	"context"
	"testing"

	"surgalt/internal/files"
	"surgalt/internal/store"
)

// Жишээ сургалт: хуучин санд ч нэг л удаа нэмэгдэж, дүрэм бүр хичээл тус бүрт, суралцагч элссэн байна.
func TestSeedRulesDemo(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemory()
	fs, err := files.New(t.TempDir(), []byte("k"), 10<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := seedDemo(ctx, st, fs); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := seedRulesDemo(ctx, st); err != nil {
			t.Fatal(err)
		}
	}
	tch, _ := st.UserByUsername(ctx, "demo")
	cs, _ := st.CoursesByTeacher(ctx, tch.ID, false)
	var demo *store.Course
	n := 0
	for i := range cs {
		if cs[i].Title == rulesDemoTitle {
			demo, n = &cs[i], n+1
		}
	}
	if n != 1 || !demo.Drip || demo.UnlockAllPaid {
		t.Fatalf("жишээ сургалт нэг л удаа, дараалалтай: %d %+v", n, demo)
	}
	ls, _ := st.LessonsByCourse(ctx, demo.ID)
	rules := map[string]bool{}
	for _, l := range ls {
		rules[l.UnlockRule] = true
	}
	for _, r := range []string{"", "view", "quiz", "active", "quiz_active", "complete", "exam", "manual"} {
		if !rules[r] {
			t.Fatalf("%q дүрэм алга", r)
		}
	}
	if len(ls) != 10 || !ls[9].AlwaysOpen || ls[5].Exam == nil {
		t.Fatalf("хичээлүүд: %d", len(ls))
	}
	s, _ := st.UserByUsername(ctx, "suragch")
	if ok, _ := st.IsEnrolled(ctx, s.ID, demo.ID); !ok {
		t.Fatal("демо суралцагч элссэн байх ёстой")
	}
}
