package coords

import (
	"sync"
	"testing"
)

func TestGeoRect_ContainsAndIntersects(t *testing.T) {
	rect := GeoRect{MinLat: 10.0, MaxLat: 20.0, MinLon: 30.0, MaxLon: 40.0}

	// Inside
	if !rect.Contains(Coordinates{Latitude: 15.0, Longitude: 35.0}) {
		t.Errorf("expected rect to contain interior point")
	}
	// On boundary
	if !rect.Contains(Coordinates{Latitude: 10.0, Longitude: 30.0}) {
		t.Errorf("expected rect to contain corner point")
	}
	// Outside
	if rect.Contains(Coordinates{Latitude: 25.0, Longitude: 35.0}) {
		t.Errorf("expected rect to not contain outside point")
	}

	// Intersecting
	other1 := GeoRect{MinLat: 15.0, MaxLat: 25.0, MinLon: 35.0, MaxLon: 45.0}
	if !rect.Intersects(other1) {
		t.Errorf("expected intersecting rects")
	}

	// Disjoint
	other2 := GeoRect{MinLat: 50.0, MaxLat: 60.0, MinLon: 70.0, MaxLon: 80.0}
	if rect.Intersects(other2) {
		t.Errorf("expected disjoint rects")
	}
}

func TestSpatialQuadtree_InsertAndSubdivide(t *testing.T) {
	// Node capacity 2, max depth 4
	qt := NewSpatialQuadtree(GlobalEarthRect(), 2, 4)

	pts := []SpatialPoint{
		{ID: "sf", Coord: Coordinates{Latitude: 37.7749, Longitude: -122.4194}},
		{ID: "la", Coord: Coordinates{Latitude: 34.0522, Longitude: -118.2437}},
		{ID: "nyc", Coord: Coordinates{Latitude: 40.7128, Longitude: -74.0060}},
		{ID: "london", Coord: Coordinates{Latitude: 51.5074, Longitude: -0.1278}},
		{ID: "paris", Coord: Coordinates{Latitude: 48.8566, Longitude: 2.3522}},
		{ID: "tokyo", Coord: Coordinates{Latitude: 35.6762, Longitude: 139.6503}},
	}

	for _, p := range pts {
		ok, err := qt.Insert(p)
		if err != nil || !ok {
			t.Fatalf("failed to insert %s: %v", p.ID, err)
		}
	}

	if qt.Count() != 6 {
		t.Errorf("expected count 6, got %d", qt.Count())
	}

	// Range query US West Coast
	westCoast := GeoRect{MinLat: 30.0, MaxLat: 45.0, MinLon: -125.0, MaxLon: -115.0}
	results := qt.QueryRange(westCoast)
	if len(results) != 2 {
		t.Errorf("expected 2 points in US West Coast, got %d", len(results))
	}
}

func TestSpatialQuadtree_QueryRadius(t *testing.T) {
	qt := NewSpatialQuadtree(GlobalEarthRect(), 4, 8)

	sf := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	oakland := Coordinates{Latitude: 37.8044, Longitude: -122.2712} // ~13 km from SF
	sanjose := Coordinates{Latitude: 37.3382, Longitude: -121.8863} // ~67 km from SF
	tokyo := Coordinates{Latitude: 35.6762, Longitude: 139.6503}   // ~8200 km

	_, _ = qt.Insert(SpatialPoint{ID: "sf", Coord: sf})
	_, _ = qt.Insert(SpatialPoint{ID: "oakland", Coord: oakland})
	_, _ = qt.Insert(SpatialPoint{ID: "sanjose", Coord: sanjose})
	_, _ = qt.Insert(SpatialPoint{ID: "tokyo", Coord: tokyo})

	// Search 25 km around SF (should find SF + Oakland, not San Jose or Tokyo)
	matches, err := qt.QueryRadius(sf, 25.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matches) != 2 {
		t.Errorf("expected 2 matches within 25 km, got %d", len(matches))
	}

	// Search 100 km around SF (should find SF, Oakland, San Jose)
	matches100, err := qt.QueryRadius(sf, 100.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches100) != 3 {
		t.Errorf("expected 3 matches within 100 km, got %d", len(matches100))
	}
}

func TestSpatialQuadtree_NearestNeighbor(t *testing.T) {
	qt := NewSpatialQuadtree(GlobalEarthRect(), 4, 8)

	_, _, err := qt.NearestNeighbor(Coordinates{Latitude: 0, Longitude: 0})
	if err != ErrEmptyQuadtree {
		t.Errorf("expected ErrEmptyQuadtree, got %v", err)
	}

	sf := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	oakland := Coordinates{Latitude: 37.8044, Longitude: -122.2712}
	nyc := Coordinates{Latitude: 40.7128, Longitude: -74.0060}

	_, _ = qt.Insert(SpatialPoint{ID: "sf", Coord: sf})
	_, _ = qt.Insert(SpatialPoint{ID: "oakland", Coord: oakland})
	_, _ = qt.Insert(SpatialPoint{ID: "nyc", Coord: nyc})

	berkeley := Coordinates{Latitude: 37.8715, Longitude: -122.2730}
	nearest, distKm, err := qt.NearestNeighbor(berkeley)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if nearest.ID != "oakland" {
		t.Errorf("expected nearest to be oakland, got %s", nearest.ID)
	}
	if distKm > 15.0 {
		t.Errorf("expected distance < 15km, got %f", distKm)
	}
}

func TestSpatialQuadtree_Concurrency(t *testing.T) {
	qt := NewSpatialQuadtree(GlobalEarthRect(), 4, 8)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				lat := float64(workerID)
				lon := float64(j)
				_, _ = qt.Insert(SpatialPoint{
					ID:    "point",
					Coord: Coordinates{Latitude: lat, Longitude: lon},
				})
			}
		}(i)
	}

	wg.Wait()

	if qt.Count() != 200 {
		t.Errorf("expected 200 points, got %d", qt.Count())
	}
}
