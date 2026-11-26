package coords

import (
	"errors"
	"testing"
	"time"
)

func TestGeofenceCircuitBreaker_PointInPolygon(t *testing.T) {
	cb := NewGeofenceCircuitBreaker(100, 3, 500*time.Millisecond)

	// Square polygon around (0,0) with size 2x2
	square := GeofencePolygon{
		ID: "box-1",
		Vertices: []Coordinates{
			{Latitude: -1, Longitude: -1},
			{Latitude: 1, Longitude: -1},
			{Latitude: 1, Longitude: 1},
			{Latitude: -1, Longitude: 1},
		},
	}

	// Point inside
	inside, err := cb.ContainsPoint(square, Coordinates{Latitude: 0, Longitude: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inside {
		t.Fatalf("expected point to be inside polygon")
	}

	// Point outside
	outside, err := cb.ContainsPoint(square, Coordinates{Latitude: 5, Longitude: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if outside {
		t.Fatalf("expected point to be outside polygon")
	}
}

func TestGeofenceCircuitBreaker_VertexExplosion(t *testing.T) {
	cb := NewGeofenceCircuitBreaker(10, 2, 500*time.Millisecond)

	// Polygon with 15 vertices > limit of 10
	vertices := make([]Coordinates, 15)
	for i := 0; i < 15; i++ {
		vertices[i] = Coordinates{Latitude: float64(i), Longitude: float64(i)}
	}

	poly := GeofencePolygon{ID: "giant-poly", Vertices: vertices}
	_, err := cb.ContainsPoint(poly, Coordinates{Latitude: 0, Longitude: 0})
	if !errors.Is(err, ErrGeofenceVertexExplosion) {
		t.Fatalf("expected ErrGeofenceVertexExplosion, got %v", err)
	}
}

func TestGeofenceCircuitBreaker_Tripping(t *testing.T) {
	cb := NewGeofenceCircuitBreaker(10, 2, 500*time.Millisecond)

	invalidPoly := GeofencePolygon{ID: "bad", Vertices: []Coordinates{{Latitude: 0, Longitude: 0}}}

	// Failure 1
	cb.ContainsPoint(invalidPoly, Coordinates{Latitude: 0, Longitude: 0})
	// Failure 2 -> should trip
	cb.ContainsPoint(invalidPoly, Coordinates{Latitude: 0, Longitude: 0})

	// Request when OPEN
	_, err := cb.ContainsPoint(invalidPoly, Coordinates{Latitude: 0, Longitude: 0})
	if !errors.Is(err, ErrGeofenceCircuitOpen) {
		t.Fatalf("expected ErrGeofenceCircuitOpen, got %v", err)
	}
}
