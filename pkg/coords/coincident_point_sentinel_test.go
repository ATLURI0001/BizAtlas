package coords

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestCoincidentPointSentinel_DistinctPoints(t *testing.T) {
	cfg := DefaultCoincidentPointConfig()
	sentinel := NewCoincidentPointSentinel(cfg)

	points := []SpatialPoint{
		{ID: "p1", Coord: Coordinates{Latitude: 37.7749, Longitude: -122.4194}},
		{ID: "p2", Coord: Coordinates{Latitude: 37.7750, Longitude: -122.4194}},
		{ID: "p3", Coord: Coordinates{Latitude: 40.7128, Longitude: -74.0060}},
	}

	report := sentinel.AnalyzePoints(points)
	if report.TotalPoints != 3 || report.UniqueCoordinates != 3 || report.CoincidentPoints != 0 {
		t.Fatalf("unexpected report for distinct points: %+v", report)
	}
	if report.SingularityDetected {
		t.Fatalf("unexpected singularity detected")
	}
}

func TestCoincidentPointSentinel_CoincidentDetection(t *testing.T) {
	cfg := DefaultCoincidentPointConfig()
	cfg.MaxClusterDensity = 5
	sentinel := NewCoincidentPointSentinel(cfg)

	// 4 points sharing the exact same coordinate
	baseCoord := Coordinates{Latitude: 37.774929, Longitude: -122.419416}
	points := make([]SpatialPoint, 4)
	for i := 0; i < 4; i++ {
		points[i] = SpatialPoint{
			ID:    fmt.Sprintf("bldg-%d", i),
			Coord: baseCoord,
		}
	}

	report := sentinel.AnalyzePoints(points)
	if report.TotalPoints != 4 {
		t.Fatalf("expected 4 total points, got %d", report.TotalPoints)
	}
	if report.UniqueCoordinates != 1 {
		t.Fatalf("expected 1 unique coordinate cluster, got %d", report.UniqueCoordinates)
	}
	if report.MaxClusterSize != 4 {
		t.Fatalf("expected max cluster size 4, got %d", report.MaxClusterSize)
	}
	if report.SingularityDetected {
		t.Fatalf("density 4 is <= max 5, should not be singularity")
	}
}

func TestCoincidentPointSentinel_SingularityThreshold(t *testing.T) {
	cfg := DefaultCoincidentPointConfig()
	cfg.MaxClusterDensity = 3
	cfg.EnableJitterDeduplication = false
	sentinel := NewCoincidentPointSentinel(cfg)

	// 5 points sharing coordinate exceeding density 3
	baseCoord := Coordinates{Latitude: 34.0522, Longitude: -118.2437}
	points := make([]SpatialPoint, 5)
	for i := 0; i < 5; i++ {
		points[i] = SpatialPoint{
			ID:    fmt.Sprintf("pt-%d", i),
			Coord: baseCoord,
		}
	}

	_, report, err := sentinel.SanitizeForTreeInsertion(points)
	if !errors.Is(err, ErrSingularityThresholdExceeded) {
		t.Fatalf("expected ErrSingularityThresholdExceeded, got %v", err)
	}
	if !report.SingularityDetected {
		t.Fatalf("expected singularityDetected to be true")
	}
}

func TestCoincidentPointSentinel_JitterSanitizationAndQuadtree(t *testing.T) {
	cfg := DefaultCoincidentPointConfig()
	cfg.MaxClusterDensity = 5
	cfg.EnableJitterDeduplication = true
	cfg.JitterScale = 1e-6
	sentinel := NewCoincidentPointSentinel(cfg)

	// 20 coincident points in a shopping mall
	baseCoord := Coordinates{Latitude: 51.5074, Longitude: -0.1278}
	points := make([]SpatialPoint, 20)
	for i := 0; i < 20; i++ {
		points[i] = SpatialPoint{
			ID:    fmt.Sprintf("mall-store-%d", i),
			Coord: baseCoord,
		}
	}

	sanitized, report, err := sentinel.SanitizeForTreeInsertion(points)
	if err != nil {
		t.Fatalf("unexpected error with jitter enabled: %v", err)
	}
	if len(sanitized) != 20 {
		t.Fatalf("expected 20 sanitized points, got %d", len(sanitized))
	}
	if report.CoincidentPoints != 20 {
		t.Fatalf("expected 20 coincident points in report, got %d", report.CoincidentPoints)
	}

	// Verify points can now be safely inserted into a Quadtree with small capacity
	qt := NewSpatialQuadtree(GlobalEarthRect(), 2, 10)
	for _, p := range sanitized {
		ok, err := qt.Insert(p)
		if !ok || err != nil {
			t.Fatalf("failed to insert sanitized point %s into quadtree: %v", p.ID, err)
		}
	}
	if qt.Count() != 20 {
		t.Fatalf("expected quadtree count 20, got %d", qt.Count())
	}
}

func TestCoincidentPointSentinel_Concurrency(t *testing.T) {
	cfg := DefaultCoincidentPointConfig()
	sentinel := NewCoincidentPointSentinel(cfg)

	var wg sync.WaitGroup
	workers := 8
	iterations := 50

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				pts := []SpatialPoint{
					{ID: "c1", Coord: Coordinates{Latitude: 10.0, Longitude: 20.0}},
					{ID: "c2", Coord: Coordinates{Latitude: 10.0, Longitude: 20.0}},
					{ID: "c3", Coord: Coordinates{Latitude: 10.0001, Longitude: 20.0001}},
				}
				_ = sentinel.AnalyzePoints(pts)
			}
		}(w)
	}

	wg.Wait()
}
