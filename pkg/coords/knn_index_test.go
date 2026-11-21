package coords

import (
	"errors"
	"testing"
)

func TestSpatialGridIndex_InsertAndCount(t *testing.T) {
	idx := NewSpatialGridIndex(0.05)

	p1 := SpatialPoint{ID: "p1", Coord: Coordinates{Latitude: 37.7749, Longitude: -122.4194}} // SF
	p2 := SpatialPoint{ID: "p2", Coord: Coordinates{Latitude: 37.7833, Longitude: -122.4167}} // SF nearby
	invalid := SpatialPoint{ID: "bad", Coord: Coordinates{Latitude: 100.0, Longitude: 0.0}}

	if err := idx.Insert(p1); err != nil {
		t.Fatalf("unexpected Insert error: %v", err)
	}
	if err := idx.Insert(p2); err != nil {
		t.Fatalf("unexpected Insert error: %v", err)
	}
	if err := idx.Insert(invalid); !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
	}

	if idx.Count() != 2 {
		t.Fatalf("expected count 2, got %d", idx.Count())
	}
}

func TestSpatialGridIndex_NearestK(t *testing.T) {
	idx := NewSpatialGridIndex(0.05)

	sfOrigin := Coordinates{Latitude: 37.7749, Longitude: -122.4194}

	points := []SpatialPoint{
		{ID: "sf-center", Coord: Coordinates{Latitude: 37.7750, Longitude: -122.4195}},  // ~0.01 km
		{ID: "oakland", Coord: Coordinates{Latitude: 37.8044, Longitude: -122.2711}},    // ~13.5 km
		{ID: "san-jose", Coord: Coordinates{Latitude: 37.3382, Longitude: -121.8863}},   // ~67 km
		{ID: "tokyo", Coord: Coordinates{Latitude: 35.6762, Longitude: 139.6503}},       // ~8280 km
	}

	if err := idx.BatchInsert(points); err != nil {
		t.Fatalf("BatchInsert failed: %v", err)
	}

	// Find top 2 nearest points within 50 km
	results, err := idx.NearestK(sfOrigin, 2, 50.0)
	if err != nil {
		t.Fatalf("NearestK failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 nearest points, got %d", len(results))
	}
	if results[0].Point.ID != "sf-center" {
		t.Errorf("expected sf-center first, got %s", results[0].Point.ID)
	}
	if results[1].Point.ID != "oakland" {
		t.Errorf("expected oakland second, got %s", results[1].Point.ID)
	}
	if results[0].DistanceKm >= results[1].DistanceKm {
		t.Errorf("expected results sorted ascending by distance")
	}

	// Test invalid k
	_, err = idx.NearestK(sfOrigin, 0, 50.0)
	if !errors.Is(err, ErrInvalidK) {
		t.Errorf("expected ErrInvalidK, got %v", err)
	}
}

func TestSpatialGridIndex_RangeSearch(t *testing.T) {
	idx := NewSpatialGridIndex(0.1)

	center := Coordinates{Latitude: 40.7128, Longitude: -74.0060} // NYC
	points := []SpatialPoint{
		{ID: "wall-st", Coord: Coordinates{Latitude: 40.7074, Longitude: -74.0113}}, // ~0.7 km
		{ID: "times-sq", Coord: Coordinates{Latitude: 40.7580, Longitude: -73.9855}}, // ~5.3 km
		{ID: "philly", Coord: Coordinates{Latitude: 39.9526, Longitude: -75.1652}},   // ~130 km
	}

	_ = idx.BatchInsert(points)

	// Range search 10 km
	matches, err := idx.RangeSearch(center, 10.0)
	if err != nil {
		t.Fatalf("RangeSearch failed: %v", err)
	}

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches within 10km, got %d", len(matches))
	}
	if matches[0].Point.ID != "wall-st" {
		t.Errorf("expected wall-st first, got %s", matches[0].Point.ID)
	}
}
