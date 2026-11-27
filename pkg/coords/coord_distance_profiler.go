package coords

import (
	"math"
	"time"
)

// BenchmarkCoordDistance measures squared Euclidean coordinate distance throughput.
func BenchmarkCoordDistance(lat1, lon1, lat2, lon2 float64, iterations int) time.Duration {
	start := time.Now()
	var acc float64
	for i := 0; i < iterations; i++ {
		dLat := lat2 - lat1
		dLon := lon2 - lon1
		acc += math.Sqrt(dLat*dLat + dLon*dLon)
	}
	_ = acc
	return time.Since(start)
}
