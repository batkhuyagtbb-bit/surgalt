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
	// gen — drop бүрт өснө; drops — сүүлийн drop-уудын нөхцөл. Сангаас ачаалж эхэлсний дараа (DELETE-ээс өмнө уншсан)
	// drop болж энэ сессийг хамарсан бол буферт оруулахгүй (keepFrom) — эс бөгөөс устгасан сесс дахин бичигдэнэ.
	gen   uint64
	drops []bufDrop
}

type bufDrop struct {
	gen  uint64
	pred func(*StudySession) bool
}

// bufDropLog — хадгалах drop-ын нөхцөлийн тоо (ачаалал хэдхэн мс; устгал зэрэг 1-ээс олон ажиллахгүй).
const bufDropLog = 32

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

// generation — сангаас ачаалахаас өмнө авна (keepFrom-д дамжуулна).
func (b *sessionBuffer) generation() uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.gen
}

// keepFrom — сангаас уншсан сессийг буферт нэмнэ, гэхдээ gen-ээс хойш энэ сессийг хамарсан drop болсон бол үгүй (false):
// уншилт DELETE-ээс өмнө байж болзошгүй тул дуудагч ErrNotFound буцаана.
func (b *sessionBuffer) keepFrom(s *StudySession, gen uint64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.gen != gen {
		if len(b.drops) == 0 || b.drops[0].gen > gen+1 { // бүртгэлээс хэтэрсэн — болгоомжтой
			return false
		}
		for _, d := range b.drops {
			if d.gen > gen && d.pred(s) {
				return false
			}
		}
	}
	if _, ok := b.m[s.ID]; !ok {
		b.m[s.ID] = &bufSess{s: deepCopy(*s), seen: time.Now()}
	}
	return true
}

func (b *sessionBuffer) beat(ctx context.Context, id string, bt SessionBeat) error {
	b.mu.Lock()
	e, gen := b.m[id], b.gen
	b.mu.Unlock()
	if e == nil {
		s, err := b.load(ctx, id)
		if err != nil {
			return err
		}
		if !b.keepFrom(s, gen) {
			return ErrNotFound
		}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	e = b.m[id]
	if e == nil { // хооронд нь drop (суралцагчийн өгөгдөл устгасан) эсвэл flush санах ойгоос гаргасан
		return ErrNotFound
	}
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

// hold — flush-ийг (fmu) зогсооно: буцаасан функцийг дуустал санах ойн сессүүд сан руу бичигдэхгүй. Суралцагчийн
// өгөгдөл устгах (course_reset.go) үед: flush-ийн авсан хуучин хуулбар DELETE-ийн дараа дахин бичигдэхээс сэргийлнэ.
func (b *sessionBuffer) hold() (release func()) {
	b.fmu.Lock()
	var once sync.Once
	return func() { once.Do(b.fmu.Unlock) }
}

// drop — pred-д тохирох сессүүдийг (бичигдээгүй beat-ийн хамт) санах ойгоос хаяна; хаясан тоог буцаана.
// Дараагийн beat сангаас ачаална — мөр нь устсан бол ErrNotFound (клиент шинэ сесс эхлүүлнэ).
func (b *sessionBuffer) drop(pred func(*StudySession) bool) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.gen++
	b.drops = append(b.drops, bufDrop{b.gen, pred})
	if len(b.drops) > bufDropLog {
		b.drops = append([]bufDrop(nil), b.drops[len(b.drops)-bufDropLog:]...)
	}
	n := 0
	for id, e := range b.m {
		if pred(&e.s) {
			delete(b.m, id)
			n++
		}
	}
	return n
}

// size — тестэд: санах ойд байгаа сессийн тоо.
func (b *sessionBuffer) size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.m)
}
