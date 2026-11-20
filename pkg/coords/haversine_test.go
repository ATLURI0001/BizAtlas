package coords

import (
	"math"
	"testing"
)

func TestHaversineDistance_KnownLocations(t *testing.T) {
	// London: 51.5074 N, 0.1278 W
	london := Coordinates{Latitude: 51.5074, Longitude: -0.1278}
	// Paris: 48.8566 N, 2.3522 E
	paris := Coordinates{Latitude: 48.8566, Longitude: 2.3522}

	// Approximate distance ~344 km
	distKm, err := HaversineDistance(london, paris)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if distKm < 340.0 || distKm > 346.0 {
		t.Fatalf("expected distance between London and Paris ~343-344 km, got %f", distKm)
	}

	distMeters, err := london.DistanceMeters(paris)
	if err != nil {
		t.Fatalf("unexpected error in DistanceMeters: %v", err)
	}
	if math.Abs(distMeters-(distKm*1000.0)) > 1e-4 {
		t.Fatalf("expected meters to be 1000x km, got %f vs %f", distMeters, distKm*1000.0)
	}

	// Identical point distance should be 0.0
	zeroDist, err := london.DistanceKm(london)
	if err != nil || zeroDist != 0.0 {
		t.Fatalf("expected zero distance for identical coordinates, got %f, err: %v", zeroDist, err)
	}
}

func TestHaversineDistance_InvalidCoordinates(t *testing.T) {
	invalidLat := Coordinates{Latitude: 95.0, Longitude: 10.0}
	valid := Coordinates{Latitude: 10.0, Longitude: 10.0}

	_, err := HaversineDistance(invalidLat, valid)
	if err != ErrInvalidCoordinates {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
	}

	invalidLon := Coordinates{Latitude: 10.0, Longitude: 200.0}
	_, err = HaversineDistance(valid, invalidLon)
	if err != ErrInvalidCoordinates {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
	}
}

func TestBoundingBox_Calculations(t *testing.T) {
	center := Coordinates{Latitude: 0.0, Longitude: 0.0} // Equator / Prime Meridian
	radiusKm := 111.0                                    // ~1 degree on equator

	minCoords, maxCoords, err := center.BoundingBox(radiusKm)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if minCoords.Latitude >= 0.0 || maxCoords.Latitude <= 0.0 {
		t.Fatalf("expected bounding box to span around equator, got minLat=%f, maxLat=%f", minCoords.Latitude, maxCoords.Latitude)
	}
	if minCoords.Longitude >= 0.0 || maxCoords.Longitude <= 0.0 {
		t.Fatalf("expected bounding box to span around meridian, got minLon=%f, maxLon=%f", minCoords.Longitude, maxCoords.Longitude)
	}

	// Invalid radius
	_, _, err = center.BoundingBox(-10.0)
	if err != ErrInvalidRadius {
		t.Fatalf("expected ErrInvalidRadius for negative radius, got %v", err)
	}
}
