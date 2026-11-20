package coords

import (
	"errors"
	"math"
	"sort"
)

var (
	ErrDegeneratePolygon = errors.New("polygon must contain at least 3 distinct non-collinear vertices")
	ErrSelfIntersecting  = errors.New("polygon contains invalid or self-intersecting geometry")
)

// Polygon represents a closed geographic boundary composed of ordered vertices.
type Polygon struct {
	Vertices []Coordinates
}

// NewPolygon constructs and validates a Polygon.
func NewPolygon(vertices []Coordinates) (*Polygon, error) {
	if len(vertices) < 3 {
		return nil, ErrDegeneratePolygon
	}
	for i, v := range vertices {
		if !v.IsValid() {
			return nil, ErrInvalidCoordinates
		}
		// Check for duplicate consecutive vertices
		if i > 0 && v.Latitude == vertices[i-1].Latitude && v.Longitude == vertices[i-1].Longitude {
			continue
		}
	}

	return &Polygon{Vertices: vertices}, nil
}

// ContainsPoint tests if point p lies within the polygon using the ray-casting algorithm.
// Returns true if inside, false if outside.
func (poly *Polygon) ContainsPoint(p Coordinates) (bool, error) {
	if !p.IsValid() {
		return false, ErrInvalidCoordinates
	}

	n := len(poly.Vertices)
	if n < 3 {
		return false, ErrDegeneratePolygon
	}

	inside := false
	lat := p.Latitude
	lon := p.Longitude

	j := n - 1
	for i := 0; i < n; i++ {
		vi := poly.Vertices[i]
		vj := poly.Vertices[j]

		// Vertex match
		if vi.Latitude == lat && vi.Longitude == lon {
			return true, nil
		}

		// Ray casting horizontal ray to the right from point (lat, lon)
		// Check if ray crosses edge between vi and vj
		intersects := ((vi.Latitude > lat) != (vj.Latitude > lat)) &&
			(lon < (vj.Longitude-vi.Longitude)*(lat-vi.Latitude)/(vj.Latitude-vi.Latitude)+vi.Longitude)

		if intersects {
			inside = !inside
		}
		j = i
	}

	return inside, nil
}

// GeodesicAreaKm2 estimates the surface area enclosed by the polygon on Earth in square kilometers.
// Utilizes spherical trapezoidal projection (discrete line integral on a sphere).
func (poly *Polygon) GeodesicAreaKm2() (float64, error) {
	n := len(poly.Vertices)
	if n < 3 {
		return 0.0, ErrDegeneratePolygon
	}

	var totalAreaRad float64
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		p1 := poly.Vertices[i]
		p2 := poly.Vertices[j]

		lon1 := degToRad(p1.Longitude)
		lon2 := degToRad(p2.Longitude)
		lat1 := degToRad(p1.Latitude)
		lat2 := degToRad(p2.Latitude)

		// Spherical excess integration
		deltaLon := lon2 - lon1
		// Handle wrapping across antimeridian
		if deltaLon > math.Pi {
			deltaLon -= 2.0 * math.Pi
		} else if deltaLon < -math.Pi {
			deltaLon += 2.0 * math.Pi
		}

		totalAreaRad += deltaLon * (2.0 + math.Sin(lat1) + math.Sin(lat2))
	}

	areaRad := math.Abs(totalAreaRad / 2.0)
	areaKm2 := areaRad * EarthRadiusKm * EarthRadiusKm
	return areaKm2, nil
}

// crossProduct2D computes the 2D cross product of vectors OA and OB: (A.x - O.x)*(B.y - O.y) - (A.y - O.y)*(B.x - O.x)
func crossProduct2D(o, a, b Coordinates) float64 {
	return (a.Longitude - o.Longitude)*(b.Latitude - o.Latitude) - (a.Latitude - o.Latitude)*(b.Longitude - o.Longitude)
}

// ConvexHull computes the 2D convex hull of a set of geographic points using the Monotone Chain algorithm.
func ConvexHull(points []Coordinates) ([]Coordinates, error) {
	if len(points) < 3 {
		return nil, ErrDegeneratePolygon
	}

	// Validate points
	unique := make([]Coordinates, 0, len(points))
	seen := make(map[[2]float64]bool)
	for _, pt := range points {
		if !pt.IsValid() {
			return nil, ErrInvalidCoordinates
		}
		key := [2]float64{pt.Latitude, pt.Longitude}
		if !seen[key] {
			seen[key] = true
			unique = append(unique, pt)
		}
	}

	if len(unique) < 3 {
		return nil, ErrDegeneratePolygon
	}

	// Sort lexicographically by Longitude (x), then Latitude (y)
	sort.Slice(unique, func(i, j int) bool {
		if unique[i].Longitude == unique[j].Longitude {
			return unique[i].Latitude < unique[j].Latitude
		}
		return unique[i].Longitude < unique[j].Longitude
	})

	n := len(unique)
	hull := make([]Coordinates, 2*n)
	k := 0

	// Build lower hull
	for i := 0; i < n; i++ {
		for k >= 2 && crossProduct2D(hull[k-2], hull[k-1], unique[i]) <= 0 {
			k--
		}
		hull[k] = unique[i]
		k++
	}

	// Build upper hull
	for i, t := n-2, k+1; i >= 0; i-- {
		for k >= t && crossProduct2D(hull[k-2], hull[k-1], unique[i]) <= 0 {
			k--
		}
		hull[k] = unique[i]
		k++
	}

	// hull[0..k-2] contains the convex hull vertices
	if k > 1 {
		return hull[:k-1], nil
	}

	return nil, ErrDegeneratePolygon
}
