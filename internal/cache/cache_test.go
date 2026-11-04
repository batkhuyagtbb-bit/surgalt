package cache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var errNF = errors.New("nf")

// 10000 зэрэг хүсэлт нэг түлхүүр дээр ирэхэд load зөвхөн нэг удаа дуудагдах ёстой.
func TestStampedeSingleLoad(t *testing.T) {
	c := New[int](time.Minute, time.Second, nil)
	var calls atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := c.GetOrLoad(context.Background(), "k", func(context.Context) (int, error) {
				calls.Add(1)
				time.Sleep(20 * time.Millisecond)
				return 42, nil
			})
			if err != nil || v != 42 {
				t.Errorf("got %d %v", v, err)
			}
		}()
	}
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatalf("load %d удаа дуудагдлаа, 1 байх ёстой", calls.Load())
	}
}

func TestNegativeCaching(t *testing.T) {
	c := New[int](time.Minute, time.Minute, func(err error) bool { return errors.Is(err, errNF) })
	var calls int
	load := func(context.Context) (int, error) { calls++; return 0, errNF }
	for i := 0; i < 5; i++ {
		if _, err := c.GetOrLoad(context.Background(), "x", load); !errors.Is(err, errNF) {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("олдоогүйг кэшлээгүй: %d", calls)
	}
	c.Delete("x")
	_, _ = c.GetOrLoad(context.Background(), "x", load)
	if calls != 2 {
		t.Fatal("Delete ажиллаагүй")
	}
}

// Ачаалал явж байх хооронд Delete дуудагдвал (өгөгдөл өөрчлөгдсөн) хуучин үр дүн кэшэд үлдэх ёсгүй.
func TestDeleteDuringLoadDropsStale(t *testing.T) {
	c := New[int](time.Minute, time.Second, nil)
	started, release := make(chan struct{}), make(chan struct{})
	go func() {
		_, _ = c.GetOrLoad(context.Background(), "k", func(context.Context) (int, error) {
			close(started)
			<-release
			return 1, nil // хуучин утга
		})
	}()
	<-started
	c.Delete("k") // энэ хооронд бичилт болж кэш хүчингүй болсон
	close(release)
	time.Sleep(20 * time.Millisecond)
	v, _ := c.GetOrLoad(context.Background(), "k", func(context.Context) (int, error) { return 2, nil })
	if v != 2 {
		t.Fatalf("хуучирсан утга кэшэд үлдлээ: %d", v)
	}
	c.Clear()
	if c.Len() != 0 {
		t.Fatal("Clear ажиллаагүй")
	}
}
