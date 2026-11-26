package coords

import (
	"errors"
	"testing"
)

func TestValidateAndSanitizeCluster_ValidHaversine(t *testing.T) {
	points := []Coordinates{
		{Latitude: 37.7749, Longitude: -122.4194}, // San Francisco
		{Latitude: 37.7752, Longitude: -122.4180},
		{Latitude: 37.7740, Longitude: -122.4200},
	}

	spec, err := ValidateAndSanitizeCluster("cluster_sf_downtown", ClusterMetricHaversine, 5.0, points)
	if err != nil {
		t.Fatalf("unexpected error validating spatial cluster: %v", err)
	}

	if spec.Version != "bizatlas/cluster-v2" {
		t.Errorf("expected version bizatlas/cluster-v2, got %s", spec.Version)
	}
	if spec.PointCount != 3 {
		t.Errorf("expected 3 points, got %d", spec.PointCount)
	}
	if spec.Centroid.Latitude < 37.77 || spec.Centroid.Latitude > 37.78 {
		t.Errorf("unexpected centroid latitude: %f", spec.Centroid.Latitude)
	}
	if spec.BoundingBox.MinLat > spec.BoundingBox.MaxLat {
		t.Errorf("invalid bbox latitude range: [%f, %f]", spec.BoundingBox.MinLat, spec.BoundingBox.MaxLat)
	}
}

func TestValidateAndSanitizeCluster_InvariantViolations(t *testing.T) {
	validPoints := []Coordinates{{Latitude: 10.0, Longitude: 20.0}}

	// Empty points
	_, errEmpty := ValidateAndSanitizeCluster("c1", ClusterMetricHaversine, 1.0, nil)
	if !errors.Is(errEmpty, ErrEmptyClusterPoints) {
		t.Errorf("expected ErrEmptyClusterPoints, got %v", errEmpty)
	}

	// Invalid metric
	_, errMetric := ValidateAndSanitizeCluster("c1", "CHEBYSHEV", 1.0, validPoints)
	if !errors.Is(errMetric, ErrInvalidMetricType) {
		t.Errorf("expected ErrInvalidMetricType, got %v", errMetric)
	}

	// Radius out of bounds
	_, errRadius := ValidateAndSanitizeCluster("c1", ClusterMetricHaversine, 5000.0, validPoints)
	if !errors.Is(errRadius, ErrClusterRadiusOutOfRange) {
		t.Errorf("expected ErrClusterRadiusOutOfRange, got %v", errRadius)
	}

	// Coordinate out of bounds
	badPoints := []Coordinates{{Latitude: 95.0, Longitude: 20.0}}
	_, errCoords := ValidateAndSanitizeCluster("c1", ClusterMetricHaversine, 1.0, badPoints)
	if !errors.Is(errCoords, ErrCoordinateBoundaryBreach) {
		t.Errorf("expected ErrCoordinateBoundaryBreach, got %v", errCoords)
	}
}

func TestMigrateClusterSpecV1ToV2(t *testing.T) {
	points := []Coordinates{
		{Latitude: 40.7128, Longitude: -74.0060}, // New York
		{Latitude: 40.7135, Longitude: -74.0050},
	}

	spec, err := MigrateClusterSpecV1ToV2("legacy_ny_cluster", 500.0, points)
	if err != nil {
		t.Fatalf("unexpected error migrating v1 cluster spec: %v", err)
	}

	if spec.MaxRadiusKm != 0.5 {
		t.Errorf("expected 500m to convert to 0.5km, got %f", spec.MaxRadiusKm)
	}
	if spec.Metric != ClusterMetricHaversine {
		t.Errorf("expected default Haversine metric, got %s", spec.Metric)
	}
}
