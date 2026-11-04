// Package cache нь процесс доторх, shard-лагдсан TTL кэш.
// singleflight-ээр нэг түлхүүр дээр зэрэг ирсэн олон мянган хүсэлтийг
// өгөгдлийн сан руу ганц асуулга болгож нэгтгэнэ (cache stampede хамгаалалт).
package cache

import (
	"context"
	"hash/maphash"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const shardCount = 64

type entry[V any] struct {
	val V
	err error
	exp time.Time
}

type shard[V any] struct {
	mu sync.RWMutex
	m  map[string]entry[V]
	// gen нь Delete/Clear бүрт өснө: ачаалал эхэлснээс хойш устгал болсон бол (өгөгдөл өөрчлөгдсөн байж болзошгүй)
	// хуучирсан үр дүнг кэшэд буцааж бичихгүй.
	gen uint64
}

type Cache[V any] struct {
	shards [shardCount]shard[V]
	seed   maphash.Seed
	ttl    time.Duration
	negTTL time.Duration
	sf     singleflight.Group
	// IsNegative нь аль алдааг богино хугацаанд кэшлэхийг шийднэ (жишээ нь ErrNotFound).
	isNegative func(error) bool
}

func New[V any](ttl, negTTL time.Duration, isNegative func(error) bool) *Cache[V] {
	c := &Cache[V]{seed: maphash.MakeSeed(), ttl: ttl, negTTL: negTTL, isNegative: isNegative}
	for i := range c.shards {
		c.shards[i].m = make(map[string]entry[V])
	}
	return c
}

func (c *Cache[V]) shard(key string) *shard[V] {
	return &c.shards[maphash.String(c.seed, key)%shardCount]
}

func (c *Cache[V]) get(key string) (entry[V], bool) {
	s := c.shard(key)
	s.mu.RLock()
	e, ok := s.m[key]
	s.mu.RUnlock()
	if !ok || time.Now().After(e.exp) {
		return e, false
	}
	return e, true
}

func (c *Cache[V]) gen(key string) uint64 {
	s := c.shard(key)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gen
}

// set нь ачаалал эхлэх үеийн gen хэвээр бол л бичнэ.
func (c *Cache[V]) set(key string, e entry[V], gen uint64) {
	s := c.shard(key)
	s.mu.Lock()
	if s.gen == gen {
		s.m[key] = e
	}
	s.mu.Unlock()
}

func (c *Cache[V]) Delete(key string) {
	s := c.shard(key)
	s.mu.Lock()
	delete(s.m, key)
	s.gen++
	s.mu.Unlock()
	c.sf.Forget(key)
}

// Clear нь бүх бичлэгийг устгана (ховор, олон түлхүүрт нөлөөлөх өөрчлөлтөд).
func (c *Cache[V]) Clear() {
	for i := range c.shards {
		s := &c.shards[i]
		s.mu.Lock()
		clear(s.m)
		s.gen++
		s.mu.Unlock()
	}
}

// Len — одоо хадгалагдаж буй бичлэгийн тоо (хугацаа нь дууссан ч Janitor цэвэрлээгүй бол орно).
func (c *Cache[V]) Len() int {
	n := 0
	for i := range c.shards {
		s := &c.shards[i]
		s.mu.RLock()
		n += len(s.m)
		s.mu.RUnlock()
	}
	return n
}

// Set — утгыг шууд хадгална (ачааллын үр дүнгээс бусад: ж. «нээгдсэн» гэдгийг тэмдэглэх).
func (c *Cache[V]) Set(key string, v V) {
	c.set(key, entry[V]{val: v, exp: time.Now().Add(c.ttl)}, c.gen(key))
}

// Peek — кэшэд байгаа (хугацаа нь дуусаагүй) утга, ачаалахгүй.
func (c *Cache[V]) Peek(key string) (V, bool) {
	e, ok := c.get(key)
	if !ok || e.err != nil {
		var zero V
		return zero, false
	}
	return e.val, true
}

// GetOrLoad нь кэшээс буцаах эсвэл load-ийг ганцхан удаа дуудна.
func (c *Cache[V]) GetOrLoad(ctx context.Context, key string, load func(context.Context) (V, error)) (V, error) {
	if e, ok := c.get(key); ok {
		return e.val, e.err
	}
	ch := c.sf.DoChan(key, func() (any, error) {
		// Өмнөх flight саяхан дүүрсэн бол дахин ачаалахгүй (double-checked).
		if e, ok := c.get(key); ok {
			return e.val, e.err
		}
		gen := c.gen(key)
		// Хүсэлт цуцлагдсан ч бусад хүлээгчид үр дүнг авах ёстой.
		lctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		v, err := load(lctx)
		switch {
		case err == nil:
			c.set(key, entry[V]{val: v, exp: time.Now().Add(c.ttl)}, gen)
		case c.isNegative != nil && c.isNegative(err):
			c.set(key, entry[V]{err: err, exp: time.Now().Add(c.negTTL)}, gen)
		}
		return v, err
	})
	select {
	case r := <-ch:
		v, _ := r.Val.(V)
		return v, r.Err
	case <-ctx.Done():
		var zero V
		return zero, ctx.Err()
	}
}

// Janitor нь хугацаа нь дууссан бичлэгүүдийг үе үе цэвэрлэнэ.
func (c *Cache[V]) Janitor(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			for i := range c.shards {
				s := &c.shards[i]
				s.mu.Lock()
				for k, e := range s.m {
					if now.After(e.exp) {
						delete(s.m, k)
					}
				}
				s.mu.Unlock()
			}
		}
	}
}
