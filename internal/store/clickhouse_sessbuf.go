package store

import (
	"context"
	"sort"
	"sync"
	"time"
)

// ---- Идэвхтэй сессийн бичилтийн буфер ----
// Суралцагч бүр 15 секунд тутам beat илгээдэг. Урьд нь beat бүр түгжээ + SessionByID (FINAL) + бүтэн мөрийн INSERT
// (async insert-ийн 50ms хүлээлттэй) хийдэг байсан тул 3 000 зэрэг суралцагч ~800 асуулга/с үүсгэнэ. Одоо идэвхтэй
// сессүүд санах ойд байж beat зөвхөн санах ойг өөрчилнө; өөрчлөгдсөн сессүүдийг sessFlushEvery тутам НЭГ INSERT-ээр
// бичнэ. SessionByID, Sessions нь санах ойн хувилбарыг давамгайлуулна (багшийн хяналт, идэвхтэй минут шинэ хэвээр).
// Процесс гэнэт унавал сүүлийн ≤5 секундын beat алдагдана; зөв зогсооход (Close) бүгд бичигдэнэ.

const (
	sessFlushEvery = 5 * time.Second
	sessIdle       = 30 * time.Minute // энэ хугацаанд хандаагүй, бичигдсэн сессийг санах ойгоос гаргана
)

type sessionBuffer struct {
	load  func(ctx context.Context, id string) (*StudySession, error) // сангаас (ClickHouse: sessionByIDDB)
	write func(ctx context.Context, ss []StudySession) error          // нэг INSERT (ClickHouse: writeSessions)
	fmu   sync.Mutex                                                  // flush-ууд дараална: зэрэгцвэл хуучин хуулбар сүүлд бичигдэж (их ver-тэй) шинийг дарна
	mu    sync.Mutex
	m     map[string]*bufSess
}

type bufSess struct {
	s     StudySession
	dirty bool
	seen  time.Time
}

func newSessionBuffer(c *ClickHouse) *sessionBuffer {
	return &sessionBuffer{load: c.sessionByIDDB, write: c.writeSessions, m: map[string]*bufSess{}}
}

func (b *sessionBuffer) run(ctx context.Context) {
	t := time.NewTicker(sessFlushEvery)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.flush(ctx)
		}
	}
}

// get — санах ойд байгаа сессийн хуулбар.
func (b *sessionBuffer) get(id string) (*StudySession, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	e := b.m[id]
	if e == nil {
		return nil, false
	}
	e.seen = time.Now()
	s := deepCopy(e.s)
	return &s, true
}

// keep — сангаас уншсан (эсвэл шинээр бичсэн) сессийг санах ойд нэмнэ (байгаа бол давхарлахгүй).
func (b *sessionBuffer) keep(s *StudySession) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.m[s.ID]; !ok {
		b.m[s.ID] = &bufSess{s: deepCopy(*s), seen: time.Now()}
	}
}

func (b *sessionBuffer) beat(ctx context.Context, id string, bt SessionBeat) error {
	b.mu.Lock()
	e := b.m[id]
	b.mu.Unlock()
	if e == nil {
		s, err := b.load(ctx, id)
		if err != nil {
			return err
		}
		b.keep(s)
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e = b.m[id]
	s := &e.s
	s.ActiveSec += bt.ActiveSec
	s.IdleSec += bt.IdleSec
	s.AwaySec += bt.AwaySec
	s.FocusSum += bt.FocusSum
	s.FocusN += bt.FocusN
	s.Camera = s.Camera || bt.Camera
	if s.Counts == nil {
		s.Counts = map[string]int{}
	}
	for k, v := range bt.Counts {
		s.Counts[k] += v
	}
	s.LastAt = time.Now()
	if bt.End != "" {
		s.Ended, s.EndReason = true, bt.End
	}
	e.dirty, e.seen = true, time.Now()
	return nil
}

// overlay — сангаас уншсан жагсаалтад санах ойн шинэ хувилбарыг орлуулна.
func (b *sessionBuffer) overlay(list []StudySession) []StudySession {
	b.mu.Lock()
	changed := false
	for i := range list {
		if e := b.m[list[i].ID]; e != nil { // санах ойнх үргэлж хамгийн шинэ (бичигдэж байх зуур ч)
			list[i] = deepCopy(e.s)
			changed = true
		}
	}
	b.mu.Unlock()
	if changed {
		sort.SliceStable(list, func(i, j int) bool { return list[i].LastAt.After(list[j].LastAt) })
	}
	return list
}

// flush — өөрчлөгдсөн сессүүдийг нэг INSERT-ээр бичиж, удаан хандаагүйг санах ойгоос гаргана.
func (b *sessionBuffer) flush(ctx context.Context) {
	b.fmu.Lock()
	defer b.fmu.Unlock()
	b.mu.Lock()
	var batch []StudySession
	var ids []string
	for id, e := range b.m {
		if e.dirty {
			batch = append(batch, deepCopy(e.s))
			ids = append(ids, id)
			e.dirty = false
		}
	}
	b.mu.Unlock()
	if len(batch) > 0 {
		wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		err := b.write(wctx, batch)
		cancel()
		if err != nil { // дараагийн удаа дахин оролдоно
			b.mu.Lock()
			for _, id := range ids {
				if e := b.m[id]; e != nil {
					e.dirty = true
				}
			}
			b.mu.Unlock()
			return
		}
	}
	now := time.Now()
	b.mu.Lock()
	for id, e := range b.m {
		if !e.dirty && (e.s.Ended || now.Sub(e.seen) > sessIdle) {
			delete(b.m, id)
		}
	}
	b.mu.Unlock()
}

// size — тестэд: санах ойд байгаа сессийн тоо.
func (b *sessionBuffer) size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.m)
}
