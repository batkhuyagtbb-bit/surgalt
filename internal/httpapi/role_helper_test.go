package httpapi

import (
	"context"
	"testing"
	"time"

	"surgalt/internal/auth"
	"surgalt/internal/store"
)

// testStore — сүүлд үүсгэсэн туршилтын store (newTestStore); тестүүд зэрэгцээ ажилладаггүй.
var testStore store.Store

// asRole: бүртгэл үргэлж багш үүсгэдэг тул «суралцагч» хэрэгтэй тест (хуучин бүртгэлүүд, эрхийн хязгаарлалт)
// өгөгдлийн санд төрлийг нь шууд солиод суралцагчийн токен авна. Бусад төрөлд "" буцаана.
func asRole(t *testing.T, id, username, role string) string {
	t.Helper()
	if role != string(store.RoleStudent) || testStore == nil {
		return ""
	}
	if _, err := testStore.UpdateUser(context.Background(), id, func(x *store.User) error { x.Role = store.RoleStudent; return nil }); err != nil {
		t.Fatalf("суралцагч болгох: %v", err)
	}
	tok, err := auth.NewSigner("test", time.Hour).SignUser(id, string(store.RoleStudent), username)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}
