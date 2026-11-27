package coords

import "errors"

var ErrGeoFenceContractViolation = errors.New("coordinates out of valid range")

// ValidateGeoFence checks latitude and longitude are within valid WGS-84 bounds.
func ValidateGeoFence(lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return ErrGeoFenceContractViolation
	}
	return nil
}
