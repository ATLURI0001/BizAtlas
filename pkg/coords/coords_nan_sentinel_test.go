package coords

import (
	"math"
	"testing"
)

func TestAssertFiniteCoords(t *testing.T) {
	if err := AssertFiniteCoords(12.97, 77.59); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := AssertFiniteCoords(math.NaN(), 77.59); err == nil {
		t.Fatal("expected NaN sentinel violation")
	}
}
