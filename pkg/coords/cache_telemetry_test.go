package coords

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestGeocodingCacheTelemetry_NominalTracking(t *testing.T) {
	telemetry := NewGeocodingCacheTelemetry(100)

	// Simulate 40 hits and 10 misses with nominal 10ms-25ms latency
	for i := 0; i < 40; i++ {
		telemetry.RecordLookup(true, time.Duration(10+i%10)*time.Millisecond, nil)
	}
	for i := 0; i < 10; i++ {
		telemetry.RecordLookup(false, time.Duration(50+i%20)*time.Millisecond, nil)
	}

	snap := telemetry.GetSnapshot()
	if snap.TotalRequests != 50 {
		t.Errorf("expected 50 total requests, got %d", snap.TotalRequests)
	}
	if snap.CacheHits != 40 {
		t.Errorf("expected 40 hits, got %d", snap.CacheHits)
	}
	if snap.CacheMisses != 10 {
		t.Errorf("expected 10 misses, got %d", snap.CacheMisses)
	}
	if snap.HitRatePercent != 80.0 {
		t.Errorf("expected 80%% hit rate, got %f", snap.HitRatePercent)
	}
	if snap.IsDegraded {
		t.Errorf("expected nominal state (not degraded), got warnings: %v", snap.WarningMessages)
	}
	if snap.P50LatencyMs <= 0 || snap.P99LatencyMs <= 0 {
		t.Errorf("expected positive latencies, got p50=%f, p99=%f", snap.P50LatencyMs, snap.P99LatencyMs)
	}
}

func TestGeocodingCacheTelemetry_DegradationTriggers(t *testing.T) {
	// 1. Low hit rate degradation (< 40%)
	tLowHit := NewGeocodingCacheTelemetry(100)
	for i := 0; i < 5; i++ {
		tLowHit.RecordLookup(true, 5*time.Millisecond, nil)
	}
	for i := 0; i < 35; i++ {
		tLowHit.RecordLookup(false, 30*time.Millisecond, nil)
	}
	snapLow := tLowHit.GetSnapshot()
	if !snapLow.IsDegraded {
		t.Error("expected degradation on low hit rate")
	}
	hasHitWarning := false
	for _, w := range snapLow.WarningMessages {
		if strings.Contains(w, "Low cache hit rate") {
			hasHitWarning = true
			break
		}
	}
	if !hasHitWarning {
		t.Errorf("missing low hit rate warning, got %v", snapLow.WarningMessages)
	}

	// 2. High latency spike ($P_{99} > 250ms$)
	tSlow := NewGeocodingCacheTelemetry(100)
	for i := 0; i < 19; i++ {
		tSlow.RecordLookup(true, 10*time.Millisecond, nil)
	}
	// 1 extreme outlier
	tSlow.RecordLookup(false, 350*time.Millisecond, nil)
	snapSlow := tSlow.GetSnapshot()
	if !snapSlow.IsDegraded {
		t.Error("expected degradation on P99 latency spike")
	}

	// 3. Elevated error rate (> 5%)
	tErr := NewGeocodingCacheTelemetry(100)
	for i := 0; i < 20; i++ {
		tErr.RecordLookup(true, 10*time.Millisecond, nil)
	}
	for i := 0; i < 5; i++ {
		tErr.RecordLookup(false, 15*time.Millisecond, errors.New("upstream timeout"))
	}
	snapErr := tErr.GetSnapshot()
	if !snapErr.IsDegraded {
		t.Error("expected degradation on high error rate")
	}
}

func TestGeocodingCacheTelemetry_EvictionAndPrometheus(t *testing.T) {
	telemetry := NewGeocodingCacheTelemetry(20)

	// Ingest 30 samples to test circular eviction
	for i := 0; i < 30; i++ {
		telemetry.RecordLookup(true, time.Duration(10+i)*time.Millisecond, nil)
	}

	snap := telemetry.GetSnapshot()
	if snap.TotalRequests != 30 {
		t.Errorf("expected 30 total requests, got %d", snap.TotalRequests)
	}

	prom := telemetry.ToPrometheusMetrics()
	if !strings.Contains(prom, "bizatlas_geocoding_requests_total 30") {
		t.Errorf("expected total requests in prometheus output, got:\n%s", prom)
	}
	if !strings.Contains(prom, "bizatlas_geocoding_cache_hit_rate_ratio") {
		t.Errorf("expected hit rate ratio in prometheus output, got:\n%s", prom)
	}
}
