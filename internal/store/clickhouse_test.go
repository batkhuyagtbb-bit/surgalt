package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"
)

func testClickHouse(t *testing.T) *ClickHouse {
	t.Helper()
	dsn := os.Getenv("SURGALT_TEST_CLICKHOUSE_DSN")
	if dsn == "" {
		t.Skip("SURGALT_TEST_CLICKHOUSE_DSN тохируулаагүй")
	}
	u, _ := url.Parse(dsn)
	var rb [6]byte
	_, _ = rand.Read(rb[:])
	db := "t_" + hex.EncodeToString(rb[:])
	u.Path = "/" + db
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ch, err := NewClickHouse(ctx, ClickHouseOptions{DSN: u.String()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cctx, c2 := context.WithTimeout(context.Background(), 30*time.Second)
		defer c2()
		_ = ch.DropDatabase(cctx, db)
		ch.Close()
	})
	return ch
}

// Олон хуулбарын чат: нэг хуулбар мессеж бичихэд нөгөө нь ClickHouse-оос санамж авч хүргэнэ.
func TestClickHouseWatchMessages(t *testing.T) {
	ch := testClickHouse(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	teacher := &User{Username: "t1", Email: "t1@x.mn", Role: RoleTeacher, DisplayName: "T"}
	if err := ch.CreateUser(ctx, teacher); err != nil {
		t.Fatal(err)
	}
	conv, err := ch.GetOrCreateConversation(ctx, teacher.ID, "g:guest1", "Зочин", "")
	if err != nil {
		t.Fatal(err)
	}
	// Watcher эхлэхээс өмнөх мессеж дахин хүргэгдэх ёсгүй.
	if err := ch.AddMessage(ctx, &Message{ConversationID: conv.ID, Sender: SenderVisitor, Body: "хуучин"}); err != nil {
		t.Fatal(err)
	}
	got := make(chan Message, 10)
	if err := ch.WatchMessages(ctx, 100*time.Millisecond, nil, func(m Message) { got <- m }); err != nil {
		t.Fatal(err)
	}
	nots := make(chan Notification, 10)
	if err := ch.WatchNotifications(ctx, 100*time.Millisecond, nil, func(n Notification) { nots <- n }); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	// "Өөр хуулбар" бичлээ гэж үзье (ижил сан).
	if err := ch.AddMessage(ctx, &Message{ConversationID: conv.ID, Sender: SenderTeacher, Body: "шинэ"}); err != nil {
		t.Fatal(err)
	}
	if err := ch.AddNotifications(ctx, []*Notification{{UserID: teacher.ID, Type: "message", Title: "Шинэ мессеж"}}); err != nil {
		t.Fatal(err)
	}
	select {
	case m := <-got:
		if m.Body != "шинэ" || m.ConversationID != conv.ID || m.TeacherID != teacher.ID {
			t.Fatalf("буруу мессеж хүрсэн: %+v", m)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watcher мессежийг 5 сек дотор хүргэсэнгүй")
	}
	select {
	case n := <-nots:
		if n.Title != "Шинэ мессеж" {
			t.Fatalf("буруу мэдэгдэл: %+v", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watcher мэдэгдлийг хүргэсэнгүй")
	}
	select {
	case m := <-got:
		t.Fatalf("давхар/хуучин мессеж хүрлээ: %+v", m)
	case <-time.After(400 * time.Millisecond):
	}
}

// Давхардсан имэйл: түгжээгүй (хоёр тусдаа ClickHouse холболт = хоёр хуулбар, Locker-гүй) үед ч
// бичсэний дараах шалгалт нэгийг нь л үлдээнэ.
func TestClickHouseDuplicateEmailWithoutLock(t *testing.T) {
	ch := testClickHouse(t)
	ctx := context.Background()
	ok, conflict := 0, 0
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func(i int) {
			u := &User{Username: "u" + string(rune('a'+i)), Email: "dup@x.mn", Role: RoleStudent, DisplayName: "D"}
			done <- ch.CreateUser(ctx, u)
		}(i)
	}
	for i := 0; i < 10; i++ {
		if err := <-done; err == nil {
			ok++
		} else if err == ErrConflict {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if ok != 1 || conflict != 9 {
		t.Fatalf("яг 1 амжилт байх ёстой: ok=%d conflict=%d", ok, conflict)
	}
	n, err := ch.count(ctx, "SELECT count() FROM users FINAL WHERE deleted = false AND email = ?", "dup@x.mn")
	if err != nil || n != 1 {
		t.Fatalf("санд яг 1 идэвхтэй мөр байх ёстой: %d %v", n, err)
	}
}
