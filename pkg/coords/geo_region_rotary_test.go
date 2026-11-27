package coords

import "testing"

func TestGeoRegionRotary(t *testing.T) {
	r := NewGeoRegionRotary(3)
	r.Push("us-east-1")
	r.Push("eu-west-1")
	r.Push("ap-south-1")
	r.Push("us-west-2") // overwrites oldest
	if r.Count() != 3 {
		t.Fatalf("expected count 3, got %d", r.Count())
	}
}
