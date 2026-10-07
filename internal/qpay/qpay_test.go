package qpay

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTokenExpiry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	// Unix хугацаагаар (QPay-ийн хэлбэр): 30 минутын дараа дуусна → 1 минутын өмнө шинэчилнэ.
	if got := tokenExpiry(now.Add(30*time.Minute).Unix(), now); !got.Equal(now.Add(29 * time.Minute)) {
		t.Fatalf("unix: %v", got.Sub(now))
	}
	// Үргэлжлэх секундээр.
	if got := tokenExpiry(600, now); !got.Equal(now.Add(9 * time.Minute)) {
		t.Fatalf("секунд: %v", got.Sub(now))
	}
	// Хэт урт бол 1 цагаар хязгаарлана.
	if got := tokenExpiry(now.Add(24*time.Hour).Unix(), now); !got.Equal(now.Add(59 * time.Minute)) {
		t.Fatalf("хязгаар: %v", got.Sub(now))
	}
}

func TestAmount(t *testing.T) {
	for raw, want := range map[string]int64{`"15000.00"`: 15000, `15000`: 15000, `"99.6"`: 100, `null`: 0, `"x"`: 0} {
		if got := amount(json.RawMessage(raw)); got != want {
			t.Fatalf("%s: %d, хүлээсэн %d", raw, got, want)
		}
	}
}
