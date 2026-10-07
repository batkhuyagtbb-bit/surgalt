// Package auth нь нууц үг болон HMAC гарын үсэгтэй токенуудыг хариуцна.
// Токен нь stateless тул хүсэлт бүрт өгөгдлийн сан руу хандахгүй.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidToken = errors.New("токен хүчингүй")

const RoleGuest = "guest"

// Claims: нэвтэрсэн хэрэглэгч (UID) эсвэл нэвтрээгүй зочин (GID).
type Claims struct {
	UID  string `json:"uid,omitempty"`
	GID  string `json:"gid,omitempty"`
	Role string `json:"role"`
	Name string `json:"name,omitempty"`
	Exp  int64  `json:"exp"`
}

func (c Claims) IsGuest() bool { return c.Role == RoleGuest }

// VisitorKey нь чатад зочныг ялгах түлхүүр.
func (c Claims) VisitorKey() string {
	if c.IsGuest() {
		return "g:" + c.GID
	}
	return "u:" + c.UID
}

type Signer struct {
	key []byte
	ttl time.Duration
}

func NewSigner(secret string, ttl time.Duration) *Signer {
	return &Signer{key: []byte(secret), ttl: ttl}
}

var b64 = base64.RawURLEncoding

func (s *Signer) sign(c Claims) (string, error) {
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	p := b64.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(p))
	return p + "." + b64.EncodeToString(mac.Sum(nil)), nil
}

// MAC нь богино, хугацаагүй гарын үсэг (төлбөрийн холбоос г.м); purpose-оор зориулалтыг тусгаарлана.
func (s *Signer) MAC(purpose, msg string) string {
	mac := hmac.New(sha256.New, append([]byte(purpose+":"), s.key...))
	mac.Write([]byte(msg))
	return b64.EncodeToString(mac.Sum(nil)[:16])
}

func (s *Signer) SignUser(uid, role, name string) (string, error) {
	return s.sign(Claims{UID: uid, Role: role, Name: name, Exp: time.Now().Add(s.ttl).Unix()})
}

// SignGuest нь нэвтрээгүй зочинд 30 хоногийн чатын токен өгнө.
func (s *Signer) SignGuest(name string) (string, Claims, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", Claims{}, err
	}
	c := Claims{GID: hex.EncodeToString(b[:]), Role: RoleGuest, Name: name, Exp: time.Now().Add(30 * 24 * time.Hour).Unix()}
	t, err := s.sign(c)
	return t, c, err
}

func (s *Signer) Verify(tok string) (Claims, error) {
	var c Claims
	p, sig, ok := strings.Cut(tok, ".")
	if !ok {
		return c, ErrInvalidToken
	}
	got, err := b64.DecodeString(sig)
	if err != nil {
		return c, ErrInvalidToken
	}
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(p))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return c, ErrInvalidToken
	}
	raw, err := b64.DecodeString(p)
	if err != nil || json.Unmarshal(raw, &c) != nil {
		return c, ErrInvalidToken
	}
	if time.Now().Unix() > c.Exp {
		return c, ErrInvalidToken
	}
	return c, nil
}

// Seal нь дурын утгыг хугацаатайгаар гарын үсэг зурж мөр болгоно (OAuth state cookie г.м).
func (s *Signer) Seal(v any, ttl time.Duration) (string, error) {
	raw, err := json.Marshal(struct {
		V   any   `json:"v"`
		Exp int64 `json:"exp"`
	}{v, time.Now().Add(ttl).Unix()})
	if err != nil {
		return "", err
	}
	p := b64.EncodeToString(raw)
	mac := hmac.New(sha256.New, append([]byte("seal:"), s.key...))
	mac.Write([]byte(p))
	return p + "." + b64.EncodeToString(mac.Sum(nil)), nil
}

func (s *Signer) Open(tok string, dst any) error {
	p, sig, ok := strings.Cut(tok, ".")
	if !ok {
		return ErrInvalidToken
	}
	got, err := b64.DecodeString(sig)
	if err != nil {
		return ErrInvalidToken
	}
	mac := hmac.New(sha256.New, append([]byte("seal:"), s.key...))
	mac.Write([]byte(p))
	if !hmac.Equal(got, mac.Sum(nil)) {
		return ErrInvalidToken
	}
	raw, err := b64.DecodeString(p)
	if err != nil {
		return ErrInvalidToken
	}
	var w struct {
		V   json.RawMessage `json:"v"`
		Exp int64           `json:"exp"`
	}
	if json.Unmarshal(raw, &w) != nil || time.Now().Unix() > w.Exp || json.Unmarshal(w.V, dst) != nil {
		return ErrInvalidToken
	}
	return nil
}

// Encrypt нь нууц өгөгдлийг (жишээ нь Google refresh token) AES-256-GCM-ээр шифрлэнэ.
func (s *Signer) Encrypt(plain string) (string, error) {
	gcm, err := s.gcm()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return b64.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil)), nil
}

func (s *Signer) Decrypt(enc string) (string, error) {
	gcm, err := s.gcm()
	if err != nil {
		return "", err
	}
	raw, err := b64.DecodeString(enc)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", ErrInvalidToken
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", ErrInvalidToken
	}
	return string(plain), nil
}

func (s *Signer) gcm() (cipher.AEAD, error) {
	key := sha256.Sum256(append([]byte("enc:"), s.key...))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
