package auth

import (
	"strings"
	"testing"
	"time"
)

func TestTokenRoundTripAndTamper(t *testing.T) {
	s := NewSigner("secret-secret-secret-secret-1234", time.Hour)
	tok, err := s.SignUser("u1", "teacher", "demo")
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.Verify(tok)
	if err != nil || c.UID != "u1" || c.Role != "teacher" {
		t.Fatalf("%+v %v", c, err)
	}
	if _, err := s.Verify(tok[:len(tok)-2] + "xx"); err == nil {
		t.Fatal("хуурамч гарын үсэг хүлээн авагдлаа")
	}
	if _, err := NewSigner("other", time.Hour).Verify(tok); err == nil {
		t.Fatal("өөр түлхүүрээр баталгаажлаа")
	}
	expired, _ := NewSigner("secret-secret-secret-secret-1234", -time.Minute).SignUser("u1", "teacher", "demo")
	if _, err := s.Verify(expired); err == nil {
		t.Fatal("хугацаа дууссан токен хүлээн авагдлаа")
	}
	g, gc, _ := s.SignGuest("Болд")
	if c, err := s.Verify(g); err != nil || !c.IsGuest() || c.VisitorKey() != "g:"+gc.GID {
		t.Fatal("зочны токен")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	s := NewSigner("k", time.Hour)
	enc, err := s.Encrypt("refresh-token")
	if err != nil || strings.Contains(enc, "refresh") {
		t.Fatal("шифрлэгдээгүй")
	}
	if p, err := s.Decrypt(enc); err != nil || p != "refresh-token" {
		t.Fatal(p, err)
	}
	if _, err := NewSigner("other", time.Hour).Decrypt(enc); err == nil {
		t.Fatal("өөр түлхүүрээр тайлагдлаа")
	}
}

func TestSealOpen(t *testing.T) {
	s := NewSigner("k", time.Hour)
	v, _ := s.Seal(map[string]string{"s": "abc"}, time.Minute)
	var out map[string]string
	if err := s.Open(v, &out); err != nil || out["s"] != "abc" {
		t.Fatal(err)
	}
	old, _ := s.Seal(1, -time.Second)
	if s.Open(old, &out) == nil {
		t.Fatal("хугацаа дууссан state хүлээн авагдлаа")
	}
}
