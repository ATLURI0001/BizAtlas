package coords

import "testing"

func TestCoordPrecisionSentinel(t *testing.T) {
	s := NewCoordPrecisionSentinel()
	if !s.Check(37.7749) {
		t.Fatal("expected precise coord to pass")
	}
	if s.Check(37.0) {
		t.Fatal("expected coarse coord to fail")
	}
	if s.Hits() != 1 {
		t.Fatalf("expected 1 hit, got %d", s.Hits())
	}
}
