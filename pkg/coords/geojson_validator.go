package coords

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

var (
	ErrEmptyGeometry         = errors.New("geojson: geometry coordinate array cannot be empty")
	ErrUnclosedLinearRing    = errors.New("geojson: linear ring is not closed (first != last)")
	ErrInsufficientVertices  = errors.New("geojson: linear ring must contain at least 4 positions")
	ErrInvalidCoordinateBounds = errors.New("geojson: position out of longitude/latitude range")
	ErrNonFiniteCoordinate   = errors.New("geojson: coordinate contains NaN or Infinity")
)

// GeoJSONPosition represents [longitude, latitude] or [longitude, latitude, elevation].
type GeoJSONPosition []float64

// GeoJSONValidationResult reports validity and specific compliance errors.
type GeoJSONValidationResult struct {
	Valid          bool     `json:"valid"`
	Errors         []string `json:"errors,omitempty"`
	RingCount      int      `json:"ring_count"`
	TotalPositions int      `json:"total_positions"`
	BoundingBox    []float64 `json:"bbox,omitempty"` // [minLon, minLat, maxLon, maxLat]
}

// ValidateLinearRing enforces RFC 7946 constraints on a single linear ring of coordinates.
func ValidateLinearRing(ring []Coordinates) (bool, error) {
	if len(ring) < 4 {
		return false, fmt.Errorf("%w: got %d points", ErrInsufficientVertices, len(ring))
	}

	for _, pt := range ring {
		if math.IsNaN(pt.Latitude) || math.IsNaN(pt.Longitude) || math.IsInf(pt.Latitude, 0) || math.IsInf(pt.Longitude, 0) {
			return false, ErrNonFiniteCoordinate
		}
		if pt.Latitude < -90.0 || pt.Latitude > 90.0 || pt.Longitude < -180.0 || pt.Longitude > 180.0 {
			return false, fmt.Errorf("%w: lat=%f lon=%f", ErrInvalidCoordinateBounds, pt.Latitude, pt.Longitude)
		}
	}

	first := ring[0]
	last := ring[len(ring)-1]
	const eps = 1e-7
	if math.Abs(first.Latitude-last.Latitude) > eps || math.Abs(first.Longitude-last.Longitude) > eps {
		return false, ErrUnclosedLinearRing
	}

	return true, nil
}

// IsCounterClockwise computes the signed area to determine polygon winding orientation.
// Returns true if vertices follow counter-clockwise (exterior ring rule under RFC 7946).
func IsCounterClockwise(ring []Coordinates) bool {
	if len(ring) < 3 {
		return false
	}
	var signedArea float64
	for i := 0; i < len(ring)-1; i++ {
		p1 := ring[i]
		p2 := ring[i+1]
		signedArea += (p2.Longitude - p1.Longitude) * (p2.Latitude + p1.Latitude)
	}
	// Under standard Cartesian projection with Y=Latitude, X=Longitude:
	// A negative signed area implies counter-clockwise traversal
	return signedArea < 0
}

// EnforceRingClosure appends the first coordinate if the ring is open.
func EnforceRingClosure(ring []Coordinates) []Coordinates {
	if len(ring) < 3 {
		return ring
	}
	first := ring[0]
	last := ring[len(ring)-1]
	const eps = 1e-7
	if math.Abs(first.Latitude-last.Latitude) > eps || math.Abs(first.Longitude-last.Longitude) > eps {
		return append(ring, first)
	}
	return ring
}

// ValidateMultiPolygonGeometry verifies an array of polygons (each an array of rings).
func ValidateMultiPolygonGeometry(polygons [][][]Coordinates) GeoJSONValidationResult {
	if len(polygons) == 0 {
		return GeoJSONValidationResult{
			Valid:  false,
			Errors: []string{"MultiPolygon must contain at least one Polygon"},
		}
	}

	var validationErrors []string
	totalRings := 0
	totalPositions := 0

	minLon := 180.0
	minLat := 90.0
	maxLon := -180.0
	maxLat := -90.0

	for polyIdx, poly := range polygons {
		if len(poly) == 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("Polygon %d has no linear rings", polyIdx))
			continue
		}

		for ringIdx, ring := range poly {
			totalRings++
			totalPositions += len(ring)

			ok, err := ValidateLinearRing(ring)
			if !ok {
				validationErrors = append(validationErrors, fmt.Sprintf("Polygon %d Ring %d invalid: %v", polyIdx, ringIdx, err))
			}

			// Update bounding box
			for _, pt := range ring {
				if pt.Longitude < minLon {
					minLon = pt.Longitude
				}
				if pt.Longitude > maxLon {
					maxLon = pt.Longitude
				}
				if pt.Latitude < minLat {
					minLat = pt.Latitude
				}
				if pt.Latitude > maxLat {
					maxLat = pt.Latitude
				}
			}
		}
	}

	valid := len(validationErrors) == 0
	var bbox []float64
	if totalPositions > 0 && minLon <= maxLon && minLat <= maxLat {
		bbox = []float64{
			math.Round(minLon*1e6) / 1e6,
			math.Round(minLat*1e6) / 1e6,
			math.Round(maxLon*1e6) / 1e6,
			math.Round(maxLat*1e6) / 1e6,
		}
	}

	return GeoJSONValidationResult{
		Valid:          valid,
		Errors:         validationErrors,
		RingCount:      totalRings,
		TotalPositions: totalPositions,
		BoundingBox:    bbox,
	}
}

// EncodeMultiPolygonGeoJSON serializes validated polygons into RFC 7946 GeoJSON format.
func EncodeMultiPolygonGeoJSON(polygons [][][]Coordinates, properties map[string]interface{}) ([]byte, error) {
	result := ValidateMultiPolygonGeometry(polygons)
	if !result.Valid {
		return nil, fmt.Errorf("cannot encode invalid geometry: %v", result.Errors)
	}

	var coordsArray [][][][]float64
	for _, poly := range polygons {
		var polyCoords [][][]float64
		for _, ring := range poly {
			var ringCoords [][]float64
			for _, pt := range ring {
				// RFC 7946 dictates [lon, lat] coordinate ordering
				ringCoords = append(ringCoords, []float64{pt.Longitude, pt.Latitude})
			}
			polyCoords = append(polyCoords, ringCoords)
		}
		coordsArray = append(coordsArray, polyCoords)
	}

	feature := map[string]interface{}{
		"type": "Feature",
		"bbox": result.BoundingBox,
		"geometry": map[string]interface{}{
			"type":        "MultiPolygon",
			"coordinates": coordsArray,
		},
		"properties": properties,
	}

	return json.Marshal(feature)
}
