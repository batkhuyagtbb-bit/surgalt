package khanbank

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// Токен (Basic auth, хоосон бие) нэг удаа авч дахин ашиглана; 401 бол шинэчлээд нэг удаа давтана;
// банкны алдааг (CGW код, Apigee fault) задлаж, татгалзсан/тодорхойгүйг ялгана; журнал тоо ч мөр ч болно.
func TestClient(t *testing.T) {
	var tokens, expire atomic.Int32
	var last map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/auth/token" {
			u, p, ok := r.BasicAuth()
			if !ok || u != "ck" || p != "sk" || r.URL.Query().Get("grant_type") != "client_credentials" || r.ContentLength != 0 {
				w.WriteHeader(401)
				return
			}
			tokens.Add(1)
			_, _ = io.WriteString(w, `{"access_token":"T`+string(rune('0'+tokens.Load()))+`","access_token_expires_in":1799,"developer_email":"dev@x.mn"}`)
			return
		}
		if r.Header.Get("Authorization") == "" || (expire.Load() == 1 && r.Header.Get("Authorization") == "Bearer T1") {
			expire.Store(0)
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/v1/accounts":
			_, _ = io.WriteString(w, `{"accounts":[{"number":"5021712565","type":"D","currency":"MNT","status":"OPEN","balance":1000,"avalaibleBalance":900,"name":"ХАРИЛЦАХ"}]}`)
		case "/v1/accounts/cam":
			if r.URL.Query().Get("acct") != "8000070728" || r.URL.Query().Get("bank_code") != "390000" {
				w.WriteHeader(400)
				_, _ = io.WriteString(w, `{"code":"CGW009","message":"Хүлээн авагчийн дансны дугаар ДНС бүртгэлгүй байна "}`)
				return
			}
			_, _ = io.WriteString(w, `{"iban":"MN570039008000070728","alias":null,"name":"Б*т     Б**д ","msgId":"1"}`)
		case "/v1/transfer/interbank":
			_ = json.NewDecoder(r.Body).Decode(&last)
			_, _ = io.WriteString(w, `{"uuid":"u1","journalNo":7148753,"systemDate":"2026/09/25"}`)
		case "/v1/transfer/domestic":
			w.WriteHeader(500)
			_, _ = io.WriteString(w, `{"fault":{"faultstring":"Gateway Timeout","detail":{"errorcode":"messaging.adaptors.http.flow.GatewayTimeout"}}}`)
		}
	}))
	defer srv.Close()
	c := New(srv.URL+"/v1", "ck", "sk")
	ctx := context.Background()

	accts, err := c.Accounts(ctx)
	if err != nil || len(accts) != 1 || accts[0].Number != "5021712565" || accts[0].Available != 900 {
		t.Fatalf("данс: %+v %v", accts, err)
	}
	if e, _ := c.DeveloperEmail(ctx); e != "dev@x.mn" || tokens.Load() != 1 {
		t.Fatalf("токен дахин ашиглах: %q %d", e, tokens.Load())
	}
	h, err := c.NameInquiry(ctx, "8000070728", "390000")
	if err != nil || h.Name != "Б*т Б**д" || h.IBAN != "MN570039008000070728" {
		t.Fatalf("ДНС: %+v %v", h, err)
	}
	if _, err := c.NameInquiry(ctx, "1", "150000"); !Rejected(err) || err.(*APIError).Code != "CGW009" {
		t.Fatalf("бүртгэлгүй данс: %v", err)
	}
	// Токен хүчингүй болсон: шинэчлээд нэг удаа давтана.
	expire.Store(1)
	res, err := c.Interbank(ctx, Transfer{FromAccount: "5021712565", ToAccount: "8000070728", ToAccountName: "Бат Болд", ToBank: "390000", ToBankBic: "MBNKMNUB",
		Amount: 12500, Description: "surgalt.mn", TransferID: "SG1", LoginName: "dev@x.mn", TranPassword: "pw"})
	if err != nil || res.Journal != "7148753" || res.UUID != "u1" || tokens.Load() != 2 {
		t.Fatalf("банк хоорондын: %+v %v токен %d", res, err, tokens.Load())
	}
	if last["toBank"] != "390000" || last["amount"] != 12500.0 || last["toCurrency"] != "MNT" || last["currency"] != "MNT" || last["tranPassword"] != "pw" || last["transferid"] != "SG1" {
		t.Fatalf("илгээсэн бие: %v", last)
	}
	// 5xx — тодорхойгүй (дахин илгээхгүй).
	if _, err := c.Domestic(ctx, Transfer{FromAccount: "1", ToAccount: "2", Amount: 1, LoginName: "a", TranPassword: "b"}); err == nil || Rejected(err) || err.(*APIError).Message != "Gateway Timeout" {
		t.Fatalf("5xx: %v", err)
	}
	// Нууц үггүй бол банк руу явуулахгүй.
	if _, err := c.Domestic(ctx, Transfer{FromAccount: "1", ToAccount: "2", Amount: 1}); !Rejected(err) {
		t.Fatalf("нууц үггүй: %v", err)
	}
	if !New("", "a", "b").Test() || New(ProductionURL, "a", "b").Test() {
		t.Fatal("орчин")
	}
}
