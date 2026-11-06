package store

import (
	"context"
	"testing"
)

// Дахин судлахад асуултын зөв хариулт буруугаар дарагдахгүй (ахиц зөвхөн сайжирна).
func TestQuizKeepsBest(t *testing.T) {
	m := NewMemory()
	ctx := context.Background()
	_ = m.SaveQuizResult(ctx, "u1", "c1", "l1", "q001", false)
	_ = m.SaveQuizResult(ctx, "u1", "c1", "l1", "q001", true)
	_ = m.SaveQuizResult(ctx, "u1", "c1", "l1", "q001", false)
	if p := m.progress[[2]string{"u1", "l1"}]; p == nil || !p.Quiz["q001"] {
		t.Fatalf("зөв хариулт буруугаар дарагдлаа: %+v", p)
	}
}
