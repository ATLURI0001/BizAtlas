package coords

import (
	"strings"
	"sync"
	"testing"
)

func TestGeocodingTelemetry_Basics(t *testing.T) {
	telemetry := NewGeocodingTelemetry(50, 2.5)

	// Record initial events
	telemetry.RecordRequest("osm", 12.5, true, false)
	telemetry.RecordRequest("osm", 18.0, false, false)
	telemetry.RecordRequest("google", 45.2, false, true)

	snap := telemetry.Snapshot()
	if snap.TotalRequests != 3 {
		t.Fatalf("expected 3 total requests, got %d", snap.TotalRequests)
	}
	if snap.CacheHits != 1 {
		t.Fatalf("expected 1 cache hit, got %d", snap.CacheHits)
	}
	if snap.CacheMisses != 2 {
		t.Fatalf("expected 2 cache misses, got %d", snap.CacheMisses)
	}
	if snap.ErrorCount != 1 {
		t.Fatalf("expected 1 error, got %d", snap.ErrorCount)
	}
	expectedRatio := 1.0 / 3.0
	if snap.CacheHitRatio < expectedRatio-0.01 || snap.CacheHitRatio > expectedRatio+0.01 {
		t.Fatalf("expected ~0.333 hit ratio, got %f", snap.CacheHitRatio)
	}
	if snap.ProviderStats["osm"] != 2 {
		t.Fatalf("expected 2 osm provider calls, got %d", snap.ProviderStats["osm"])
	}
	if snap.ProviderStats["google"] != 1 {
		t.Fatalf("expected 1 google provider call, got %d", snap.ProviderStats["google"])
	}
}

func TestGeocodingTelemetry_Percentiles(t *testing.T) {
	telemetry := NewGeocodingTelemetry(100, 3.0)

	// Feed 100 uniformly spaced samples: 1.0 to 100.0
	for i := 1; i <= 100; i++ {
		telemetry.RecordRequest("nominatim", float64(i), i%2 == 0, false)
	}

	snap := telemetry.Snapshot()
	if snap.P50LatencyMs < 49.0 || snap.P50LatencyMs > 52.0 {
		t.Errorf("expected P50 around 50.5, got %f", snap.P50LatencyMs)
	}
	if snap.P90LatencyMs < 89.0 || snap.P90LatencyMs > 91.5 {
		t.Errorf("expected P90 around 90.1, got %f", snap.P90LatencyMs)
	}
	if snap.P99LatencyMs < 98.0 || snap.P99LatencyMs > 100.0 {
		t.Errorf("expected P99 around 99.0, got %f", snap.P99LatencyMs)
	}
}

func TestGeocodingTelemetry_AnomalyDetection(t *testing.T) {
	telemetry := NewGeocodingTelemetry(100, 2.5)

	// Not enough samples initially (< 10)
	isAnomaly, _, _, _ := telemetry.DetectLatencyAnomaly(500.0)
	if isAnomaly {
		t.Fatal("expected no anomaly flag when observations < 10")
	}

	// Feed 30 observations centered around 20ms (between 18ms and 22ms)
	for i := 0; i < 30; i++ {
		lat := 20.0 + float64(i%5) - 2.0 // 18, 19, 20, 21, 22
		telemetry.RecordRequest("local", lat, true, false)
	}

	// Test normal observation
	isAnomaly, zScore, mean, stdDev := telemetry.DetectLatencyAnomaly(21.0)
	if isAnomaly {
		t.Fatalf("expected 21ms not to be an anomaly, zScore=%f, mean=%f, stdDev=%f", zScore, mean, stdDev)
	}

	// Test spike observation (e.g. 150ms when mean is ~20 and stdDev is ~1.4)
	isAnomaly, zScore, _, _ = telemetry.DetectLatencyAnomaly(150.0)
	if !isAnomaly {
		t.Fatalf("expected 150ms to trigger anomaly flag, zScore=%f", zScore)
	}
}

func TestGeocodingTelemetry_PrometheusExport(t *testing.T) {
	telemetry := NewGeocodingTelemetry(50, 3.0)
	telemetry.RecordRequest("osm", 15.0, true, false)
	telemetry.RecordRequest("osm", 25.0, false, false)

	prom := telemetry.ExportPrometheus()
	expectedMetrics := []string{
		"bizatlas_geocoding_requests_total 2",
		"bizatlas_geocoding_cache_hits_total 1",
		"bizatlas_geocoding_cache_misses_total 1",
		"bizatlas_geocoding_cache_hit_ratio 0.5000",
		"bizatlas_geocoding_duration_ms{quantile=\"0.50\"}",
		"bizatlas_geocoding_provider_requests_total{provider=\"osm\"} 2",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(prom, metric) {
			t.Errorf("expected prometheus output to contain '%s', got:\n%s", metric, prom)
		}
	}
}

func TestGeocodingTelemetry_Concurrency(t *testing.T) {
	telemetry := NewGeocodingTelemetry(500, 3.0)
	var wg sync.WaitGroup

	// Run concurrent writers and readers
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				telemetry.RecordRequest("worker", float64(j*2), j%3 == 0, false)
				_ = telemetry.Snapshot()
				_, _, _, _ = telemetry.DetectLatencyAnomaly(50.0)
			}
		}(i)
	}

	wg.Wait()
	snap := telemetry.Snapshot()
	if snap.TotalRequests != 1000 {
		t.Fatalf("expected 1000 total requests after concurrent run, got %d", snap.TotalRequests)
	}
}
