package coords

import (
	"errors"
	"fmt"
	"math"
)

var (
	// ErrPolygonUnclosed indicates the first and last coordinates do not match.
	ErrPolygonUnclosed = errors.New("polygon_contract: linear ring must be closed (first and last coordinate must be identical)")
	// ErrPolygonRingInsufficient indicates fewer than 4 coordinates in a linear ring.
	ErrPolygonRingInsufficient = errors.New("polygon_contract: linear ring must contain at least 4 coordinates (3 distinct vertices + 1 closing vertex)")
	// ErrCoordinatesOutOfBounds indicates latitude outside [-90, 90] or longitude outside [-180, 180].
	ErrCoordinatesOutOfBounds = errors.New("polygon_contract: coordinate values out of geographic bounds")
	// ErrDegenerateRing indicates polygon ring has zero area.
	ErrDegenerateRing = errors.New("polygon_contract: polygon ring has zero or degenerate surface area")
	// ErrInvalidWindingOrder indicates external ring is clockwise or interior ring is counterclockwise when strict is enabled.
	ErrInvalidWindingOrder = errors.New("polygon_contract: RFC 7946 winding order violation (exterior ring must be CCW, interior rings CW)")
)

// Point2D represents a geographic coordinate [longitude, latitude].
type Point2D struct {
	Lon float64
	Lat float64
}

// BoundingBox2D represents a 2D spatial bounding box [minLon, minLat, maxLon, maxLat].
type BoundingBox2D struct {
	MinLon float64
	MinLat float64
	MaxLon float64
	MaxLat float64
}

// PolygonSpec represents an ingress polygon contract payload.
type PolygonSpec struct {
	Version   string
	Exterior  []Point2D
	Holes     [][]Point2D
	BBox      BoundingBox2D
	AreaSqDeg float64
}

// CalculatePolygonSignedArea computes signed area using the shoelace formula.
// Returns positive for CCW (counterclockwise) and negative for CW (clockwise).
func CalculatePolygonSignedArea(ring []Point2D) float64 {
	if len(ring) < 3 {
		return 0.0
	}
	var area float64
	n := len(ring)
	for i := 0; i < n-1; i++ {
		p1 := ring[i]
		p2 := ring[i+1]
		area += (p2.Lon - p1.Lon) * (p2.Lat + p1.Lat)
	}
	return -area / 2.0
}

// IsPolygonCCW checks whether a linear ring follows CCW orientation.
func IsPolygonCCW(ring []Point2D) bool {
	return CalculatePolygonSignedArea(ring) > 0
}

// EnforceWindingOrder ensures the ring matches the desired CCW orientation.
// If invert is needed, it reverses the vertices.
func EnforceWindingOrder(ring []Point2D, wantCCW bool) []Point2D {
	isCCW := IsPolygonCCW(ring)
	if isCCW == wantCCW {
		return ring
	}
	// Reverse ring order while preserving closure
	reversed := make([]Point2D, len(ring))
	for i := range ring {
		reversed[i] = ring[len(ring)-1-i]
	}
	return reversed
}

// ValidateAndSanitizeLinearRing checks coordinate validity, closure, and returns bbox.
func ValidateAndSanitizeLinearRing(rawPoints [][]float64) ([]Point2D, BoundingBox2D, error) {
	if len(rawPoints) < 4 {
		return nil, BoundingBox2D{}, ErrPolygonRingInsufficient
	}

	first := rawPoints[0]
	last := rawPoints[len(rawPoints)-1]
	if len(first) < 2 || len(last) < 2 {
		return nil, BoundingBox2D{}, ErrCoordinatesOutOfBounds
	}

	// Closure check
	if math.Abs(first[0]-last[0]) > 1e-9 || math.Abs(first[1]-last[1]) > 1e-9 {
		return nil, BoundingBox2D{}, ErrPolygonUnclosed
	}

	minLon := math.MaxFloat64
	minLat := math.MaxFloat64
	maxLon := -math.MaxFloat64
	maxLat := -math.MaxFloat64

	ring := make([]Point2D, len(rawPoints))
	for i, coord := range rawPoints {
		if len(coord) < 2 {
			return nil, BoundingBox2D{}, ErrCoordinatesOutOfBounds
		}
		lon := coord[0]
		lat := coord[1]

		if lon < -180.0 || lon > 180.0 || lat < -90.0 || lat > 90.0 {
			return nil, BoundingBox2D{}, fmt.Errorf("%w: [%f, %f]", ErrCoordinatesOutOfBounds, lon, lat)
		}

		if lon < minLon {
			minLon = lon
		}
		if lon > maxLon {
			maxLon = lon
		}
		if lat < minLat {
			minLat = lat
		}
		if lat > maxLat {
			maxLat = lat
		}

		ring[i] = Point2D{Lon: lon, Lat: lat}
	}

	bbox := BoundingBox2D{
		MinLon: minLon,
		MinLat: minLat,
		MaxLon: maxLon,
		MaxLat: maxLat,
	}

	return ring, bbox, nil
}

// ValidateAndSanitizePolygon enforces RFC 7946 winding order and bounding box contracts.
func ValidateAndSanitizePolygon(rawExterior [][]float64, rawHoles [][][]float64, autoFixWinding bool) (*PolygonSpec, error) {
	extRing, bbox, err := ValidateAndSanitizeLinearRing(rawExterior)
	if err != nil {
		return nil, err
	}

	signedArea := CalculatePolygonSignedArea(extRing)
	if math.Abs(signedArea) < 1e-12 {
		return nil, ErrDegenerateRing
	}

	// RFC 7946: exterior must be CCW
	if !IsPolygonCCW(extRing) {
		if autoFixWinding {
			extRing = EnforceWindingOrder(extRing, true)
		} else {
			return nil, fmt.Errorf("%w: exterior ring is clockwise", ErrInvalidWindingOrder)
		}
	}

	sanitizedHoles := make([][]Point2D, 0, len(rawHoles))
	for i, holeRaw := range rawHoles {
		holeRing, _, err := ValidateAndSanitizeLinearRing(holeRaw)
		if err != nil {
			return nil, fmt.Errorf("hole[%d]: %w", i, err)
		}
		// RFC 7946: interior rings must be CW
		if IsPolygonCCW(holeRing) {
			if autoFixWinding {
				holeRing = EnforceWindingOrder(holeRing, false)
			} else {
				return nil, fmt.Errorf("%w: interior ring[%d] is counterclockwise", ErrInvalidWindingOrder, i)
			}
		}
		sanitizedHoles = append(sanitizedHoles, holeRing)
	}

	return &PolygonSpec{
		Version:   "geojson-polygon/v2",
		Exterior:  extRing,
		Holes:     sanitizedHoles,
		BBox:      bbox,
		AreaSqDeg: math.Abs(signedArea),
	}, nil
}
