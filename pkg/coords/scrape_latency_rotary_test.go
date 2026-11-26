package coords

import "testing"

func TestScrapeLatencyRotary(t *testing.T) {
	r := NewScrapeLatencyRotary(2)
	r.Record(10)
	r.Record(20)
	r.Record(30)
	if got := r.Avg(); got != 25 {
		t.Fatalf("expected avg 25, got %v", got)
	}
}
