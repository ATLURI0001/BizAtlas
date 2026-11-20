package coords

import (
	"errors"
	"math"
	"testing"
)

func TestPolygon_Validation(t *testing.T) {
	// Too few vertices
	_, errFew := NewPolygon([]Coordinates{
		{Latitude: 10, Longitude: 10},
		{Latitude: 20, Longitude: 20},
	})
	if !errors.Is(errFew, ErrDegeneratePolygon) {
		t.Fatalf("expected ErrDegeneratePolygon, got %v", errFew)
	}

	// Invalid coordinates
	_, errInvalid := NewPolygon([]Coordinates{
		{Latitude: 10, Longitude: 10},
		{Latitude: 120, Longitude: 20}, // invalid lat
		{Latitude: 20, Longitude: 30},
	})
	if !errors.Is(errInvalid, ErrInvalidCoordinates) {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", errInvalid)
	}
}

func TestPolygon_ContainsPoint(t *testing.T) {
	// Square box around lat [10, 20], lon [30, 40]
	poly, err := NewPolygon([]Coordinates{
		{Latitude: 10, Longitude: 30},
		{Latitude: 20, Longitude: 30},
		{Latitude: 20, Longitude: 40},
		{Latitude: 10, Longitude: 40},
	})
	if err != nil {
		t.Fatalf("unexpected error creating polygon: %v", err)
	}

	// Inside center point
	inside, err := poly.ContainsPoint(Coordinates{Latitude: 15, Longitude: 35})
	if err != nil || !inside {
		t.Fatalf("expected point to be inside, got inside=%v, err=%v", inside, err)
	}

	// Outside point to the west
	outsideWest, err := poly.ContainsPoint(Coordinates{Latitude: 15, Longitude: 25})
	if err != nil || outsideWest {
		t.Fatalf("expected point to be outside, got inside=%v, err=%v", outsideWest, err)
	}

	// Outside point to the north
	outsideNorth, err := poly.ContainsPoint(Coordinates{Latitude: 25, Longitude: 35})
	if err != nil || outsideNorth {
		t.Fatalf("expected point to be outside, got inside=%v, err=%v", outsideNorth, err)
	}

	// Vertex point
	vertexMatch, err := poly.ContainsPoint(Coordinates{Latitude: 10, Longitude: 30})
	if err != nil || !vertexMatch {
		t.Fatalf("expected vertex point to return true, got %v", vertexMatch)
	}
}

func TestPolygon_GeodesicArea(t *testing.T) {
	// 1-degree square at equator: [0, 0] to [1, 1]
	poly, err := NewPolygon([]Coordinates{
		{Latitude: 0, Longitude: 0},
		{Latitude: 1, Longitude: 0},
		{Latitude: 1, Longitude: 1},
		{Latitude: 0, Longitude: 1},
	})
	if err != nil {
		t.Fatalf("unexpected polygon error: %v", err)
	}

	area, err := poly.GeodesicAreaKm2()
	if err != nil {
		t.Fatalf("unexpected area error: %v", err)
	}

	// Theoretical area: ~12,364 km^2 (within 1% spherical accuracy)
	if area < 12000.0 || area > 12500.0 {
		t.Fatalf("expected geodesic area ~12364 km^2, got %f km^2", area)
	}
}

func TestPolygon_ConvexHull(t *testing.T) {
	points := []Coordinates{
		{Latitude: 0, Longitude: 0},   // hull vertex
		{Latitude: 0, Longitude: 10},  // hull vertex
		{Latitude: 10, Longitude: 10}, // hull vertex
		{Latitude: 10, Longitude: 0},  // hull vertex
		{Latitude: 5, Longitude: 5},   // internal point
		{Latitude: 2, Longitude: 2},   // internal point
	}

	hull, err := ConvexHull(points)
	if err != nil {
		t.Fatalf("unexpected convex hull error: %v", err)
	}

	if len(hull) != 4 {
		t.Fatalf("expected 4 vertices in convex hull, got %d: %+v", len(hull), hull)
	}

	// Verify all 4 corner points exist in hull
	for _, expectedCorner := range []Coordinates{
		{Latitude: 0, Longitude: 0},
		{Latitude: 0, Longitude: 10},
		{Latitude: 10, Longitude: 10},
		{Latitude: 10, Longitude: 0},
	} {
		found := false
		for _, h := range hull {
			if math.Abs(h.Latitude-expectedCorner.Latitude) < 1e-6 &&
				math.Abs(h.Longitude-expectedCorner.Longitude) < 1e-6 {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected corner %+v not found in hull", expectedCorner)
		}
	}
}
