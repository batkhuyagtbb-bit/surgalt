package ratelimit

import "testing"

func TestBurstThenBlock(t *testing.T) {
	l := New(1, 5)
	allowed := 0
	for i := 0; i < 20; i++ {
		if l.Allow("1.2.3.4") {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("burst 5 байх ёстой, %d", allowed)
	}
	if !l.Allow("5.6.7.8") {
		t.Fatal("өөр IP хязгаарлагдах ёсгүй")
	}
	if !New(0, 0).Allow("x") {
		t.Fatal("rate 0 бол хязгааргүй")
	}
}
