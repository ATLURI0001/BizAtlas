package coords

import "time"

// BenchmarkPlaceDedup measures place ID deduplication throughput.
func BenchmarkPlaceDedup(ids []string, iterations int) (unique int, elapsed time.Duration) {
	start := time.Now()
	for i := 0; i < iterations; i++ {
		seen := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			seen[id] = struct{}{}
		}
		unique = len(seen)
	}
	return unique, time.Since(start)
}
