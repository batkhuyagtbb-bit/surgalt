package store

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// NotificationProof — тодорхой мэдэгдэл (төрөл + холбоос) эдгээр хэрэглэгчийн аль нэгэнд since-ээс хойш очсон эсэх:
// үйлдлийг зөвхөн бодит хүсэлтийн хариуд зөвшөөрөх баталгаа (ж: нээлгэх хүсэлт → багш «Нээх», access_request.go).
type NotificationProof interface {
	HasNotification(ctx context.Context, userIDs []string, typ, link string, since time.Time) (bool, error)
}

// idSince — NewID нь эхний 4 байтдаа unix секундтэй (hex) тул энэ мөчөөс хойших ID бүр үүнээс их буюу тэнцүү.
func idSince(t time.Time) string {
	var b [12]byte
	binary.BigEndian.PutUint32(b[0:4], uint32(t.Unix()))
	return hex.EncodeToString(b[:])
}

// HasNotification — notifications нь ORDER BY (user_id, id): user_id IN (…) ба id-ийн мужаар (ID цагаар өсдөг)
// анхдагч түлхүүрээр л уншина.
func (c *ClickHouse) HasNotification(ctx context.Context, userIDs []string, typ, link string, since time.Time) (bool, error) {
	if len(userIDs) == 0 {
		return false, nil
	}
	args := make([]any, 0, len(userIDs)+4)
	for _, u := range userIDs {
		args = append(args, u)
	}
	args = append(args, idSince(since), typ, link, since.UTC())
	q := "SELECT 1 FROM notifications WHERE user_id IN (" + strings.TrimSuffix(strings.Repeat("?,", len(userIDs)), ",") +
		") AND id >= ? AND type = ? AND link = ? AND created_at >= ? LIMIT 1"
	found := false
	err := c.query(ctx, q, args, func(r driver.Rows) error {
		var one uint8
		found = true
		return r.Scan(&one)
	})
	return found && err == nil, err
}

func (m *Memory) HasNotification(_ context.Context, userIDs []string, typ, link string, since time.Time) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, u := range userIDs {
		for _, n := range m.notifs[u] {
			if n.Type == typ && n.Link == link && !n.CreatedAt.Before(since) {
				return true, nil
			}
		}
	}
	return false, nil
}
