package httpapi

import "testing"

func TestFixBlockID(t *testing.T) {
	seen := map[string]bool{}
	var got []string
	for _, id := range []string{"r1t", "AB-cd", "a001", "a001", "a001", "a0011", "", "abcdefghijklmnopqrst"} {
		f := fixBlockID(id, seen)
		if !ValidBlockID(f) || seen[f] {
			t.Fatalf("%q → %q буруу эсвэл давхардсан", id, f)
		}
		seen[f] = true
		got = append(got, f)
	}
	want := []string{"r1t0", "abcd", "a001", "a0011", "a0012", "a00111", "0000", "abcdefghijklmnop"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%d: %q, хүлээсэн %q (%v)", i, got[i], want[i], got)
		}
	}
}
