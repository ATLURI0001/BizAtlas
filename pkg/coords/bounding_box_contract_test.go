package coords

import "testing"

func TestBoundingBoxContract(t *testing.T) {
	c := NewBoundingBoxContract(-90.0, 90.0)
	if err := c.Validate(37.7749); err != nil {
		t.Fatal(err)
	}
	if err := c.Validate(120.0); err == nil {
		t.Fatal("expected error for out-of-bounds latitude")
	}
}
