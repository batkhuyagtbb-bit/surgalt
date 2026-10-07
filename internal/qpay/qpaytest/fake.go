// Package qpaytest нь тестэд зориулсан хуурамч QPay сервер (httptest шиг).
package qpaytest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

const (
	User = "TEST_MERCHANT"
	Pass = "123456"
)

type Invoice struct {
	ID       string
	SenderNo string
	Amount   int64
	Desc     string
	Callback string
	Paid     bool
}

type Server struct {
	*httptest.Server
	mu       sync.Mutex
	invoices []*Invoice
	token    string
	n        int
	Auths    int // хэдэн удаа нэвтэрсэн
}

func New() *Server {
	f := &Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/token", f.auth)
	mux.HandleFunc("POST /invoice", f.invoice)
	mux.HandleFunc("POST /payment/check", f.check)
	f.Server = httptest.NewServer(mux)
	return f
}

// Pay нь нэхэмжлэхийг төлөгдсөн болгоно (суралцагч банкны апп-аар уншуулсан мэт).
func (f *Server) Pay(invoiceID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, in := range f.invoices {
		if in.ID == invoiceID {
			in.Paid = true
		}
	}
}

func (f *Server) Invoices() []Invoice {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Invoice, len(f.invoices))
	for i, in := range f.invoices {
		out[i] = *in
	}
	return out
}

// ExpireToken нь одоогийн токеныг хүчингүй болгоно (дахин нэвтрэхийг шалгахад).
func (f *Server) ExpireToken() {
	f.mu.Lock()
	f.token = "expired"
	f.mu.Unlock()
}

func (f *Server) auth(w http.ResponseWriter, r *http.Request) {
	u, p, ok := r.BasicAuth()
	if !ok || u != User || p != Pass {
		http.Error(w, `{"error":"NO_CREDENDIALS"}`, http.StatusUnauthorized)
		return
	}
	f.mu.Lock()
	f.n++
	f.Auths++
	f.token = fmt.Sprintf("tok-%d", f.n)
	tok := f.token
	f.mu.Unlock()
	writeJSON(w, map[string]any{"token_type": "bearer", "access_token": tok, "refresh_token": "r", "expires_in": time.Now().Add(time.Hour).Unix()})
}

func (f *Server) authorized(w http.ResponseWriter, r *http.Request) bool {
	f.mu.Lock()
	ok := f.token != "" && r.Header.Get("Authorization") == "Bearer "+f.token
	f.mu.Unlock()
	if !ok {
		http.Error(w, `{"error":"AUTHENTICATION_FAILED"}`, http.StatusUnauthorized)
	}
	return ok
}

func (f *Server) invoice(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(w, r) {
		return
	}
	var in struct {
		Code     string `json:"invoice_code"`
		SenderNo string `json:"sender_invoice_no"`
		Desc     string `json:"invoice_description"`
		Amount   int64  `json:"amount"`
		Callback string `json:"callback_url"`
	}
	if json.NewDecoder(r.Body).Decode(&in) != nil || in.Code == "" || in.Amount <= 0 || in.SenderNo == "" {
		http.Error(w, `{"error":"INVALID_REQUEST"}`, http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.n++
	inv := &Invoice{ID: fmt.Sprintf("inv-%d", f.n), SenderNo: in.SenderNo, Amount: in.Amount, Desc: in.Desc, Callback: in.Callback}
	f.invoices = append(f.invoices, inv)
	f.mu.Unlock()
	writeJSON(w, map[string]any{"invoice_id": inv.ID, "qr_text": "0002010102121531" + strings.ToUpper(inv.ID), "qr_image": "",
		"qPay_shortUrl": "https://s.qpay.mn/" + inv.ID,
		"urls": []map[string]string{
			{"name": "Khan bank", "description": "Хаан банк", "logo": "https://qpay.mn/q/logo/khanbank.png", "link": "khanbank://q?qPay_QRcode=" + inv.ID},
			{"name": "Golomt bank", "description": "Голомт банк", "logo": "https://qpay.mn/q/logo/golomtbank.png", "link": "golomtbank://q?qPay_QRcode=" + inv.ID},
		}})
}

func (f *Server) check(w http.ResponseWriter, r *http.Request) {
	if !f.authorized(w, r) {
		return
	}
	var in struct {
		ObjectType string `json:"object_type"`
		ObjectID   string `json:"object_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&in)
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := []map[string]any{}
	var paid int64
	for _, inv := range f.invoices {
		if inv.ID == in.ObjectID && inv.Paid {
			paid += inv.Amount
			rows = append(rows, map[string]any{"payment_id": "p-" + inv.ID, "payment_status": "PAID", "payment_amount": fmt.Sprintf("%d.00", inv.Amount), "payment_currency": "MNT"})
		}
	}
	writeJSON(w, map[string]any{"count": len(rows), "paid_amount": paid, "rows": rows})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
