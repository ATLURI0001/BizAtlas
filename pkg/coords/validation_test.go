package coords

import (
	"strings"
	"testing"
)

func TestCoordinates_Validation(t *testing.T) {
	valid := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	if !valid.IsValid() {
		t.Fatal("expected San Francisco coordinates to be valid")
	}

	str := valid.String()
	if !strings.Contains(str, "37.774900") || !strings.Contains(str, "-122.419400") {
		t.Fatalf("unexpected formatted string: %s", str)
	}

	invalidLat := Coordinates{Latitude: -91.0, Longitude: 0.0}
	if invalidLat.IsValid() {
		t.Fatal("expected latitude < -90 to be invalid")
	}

	invalidLon := Coordinates{Latitude: 0.0, Longitude: 181.0}
	if invalidLon.IsValid() {
		t.Fatal("expected longitude > 180 to be invalid")
	}
}
