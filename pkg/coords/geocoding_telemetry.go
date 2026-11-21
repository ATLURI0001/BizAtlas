package coords

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

// TelemetrySnapshot represents an immutable point-in-time metrics report.
type TelemetrySnapshot struct {
	TotalRequests   uint64             `json:"total_requests"`
	CacheHits       uint64             `json:"cache_hits"`
	CacheMisses     uint64             `json:"cache_misses"`
	CacheHitRatio   float64            `json:"cache_hit_ratio"`
	ErrorCount      uint64             `json:"error_count"`
	P50LatencyMs    float64            `json:"p50_latency_ms"`
	P90LatencyMs    float64            `json:"p90_latency_ms"`
	P95LatencyMs    float64            `json:"p95_latency_ms"`
	P99LatencyMs    float64            `json:"p99_latency_ms"`
	MeanLatencyMs   float64            `json:"mean_latency_ms"`
	StdDevLatencyMs float64            `json:"std_dev_latency_ms"`
	ProviderStats   map[string]uint64  `json:"provider_stats"`
	Timestamp       time.Time          `json:"timestamp"`
}

// GeocodingTelemetry tracks geocoding ingress performance, latency percentiles, and anomalies.
type GeocodingTelemetry struct {
	mu               sync.RWMutex
	windowSize       int
	anomalyThreshold float64
	totalRequests    uint64
	cacheHits        uint64
	cacheMisses      uint64
	errorCount       uint64
	providerCounts   map[string]uint64
	recentLatencies  []float64
	headIndex        int
	count            int
}

// NewGeocodingTelemetry initializes a telemetry collector with an observation window and anomaly z-score threshold.
func NewGeocodingTelemetry(windowSize int, anomalyThreshold float64) *GeocodingTelemetry {
	if windowSize <= 0 {
		windowSize = 1000
	}
	if anomalyThreshold <= 0.0 {
		anomalyThreshold = 3.0
	}
	return &GeocodingTelemetry{
		windowSize:       windowSize,
		anomalyThreshold: anomalyThreshold,
		providerCounts:   make(map[string]uint64),
		recentLatencies:  make([]float64, windowSize),
	}
}

// RecordRequest captures latency, cache outcome, provider routing, and error state for a geocoding event.
func (t *GeocodingTelemetry) RecordRequest(provider string, durationMs float64, cacheHit bool, isError bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.totalRequests++
	if cacheHit {
		t.cacheHits++
	} else {
		t.cacheMisses++
	}
	if isError {
		t.errorCount++
	}

	if provider != "" {
		t.providerCounts[provider]++
	}

	t.recentLatencies[t.headIndex] = durationMs
	t.headIndex = (t.headIndex + 1) % t.windowSize
	if t.count < t.windowSize {
		t.count++
	}
}

// DetectLatencyAnomaly checks whether a candidate duration exceeds the statistical z-score threshold.
func (t *GeocodingTelemetry) DetectLatencyAnomaly(durationMs float64) (isAnomaly bool, zScore float64, mean float64, stdDev float64) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.count < 10 {
		// Insufficient observations for statistically valid z-score
		return false, 0.0, 0.0, 0.0
	}

	var sum float64
	for i := 0; i < t.count; i++ {
		sum += t.recentLatencies[i]
	}
	mean = sum / float64(t.count)

	var varianceSum float64
	for i := 0; i < t.count; i++ {
		diff := t.recentLatencies[i] - mean
		varianceSum += diff * diff
	}
	rawStdDev := math.Sqrt(varianceSum / float64(t.count))
	// Apply floor to prevent zero-division when variance is tiny
	floor := math.Max(0.05*mean, 0.1)
	if rawStdDev < floor {
		stdDev = floor
	} else {
		stdDev = rawStdDev
	}

	zScore = (durationMs - mean) / stdDev
	isAnomaly = zScore > t.anomalyThreshold
	return isAnomaly, zScore, mean, stdDev
}

// Snapshot returns a thread-safe summary of all aggregated telemetry data.
func (t *GeocodingTelemetry) Snapshot() TelemetrySnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	providers := make(map[string]uint64, len(t.providerCounts))
	for k, v := range t.providerCounts {
		providers[k] = v
	}

	var ratio float64
	totalCacheOps := t.cacheHits + t.cacheMisses
	if totalCacheOps > 0 {
		ratio = float64(t.cacheHits) / float64(totalCacheOps)
	}

	snap := TelemetrySnapshot{
		TotalRequests: t.totalRequests,
		CacheHits:     t.cacheHits,
		CacheMisses:   t.cacheMisses,
		CacheHitRatio: ratio,
		ErrorCount:    t.errorCount,
		ProviderStats: providers,
		Timestamp:     time.Now().UTC(),
	}

	if t.count == 0 {
		return snap
	}

	latencies := make([]float64, t.count)
	copy(latencies, t.recentLatencies[:t.count])
	sort.Float64s(latencies)

	var sum float64
	for _, lat := range latencies {
		sum += lat
	}
	snap.MeanLatencyMs = sum / float64(t.count)

	var varSum float64
	for _, lat := range latencies {
		diff := lat - snap.MeanLatencyMs
		varSum += diff * diff
	}
	snap.StdDevLatencyMs = math.Sqrt(varSum / float64(t.count))

	snap.P50LatencyMs = computePercentile(latencies, 50.0)
	snap.P90LatencyMs = computePercentile(latencies, 90.0)
	snap.P95LatencyMs = computePercentile(latencies, 95.0)
	snap.P99LatencyMs = computePercentile(latencies, 99.0)

	return snap
}

func computePercentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0.0
	}
	if p <= 0.0 {
		return sorted[0]
	}
	if p >= 100.0 {
		return sorted[len(sorted)-1]
	}
	idx := (p / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(idx))
	upper := int(math.Ceil(idx))
	if lower == upper {
		return sorted[lower]
	}
	weight := idx - float64(lower)
	return sorted[lower]*(1.0-weight) + sorted[upper]*weight
}

// ExportPrometheus serializes telemetry metrics into standard Prometheus exposition format.
func (t *GeocodingTelemetry) ExportPrometheus() string {
	snap := t.Snapshot()

	var sb strings.Builder
	sb.WriteString("# HELP bizatlas_geocoding_requests_total Total number of geocoding requests received.\n")
	sb.WriteString("# TYPE bizatlas_geocoding_requests_total counter\n")
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_requests_total %d\n", snap.TotalRequests))

	sb.WriteString("# HELP bizatlas_geocoding_cache_hits_total Total number of geocoding cache hits.\n")
	sb.WriteString("# TYPE bizatlas_geocoding_cache_hits_total counter\n")
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_cache_hits_total %d\n", snap.CacheHits))

	sb.WriteString("# HELP bizatlas_geocoding_cache_misses_total Total number of geocoding cache misses.\n")
	sb.WriteString("# TYPE bizatlas_geocoding_cache_misses_total counter\n")
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_cache_misses_total %d\n", snap.CacheMisses))

	sb.WriteString("# HELP bizatlas_geocoding_cache_hit_ratio Proportion of requests served from cache.\n")
	sb.WriteString("# TYPE bizatlas_geocoding_cache_hit_ratio gauge\n")
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_cache_hit_ratio %.4f\n", snap.CacheHitRatio))

	sb.WriteString("# HELP bizatlas_geocoding_errors_total Total number of geocoding lookup failures.\n")
	sb.WriteString("# TYPE bizatlas_geocoding_errors_total counter\n")
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_errors_total %d\n", snap.ErrorCount))

	sb.WriteString("# HELP bizatlas_geocoding_duration_ms Geocoding latency percentiles in milliseconds.\n")
	sb.WriteString("# TYPE bizatlas_geocoding_duration_ms gauge\n")
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_duration_ms{quantile=\"0.50\"} %.2f\n", snap.P50LatencyMs))
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_duration_ms{quantile=\"0.90\"} %.2f\n", snap.P90LatencyMs))
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_duration_ms{quantile=\"0.95\"} %.2f\n", snap.P95LatencyMs))
	sb.WriteString(fmt.Sprintf("bizatlas_geocoding_duration_ms{quantile=\"0.99\"} %.2f\n", snap.P99LatencyMs))

	if len(snap.ProviderStats) > 0 {
		sb.WriteString("# HELP bizatlas_geocoding_provider_requests_total Total requests partitioned by upstream provider.\n")
		sb.WriteString("# TYPE bizatlas_geocoding_provider_requests_total counter\n")
		for prov, count := range snap.ProviderStats {
			sb.WriteString(fmt.Sprintf("bizatlas_geocoding_provider_requests_total{provider=\"%s\"} %d\n", prov, count))
		}
	}

	return sb.String()
}
