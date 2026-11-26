package coords

import "fmt"

// ValidateGeoBBox enforces min < max bounds for geographic bounding boxes.
func ValidateGeoBBox(minLat, minLon, maxLat, maxLon float64) error {
	if minLat >= maxLat || minLon >= maxLon {
		return fmt.Errorf("contract violation: invalid bbox (%v,%v)-(%v,%v)", minLat, minLon, maxLat, maxLon)
	}
	return nil
}
