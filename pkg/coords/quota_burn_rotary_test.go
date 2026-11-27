package coords

import "testing"

func TestQuotaBurnRotary(t *testing.T) {
	r := NewQuotaBurnRotary(2)
	r.Record(1)
	r.Record(3)
	r.Record(5)
	if got := r.Avg(); got != 4 {
		t.Fatalf("expected 4, got %v", got)
	}
}
