package coords

import (
	"errors"
	"math"
)

const (
	// EarthRadiusKm represents the mean volumetric radius of Earth in kilometers (WGS 84).
	EarthRadiusKm = 6371.0088
)

var (
	// ErrInvalidCoordinates is returned when coordinates violate latitude/longitude ranges or contain non-finite values.
	ErrInvalidCoordinates = errors.New("invalid coordinates: latitude must be in [-90, 90] and longitude in [-180, 180]")
	// ErrInvalidRadius is returned when a negative or non-finite radius is provided.
	ErrInvalidRadius = errors.New("invalid radius: must be non-negative and finite")
)

// degToRad converts degrees to radians.
func degToRad(deg float64) float64 {
	return deg * (math.Pi / 180.0)
}

// radToDeg converts radians to degrees.
func radToDeg(rad float64) float64 {
	return rad * (180.0 / math.Pi)
}

// HaversineDistance calculates the great-circle distance between two geographic points in kilometers.
func HaversineDistance(c1, c2 Coordinates) (float64, error) {
	if !c1.IsValid() || !c2.IsValid() {
		return 0.0, ErrInvalidCoordinates
	}

	lat1Rad := degToRad(c1.Latitude)
	lat2Rad := degToRad(c2.Latitude)
	deltaLat := degToRad(c2.Latitude - c1.Latitude)
	deltaLon := degToRad(c2.Longitude - c1.Longitude)

	// Haversine formula
	sinHalfDeltaLat := math.Sin(deltaLat / 2.0)
	sinHalfDeltaLon := math.Sin(deltaLon / 2.0)

	a := (sinHalfDeltaLat * sinHalfDeltaLat) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*(sinHalfDeltaLon*sinHalfDeltaLon)

	// Clamp 'a' to [0.0, 1.0] to protect against floating point inaccuracy in asin/atan2
	if a < 0.0 {
		a = 0.0
	} else if a > 1.0 {
		a = 1.0
	}

	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))
	return EarthRadiusKm * c, nil
}

// DistanceKm returns the great-circle distance in kilometers from c to other.
func (c Coordinates) DistanceKm(other Coordinates) (float64, error) {
	return HaversineDistance(c, other)
}

// DistanceMeters returns the great-circle distance in meters from c to other.
func (c Coordinates) DistanceMeters(other Coordinates) (float64, error) {
	km, err := HaversineDistance(c, other)
	if err != nil {
		return 0.0, err
	}
	return km * 1000.0, nil
}

// BoundingBox computes the minimum and maximum geographic coordinates encompassing a circular region of radiusKm.
func (c Coordinates) BoundingBox(radiusKm float64) (minCoords, maxCoords Coordinates, err error) {
	if !c.IsValid() {
		return Coordinates{}, Coordinates{}, ErrInvalidCoordinates
	}
	if radiusKm < 0.0 || math.IsNaN(radiusKm) || math.IsInf(radiusKm, 0) {
		return Coordinates{}, Coordinates{}, ErrInvalidRadius
	}

	angularDistance := radiusKm / EarthRadiusKm
	latRad := degToRad(c.Latitude)

	minLat := c.Latitude - radToDeg(angularDistance)
	maxLat := c.Latitude + radToDeg(angularDistance)

	var minLon, maxLon float64
	// If poles are contained, longitude spans the entire earth [-180, 180]
	if minLat > -90.0 && maxLat < 90.0 {
		deltaLon := radToDeg(math.Asin(math.Sin(angularDistance) / math.Cos(latRad)))
		minLon = c.Longitude - deltaLon
		maxLon = c.Longitude + deltaLon

		// Wrap longitudes into [-180, 180]
		if minLon < -180.0 {
			minLon += 360.0
		}
		if maxLon > 180.0 {
			maxLon -= 360.0
		}
	} else {
		// Encompasses pole
		if minLat < -90.0 {
			minLat = -90.0
		}
		if maxLat > 90.0 {
			maxLat = 90.0
		}
		minLon = -180.0
		maxLon = 180.0
	}

	return Coordinates{Latitude: minLat, Longitude: minLon}, Coordinates{Latitude: maxLat, Longitude: maxLon}, nil
}
