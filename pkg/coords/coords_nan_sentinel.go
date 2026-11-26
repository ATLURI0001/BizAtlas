package coords

import (
	"fmt"
	"math"
)

// AssertFiniteCoords rejects NaN or Inf coordinates.
func AssertFiniteCoords(lat, lon float64) error {
	if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) {
		return fmt.Errorf("sentinel violation: non-finite coordinates (%v, %v)", lat, lon)
	}
	return nil
}
