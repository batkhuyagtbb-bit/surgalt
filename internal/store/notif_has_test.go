package store

import (
	"context"
	"testing"
	"time"
)

// HasNotification: хэрэглэгчдийн аль нэгэнд, яг ижил төрөл ба холбоостой, since-ээс хойших мэдэгдэл.
func TestHasNotificationMemory(t *testing.T) {
	ctx := context.Background()
	m := NewMemory()
	_ = m.AddNotifications(ctx, []*Notification{
		{UserID: "t1", Type: "access_request", Link: "grant:u1:c1:"},
		{UserID: "t2", Type: "access_request", Link: "grant:u2:c1:l1"},
		{UserID: "t1", Type: "enroll", Link: "grant:u3:c1:"},
	})
	hour := time.Hour
	for name, tc := range map[string]struct {
		users []string
		typ   string
		link  string
		since time.Duration
		want  bool
	}{
		"эзэмшигчид":        {[]string{"t1", "t2"}, "access_request", "grant:u1:c1:", -hour, true},
		"хамтран заагчид":   {[]string{"t1", "t2"}, "access_request", "grant:u2:c1:l1", -hour, true},
		"өөр багшид":        {[]string{"t3"}, "access_request", "grant:u1:c1:", -hour, false},
		"холбоос яг таарна": {[]string{"t1"}, "access_request", "grant:u1:c1:l1", -hour, false},
		"өөр төрөл":         {[]string{"t1"}, "access_request", "grant:u3:c1:", -hour, false},
		"хугацаа өнгөрсөн":  {[]string{"t1"}, "access_request", "grant:u1:c1:", hour, false},
		"хэрэглэгчгүй":      {nil, "access_request", "grant:u1:c1:", -hour, false},
	} {
		got, err := m.HasNotification(ctx, tc.users, tc.typ, tc.link, time.Now().Add(tc.since))
		if err != nil || got != tc.want {
			t.Fatalf("%s: %v %v", name, got, err)
		}
	}
	// idSince: тэр мөчөөс хойш үүссэн ID бүр түүнээс багагүй (ClickHouse-ийн id мужийн шүүлт).
	since := time.Now().Add(-time.Second)
	if id := NewID(); id < idSince(since) || idSince(time.Now().Add(time.Hour)) <= id {
		t.Fatalf("idSince: %s %s", id, idSince(since))
	}
}
