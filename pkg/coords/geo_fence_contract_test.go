package coords

import "testing"

func TestValidateGeoFence(t *testing.T) {
	if err := ValidateGeoFence(37.7749, -122.4194); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateGeoFence(91.0, 0.0); err == nil {
		t.Fatal("expected error for out-of-range latitude")
	}
}
