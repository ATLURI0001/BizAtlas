package coords

import "testing"

func TestBenchmarkCoordDistance(t *testing.T) {
	d := BenchmarkCoordDistance(12.97, 77.59, 13.08, 80.27, 500)
	if d < 0 {
		t.Fatalf("expected non-negative duration, got %v", d)
	}
}
