package store

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// lockerExclusion: N goroutine нэг түлхүүрт зэрэг орохыг оролдоход яг нэг нэгээрээ орж байгаа эсэх.
func lockerExclusion(t *testing.T, l Locker) {
	t.Helper()
	var wg sync.WaitGroup
	var inside, maxInside, total int
	var mu sync.Mutex
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			unlock, err := l.Lock(ctx, "users")
			if err != nil {
				t.Errorf("lock: %v", err)
				return
			}
			mu.Lock()
			inside++
			if inside > maxInside {
				maxInside = inside
			}
			total++
			mu.Unlock()
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			inside--
			mu.Unlock()
			unlock()
		}()
	}
	wg.Wait()
	if maxInside != 1 || total != 20 {
		t.Fatalf("түгжээ алдагдсан: зэрэг %d, нийт %d", maxInside, total)
	}
}

func TestLocalLocker(t *testing.T) { lockerExclusion(t, &LocalLocker{}) }

// SURGALT_TEST_KEEPER=127.0.0.1:9181 өгвөл жинхэнэ ClickHouse Keeper дээр: хоёр тусдаа холболт
// (= хоёр app хуулбар) ижил түлхүүрт нэг нэгээ хүлээж байгааг шалгана.
func TestKeeperLocker(t *testing.T) {
	addr := os.Getenv("SURGALT_TEST_KEEPER")
	if addr == "" {
		t.Skip("SURGALT_TEST_KEEPER тохируулаагүй")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	a, err := NewKeeperLocker(ctx, strings.Split(addr, ","), "/surgalt_test")
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewKeeperLocker(ctx, strings.Split(addr, ","), "/surgalt_test")
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	// Хоёр хуулбар: хагас нь a, хагас нь b-ээр түгжинэ — ялгаа байх ёсгүй.
	var wg sync.WaitGroup
	var inside, maxInside int
	var mu sync.Mutex
	for i := 0; i < 12; i++ {
		l := Locker(a)
		if i%2 == 1 {
			l = b
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			unlock, err := l.Lock(ctx, "users")
			if err != nil {
				t.Errorf("keeper lock: %v", err)
				return
			}
			mu.Lock()
			inside++
			if inside > maxInside {
				maxInside = inside
			}
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			inside--
			mu.Unlock()
			unlock()
		}()
	}
	wg.Wait()
	if maxInside != 1 {
		t.Fatalf("Keeper түгжээ алдагдсан: зэрэг %d", maxInside)
	}
	// Хугацаа дууссан ctx: түгжээ бусдад байхад алдаа буцаана, гацахгүй.
	unlock, err := a.Lock(ctx, "busy")
	if err != nil {
		t.Fatal(err)
	}
	sctx, scancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer scancel()
	if _, err := b.Lock(sctx, "busy"); err == nil {
		t.Fatal("эзлэгдсэн түгжээ ctx дуусахад алдаа өгөх ёстой")
	}
	unlock()
}
