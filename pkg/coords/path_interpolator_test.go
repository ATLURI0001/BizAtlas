package coords

import (
	"math"
	"testing"
)

func TestInitialBearing(t *testing.T) {
	origin := Coordinates{Latitude: 0, Longitude: 0}
	north := Coordinates{Latitude: 10, Longitude: 0}
	east := Coordinates{Latitude: 0, Longitude: 10}
	south := Coordinates{Latitude: -10, Longitude: 0}
	west := Coordinates{Latitude: 0, Longitude: -10}

	tests := []struct {
		name        string
		c1, c2      Coordinates
		expectedDeg float64
	}{
		{"North", origin, north, 0.0},
		{"East", origin, east, 90.0},
		{"South", origin, south, 180.0},
		{"West", origin, west, 270.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := InitialBearing(tt.c1, tt.c2)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if math.Abs(b-tt.expectedDeg) > 0.01 {
				t.Errorf("expected bearing %.2f, got %.2f", tt.expectedDeg, b)
			}
		})
	}

	// Invalid coordinates
	_, err := InitialBearing(Coordinates{Latitude: 95, Longitude: 0}, origin)
	if err != ErrInvalidCoordinates {
		t.Errorf("expected ErrInvalidCoordinates, got %v", err)
	}
}

func TestTotalPathDistanceMeters(t *testing.T) {
	// Equator segment: 0,0 to 0,1 degree
	p1 := Coordinates{Latitude: 0, Longitude: 0}
	p2 := Coordinates{Latitude: 0, Longitude: 1}
	p3 := Coordinates{Latitude: 0, Longitude: 2}

	dist, err := TotalPathDistanceMeters([]Coordinates{p1, p2, p3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 1 deg on equator is approx 111.319 km -> 2 deg is approx 222.6 km
	if dist < 220000 || dist > 225000 {
		t.Errorf("expected ~222,600m, got %.2f", dist)
	}

	// Single point or empty path
	zeroDist, err := TotalPathDistanceMeters([]Coordinates{p1})
	if err != nil || zeroDist != 0 {
		t.Errorf("expected 0 for single point path, got %v, err=%v", zeroDist, err)
	}
}

func TestInterpolateSegment_And_Path(t *testing.T) {
	sf := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	la := Coordinates{Latitude: 34.0522, Longitude: -118.2437}

	// Distance between SF and LA is ~559 km (559,000m)
	totalDistKm, err := HaversineDistance(sf, la)
	if err != nil {
		t.Fatalf("HaversineDistance failed: %v", err)
	}
	_ = totalDistKm

	// Interpolate with max 50km (50,000m) segments
	maxStep := 50000.0
	points, err := InterpolateSegment(sf, la, maxStep)
	if err != nil {
		t.Fatalf("InterpolateSegment failed: %v", err)
	}

	if len(points) < 11 {
		t.Fatalf("expected at least 11 points for 559km route, got %d", len(points))
	}

	// Verify endpoints match
	if points[0] != sf || points[len(points)-1] != la {
		t.Errorf("endpoints mismatch: first=%v, last=%v", points[0], points[len(points)-1])
	}

	// Verify each consecutive step is within maxStepMeters (+ small epsilon for floating point)
	for i := 0; i < len(points)-1; i++ {
		dKm, err := HaversineDistance(points[i], points[i+1])
		if err != nil {
			t.Fatalf("step %d distance failed: %v", i, err)
		}
		stepM := dKm * 1000.0
		if stepM > maxStep+1.0 {
			t.Errorf("step %d exceeded maxStep: %.2fm > %.2fm", i, stepM, maxStep)
		}
	}

	// Test multi-segment InterpolatePath
	ny := Coordinates{Latitude: 40.7128, Longitude: -74.0060}
	multiPath, err := InterpolatePath([]Coordinates{sf, la, ny}, 100000.0)
	if err != nil {
		t.Fatalf("InterpolatePath failed: %v", err)
	}
	if len(multiPath) < 40 {
		t.Errorf("expected densified multiPath >= 40 points, got %d", len(multiPath))
	}
}

func TestCrossTrackDistanceMeters(t *testing.T) {
	start := Coordinates{Latitude: 0, Longitude: 0}
	end := Coordinates{Latitude: 0, Longitude: 10}

	// Point right on the equator
	onPath := Coordinates{Latitude: 0, Longitude: 5}
	d, err := CrossTrackDistanceMeters(onPath, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d > 0.01 {
		t.Errorf("expected 0 cross-track distance, got %.4fm", d)
	}

	// Point 1 degree north of equator (approx 111.3 km)
	offPath := Coordinates{Latitude: 1, Longitude: 5}
	d, err = CrossTrackDistanceMeters(offPath, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d < 110000 || d > 112000 {
		t.Errorf("expected ~111,300m cross track distance, got %.2fm", d)
	}
}

func TestSimplifyPath_RDP(t *testing.T) {
	// Test line with strictly collinear points
	p1 := Coordinates{Latitude: 0, Longitude: 0}
	p2 := Coordinates{Latitude: 0, Longitude: 1} // Collinear
	p3 := Coordinates{Latitude: 0, Longitude: 2} // Collinear
	p4 := Coordinates{Latitude: 0, Longitude: 3} // Collinear
	p5 := Coordinates{Latitude: 0, Longitude: 4} // Collinear
	p6 := Coordinates{Latitude: 0, Longitude: 5} // End

	collinearPath := []Coordinates{p1, p2, p3, p4, p5, p6}

	// Simplification with tolerance 10,000m (10km):
	// All intermediate collinear points on the equator should be removed.
	simplified, err := SimplifyPath(collinearPath, 10000.0)
	if err != nil {
		t.Fatalf("SimplifyPath failed: %v", err)
	}

	if len(simplified) != 2 {
		t.Fatalf("expected 2 points (endpoints only), got %d: %v", len(simplified), simplified)
	}
	if simplified[0] != p1 || simplified[1] != p6 {
		t.Errorf("unexpected endpoints: %v", simplified)
	}

	// Now add a sharp detour spike at index 3
	spike := Coordinates{Latitude: 2, Longitude: 2.5} // ~222km north
	withSpike := []Coordinates{p1, p2, p3, spike, p4, p5, p6}
	// 190km tolerance: intermediate points on the leg toward the spike (~178km off-track)
	// are simplified away, while the 222km spike itself is retained.
	simplifiedSpike, err := SimplifyPath(withSpike, 190000.0)
	if err != nil {
		t.Fatalf("SimplifyPath with spike failed: %v", err)
	}

	// Should retain p1, spike, and p6
	if len(simplifiedSpike) != 3 {
		t.Fatalf("expected 3 points (p1, spike, p6), got %d: %v", len(simplifiedSpike), simplifiedSpike)
	}
	if simplifiedSpike[0] != p1 || simplifiedSpike[1] != spike || simplifiedSpike[2] != p6 {
		t.Errorf("unexpected simplified points with spike: %v", simplifiedSpike)
	}
}

func TestSimplifyPath_ChaosStress(t *testing.T) {
	// Generate a 1,000-point path to verify stack safety and performance
	path := make([]Coordinates, 1000)
	for i := 0; i < 1000; i++ {
		lat := float64(i) * 0.01
		lon := math.Sin(float64(i)*0.1) * 0.5
		path[i] = Coordinates{Latitude: lat, Longitude: lon}
	}

	simplified, err := SimplifyPath(path, 5000.0)
	if err != nil {
		t.Fatalf("SimplifyPath on 1000 pts failed: %v", err)
	}
	if len(simplified) >= 1000 {
		t.Errorf("expected simplified path to reduce points, got %d", len(simplified))
	}

	// Edge case: empty and 2 points
	res, err := SimplifyPath(nil, 100.0)
	if err != nil || len(res) != 0 {
		t.Errorf("expected empty result for nil, got %v, err=%v", res, err)
	}

	// Invalid tolerance
	_, err = SimplifyPath(path[:2], -10.0)
	if err != ErrInvalidTolerance {
		t.Errorf("expected ErrInvalidTolerance, got %v", err)
	}
}
