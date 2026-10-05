// Package ratelimit нь IP тус бүрийн token-bucket хязгаарлагч (shard-лагдсан, lock contention багатай).
package ratelimit

import (
	"context"
	"hash/maphash"
	"sync"
	"time"
)

const shards = 64

type bucket struct {
	tokens float64
	last   time.Time
}

type shard struct {
	mu sync.Mutex
	m  map[string]*bucket
}

type Limiter struct {
	rate, burst float64
	seed        maphash.Seed
	shards      [shards]shard
}

// New: rate<=0 бол хязгааргүй (Allow үргэлж true).
func New(ratePerSec, burst float64) *Limiter {
	l := &Limiter{rate: ratePerSec, burst: burst, seed: maphash.MakeSeed()}
	for i := range l.shards {
		l.shards[i].m = make(map[string]*bucket)
	}
	return l
}

func (l *Limiter) Allow(key string) bool {
	if l == nil || l.rate <= 0 {
		return true
	}
	s := &l.shards[maphash.String(l.seed, key)%shards]
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.m[key]
	if !ok {
		s.m[key] = &bucket{tokens: l.burst - 1, last: now}
		return true
	}
	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Janitor нь удаан идэвхгүй байсан bucket-уудыг устгаж санах ойг хэмнэнэ.
func (l *Limiter) Janitor(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			for i := range l.shards {
				s := &l.shards[i]
				s.mu.Lock()
				for k, b := range s.m {
					if now.Sub(b.last) > 3*time.Minute {
						delete(s.m, k)
					}
				}
				s.mu.Unlock()
			}
		}
	}
}
