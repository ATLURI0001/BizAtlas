package coords

import (
	"encoding/json"
	"math"
	"testing"
)

func TestPointToFeature(t *testing.T) {
	c := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	props := map[string]interface{}{"name": "San Francisco"}

	feat, err := PointToFeature(c, props)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if feat.Type != TypeFeature {
		t.Errorf("expected type Feature, got %s", feat.Type)
	}
	if feat.Geometry.Type != TypePoint {
		t.Errorf("expected geometry type Point, got %s", feat.Geometry.Type)
	}

	var coords [2]float64
	if err := json.Unmarshal(feat.Geometry.Coordinates, &coords); err != nil {
		t.Fatalf("failed unmarshaling coords: %v", err)
	}

	if coords[0] != c.Longitude || coords[1] != c.Latitude {
		t.Errorf("expected [lon, lat] [%f, %f], got [%f, %f]", c.Longitude, c.Latitude, coords[0], coords[1])
	}

	invalid := Coordinates{Latitude: 100.0, Longitude: 0.0}
	if _, err := PointToFeature(invalid, nil); err != ErrInvalidCoordinates {
		t.Errorf("expected ErrInvalidCoordinates, got %v", err)
	}
}

func TestLineStringToFeature(t *testing.T) {
	pts := []Coordinates{
		{Latitude: 10.0, Longitude: 20.0},
		{Latitude: 15.0, Longitude: 25.0},
		{Latitude: 12.0, Longitude: 30.0},
	}

	feat, err := LineStringToFeature(pts, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if feat.Geometry.Type != TypeLineString {
		t.Errorf("expected LineString, got %s", feat.Geometry.Type)
	}
	if len(feat.BBox) != 4 {
		t.Fatalf("expected bbox of length 4, got %v", feat.BBox)
	}
	// bbox is [minLon, minLat, maxLon, maxLat] -> [20.0, 10.0, 30.0, 15.0]
	if feat.BBox[0] != 20.0 || feat.BBox[1] != 10.0 || feat.BBox[2] != 30.0 || feat.BBox[3] != 15.0 {
		t.Errorf("unexpected bbox values: %v", feat.BBox)
	}

	// Less than 2 points
	if _, err := LineStringToFeature([]Coordinates{{Latitude: 10, Longitude: 10}}, nil); err == nil {
		t.Errorf("expected error for single point linestring")
	}
}

func TestPolygonToFeature(t *testing.T) {
	verts := []Coordinates{
		{Latitude: 0, Longitude: 0},
		{Latitude: 0, Longitude: 10},
		{Latitude: 10, Longitude: 10},
		{Latitude: 10, Longitude: 0},
	}
	poly, err := NewPolygon(verts)
	if err != nil {
		t.Fatalf("failed creating polygon: %v", err)
	}

	feat, err := PolygonToFeature(poly, map[string]interface{}{"zone": "alpha"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if feat.Geometry.Type != TypePolygon {
		t.Errorf("expected Polygon, got %s", feat.Geometry.Type)
	}

	var rings [][][2]float64
	if err := json.Unmarshal(feat.Geometry.Coordinates, &rings); err != nil {
		t.Fatalf("failed unmarshaling polygon coords: %v", err)
	}

	if len(rings) != 1 {
		t.Fatalf("expected 1 linear ring, got %d", len(rings))
	}
	// Ring must be closed (5 coordinates)
	if len(rings[0]) != 5 {
		t.Fatalf("expected 5 closed vertices, got %d", len(rings[0]))
	}
	if rings[0][0] != rings[0][4] {
		t.Errorf("expected first and last ring coordinates to be equal")
	}
}

func TestEncodeFeatureCollection(t *testing.T) {
	p1, _ := PointToFeature(Coordinates{Latitude: 10, Longitude: 10}, nil)
	p2, _ := PointToFeature(Coordinates{Latitude: 20, Longitude: 30}, nil)

	fc, err := EncodeFeatureCollection([]GeoJSONFeature{*p1, *p2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fc.Type != TypeFeatureCollection {
		t.Errorf("expected FeatureCollection, got %s", fc.Type)
	}
	if len(fc.Features) != 2 {
		t.Errorf("expected 2 features, got %d", len(fc.Features))
	}
	// bbox [10, 10, 30, 20]
	if fc.BBox[0] != 10 || fc.BBox[1] != 10 || fc.BBox[2] != 30 || fc.BBox[3] != 20 {
		t.Errorf("unexpected consolidated bbox: %v", fc.BBox)
	}
}

func TestDecodeFeaturePoints(t *testing.T) {
	geojsonRaw := `{
		"type": "FeatureCollection",
		"features": [
			{
				"type": "Feature",
				"geometry": {
					"type": "Point",
					"coordinates": [12.5, 45.2]
				}
			},
			{
				"type": "Feature",
				"geometry": {
					"type": "LineString",
					"coordinates": [
						[10.0, 20.0],
						[11.0, 21.0]
					]
				}
			}
		]
	}`

	pts, err := DecodeFeaturePoints([]byte(geojsonRaw))
	if err != nil {
		t.Fatalf("unexpected error decoding points: %v", err)
	}

	if len(pts) != 3 {
		t.Fatalf("expected 3 points, got %d", len(pts))
	}

	if pts[0].Latitude != 45.2 || pts[0].Longitude != 12.5 {
		t.Errorf("mismatch on point 0: %+v", pts[0])
	}
}

func TestEncodeDecodePolyline(t *testing.T) {
	original := []Coordinates{
		{Latitude: 38.5, Longitude: -120.2},
		{Latitude: 40.7, Longitude: -120.95},
		{Latitude: 43.252, Longitude: -126.453},
	}

	encoded, err := EncodePolyline(original)
	if err != nil {
		t.Fatalf("failed encoding polyline: %v", err)
	}

	if encoded == "" {
		t.Fatalf("encoded polyline should not be empty")
	}

	decoded, err := DecodePolyline(encoded)
	if err != nil {
		t.Fatalf("failed decoding polyline: %v", err)
	}

	if len(decoded) != len(original) {
		t.Fatalf("expected %d points, got %d", len(original), len(decoded))
	}

	for i := range original {
		if math.Abs(decoded[i].Latitude-original[i].Latitude) > 1e-4 {
			t.Errorf("point %d lat mismatch: %f vs %f", i, decoded[i].Latitude, original[i].Latitude)
		}
		if math.Abs(decoded[i].Longitude-original[i].Longitude) > 1e-4 {
			t.Errorf("point %d lon mismatch: %f vs %f", i, decoded[i].Longitude, original[i].Longitude)
		}
	}
}

func TestPolylineErrors(t *testing.T) {
	if _, err := EncodePolyline(nil); err != ErrEmptyCoordinates {
		t.Errorf("expected ErrEmptyCoordinates, got %v", err)
	}

	if _, err := DecodePolyline(""); err != ErrInvalidPolyline {
		t.Errorf("expected ErrInvalidPolyline, got %v", err)
	}

	// Truncated/corrupted polyline
	if _, err := DecodePolyline("~invalid!char"); err != ErrInvalidPolyline {
		t.Errorf("expected ErrInvalidPolyline for corrupted input, got %v", err)
	}
}

func TestNormalizeCoordinates(t *testing.T) {
	// Latitude clamping
	c1 := NormalizeCoordinates(95.0, 50.0)
	if c1.Latitude != 90.0 {
		t.Errorf("expected latitude clamped to 90.0, got %f", c1.Latitude)
	}

	c2 := NormalizeCoordinates(-105.0, 50.0)
	if c2.Latitude != -90.0 {
		t.Errorf("expected latitude clamped to -90.0, got %f", c2.Latitude)
	}

	// Longitude wrapping
	c3 := NormalizeCoordinates(10.0, 190.0)
	if c3.Longitude != -170.0 {
		t.Errorf("expected longitude wrapped to -170.0, got %f", c3.Longitude)
	}

	c4 := NormalizeCoordinates(10.0, -200.0)
	if c4.Longitude != 160.0 {
		t.Errorf("expected longitude wrapped to 160.0, got %f", c4.Longitude)
	}
}
