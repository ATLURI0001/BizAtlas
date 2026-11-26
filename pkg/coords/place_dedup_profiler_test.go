package coords

import "testing"

func TestBenchmarkPlaceDedup(t *testing.T) {
	u, d := BenchmarkPlaceDedup([]string{"a", "b", "a", "c"}, 100)
	if u != 3 || d < 0 {
		t.Fatalf("unexpected dedup result: %d, %v", u, d)
	}
}
