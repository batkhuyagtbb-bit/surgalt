package store

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
)

// Locker — ClickHouse store-ийн "шалгаад бич" хэсгүүдийг (имэйлийн давхардал, төлбөрийн яг нэг удаа,
// pending захиалгын дедуп, явцын RMW) хамгаалах түгжээ.
//   - LocalLocker: процесс доторх мутекс — нэг app хуулбартай үед хангалттай.
//   - KeeperLocker: ClickHouse Keeper (ZooKeeper-compatible) дээрх түгжээ — олон хуулбар ижил
//     түлхүүрт нэг нэгээ хүлээнэ. Redis г.м нэмэлт сан хэрэггүй: Keeper нь ClickHouse-ийн нэг хэсэг.
type Locker interface {
	Lock(ctx context.Context, key string) (unlock func(), err error)
}

// LocalLocker — түлхүүр бүрт нэг sync.Mutex.
type LocalLocker struct{ m sync.Map }

func (l *LocalLocker) Lock(_ context.Context, key string) (func(), error) {
	v, _ := l.m.LoadOrStore(key, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock, nil
}

// KeeperLocker — Keeper дээр /<prefix>/locks/<key> зам дор ephemeral sequential node-оор түгжинэ
// (стандарт ZooKeeper lock recipe). Процесс унавал сесс дуусч, түгжээ өөрөө суларна.
type KeeperLocker struct {
	conn   *zk.Conn
	prefix string
	local  LocalLocker // нэг процесс доторх давхар хүлээлтийг хямдаар шийднэ
}

// NewKeeperLocker: addrs — "host:port,host:port"; prefix — ж. "/surgalt".
func NewKeeperLocker(ctx context.Context, addrs []string, prefix string) (*KeeperLocker, error) {
	prefix = "/" + strings.Trim(prefix, "/")
	conn, events, err := zk.Connect(addrs, 30*time.Second, zk.WithLogInfo(false), zk.WithLogger(quietLogger{}))
	if err != nil {
		return nil, fmt.Errorf("keeper connect: %w", err)
	}
	// Холбогдтол хүлээнэ (ctx-ийн хугацаанд).
	for {
		if conn.State() == zk.StateHasSession {
			break
		}
		select {
		case <-ctx.Done():
			conn.Close()
			return nil, fmt.Errorf("keeper: %w", ctx.Err())
		case <-events:
		case <-time.After(200 * time.Millisecond):
		}
	}
	for _, p := range []string{prefix, prefix + "/locks"} {
		if _, err := conn.Create(p, nil, 0, zk.WorldACL(zk.PermAll)); err != nil && err != zk.ErrNodeExists {
			conn.Close()
			return nil, fmt.Errorf("keeper create %s: %w", p, err)
		}
	}
	return &KeeperLocker{conn: conn, prefix: prefix}, nil
}

func (k *KeeperLocker) Lock(ctx context.Context, key string) (func(), error) {
	unlockLocal, _ := k.local.Lock(ctx, key)
	path := k.prefix + "/locks/" + sanitizeNode(key)
	l := zk.NewLock(k.conn, path, zk.WorldACL(zk.PermAll))
	done := make(chan error, 1)
	go func() { done <- l.Lock() }()
	select {
	case err := <-done:
		if err != nil {
			unlockLocal()
			return nil, fmt.Errorf("keeper lock %s: %w", key, err)
		}
	case <-ctx.Done():
		go func() {
			if <-done == nil {
				_ = l.Unlock()
			}
		}()
		unlockLocal()
		return nil, ctx.Err()
	}
	return func() {
		_ = l.Unlock()
		unlockLocal()
	}, nil
}

func (k *KeeperLocker) Close() { k.conn.Close() }

// sanitizeNode: Keeper-ийн зам дахь нэрэнд "/" байж болохгүй.
func sanitizeNode(key string) string {
	r := strings.NewReplacer("/", "_", " ", "_")
	return r.Replace(key)
}

// quietLogger — go-zookeeper-ийн холболт хаагдах үеийн мэдээллийг дарна.
type quietLogger struct{}

func (quietLogger) Printf(string, ...any) {}
