// Package qpay нь QPay merchant API v2-ийн жижиг клиент: нэхэмжлэх (QR) үүсгэх, төлбөр шалгах.
// Баримт бичиг: https://developer.qpay.mn — sandbox: https://merchant-sandbox.qpay.mn/v2
package qpay

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const DefaultURL = "https://merchant.qpay.mn/v2"

type Client struct {
	BaseURL     string // жишээ нь https://merchant.qpay.mn/v2
	Username    string
	Password    string
	InvoiceCode string
	HTTP        *http.Client

	mu     sync.Mutex
	token  string
	expiry time.Time
}

func New(baseURL, username, password, invoiceCode string) *Client {
	if baseURL == "" {
		baseURL = DefaultURL
	}
	return &Client{BaseURL: strings.TrimRight(baseURL, "/"), Username: username, Password: password, InvoiceCode: invoiceCode,
		HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// BankURL — банкны апп-ыг шууд нээх холбоос (утсан дээр QR уншуулах шаардлагагүй).
type BankURL struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Link        string `json:"link"`
}

type Invoice struct {
	ID       string    `json:"invoice_id"`
	QRText   string    `json:"qr_text"`
	ShortURL string    `json:"qPay_shortUrl"`
	URLs     []BankURL `json:"urls"`
}

type InvoiceRequest struct {
	SenderNo    string // манай захиалгын дугаар
	Description string
	Amount      int64
	CallbackURL string
}

var ErrAuth = errors.New("qpay: нэвтрэх эрх буруу")

// CreateInvoice нь нэхэмжлэх үүсгэж QR-ын текст ба банкны апп-уудын холбоосыг буцаана.
func (c *Client) CreateInvoice(ctx context.Context, in InvoiceRequest) (*Invoice, error) {
	body := map[string]any{
		"invoice_code":          c.InvoiceCode,
		"sender_invoice_no":     in.SenderNo,
		"invoice_receiver_code": "terminal",
		"invoice_description":   truncate(in.Description, 250),
		"amount":                in.Amount,
		"callback_url":          in.CallbackURL,
	}
	var out Invoice
	if err := c.call(ctx, "/invoice", body, &out); err != nil {
		return nil, err
	}
	if out.ID == "" || out.QRText == "" {
		return nil, errors.New("qpay: нэхэмжлэхийн хариу хоосон")
	}
	return &out, nil
}

// PaidAmount нь нэхэмжлэхэд төлөгдсөн нийт дүнг буцаана (төлөгдөөгүй бол 0).
func (c *Client) PaidAmount(ctx context.Context, invoiceID string) (int64, error) {
	var out struct {
		Count      int         `json:"count"`
		PaidAmount json.Number `json:"paid_amount"`
		Rows       []struct {
			Status string          `json:"payment_status"`
			Amount json.RawMessage `json:"payment_amount"` // мөр эсвэл тоо
		} `json:"rows"`
	}
	body := map[string]any{"object_type": "INVOICE", "object_id": invoiceID, "offset": map[string]int{"page_number": 1, "page_limit": 100}}
	if err := c.call(ctx, "/payment/check", body, &out); err != nil {
		return 0, err
	}
	var sum int64
	for _, r := range out.Rows {
		if strings.EqualFold(r.Status, "PAID") {
			sum += amount(r.Amount)
		}
	}
	if sum == 0 && len(out.Rows) == 0 { // зарим хариунд зөвхөн нийт дүн ирдэг
		if f, err := out.PaidAmount.Float64(); err == nil && f > 0 {
			sum = int64(f + 0.5)
		}
	}
	return sum, nil
}

// call нь токентой POST хийнэ; токен хүчингүй болсон бол нэг удаа дахин нэвтэрч оролдоно.
func (c *Client) call(ctx context.Context, path string, body, out any) error {
	for attempt := 0; ; attempt++ {
		tok, err := c.accessToken(ctx, attempt > 0)
		if err != nil {
			return err
		}
		raw, _ := json.Marshal(body)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(raw))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+tok)
		res, err := c.HTTP.Do(req)
		if err != nil {
			return fmt.Errorf("qpay %s: %w", path, err)
		}
		data, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
		res.Body.Close()
		if res.StatusCode == http.StatusUnauthorized && attempt == 0 {
			continue
		}
		if res.StatusCode/100 != 2 {
			return fmt.Errorf("qpay %s: %d %s", path, res.StatusCode, truncate(string(data), 300))
		}
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("qpay %s: %w", path, err)
		}
		return nil
	}
}

func (c *Client) accessToken(ctx context.Context, force bool) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !force && c.token != "" && time.Now().Before(c.expiry) {
		return c.token, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/auth/token", nil)
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(c.Username, c.Password)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("qpay auth: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return "", ErrAuth
	}
	if res.StatusCode/100 != 2 {
		data, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return "", fmt.Errorf("qpay auth: %d %s", res.StatusCode, truncate(string(data), 300))
	}
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err := json.NewDecoder(res.Body).Decode(&t); err != nil || t.AccessToken == "" {
		return "", errors.New("qpay auth: токен ирсэнгүй")
	}
	c.token, c.expiry = t.AccessToken, tokenExpiry(t.ExpiresIn, time.Now())
	return c.token, nil
}

// tokenExpiry: QPay "expires_in"-ийг Unix хугацаагаар (секунд) өгдөг; үргэлжлэх секунд байвал түүгээр.
// Аль ч тохиолдолд 1 минутын өмнө шинэчилнэ, 1 цагаас удаан хадгалахгүй.
func tokenExpiry(v int64, now time.Time) time.Time {
	exp := now.Add(time.Duration(v) * time.Second)
	if v > 1_000_000_000 {
		exp = time.Unix(v, 0)
	}
	if limit := now.Add(time.Hour); exp.After(limit) {
		exp = limit
	}
	return exp.Add(-time.Minute)
}

func amount(raw json.RawMessage) int64 {
	s := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(f + 0.5)
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
