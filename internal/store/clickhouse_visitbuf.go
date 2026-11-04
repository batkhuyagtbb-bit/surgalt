package store

import (
	"context"
	"sync"
	"time"
)

// ---- Хуудас үзэлтийн хугацааны буфер ----
// Нээлттэй хуудас бүр 60 секунд тутам (мөн таб нуухад) «хэр удаан байсан»-аа илгээдэг (POST /api/views/{id}/end).
// Урьд нь бүрт нь VisitByID (sort key биш id-гаар FINAL) + INSERT (50ms хүлээлт) хийдэг байв. Одоо шинэ үзэлт шууд
// бичигдэж санах ойд үлдэнэ; хугацааны шинэчлэл санах ойд хийгдээд 5 секунд тутам НЭГ INSERT-ээр бичигдэнэ.

const visitIdle = 2 * time.Hour

type visitBuffer struct {
	load  func(ctx context.Context, id string) (*PageVisit, error)
	write func(ctx context.Context, vs []PageVisit) error
	fmu   sync.Mutex // flush-ууд дараална: зэрэгцвэл хуучин хуулбар сүүлд бичигдэж (их ver-тэй) шинийг дарна
	mu    sync.Mutex
	m     map[string]*bufVisit
}

type bufVisit struct {
	v     PageVisit
	dirty bool
	seen  time.Time
}

func newVisitBuffer(c *ClickHouse) *visitBuffer {
	return &visitBuffer{load: c.visitByIDDB, write: c.writeVisits, m: map[string]*bufVisit{}}
}

func (b *visitBuffer) run(ctx context.Context) {
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

func (b *visitBuffer) get(ctx context.Context, id string) (*PageVisit, error) {
	b.mu.Lock()
	if e := b.m[id]; e != nil {
		e.seen = time.Now()
		v := e.v
		b.mu.Unlock()
		return &v, nil
	}
	b.mu.Unlock()
	v, err := b.load(ctx, id)
	if err != nil {
		return nil, err
	}
	b.keep(*v, false)
	return v, nil
}

// keep — санах ойд хадгална; dirty бол дараагийн flush-д бичигдэнэ.
func (b *visitBuffer) keep(v PageVisit, dirty bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if e := b.m[v.ID]; e != nil {
		e.v, e.seen = v, time.Now()
		e.dirty = e.dirty || dirty
		return
	}
	b.m[v.ID] = &bufVisit{v: v, dirty: dirty, seen: time.Now()}
}

func (b *visitBuffer) overlay(list []PageVisit) []PageVisit {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range list {
		if e := b.m[list[i].ID]; e != nil { // санах ойнх үргэлж хамгийн шинэ (бичигдэж байх зуур ч)
			list[i] = e.v
		}
	}
	return list
}

func (b *visitBuffer) flush(ctx context.Context) {
	b.fmu.Lock()
	defer b.fmu.Unlock()
	b.mu.Lock()
	var batch []PageVisit
	for _, e := range b.m {
		if e.dirty {
			batch = append(batch, e.v)
			e.dirty = false
		}
	}
	b.mu.Unlock()
	if len(batch) > 0 {
		wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		err := b.write(wctx, batch)
		cancel()
		if err != nil {
			b.mu.Lock()
			for _, v := range batch {
				if e := b.m[v.ID]; e != nil {
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
		if !e.dirty && now.Sub(e.seen) > visitIdle {
			delete(b.m, id)
		}
	}
	b.mu.Unlock()
}
