package coords

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// GeocodingCacheSnapshot provides an immutable snapshot of geocoding cache operational health.
type GeocodingCacheSnapshot struct {
	TotalRequests   uint64
	CacheHits       uint64
	CacheMisses     uint64
	UpstreamErrors  uint64
	HitRatePercent  float64
	P50LatencyMs    float64
	P90LatencyMs    float64
	P95LatencyMs    float64
	P99LatencyMs    float64
	MeanLatencyMs   float64
	IsDegraded      bool
	WarningMessages []string
}

// GeocodingCacheTelemetry collects real-time hit rates, latency distributions, and degradation signals.
type GeocodingCacheTelemetry struct {
	totalRequests  atomic.Uint64
	cacheHits      atomic.Uint64
	cacheMisses    atomic.Uint64
	upstreamErrors atomic.Uint64

	minHitRateThreshold float64 // e.g. 0.40 (40%)
	maxP99LatencyMs     float64 // e.g. 250.0 ms
	maxErrorRate        float64 // e.g. 0.05 (5%)

	mu             sync.RWMutex
	latencySamples []float64 // in milliseconds
	maxSamples     int
	startTime      time.Time
}

// NewGeocodingCacheTelemetry creates a configured telemetry collector with default degradation gates.
func NewGeocodingCacheTelemetry(maxSamples int) *GeocodingCacheTelemetry {
	if maxSamples <= 0 {
		maxSamples = 10000
	}
	return &GeocodingCacheTelemetry{
		minHitRateThreshold: 0.40,
		maxP99LatencyMs:     250.0,
		maxErrorRate:        0.05,
		latencySamples:      make([]float64, 0, maxSamples),
		maxSamples:          maxSamples,
		startTime:           time.Now(),
	}
}

// RecordLookup records a completed geocoding request outcome and elapsed duration.
func (t *GeocodingCacheTelemetry) RecordLookup(cacheHit bool, duration time.Duration, err error) {
	t.totalRequests.Add(1)

	if err != nil {
		t.upstreamErrors.Add(1)
	}

	if cacheHit {
		t.cacheHits.Add(1)
	} else {
		t.cacheMisses.Add(1)
	}

	ms := float64(duration.Nanoseconds()) / 1e6

	t.mu.Lock()
	if len(t.latencySamples) >= t.maxSamples {
		// Circular eviction of oldest 20%
		evict := t.maxSamples / 5
		t.latencySamples = append(t.latencySamples[:0], t.latencySamples[evict:]...)
	}
	t.latencySamples = append(t.latencySamples, ms)
	t.mu.Unlock()
}

// GetSnapshot calculates non-parametric percentiles and determines operational degradation.
func (t *GeocodingCacheTelemetry) GetSnapshot() GeocodingCacheSnapshot {
	total := t.totalRequests.Load()
	hits := t.cacheHits.Load()
	misses := t.cacheMisses.Load()
	errors := t.upstreamErrors.Load()

	hitRate := 0.0
	if hits+misses > 0 {
		hitRate = (float64(hits) / float64(hits+misses)) * 100.0
	}

	t.mu.RLock()
	n := len(t.latencySamples)
	samples := make([]float64, n)
	copy(samples, t.latencySamples)
	t.mu.RUnlock()

	var p50, p90, p95, p99, mean float64
	if n > 0 {
		sort.Float64s(samples)
		var sum float64
		for _, s := range samples {
			sum += s
		}
		mean = sum / float64(n)

		pct := func(p float64) float64 {
			idx := int(math.Ceil((p / 100.0) * float64(n))) - 1
			if idx < 0 {
				idx = 0
			}
			if idx >= n {
				idx = n - 1
			}
			return samples[idx]
		}
		p50 = pct(50)
		p90 = pct(90)
		p95 = pct(95)
		p99 = pct(99)
	}

	// Diagnostic evaluation
	warnings := make([]string, 0)
	isDegraded := false

	// Hit rate warning only after sufficient warmup (>30 queries)
	if total >= 30 && hitRate < t.minHitRateThreshold*100.0 {
		warnings = append(warnings, fmt.Sprintf("Low cache hit rate: %.1f%% (threshold: %.1f%%)", hitRate, t.minHitRateThreshold*100.0))
		isDegraded = true
	}

	if n >= 10 && p99 > t.maxP99LatencyMs {
		warnings = append(warnings, fmt.Sprintf("High P99 latency: %.2f ms (threshold: %.2f ms)", p99, t.maxP99LatencyMs))
		isDegraded = true
	}

	if total >= 20 {
		errRate := float64(errors) / float64(total)
		if errRate > t.maxErrorRate {
			warnings = append(warnings, fmt.Sprintf("Elevated upstream error rate: %.1f%% (threshold: %.1f%%)", errRate*100.0, t.maxErrorRate*100.0))
			isDegraded = true
		}
	}

	return GeocodingCacheSnapshot{
		TotalRequests:   total,
		CacheHits:       hits,
		CacheMisses:     misses,
		UpstreamErrors:  errors,
		HitRatePercent:  math.Round(hitRate*100) / 100,
		P50LatencyMs:    math.Round(p50*100) / 100,
		P90LatencyMs:    math.Round(p90*100) / 100,
		P95LatencyMs:    math.Round(p95*100) / 100,
		P99LatencyMs:    math.Round(p99*100) / 100,
		MeanLatencyMs:   math.Round(mean*100) / 100,
		IsDegraded:      isDegraded,
		WarningMessages: warnings,
	}
}

// ToPrometheusMetrics exports standard Prometheus exposition format string.
func (t *GeocodingCacheTelemetry) ToPrometheusMetrics() string {
	snap := t.GetSnapshot()
	var b strings.Builder

	b.WriteString("# HELP bizatlas_geocoding_requests_total Total geocoding requests observed\n")
	b.WriteString("# TYPE bizatlas_geocoding_requests_total counter\n")
	b.WriteString(fmt.Sprintf("bizatlas_geocoding_requests_total %d\n", snap.TotalRequests))

	b.WriteString("# HELP bizatlas_geocoding_cache_hits_total Total geocoding cache hits\n")
	b.WriteString("# TYPE bizatlas_geocoding_cache_hits_total counter\n")
	b.WriteString(fmt.Sprintf("bizatlas_geocoding_cache_hits_total %d\n", snap.CacheHits))

	b.WriteString("# HELP bizatlas_geocoding_cache_misses_total Total geocoding cache misses\n")
	b.WriteString("# TYPE bizatlas_geocoding_cache_misses_total counter\n")
	b.WriteString(fmt.Sprintf("bizatlas_geocoding_cache_misses_total %d\n", snap.CacheMisses))

	b.WriteString("# HELP bizatlas_geocoding_cache_hit_rate_ratio Ratio of cache hits to total lookups\n")
	b.WriteString("# TYPE bizatlas_geocoding_cache_hit_rate_ratio gauge\n")
	b.WriteString(fmt.Sprintf("bizatlas_geocoding_cache_hit_rate_ratio %.4f\n", snap.HitRatePercent/100.0))

	b.WriteString("# HELP bizatlas_geocoding_latency_p99_ms 99th percentile response latency in milliseconds\n")
	b.WriteString("# TYPE bizatlas_geocoding_latency_p99_ms gauge\n")
	b.WriteString(fmt.Sprintf("bizatlas_geocoding_latency_p99_ms %.2f\n", snap.P99LatencyMs))

	b.WriteString("# HELP bizatlas_geocoding_degraded Binary indicator of operational degradation\n")
	b.WriteString("# TYPE bizatlas_geocoding_degraded gauge\n")
	deg := 0
	if snap.IsDegraded {
		deg = 1
	}
	b.WriteString(fmt.Sprintf("bizatlas_geocoding_degraded %d\n", deg))

	return b.String()
}
