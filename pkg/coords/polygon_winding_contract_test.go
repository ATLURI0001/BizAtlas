package coords

import (
	"errors"
	"testing"
)

func TestValidateAndSanitizePolygon_ValidCCW(t *testing.T) {
	// Counterclockwise square around null island [0,0]
	// [0,0] -> [1,0] -> [1,1] -> [0,1] -> [0,0]
	exterior := [][]float64{
		{0.0, 0.0},
		{1.0, 0.0},
		{1.0, 1.0},
		{0.0, 1.0},
		{0.0, 0.0},
	}

	spec, err := ValidateAndSanitizePolygon(exterior, nil, false)
	if err != nil {
		t.Fatalf("expected valid polygon, got %v", err)
	}

	if spec.Version != "geojson-polygon/v2" {
		t.Fatalf("expected version v2, got %s", spec.Version)
	}
	if spec.BBox.MinLon != 0.0 || spec.BBox.MaxLon != 1.0 {
		t.Fatalf("unexpected lon bounds: [%f, %f]", spec.BBox.MinLon, spec.BBox.MaxLon)
	}
	if spec.AreaSqDeg != 1.0 {
		t.Fatalf("expected area 1.0, got %f", spec.AreaSqDeg)
	}
}

func TestValidateAndSanitizePolygon_AutoFixClockwiseExterior(t *testing.T) {
	// Clockwise square
	// [0,0] -> [0,1] -> [1,1] -> [1,0] -> [0,0]
	clockwiseExterior := [][]float64{
		{0.0, 0.0},
		{0.0, 1.0},
		{1.0, 1.0},
		{1.0, 0.0},
		{0.0, 0.0},
	}

	// In strict mode (autoFix=false), must error
	_, err := ValidateAndSanitizePolygon(clockwiseExterior, nil, false)
	if !errors.Is(err, ErrInvalidWindingOrder) {
		t.Fatalf("expected ErrInvalidWindingOrder in strict mode, got %v", err)
	}

	// With autoFix=true, should normalize to CCW
	spec, err := ValidateAndSanitizePolygon(clockwiseExterior, nil, true)
	if err != nil {
		t.Fatalf("expected auto-fix success, got %v", err)
	}

	if !IsPolygonCCW(spec.Exterior) {
		t.Fatalf("expected exterior to be corrected to CCW")
	}
}

func TestValidateAndSanitizePolygon_UnclosedAndBounds(t *testing.T) {
	// Unclosed ring
	unclosed := [][]float64{
		{0.0, 0.0},
		{1.0, 0.0},
		{1.0, 1.0},
		{0.0, 1.0},
	}
	_, err := ValidateAndSanitizePolygon(unclosed, nil, false)
	if !errors.Is(err, ErrPolygonUnclosed) {
		t.Fatalf("expected ErrPolygonUnclosed, got %v", err)
	}

	// Out of bounds coordinates (lon 200)
	outOfBounds := [][]float64{
		{0.0, 0.0},
		{200.0, 0.0},
		{200.0, 1.0},
		{0.0, 0.0},
	}
	_, err = ValidateAndSanitizePolygon(outOfBounds, nil, false)
	if !errors.Is(err, ErrCoordinatesOutOfBounds) {
		t.Fatalf("expected ErrCoordinatesOutOfBounds, got %v", err)
	}
}
