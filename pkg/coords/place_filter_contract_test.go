package coords

import "testing"

func TestPlaceFilterContract(t *testing.T) {
	c := &PlaceFilterContract{}
	valid := Place{Name: "Test Cafe", Latitude: 12.9716, Longitude: 77.5946, Rating: 4.2}
	if err := c.Validate(valid); err != nil {
		t.Fatalf("expected valid place, got error: %v", err)
	}
	bad := Place{Name: "", Latitude: 200, Longitude: 77.5946, Rating: 6}
	if err := c.Validate(bad); err == nil {
		t.Fatal("expected contract violation for empty name")
	}
}
