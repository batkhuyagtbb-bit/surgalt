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

// Урт цонхтой хязгаарын bucket дахин дүүрэхээс өмнө цэвэрлэгдэхгүй (цэвэрлэгдвэл шинээр бүтэн burst авна).
func TestIdleCoversRefill(t *testing.T) {
	if d := New(1, 5).idle(); d.Minutes() != 3 {
		t.Fatalf("богино цонх: %v", d)
	}
	if d := New(30.0/3600, 30).idle(); d.Hours() != 1 {
		t.Fatalf("цагт 30: %v", d)
	}
	if d := New(0, 0).idle(); d.Minutes() != 3 {
		t.Fatalf("хязгааргүй: %v", d)
	}
}
