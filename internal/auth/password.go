package auth

import (
	"context"
	"runtime"

	"golang.org/x/crypto/bcrypt"
)

// bcrypt нь CPU их иддэг. Зэрэг ажиллах тоог хязгаарлаж, нэвтрэлтийн
// оргил ачаалал бусад (уншилтын) хүсэлтийг боомилохоос сэргийлнэ.
var hashSem = make(chan struct{}, max(2, runtime.NumCPU()))

const cost = 10

// dummyHash нь байхгүй имэйлээр нэвтрэхэд ч ижил хугацаа зарцуулахад хэрэглэгдэнэ.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("surgalt-dummy-password"), cost)

func acquire(ctx context.Context) error {
	select {
	case hashSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func HashPassword(ctx context.Context, pw string) (string, error) {
	if err := acquire(ctx); err != nil {
		return "", err
	}
	defer func() { <-hashSem }()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), cost)
	return string(h), err
}

// CheckPassword; hash хоосон бол dummy-тэй харьцуулж false буцаана.
func CheckPassword(ctx context.Context, hash, pw string) (bool, error) {
	if err := acquire(ctx); err != nil {
		return false, err
	}
	defer func() { <-hashSem }()
	if hash == "" {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(pw))
		return false, nil
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil, nil
}
