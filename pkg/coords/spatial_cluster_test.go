package coords

import (
	"testing"
)

func TestDBSCAN_Clustering(t *testing.T) {
	// Cluster 1: New York Midtown points (~50-100m apart)
	c1p1 := Coordinates{Latitude: 40.758896, Longitude: -73.985130} // Times Square
	c1p2 := Coordinates{Latitude: 40.759200, Longitude: -73.984800}
	c1p3 := Coordinates{Latitude: 40.758500, Longitude: -73.985500}

	// Cluster 2: Brooklyn points
	c2p1 := Coordinates{Latitude: 40.678178, Longitude: -73.944158}
	c2p2 := Coordinates{Latitude: 40.678500, Longitude: -73.944500}
	c2p3 := Coordinates{Latitude: 40.677900, Longitude: -73.943800}

	// Outlier / Noise: Paris, France
	noisePt := Coordinates{Latitude: 48.8566, Longitude: 2.3522}

	points := []Coordinates{c1p1, c1p2, c1p3, c2p1, c2p2, c2p3, noisePt}

	// Radius: 300 meters, MinPts: 3
	result, err := DBSCAN(points, 300.0, 3)
	if err != nil {
		t.Fatalf("unexpected DBSCAN error: %v", err)
	}

	if len(result.Clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(result.Clusters))
	}
	if result.TotalNoise != 1 {
		t.Fatalf("expected 1 noise point, got %d", result.TotalNoise)
	}

	// Check noise point matches Paris
	if result.Noise[0].Latitude != noisePt.Latitude {
		t.Errorf("expected noise point to be Paris, got %+v", result.Noise[0])
	}

	// Check cluster centroids exist and radius is positive
	for _, cl := range result.Clusters {
		if len(cl.Points) != 3 {
			t.Errorf("expected 3 points in cluster, got %d", len(cl.Points))
		}
		if cl.RadiusM <= 0 {
			t.Errorf("expected positive cluster radius, got %f", cl.RadiusM)
		}
		if !cl.Centroid.IsValid() {
			t.Errorf("expected valid centroid, got %+v", cl.Centroid)
		}
	}
}

func TestDBSCAN_EdgeCasesAndValidation(t *testing.T) {
	// Empty points
	_, err := DBSCAN([]Coordinates{}, 100, 2)
	if err != ErrEmptyPointSet {
		t.Errorf("expected ErrEmptyPointSet, got %v", err)
	}

	// Invalid params
	_, err = DBSCAN([]Coordinates{{Latitude: 0, Longitude: 0}}, -10, 2)
	if err != ErrInvalidDBSCANParams {
		t.Errorf("expected ErrInvalidDBSCANParams for negative epsilon, got %v", err)
	}

	_, err = DBSCAN([]Coordinates{{Latitude: 0, Longitude: 0}}, 100, 0)
	if err != ErrInvalidDBSCANParams {
		t.Errorf("expected ErrInvalidDBSCANParams for minPts=0, got %v", err)
	}

	// Invalid coordinate in set
	_, err = DBSCAN([]Coordinates{{Latitude: 95.0, Longitude: 0}}, 100, 1)
	if err != ErrInvalidCoordinates {
		t.Errorf("expected ErrInvalidCoordinates, got %v", err)
	}
}
