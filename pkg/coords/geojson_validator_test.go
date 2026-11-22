package coords

import (
	"encoding/json"
	"testing"
)

func TestValidateLinearRing_ValidRing(t *testing.T) {
	ring := []Coordinates{
		{Latitude: 37.7749, Longitude: -122.4194},
		{Latitude: 37.7750, Longitude: -122.4180},
		{Latitude: 37.7740, Longitude: -122.4180},
		{Latitude: 37.7749, Longitude: -122.4194}, // closed
	}

	ok, err := ValidateLinearRing(ring)
	if !ok || err != nil {
		t.Fatalf("expected valid ring, got err: %v", err)
	}
}

func TestValidateLinearRing_Errors(t *testing.T) {
	// 1. Insufficient vertices (<4)
	tooFew := []Coordinates{
		{Latitude: 10, Longitude: 10},
		{Latitude: 10, Longitude: 11},
		{Latitude: 10, Longitude: 10},
	}
	ok, err := ValidateLinearRing(tooFew)
	if ok || err == nil {
		t.Fatal("expected error on <4 vertices")
	}

	// 2. Unclosed ring (first != last)
	unclosed := []Coordinates{
		{Latitude: 10, Longitude: 10},
		{Latitude: 10, Longitude: 11},
		{Latitude: 11, Longitude: 11},
		{Latitude: 11, Longitude: 10},
	}
	ok, err = ValidateLinearRing(unclosed)
	if ok || err == nil {
		t.Fatal("expected error on unclosed ring")
	}

	// 3. Out-of-bounds coordinates
	outOfBounds := []Coordinates{
		{Latitude: 95.0, Longitude: 10},
		{Latitude: 10, Longitude: 11},
		{Latitude: 11, Longitude: 11},
		{Latitude: 95.0, Longitude: 10},
	}
	ok, err = ValidateLinearRing(outOfBounds)
	if ok || err == nil {
		t.Fatal("expected error on lat > 90")
	}
}

func TestEnforceRingClosure_AndWinding(t *testing.T) {
	openRing := []Coordinates{
		{Latitude: 0, Longitude: 0},
		{Latitude: 0, Longitude: 1},
		{Latitude: 1, Longitude: 1},
	}

	closed := EnforceRingClosure(openRing)
	if len(closed) != 4 {
		t.Fatalf("expected 4 vertices after closure, got %d", len(closed))
	}
	if closed[0] != closed[3] {
		t.Errorf("expected closed ring, got %v != %v", closed[0], closed[3])
	}
}

func TestValidateMultiPolygonGeometry_ValidAndBBox(t *testing.T) {
	poly1 := [][]Coordinates{
		{
			{Latitude: 10.0, Longitude: 20.0},
			{Latitude: 10.0, Longitude: 25.0},
			{Latitude: 15.0, Longitude: 25.0},
			{Latitude: 10.0, Longitude: 20.0},
		},
	}

	poly2 := [][]Coordinates{
		{
			{Latitude: 30.0, Longitude: 40.0},
			{Latitude: 30.0, Longitude: 45.0},
			{Latitude: 35.0, Longitude: 45.0},
			{Latitude: 30.0, Longitude: 40.0},
		},
	}

	multiPoly := [][][]Coordinates{poly1, poly2}
	res := ValidateMultiPolygonGeometry(multiPoly)

	if !res.Valid {
		t.Fatalf("expected valid MultiPolygon, got errors: %v", res.Errors)
	}
	if res.RingCount != 2 {
		t.Errorf("expected 2 rings, got %d", res.RingCount)
	}

	// Bounding box [minLon, minLat, maxLon, maxLat]
	expectedBBox := []float64{20.0, 10.0, 45.0, 35.0}
	if len(res.BoundingBox) != 4 {
		t.Fatalf("expected 4-element bbox, got %v", res.BoundingBox)
	}
	for i := 0; i < 4; i++ {
		if res.BoundingBox[i] != expectedBBox[i] {
			t.Errorf("bbox[%d]: expected %f, got %f", i, expectedBBox[i], res.BoundingBox[i])
		}
	}
}

func TestEncodeMultiPolygonGeoJSON(t *testing.T) {
	poly := [][]Coordinates{
		{
			{Latitude: 37.77, Longitude: -122.42},
			{Latitude: 37.77, Longitude: -122.40},
			{Latitude: 37.79, Longitude: -122.40},
			{Latitude: 37.77, Longitude: -122.42},
		},
	}

	multi := [][][]Coordinates{poly}
	props := map[string]interface{}{"zone": "downtown", "id": 101}

	bytes, err := EncodeMultiPolygonGeoJSON(multi, props)
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	if parsed["type"] != "Feature" {
		t.Errorf("expected Feature type, got %v", parsed["type"])
	}

	geom := parsed["geometry"].(map[string]interface{})
	if geom["type"] != "MultiPolygon" {
		t.Errorf("expected MultiPolygon geometry, got %v", geom["type"])
	}
}
