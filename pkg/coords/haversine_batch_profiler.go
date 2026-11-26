package coords

import (
	"math"
	"sort"
	"time"
)

const earthRadiusKm = 6371.0

// HaversineBatchResult aggregates spatial distance calculation throughput.
type HaversineBatchResult struct {
	PairCount         int
	P50Microseconds   float64
	P90Microseconds   float64
	ThroughputOpsSec  int64
}

// HaversineBatchProfiler benchmarks fast spherical distance calculations.
type HaversineBatchProfiler struct{}

// FastDistanceKm computes great-circle distance in kilometers.
func FastDistanceKm(a, b Coordinates) float64 {
	dLat := (b.Latitude - a.Latitude) * math.Pi / 180.0
	dLon := (b.Longitude - a.Longitude) * math.Pi / 180.0
	lat1 := a.Latitude * math.Pi / 180.0
	lat2 := b.Latitude * math.Pi / 180.0

	sinLat := math.Sin(dLat * 0.5)
	sinLon := math.Sin(dLon * 0.5)
	h := sinLat*sinLat + math.Cos(lat1)*math.Cos(lat2)*sinLon*sinLon
	return 2.0 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

// RunBenchmark profiles batched Haversine distance calculations.
func (p *HaversineBatchProfiler) RunBenchmark(origin Coordinates, targets []Coordinates, iterations int) HaversineBatchResult {
	if iterations <= 0 {
		iterations = 30
	}
	innerBatch := 25
	latencies := make([]float64, iterations)
	tStart := time.Now()

	for i := 0; i < iterations; i++ {
		start := time.Now()
		for b := 0; b < innerBatch; b++ {
			for _, t := range targets {
				_ = FastDistanceKm(origin, t)
			}
		}
		latencies[i] = float64(time.Since(start).Nanoseconds()) / 1000.0 / float64(innerBatch)
	}

	totalSec := time.Since(tStart).Seconds()
	sort.Float64s(latencies)
	totalOps := int64(len(targets)) * int64(iterations) * int64(innerBatch)

	return HaversineBatchResult{
		PairCount:        len(targets),
		P50Microseconds:  math.Round(latencies[int(float64(iterations)*0.5)]*100) / 100,
		P90Microseconds:  math.Round(latencies[int(float64(iterations)*0.9)]*100) / 100,
		ThroughputOpsSec: int64(float64(totalOps) / math.Max(1e-6, totalSec)),
	}
}
