package coords

import (
	"errors"
	"math"
	"sync"
	"testing"
)

func TestEncodeDecodeGeohash_Roundtrip(t *testing.T) {
	lat := 37.774929
	lon := -122.419416

	hash, err := EncodeGeohash(lat, lon, 7)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}

	if len(hash) != 7 {
		t.Fatalf("expected geohash length 7, got %d (%s)", len(hash), hash)
	}

	decLat, decLon, bbox, err := DecodeGeohash(hash)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	// Precision 7 error tolerance ~150 meters
	if math.Abs(decLat-lat) > 0.005 {
		t.Errorf("decoded lat %f drifted from %f", decLat, lat)
	}
	if math.Abs(decLon-lon) > 0.005 {
		t.Errorf("decoded lon %f drifted from %f", decLon, lon)
	}

	if lat < bbox.MinLat || lat > bbox.MaxLat || lon < bbox.MinLon || lon > bbox.MaxLon {
		t.Errorf("original coordinate not contained inside decoded bounding box: %v", bbox)
	}
}

func TestEncodeGeohash_ValidationErrors(t *testing.T) {
	_, err := EncodeGeohash(95.0, 0.0, 6)
	if !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
	}

	_, err = EncodeGeohash(0.0, 190.0, 6)
	if !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
	}

	_, err = EncodeGeohash(0.0, 0.0, 0)
	if !errors.Is(err, ErrInvalidPrecision) {
		t.Fatalf("expected ErrInvalidPrecision, got %v", err)
	}
}

func TestGetGeohashNeighbors(t *testing.T) {
	center := "9q8yy"
	neighbors, err := GetGeohashNeighbors(center)
	if err != nil {
		t.Fatalf("unexpected error getting neighbors: %v", err)
	}

	if len(neighbors) != 8 {
		t.Fatalf("expected exactly 8 neighbors, got %d (%v)", len(neighbors), neighbors)
	}

	// Verify all neighbors have same precision
	for _, n := range neighbors {
		if len(n) != len(center) {
			t.Errorf("neighbor %s precision mismatch with center %s", n, center)
		}
	}
}

func TestGeohashSpatialIndex_SearchRadius(t *testing.T) {
	idx := NewGeohashSpatialIndex(6)

	// Points around Downtown SF (37.7749, -122.4194)
	sfCenter := GeoPoint{ID: "sf-center", Lat: 37.7749, Lon: -122.4194}
	sfNearby := GeoPoint{ID: "sf-nearby", Lat: 37.7790, Lon: -122.4180} // ~0.5 km away
	oakland := GeoPoint{ID: "oakland", Lat: 37.8044, Lon: -122.2711}    // ~13 km away

	if err := idx.Insert(sfCenter); err != nil {
		t.Fatal(err)
	}
	if err := idx.Insert(sfNearby); err != nil {
		t.Fatal(err)
	}
	if err := idx.Insert(oakland); err != nil {
		t.Fatal(err)
	}

	// Search 2.0 km radius around SF
	results, err := idx.SearchRadius(37.7750, -122.4190, 2.0)
	if err != nil {
		t.Fatalf("search radius error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 nearby points within 2km, got %d", len(results))
	}

	foundMap := map[string]bool{}
	for _, r := range results {
		foundMap[r.ID] = true
	}
	if !foundMap["sf-center"] || !foundMap["sf-nearby"] {
		t.Errorf("missing expected points in result set: %v", foundMap)
	}
	if foundMap["oakland"] {
		t.Errorf("oakland unexpectedly returned inside 2km search radius")
	}
}

func TestGeohashSpatialIndex_Concurrency(t *testing.T) {
	idx := NewGeohashSpatialIndex(6)
	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				pt := GeoPoint{
					ID:  string(rune('A' + workerID)) + string(rune('0' + j)),
					Lat: 37.0 + float64(workerID)*0.01,
					Lon: -122.0 + float64(j)*0.01,
				}
				_ = idx.Insert(pt)
				_, _ = idx.SearchRadius(37.05, -122.05, 5.0)
			}
		}(i)
	}

	wg.Wait()
}
