package coords

import "testing"

func TestHaversineBatchProfiler_Benchmark(t *testing.T) {
	p := &HaversineBatchProfiler{}
	origin := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	targets := []Coordinates{
		{Latitude: 34.0522, Longitude: -118.2437},
		{Latitude: 40.7128, Longitude: -74.0060},
	}

	res := p.RunBenchmark(origin, targets, 20)
	if res.PairCount != 2 || res.ThroughputOpsSec <= 0 {
		t.Fatalf("unexpected benchmark result: %+v", res)
	}
}
