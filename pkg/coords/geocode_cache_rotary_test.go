package coords

import "testing"

func TestGeocodeCacheRotary(t *testing.T) {
	r := NewGeocodeCacheRotary(2)
	r.Push("a")
	r.Push("b")
	r.Push("c")
	if r.Count() != 2 {
		t.Fatalf("expected 2, got %d", r.Count())
	}
}
