package coords

import "testing"

func TestValidatePaginationCursor(t *testing.T) {
	if err := ValidatePaginationCursor(0, 20); err != nil {
		t.Fatalf("expected valid cursor, got %v", err)
	}
	if err := ValidatePaginationCursor(-1, 20); err == nil {
		t.Fatal("expected negative offset violation")
	}
}
