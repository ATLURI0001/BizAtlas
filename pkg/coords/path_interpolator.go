package coords

import (
	"errors"
	"fmt"
	"math"
)

var (
	// ErrInvalidStep is returned when the interpolation step distance is non-positive or non-finite.
	ErrInvalidStep = errors.New("interpolation step distance must be positive and finite")
	// ErrInvalidTolerance is returned when simplification tolerance is non-positive or non-finite.
	ErrInvalidTolerance = errors.New("simplification tolerance must be positive and finite")
	// ErrInsufficientWaypoints is returned when an operation requires at least two waypoints.
	ErrInsufficientWaypoints = errors.New("path must contain at least two valid coordinates")
)

// InitialBearing calculates the initial forward azimuth (in degrees [0, 360)) from c1 to c2 along a great circle.
func InitialBearing(c1, c2 Coordinates) (float64, error) {
	if !c1.IsValid() || !c2.IsValid() {
		return 0, ErrInvalidCoordinates
	}

	lat1 := degToRad(c1.Latitude)
	lat2 := degToRad(c2.Latitude)
	dLon := degToRad(c2.Longitude - c1.Longitude)

	y := math.Sin(dLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(dLon)

	bearingRad := math.Atan2(y, x)
	bearingDeg := math.Mod(radToDeg(bearingRad)+360.0, 360.0)
	return bearingDeg, nil
}

// TotalPathDistanceMeters computes the cumulative great-circle distance along a series of waypoints in meters.
func TotalPathDistanceMeters(path []Coordinates) (float64, error) {
	if len(path) < 2 {
		return 0, nil
	}

	var totalKm float64
	for i := 0; i < len(path)-1; i++ {
		d, err := HaversineDistance(path[i], path[i+1])
		if err != nil {
			return 0, fmt.Errorf("distance error between waypoint %d and %d: %w", i, i+1, err)
		}
		totalKm += d
	}

	return totalKm * 1000.0, nil
}

// InterpolateSegment generates intermediate points between start and end spaced by at most maxStepMeters.
// It uses spherical linear interpolation (SLERP) along the great circle connecting the two points.
func InterpolateSegment(start, end Coordinates, maxStepMeters float64) ([]Coordinates, error) {
	if !start.IsValid() || !end.IsValid() {
		return nil, ErrInvalidCoordinates
	}
	if maxStepMeters <= 0 || math.IsNaN(maxStepMeters) || math.IsInf(maxStepMeters, 0) {
		return nil, ErrInvalidStep
	}

	distKm, err := HaversineDistance(start, end)
	if err != nil {
		return nil, err
	}
	distMeters := distKm * 1000.0

	if distMeters <= maxStepMeters {
		return []Coordinates{start, end}, nil
	}

	numSegments := int(math.Ceil(distMeters / maxStepMeters))
	delta := distKm / EarthRadiusKm // angular distance in radians

	lat1 := degToRad(start.Latitude)
	lon1 := degToRad(start.Longitude)
	lat2 := degToRad(end.Latitude)
	lon2 := degToRad(end.Longitude)

	sinDelta := math.Sin(delta)
	if math.Abs(sinDelta) < 1e-12 {
		return []Coordinates{start, end}, nil
	}

	result := make([]Coordinates, 0, numSegments+1)
	result = append(result, start)

	for i := 1; i < numSegments; i++ {
		f := float64(i) / float64(numSegments)
		a := math.Sin((1.0-f)*delta) / sinDelta
		b := math.Sin(f*delta) / sinDelta

		x := a*math.Cos(lat1)*math.Cos(lon1) + b*math.Cos(lat2)*math.Cos(lon2)
		y := a*math.Cos(lat1)*math.Sin(lon1) + b*math.Cos(lat2)*math.Sin(lon2)
		z := a*math.Sin(lat1) + b*math.Sin(lat2)

		interLat := math.Atan2(z, math.Sqrt(x*x+y*y))
		interLon := math.Atan2(y, x)

		interCoord := Coordinates{
			Latitude:  radToDeg(interLat),
			Longitude: radToDeg(interLon),
		}
		result = append(result, interCoord)
	}

	result = append(result, end)
	return result, nil
}

// InterpolatePath densifies a multi-segment waypoint route such that no two adjacent points exceed maxStepMeters.
func InterpolatePath(path []Coordinates, maxStepMeters float64) ([]Coordinates, error) {
	if len(path) == 0 {
		return nil, nil
	}
	if len(path) == 1 {
		if !path[0].IsValid() {
			return nil, ErrInvalidCoordinates
		}
		return []Coordinates{path[0]}, nil
	}
	if maxStepMeters <= 0 || math.IsNaN(maxStepMeters) || math.IsInf(maxStepMeters, 0) {
		return nil, ErrInvalidStep
	}

	var densified []Coordinates
	for i := 0; i < len(path)-1; i++ {
		segment, err := InterpolateSegment(path[i], path[i+1], maxStepMeters)
		if err != nil {
			return nil, err
		}

		if i == 0 {
			densified = append(densified, segment...)
		} else {
			// Avoid duplicate waypoint at segment boundary
			densified = append(densified, segment[1:]...)
		}
	}

	return densified, nil
}

// CrossTrackDistanceMeters computes the cross-track distance (perpendicular distance) in meters from
// point pt to the great circle arc connecting start and end.
func CrossTrackDistanceMeters(pt, start, end Coordinates) (float64, error) {
	if !pt.IsValid() || !start.IsValid() || !end.IsValid() {
		return 0, ErrInvalidCoordinates
	}

	dist13Km, err := HaversineDistance(start, pt)
	if err != nil {
		return 0, err
	}
	dist13 := dist13Km / EarthRadiusKm // angular distance in radians

	bearing13Deg, err := InitialBearing(start, pt)
	if err != nil {
		return 0, err
	}
	bearing12Deg, err := InitialBearing(start, end)
	if err != nil {
		return 0, err
	}

	theta13 := degToRad(bearing13Deg)
	theta12 := degToRad(bearing12Deg)

	sinDxt := math.Sin(dist13) * math.Sin(theta13-theta12)
	dxtRad := math.Asin(math.Max(-1.0, math.Min(1.0, sinDxt)))

	return math.Abs(dxtRad) * EarthRadiusKm * 1000.0, nil
}

// SimplifyPath applies the Ramer-Douglas-Peucker algorithm using cross-track geodesic distance
// to reduce vertex density while preserving overall trajectory within toleranceMeters.
// An iterative stack-based approach is used to prevent stack overflows on deep paths.
func SimplifyPath(path []Coordinates, toleranceMeters float64) ([]Coordinates, error) {
	if toleranceMeters <= 0 || math.IsNaN(toleranceMeters) || math.IsInf(toleranceMeters, 0) {
		return nil, ErrInvalidTolerance
	}
	if len(path) <= 2 {
		for _, c := range path {
			if !c.IsValid() {
				return nil, ErrInvalidCoordinates
			}
		}
		out := make([]Coordinates, len(path))
		copy(out, path)
		return out, nil
	}

	// Validate all coordinates first
	for _, c := range path {
		if !c.IsValid() {
			return nil, ErrInvalidCoordinates
		}
	}

	keep := make([]bool, len(path))
	keep[0] = true
	keep[len(path)-1] = true

	type rangePair struct {
		start int
		end   int
	}

	// Iterative stack avoids recursive call stack exhaustion
	stack := []rangePair{{start: 0, end: len(path) - 1}}

	for len(stack) > 0 {
		curr := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		maxDist := 0.0
		maxIdx := -1

		startPt := path[curr.start]
		endPt := path[curr.end]

		for i := curr.start + 1; i < curr.end; i++ {
			d, err := CrossTrackDistanceMeters(path[i], startPt, endPt)
			if err != nil {
				continue
			}
			if d > maxDist {
				maxDist = d
				maxIdx = i
			}
		}

		if maxDist > toleranceMeters && maxIdx != -1 {
			keep[maxIdx] = true
			if maxIdx-curr.start > 1 {
				stack = append(stack, rangePair{start: curr.start, end: maxIdx})
			}
			if curr.end-maxIdx > 1 {
				stack = append(stack, rangePair{start: maxIdx, end: curr.end})
			}
		}
	}

	result := make([]Coordinates, 0, len(path))
	for i, shouldKeep := range keep {
		if shouldKeep {
			result = append(result, path[i])
		}
	}

	return result, nil
}
