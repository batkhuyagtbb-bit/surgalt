// Package khanbank — Хаан банкны Corporate Gateway (CGW Developer manual v1.1.7): токен, данс,
// ДНС-ээс данс эзэмшигчийн нэр лавлах, банк доторх (CG007) болон банк хоорондын (CG008) шилжүүлэг.
//
// Тестийн орчин: https://doob.world:6442/v1, бодит: https://api.khanbank.com/v1.
// Шилжүүлгийг хэзээ ч автоматаар давтахгүй: transferid давтагдашгүй гэсэн баталгаа гарын авлагад байхгүй тул
// хариу тодорхойгүй (timeout, 5xx) үед дахин илгээвэл хоёр удаа шилжиж болно. Зөвхөн 401-ийг (банк хүсэлтийг
// хүлээж аваагүй) токен шинэчлээд нэг удаа давтана.
package khanbank

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	TestURL       = "https://doob.world:6442/v1"
	ProductionURL = "https://api.khanbank.com/v1"
)

// Client — нэг байгууллагын CGW эрх (consumer key, secret key). Токеныг хугацаа дуустал дахин ашиглана.
type Client struct {
	base, key, secret string
	HTTP              *http.Client

	mu    sync.Mutex
	tok   string
	email string // токен олгосон и-мэйл = шилжүүлгийн loginName
	exp   time.Time
}

func New(base, key, secret string) *Client {
	if base == "" {
		base = TestURL
	}
	// Банкны сервер (doob.world) зөвхөн RSA түлхүүр солилцоот TLS 1.2 шифр дэмждэг; Go 1.22-оос анхдагчаар
	// унтарсан тул зөвхөн энэ клиентэд нэмж зөвшөөрнө (ECDHE-г түрүүлж санал болгоно).
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, CipherSuites: []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256, tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256, tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	}}
	return &Client{base: strings.TrimRight(base, "/"), key: key, secret: secret, HTTP: &http.Client{Timeout: 45 * time.Second, Transport: tr}}
}

// Test — тестийн орчин эсэх (бодит мөнгө хөдлөхгүй).
func (c *Client) Test() bool { return !strings.Contains(c.base, "api.khanbank.com") }

// APIError — банкны хариу (HTTP төлөв, CGW код, тайлбар).
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("Хаан банк %d %s: %s", e.Status, e.Code, e.Message)
	}
	return fmt.Sprintf("Хаан банк %d: %s", e.Status, e.Message)
}

// Rejected — банк хүсэлтийг шалгаад татгалзсан (4xx): мөнгө хөдлөөгүй нь тодорхой, засаад дахин илгээж болно.
// Бусад алдаа (сүлжээ, timeout, 5xx) тодорхойгүй — хуулгаас шалгахаас өмнө дахин илгээхгүй.
func Rejected(err error) bool {
	var e *APIError
	return errors.As(err, &e) && e.Status >= 400 && e.Status < 500
}

func parseErr(status int, body []byte) *APIError {
	e := &APIError{Status: status}
	var b struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Fault   struct {
			Faultstring string `json:"faultstring"`
			Detail      struct {
				Errorcode string `json:"errorcode"`
			} `json:"detail"`
		} `json:"fault"`
	}
	if json.Unmarshal(body, &b) == nil {
		e.Code, e.Message = b.Code, strings.TrimSpace(b.Message)
		if e.Message == "" {
			e.Code, e.Message = b.Fault.Detail.Errorcode, b.Fault.Faultstring
		}
	}
	if e.Message == "" {
		e.Message = strings.TrimSpace(string(body))
		if len(e.Message) > 200 {
			e.Message = e.Message[:200]
		}
	}
	return e
}

// token — CG001: POST /auth/token?grant_type=client_credentials (Basic auth, хоосон бие).
func (c *Client) token(ctx context.Context) (string, string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.tok != "" && time.Now().Before(c.exp) {
		return c.tok, c.email, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/auth/token?grant_type=client_credentials", strings.NewReader(""))
	if err != nil {
		return "", "", err
	}
	req.SetBasicAuth(c.key, c.secret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.HTTP.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return "", "", parseErr(res.StatusCode, body)
	}
	var t struct {
		Token   string `json:"access_token"`
		Expires int    `json:"access_token_expires_in"`
		Email   string `json:"developer_email"`
	}
	if err := json.Unmarshal(body, &t); err != nil || t.Token == "" {
		return "", "", &APIError{Status: res.StatusCode, Message: "токен ирсэнгүй"}
	}
	if t.Expires <= 0 {
		t.Expires = 1800
	}
	c.tok, c.email, c.exp = t.Token, t.Email, time.Now().Add(time.Duration(t.Expires-60)*time.Second)
	return c.tok, c.email, nil
}

// DeveloperEmail — токен олгосон и-мэйл (шилжүүлгийн loginName).
func (c *Client) DeveloperEmail(ctx context.Context) (string, error) {
	_, e, err := c.token(ctx)
	return e, err
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, in, out any) error {
	var payload []byte
	if in != nil {
		var err error
		if payload, err = json.Marshal(in); err != nil {
			return err
		}
	}
	for attempt := 0; ; attempt++ {
		tok, _, err := c.token(ctx)
		if err != nil {
			return err
		}
		u := c.base + path
		if len(q) > 0 {
			u += "?" + q.Encode()
		}
		req, err := http.NewRequestWithContext(ctx, method, u, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		if in != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		res, err := c.HTTP.Do(req)
		if err != nil {
			return err
		}
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
		res.Body.Close()
		if res.StatusCode == http.StatusUnauthorized && attempt == 0 { // токен хүчингүй болсон: банк хүсэлтийг хүлээж аваагүй
			c.mu.Lock()
			c.tok = ""
			c.mu.Unlock()
			continue
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return parseErr(res.StatusCode, body)
		}
		if out == nil || len(bytes.TrimSpace(body)) == 0 {
			return nil
		}
		return json.Unmarshal(body, out)
	}
}

// Account — байгууллагын данс (CG002).
type Account struct {
	Number    string  `json:"number"`
	Type      string  `json:"type"` // D — харилцах/хадгаламж, L — зээл
	Currency  string  `json:"currency"`
	Status    string  `json:"status"`
	Name      string  `json:"name"`
	Balance   float64 `json:"balance"`
	Available float64 `json:"avalaibleBalance"` // банкны бичлэгээрээ (avalaible)
	IBAN      string  `json:"ibanAcctNo"`
}

// Accounts — GET /accounts (хариу {"accounts":[…]}).
func (c *Client) Accounts(ctx context.Context) ([]Account, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/accounts", nil, nil, &raw); err != nil {
		return nil, err
	}
	var env struct {
		Accounts []Account `json:"accounts"`
	}
	if json.Unmarshal(raw, &env) == nil && env.Accounts != nil {
		return env.Accounts, nil
	}
	var list []Account
	return list, json.Unmarshal(raw, &list)
}

// Holder — ДНС-ийн лавлагаа: далдалсан нэр, IBAN.
type Holder struct {
	Name string `json:"name"`
	IBAN string `json:"iban"`
}

// NameInquiry — CG006: GET /accounts/cam?acct=&bank_code= (бусад банкных ч болно).
func (c *Client) NameInquiry(ctx context.Context, acct, bankCode string) (*Holder, error) {
	var h Holder
	if err := c.do(ctx, http.MethodGet, "/accounts/cam", url.Values{"acct": {acct}, "bank_code": {bankCode}}, nil, &h); err != nil {
		return nil, err
	}
	h.Name = strings.Join(strings.Fields(h.Name), " ")
	return &h, nil
}

// Transfer — нэг шилжүүлэг. Банк доторх бол ToAccountName, ToBank, ToBankBic хэрэггүй.
type Transfer struct {
	FromAccount   string `json:"fromAccount"`
	ToAccount     string `json:"toAccount"`
	ToAccountName string `json:"toAccountName,omitempty"` // CG008: хүлээн авагчийн нэр овог (ДНС-тэй тулгана)
	ToBank        string `json:"toBank,omitempty"`        // CG008: банкны код (жишээ нь 150000)
	ToBankBic     string `json:"toBankBic,omitempty"`     // CG008: BIC (жишээ нь GLMTMNUB)
	ToCurrency    string `json:"toCurrency"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Description   string `json:"description"`
	TransferID    string `json:"transferid"`
	LoginName     string `json:"loginName"`
	TranPassword  string `json:"tranPassword"`
}

// TransferResult — гүйлгээний код, журналын дугаар, огноо.
type TransferResult struct {
	UUID       string `json:"uuid"`
	Journal    string `json:"-"`
	SystemDate string `json:"systemDate"`
}

func (r *TransferResult) UnmarshalJSON(b []byte) error {
	var w struct {
		UUID       string          `json:"uuid"`
		JournalNo  json.RawMessage `json:"journalNo"` // бодит орчинд тоо, тестэд мөр
		SystemDate string          `json:"systemDate"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	r.UUID, r.SystemDate, r.Journal = w.UUID, w.SystemDate, strings.Trim(string(w.JournalNo), `" `)
	if r.Journal == "null" {
		r.Journal = ""
	}
	return nil
}

// Domestic — CG007: Хаан банкны данс руу.
func (c *Client) Domestic(ctx context.Context, t Transfer) (*TransferResult, error) {
	t.ToAccountName, t.ToBank, t.ToBankBic = "", "", ""
	return c.transfer(ctx, "/transfer/domestic", t)
}

// Interbank — CG008: бусад банкны данс руу (зөвхөн MNT; 09:30–16:30-аас хойш бол дараагийн ажлын өдөр).
func (c *Client) Interbank(ctx context.Context, t Transfer) (*TransferResult, error) {
	return c.transfer(ctx, "/transfer/interbank", t)
}

func (c *Client) transfer(ctx context.Context, path string, t Transfer) (*TransferResult, error) {
	switch {
	case t.FromAccount == "" || t.ToAccount == "" || t.Amount <= 0:
		return nil, &APIError{Status: http.StatusBadRequest, Code: "local", Message: "данс, дүн дутуу"}
	case t.LoginName == "" || t.TranPassword == "":
		return nil, &APIError{Status: http.StatusBadRequest, Code: "local", Message: "гүйлгээний нэвтрэх нэр, нууц үг тохируулаагүй"}
	}
	if t.Currency == "" {
		t.Currency = "MNT"
	}
	if t.ToCurrency == "" {
		t.ToCurrency = "MNT"
	}
	var r TransferResult
	if err := c.do(ctx, http.MethodPost, path, nil, t, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
