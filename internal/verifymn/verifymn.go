// Package verifymn — Verify.MN (https://verify.mn): утасны дугаарыг хэрэглэгчийн өөрийн утаснаас илгээсэн SMS-ээр
// баталгаажуулна (Mobile-Originated). Хөгжүүлэгч код үүсгэж сесс нээнэ → хэрэглэгч 144773 руу тэр кодыг SMS-ээр
// илгээнэ → Verify.MN тухайн дугаараас ирснийг шалгаж сессийг VERIFIED болгоно. SMS-ийн төлбөрийг илгээгч төлнө.
package verifymn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultURL = "https://api.verify.mn"

// Сессийн төлөв.
const (
	Pending  = "PENDING"
	Verified = "VERIFIED"
	Expired  = "EXPIRED"
)

type Client struct {
	Base string
	Key  string
	HC   *http.Client
}

func New(base, key string) *Client {
	if base == "" {
		base = DefaultURL
	}
	return &Client{Base: strings.TrimRight(base, "/"), Key: key, HC: &http.Client{Timeout: 15 * time.Second}}
}

// Session — POST /sessions-ийн хариу.
type Session struct {
	SessionID   string    `json:"sessionId"`
	Phone       string    `json:"phone"`
	Shortcode   string    `json:"shortcode"`
	Text        string    `json:"text"`
	SMSURI      string    `json:"smsUri"`
	Instruction string    `json:"displayInstruction"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

// Status — GET /sessions/:id-ийн хариу (баталгаажсан эсэхийг зөвхөн үүгээр шийднэ).
type Status struct {
	SessionID      string     `json:"sessionId"`
	SessionStatus  string     `json:"sessionStatus"`
	CallbackStatus string     `json:"callbackStatus"`
	VerifiedAt     *time.Time `json:"verifiedAt"`
	ExpiresAt      time.Time  `json:"expiresAt"`
}

// APIError — Verify.MN-ийн алдааны хариу.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("verify.mn %d: %s", e.Status, e.Message) }

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.Base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Key)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.HC.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode < 200 || res.StatusCode > 299 {
		var e struct {
			Message string `json:"message"`
			Error   any    `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		msg := e.Message
		if msg == "" {
			if s, ok := e.Error.(string); ok {
				msg = s
			} else {
				msg = strings.TrimSpace(string(raw))
				if len(msg) > 200 {
					msg = msg[:200]
				}
			}
		}
		return &APIError{Status: res.StatusCode, Message: msg}
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("verify.mn: хариу уншиж чадсангүй: %w", err)
		}
	}
	return nil
}

// Create — phone (8 оронтой) дугаараас text-ийг 144773 руу илгээхийг хүлээх сесс (5 минут). callback, reply хоосон
// байж болно; reply — баталгаажсаны дараа хэрэглэгчид очих хариу SMS (латин үсэг, 160 тэмдэгт хүртэл; Lime-д очдоггүй).
func (c *Client) Create(ctx context.Context, phone, text, callback, reply string) (*Session, error) {
	body := map[string]string{"phone": phone, "text": text}
	if callback != "" {
		body["callback"] = callback
	}
	if reply != "" {
		body["responseSms"] = reply
	}
	var s Session
	if err := c.do(ctx, http.MethodPost, "/sessions", body, &s); err != nil {
		return nil, err
	}
	if s.SessionID == "" {
		return nil, fmt.Errorf("verify.mn: sessionId ирсэнгүй")
	}
	if s.Shortcode == "" {
		s.Shortcode = "144773"
	}
	if s.Text == "" {
		s.Text = text
	}
	if s.SMSURI == "" {
		s.SMSURI = "sms:" + s.Shortcode + "?body=" + url.QueryEscape(s.Text)
	}
	return &s, nil
}

// Status — сессийн жинхэнэ төлөв.
func (c *Client) Status(ctx context.Context, id string) (*Status, error) {
	var s Status
	if err := c.do(ctx, http.MethodGet, "/sessions/"+url.PathEscape(id), nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
