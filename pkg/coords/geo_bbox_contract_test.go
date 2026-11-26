package coords

import "testing"

func TestValidateGeoBBox(t *testing.T) {
	if err := ValidateGeoBBox(10, 20, 30, 40); err != nil {
		t.Fatalf("expected valid bbox, got %v", err)
	}
	if err := ValidateGeoBBox(30, 20, 10, 40); err == nil {
		t.Fatal("expected bbox contract violation")
	}
}
