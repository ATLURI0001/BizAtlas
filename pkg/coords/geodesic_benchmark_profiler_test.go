package coords

import (
	"strings"
	"testing"
)

func TestGeodesicBenchmarkProfiler_AlgorithmsAccuracy(t *testing.T) {
	// SF to Oakland (~13 km)
	sf := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	oakland := Coordinates{Latitude: 37.8044, Longitude: -122.2711}

	hav, err := HaversineDistance(sf, oakland)
	if err != nil {
		t.Fatalf("Haversine failed: %v", err)
	}

	cos, err := LawOfCosinesDistance(sf, oakland)
	if err != nil {
		t.Fatalf("LawOfCosines failed: %v", err)
	}

	equi, err := EquirectangularDistance(sf, oakland)
	if err != nil {
		t.Fatalf("Equirectangular failed: %v", err)
	}

	// All should be ~13.5 - 14.5 km
	if hav < 13.0 || hav > 15.0 {
		t.Errorf("unexpected Haversine distance: %.2f km", hav)
	}
	// Spherical Law of Cosines should match Haversine within 0.001 km
	if diff := hav - cos; diff < -0.01 || diff > 0.01 {
		t.Errorf("Law of Cosines discrepancy too high: %.4f km", diff)
	}
	// Equirectangular local approx should match within 0.1 km for short distance
	if diff := hav - equi; diff < -0.2 || diff > 0.2 {
		t.Errorf("Equirectangular discrepancy too high for local distance: %.4f km", diff)
	}
}

func TestGeodesicBenchmarkProfiler_SyntheticGeneration(t *testing.T) {
	profiler := NewGeodesicBenchmarkProfiler()
	center := Coordinates{Latitude: 40.7128, Longitude: -74.0060} // NYC
	points := profiler.GenerateSyntheticPoints(50, center, 10.0)

	if len(points) != 50 {
		t.Fatalf("expected 50 points, got %d", len(points))
	}
	for i, pt := range points {
		if !pt.IsValid() {
			t.Fatalf("point %d is invalid: %+v", i, pt)
		}
	}
}

func TestGeodesicBenchmarkProfiler_ComparativeSuite(t *testing.T) {
	profiler := NewGeodesicBenchmarkProfiler()
	results, report := profiler.RunComparativeSuite(30, 20.0) // 30*29/2 = 435 pairs

	if len(results) != 3 {
		t.Fatalf("expected 3 benchmark results, got %d", len(results))
	}

	for _, r := range results {
		if r.PairCount != 435 {
			t.Errorf("expected 435 pairs, got %d", r.PairCount)
		}
		if r.OpsPerSec <= 0 {
			t.Errorf("expected positive OpsPerSec, got %.2f", r.OpsPerSec)
		}
		if r.MeanDurationNs <= 0 {
			t.Errorf("expected positive MeanDurationNs")
		}
	}

	if !strings.Contains(report, "# Geodesic Distance Matrix Benchmark & Spatial Profile") {
		t.Errorf("missing title in markdown report")
	}
	if !strings.Contains(report, "| **Haversine** |") {
		t.Errorf("missing Haversine in table")
	}
	if !strings.Contains(report, "| **EquirectangularApprox** |") {
		t.Errorf("missing EquirectangularApprox in table")
	}
}
