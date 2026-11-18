package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeSessDB struct {
	mu     sync.Mutex
	rows   map[string]StudySession
	loads  int
	writes int
	fail   bool
}

func (f *fakeSessDB) load(_ context.Context, id string) (*StudySession, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loads++
	s, ok := f.rows[id]
	if !ok {
		return nil, ErrNotFound
	}
	s = deepCopy(s)
	return &s, nil
}

func (f *fakeSessDB) write(_ context.Context, ss []StudySession) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return errors.New("сан унасан")
	}
	f.writes++
	for _, s := range ss {
		f.rows[s.ID] = deepCopy(s)
	}
	return nil
}

func TestSessionBuffer(t *testing.T) {
	ctx := context.Background()
	db := &fakeSessDB{rows: map[string]StudySession{"s1": {ID: "s1", UserID: "u", LessonID: "l", Counts: map[string]int{}, LastAt: time.Now().Add(-time.Minute)}}}
	b := &sessionBuffer{load: db.load, write: db.write, m: map[string]*bufSess{}}

	// 40 beat (10 минут) — санд нэг ч удаа бичихгүй, нэг л удаа уншина.
	for i := 0; i < 40; i++ {
		if err := b.beat(ctx, "s1", SessionBeat{ActiveSec: 10, IdleSec: 5, Counts: map[string]int{"tab_switch": 1}}); err != nil {
			t.Fatal(err)
		}
	}
	if db.loads != 1 || db.writes != 0 {
		t.Fatalf("beat бүрт сан руу хандсан: уншилт %d, бичилт %d", db.loads, db.writes)
	}
	s, ok := b.get("s1")
	if !ok || s.ActiveSec != 400 || s.IdleSec != 200 || s.Counts["tab_switch"] != 40 {
		t.Fatalf("санах ойн сесс: %+v", s)
	}
	// Буцаасан хуулбарыг өөрчилсөн ч буфер өөрчлөгдөхгүй.
	s.Counts["tab_switch"] = 999
	if s2, _ := b.get("s1"); s2.Counts["tab_switch"] != 40 {
		t.Fatal("хуулбар биш, буферийн map буцаасан")
	}
	// Sessions() — сангийн хуучин мөрийг санах ойнхоор орлуулна.
	list := b.overlay([]StudySession{{ID: "s1", ActiveSec: 0}, {ID: "other", ActiveSec: 7}})
	for _, x := range list {
		if x.ID == "s1" && x.ActiveSec != 400 {
			t.Fatalf("overlay хийгдээгүй: %d", x.ActiveSec)
		}
	}
	// Сан алдаа өгвөл алдагдахгүй — дараагийн flush-д дахин бичнэ.
	db.fail = true
	b.flush(ctx)
	if db.writes != 0 || b.size() != 1 {
		t.Fatal("алдааны үед сесс алдагдлаа")
	}
	db.fail = false
	b.flush(ctx)
	if db.writes != 1 || db.rows["s1"].ActiveSec != 400 {
		t.Fatalf("flush: бичилт %d, санд %d", db.writes, db.rows["s1"].ActiveSec)
	}
	b.flush(ctx) // өөрчлөлтгүй — дахин бичихгүй
	if db.writes != 1 {
		t.Fatal("өөрчлөгдөөгүй сессийг дахин бичлээ")
	}
	// Дууссан сесс бичигдсэний дараа санах ойгоос гарна.
	_ = b.beat(ctx, "s1", SessionBeat{ActiveSec: 3, End: "closed"})
	b.flush(ctx)
	if b.size() != 0 || !db.rows["s1"].Ended || db.rows["s1"].ActiveSec != 403 {
		t.Fatalf("дууссан сесс: буферт %d, санд %+v", b.size(), db.rows["s1"])
	}
	// Олдоогүй сесс.
	if err := b.beat(ctx, "nope", SessionBeat{ActiveSec: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("олдоогүй сесс: %v", err)
	}
}

// Зэрэг beat — уралдаангүй (go test -race).
func TestSessionBufferConcurrent(t *testing.T) {
	ctx := context.Background()
	db := &fakeSessDB{rows: map[string]StudySession{}}
	for i := 0; i < 50; i++ {
		id := string(rune('a'+i%26)) + string(rune('A'+i/26))
		db.rows[id] = StudySession{ID: id, Counts: map[string]int{}}
	}
	b := &sessionBuffer{load: db.load, write: db.write, m: map[string]*bufSess{}}
	var ids []string
	for id := range db.rows {
		ids = append(ids, id)
	}
	var wg sync.WaitGroup
	for _, id := range ids {
		for k := 0; k < 4; k++ {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				for j := 0; j < 25; j++ {
					_ = b.beat(ctx, id, SessionBeat{ActiveSec: 1})
					if j%10 == 0 {
						b.flush(ctx)
					}
				}
			}(id)
		}
	}
	wg.Wait()
	b.flush(ctx)
	for id, s := range db.rows {
		if s.ActiveSec != 100 {
			t.Fatalf("%s: %d секунд (100 байх ёстой)", id, s.ActiveSec)
		}
	}
}

func TestDeepCopy(t *testing.T) {
	l := Lesson{ID: "l", Blocks: []Block{{ID: "b", Type: "quiz", Quiz: &Quiz{Question: "?", Options: []string{"a", "b"}}}}}
	c := deepCopy(l)
	c.Blocks[0].Quiz.Options[0] = "Z"
	c.Blocks[0].ID = "x"
	if l.Blocks[0].Quiz.Options[0] != "a" || l.Blocks[0].ID != "b" {
		t.Fatal("гүн хуулбар биш")
	}
	u := User{ID: "u", Subjects: []string{"мат"}}
	cu := deepCopy(u)
	cu.Subjects[0] = "физ"
	if u.Subjects[0] != "мат" {
		t.Fatal("хэрэглэгчийн slice хуваалцагдлаа")
	}
}

func TestLocalLockerFreesKeys(t *testing.T) {
	var l LocalLocker
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			unlock, _ := l.Lock(context.Background(), "k"+string(rune('a'+i%5)))
			unlock()
		}(i)
	}
	wg.Wait()
	if l.size() != 0 {
		t.Fatalf("суларсан түлхүүр санах ойд үлдлээ: %d", l.size())
	}
}

// Суралцагчийн өгөгдөл устгах үед: hold нь flush-ийг зогсоож, drop нь тохирох сессүүдийг (бичигдээгүй beat-ийн хамт)
// хаяна — сангаас устгасан сессийг буфер дахин бичихгүй, дараагийн beat ErrNotFound.
func TestSessionBufferDrop(t *testing.T) {
	ctx := context.Background()
	db := &fakeSessDB{rows: map[string]StudySession{
		"s1": {ID: "s1", UserID: "u1", CourseID: "c1", Counts: map[string]int{}},
		"s2": {ID: "s2", UserID: "u1", CourseID: "c2", Counts: map[string]int{}},
		"s3": {ID: "s3", UserID: "u2", CourseID: "c1", Counts: map[string]int{}},
	}}
	b := &sessionBuffer{load: db.load, write: db.write, m: map[string]*bufSess{}}
	for _, id := range []string{"s1", "s2", "s3"} {
		if err := b.beat(ctx, id, SessionBeat{ActiveSec: 10}); err != nil {
			t.Fatal(err)
		}
	}
	release := b.hold()
	done := make(chan struct{})
	go func() { b.flush(ctx); close(done) }()
	select {
	case <-done:
		t.Fatal("hold-ийн үед flush бичлээ")
	case <-time.After(50 * time.Millisecond):
	}
	if n := b.drop(func(s *StudySession) bool { return s.UserID == "u1" && s.CourseID == "c1" }); n != 1 {
		t.Fatalf("хаясан сесс: %d", n)
	}
	db.mu.Lock()
	delete(db.rows, "s1") // сангаас устгасан (DELETE)
	db.mu.Unlock()
	release()
	release() // давтан дуудахад аюулгүй
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("hold тавьсны дараа flush үргэлжилсэнгүй")
	}
	db.mu.Lock()
	_, s1 := db.rows["s1"]
	s2, s3 := db.rows["s2"].ActiveSec, db.rows["s3"].ActiveSec
	db.mu.Unlock()
	if s1 || s2 != 10 || s3 != 10 {
		t.Fatalf("flush: s1 дахин бичигдсэн=%v, s2=%d, s3=%d", s1, s2, s3)
	}
	if _, ok := b.get("s1"); ok {
		t.Fatal("хаясан сесс буферт үлдлээ")
	}
	if err := b.beat(ctx, "s1", SessionBeat{ActiveSec: 5}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("устгасан сессийн beat: %v", err)
	}
	if err := b.beat(ctx, "s2", SessionBeat{ActiveSec: 5}); err != nil {
		t.Fatalf("бусад сесс: %v", err)
	}
}

// B4: сангаас ачаалж эхэлсний дараа (DELETE-ээс өмнө уншсан) drop болж энэ сессийг хамарвал буферт дахин орохгүй —
// beat ErrNotFound; өөр суралцагчийн зэрэг ачаалсан сесс хэвийн.
func TestSessionBufferDropDuringLoad(t *testing.T) {
	ctx := context.Background()
	rows := map[string]StudySession{
		"s1": {ID: "s1", UserID: "u1", CourseID: "c1", Counts: map[string]int{}},
		"s2": {ID: "s2", UserID: "u2", CourseID: "c1", Counts: map[string]int{}},
	}
	started, proceed := make(chan string, 2), make(chan struct{})
	db := &fakeSessDB{rows: map[string]StudySession{}}
	b := &sessionBuffer{write: db.write, m: map[string]*bufSess{}, load: func(_ context.Context, id string) (*StudySession, error) {
		s := deepCopy(rows[id]) // уншилт DELETE-ээс өмнө
		started <- id
		<-proceed
		return &s, nil
	}}
	errs := make(chan error, 2)
	for _, id := range []string{"s1", "s2"} {
		go func() { errs <- b.beat(ctx, id, SessionBeat{ActiveSec: 5}) }()
	}
	<-started
	<-started
	b.drop(func(s *StudySession) bool { return s.UserID == "u1" && s.CourseID == "c1" })
	close(proceed)
	got := map[bool]int{}
	for range 2 {
		got[errors.Is(<-errs, ErrNotFound)]++
	}
	if got[true] != 1 || got[false] != 1 {
		t.Fatalf("нэг нь ErrNotFound, нөгөө нь амжилттай байх ёстой: %v", got)
	}
	if _, ok := b.get("s1"); ok {
		t.Fatal("устгасан сесс буферт дахин орлоо")
	}
	if s, ok := b.get("s2"); !ok || s.ActiveSec != 5 {
		t.Fatalf("өөр суралцагчийн сесс: %+v %v", s, ok)
	}
	// Бүртгэлээс хэтэрсэн олон drop — болгоомжтой татгалзана.
	gen := b.generation()
	for range bufDropLog + 1 {
		b.drop(func(*StudySession) bool { return false })
	}
	if b.keepFrom(&StudySession{ID: "s9", UserID: "u9"}, gen) {
		t.Fatal("бүртгэлээс хэтэрсэн үед буферт оруулах ёсгүй")
	}
	if !b.keepFrom(&StudySession{ID: "s9", UserID: "u9"}, b.generation()) {
		t.Fatal("drop-гүй үед буферт орох ёстой")
	}
}
