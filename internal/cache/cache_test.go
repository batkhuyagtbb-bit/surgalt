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
