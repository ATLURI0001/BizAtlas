package coords

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// QuadtreeLatencyStats holds percentile distribution for quadtree operations.
type QuadtreeLatencyStats struct {
	SampleCount     int     `json:"sample_count"`
	MinMicroseconds float64 `json:"min_us"`
	P50Microseconds float64 `json:"p50_us"`
	P90Microseconds float64 `json:"p90_us"`
	P99Microseconds float64 `json:"p99_us"`
	MaxMicroseconds float64 `json:"max_us"`
	MeanMicroseconds float64 `json:"mean_us"`
}

// QuadtreeTelemetrySnapshot summarizes overall performance of the spatial quadtree index.
type QuadtreeTelemetrySnapshot struct {
	TotalInsertions     int64                `json:"total_insertions"`
	TotalRangeQueries   int64                `json:"total_range_queries"`
	TotalRadiusQueries  int64                `json:"total_radius_queries"`
	InsertionRatePerSec float64              `json:"insertion_rate_per_sec"`
	InsertLatencyStats  QuadtreeLatencyStats `json:"insert_latency_stats"`
	QueryLatencyStats   QuadtreeLatencyStats `json:"query_latency_stats"`
	ActivePointsCount   int                  `json:"active_points_count"`
}

type timedEvent struct {
	timestamp  time.Time
	durationUs float64
}

// QuadtreeTelemetryRotary maintains sliding window telemetry for spatial quadtree operations.
type QuadtreeTelemetryRotary struct {
	mu             sync.RWMutex
	capacity       int
	insertEvents   []timedEvent
	queryEvents    []timedEvent
	totalInserts   int64
	totalRangeQ    int64
	totalRadiusQ   int64
	tree           *SpatialQuadtree
	firstRecorded  time.Time
}

// NewQuadtreeTelemetryRotary initializes a telemetry rotary bound to a SpatialQuadtree.
func NewQuadtreeTelemetryRotary(tree *SpatialQuadtree, capacity int) *QuadtreeTelemetryRotary {
	if capacity <= 0 {
		capacity = 10000
	}
	return &QuadtreeTelemetryRotary{
		capacity:      capacity,
		insertEvents:  make([]timedEvent, 0, capacity),
		queryEvents:   make([]timedEvent, 0, capacity),
		tree:          tree,
		firstRecorded: time.Now(),
	}
}

// RecordInsert tracks the execution of a point insertion.
func (r *QuadtreeTelemetryRotary) RecordInsert(duration time.Duration) {
	atomic.AddInt64(&r.totalInserts, 1)
	us := float64(duration.Nanoseconds()) / 1e3

	r.mu.Lock()
	defer r.mu.Unlock()

	r.insertEvents = append(r.insertEvents, timedEvent{timestamp: time.Now(), durationUs: us})
	if len(r.insertEvents) > r.capacity {
		r.insertEvents = r.insertEvents[1:]
	}
}

// RecordRangeQuery tracks the execution of a bounding-box range query.
func (r *QuadtreeTelemetryRotary) RecordRangeQuery(duration time.Duration) {
	atomic.AddInt64(&r.totalRangeQ, 1)
	us := float64(duration.Nanoseconds()) / 1e3

	r.mu.Lock()
	defer r.mu.Unlock()

	r.queryEvents = append(r.queryEvents, timedEvent{timestamp: time.Now(), durationUs: us})
	if len(r.queryEvents) > r.capacity {
		r.queryEvents = r.queryEvents[1:]
	}
}

// RecordRadiusQuery tracks the execution of a distance-based radius query.
func (r *QuadtreeTelemetryRotary) RecordRadiusQuery(duration time.Duration) {
	atomic.AddInt64(&r.totalRadiusQ, 1)
	us := float64(duration.Nanoseconds()) / 1e3

	r.mu.Lock()
	defer r.mu.Unlock()

	r.queryEvents = append(r.queryEvents, timedEvent{timestamp: time.Now(), durationUs: us})
	if len(r.queryEvents) > r.capacity {
		r.queryEvents = r.queryEvents[1:]
	}
}

// MonitoredInsert wraps tree.Insert with automatic telemetry tracking.
func (r *QuadtreeTelemetryRotary) MonitoredInsert(p SpatialPoint) (bool, error) {
	t0 := time.Now()
	inserted, err := r.tree.Insert(p)
	r.RecordInsert(time.Since(t0))
	return inserted, err
}

// MonitoredQueryRange wraps tree.QueryRange with automatic telemetry tracking.
func (r *QuadtreeTelemetryRotary) MonitoredQueryRange(rect GeoRect) []SpatialPoint {
	t0 := time.Now()
	res := r.tree.QueryRange(rect)
	r.RecordRangeQuery(time.Since(t0))
	return res
}

// MonitoredQueryRadius wraps tree.QueryRadius with automatic telemetry tracking.
func (r *QuadtreeTelemetryRotary) MonitoredQueryRadius(center Coordinates, radiusKm float64) ([]SpatialPoint, error) {
	t0 := time.Now()
	res, err := r.tree.QueryRadius(center, radiusKm)
	r.RecordRadiusQuery(time.Since(t0))
	return res, err
}

func calculateLatencyStats(events []timedEvent) QuadtreeLatencyStats {
	n := len(events)
	if n == 0 {
		return QuadtreeLatencyStats{}
	}

	samples := make([]float64, n)
	var sum float64
	for i, ev := range events {
		samples[i] = ev.durationUs
		sum += ev.durationUs
	}
	sort.Float64s(samples)

	percentile := func(p float64) float64 {
		idx := int(math.Ceil((p / 100.0) * float64(n))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= n {
			idx = n - 1
		}
		return samples[idx]
	}

	return QuadtreeLatencyStats{
		SampleCount:     n,
		MinMicroseconds: math.Round(samples[0]*100) / 100,
		P50Microseconds: math.Round(percentile(50.0)*100) / 100,
		P90Microseconds: math.Round(percentile(90.0)*100) / 100,
		P99Microseconds: math.Round(percentile(99.0)*100) / 100,
		MaxMicroseconds: math.Round(samples[n-1]*100) / 100,
		MeanMicroseconds: math.Round((sum/float64(n))*100) / 100,
	}
}

// GetSnapshot calculates the overall telemetry profile.
func (r *QuadtreeTelemetryRotary) GetSnapshot() QuadtreeTelemetrySnapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now()
	elapsedSec := now.Sub(r.firstRecorded).Seconds()
	totalIns := atomic.LoadInt64(&r.totalInserts)
	rate := 0.0
	if elapsedSec > 0 {
		rate = float64(totalIns) / elapsedSec
	}

	pointCount := 0
	if r.tree != nil {
		pointCount = r.tree.Count()
	}

	return QuadtreeTelemetrySnapshot{
		TotalInsertions:     totalIns,
		TotalRangeQueries:   atomic.LoadInt64(&r.totalRangeQ),
		TotalRadiusQueries:  atomic.LoadInt64(&r.totalRadiusQ),
		InsertionRatePerSec: math.Round(rate*100) / 100,
		InsertLatencyStats:  calculateLatencyStats(r.insertEvents),
		QueryLatencyStats:   calculateLatencyStats(r.queryEvents),
		ActivePointsCount:   pointCount,
	}
}

// FormatMarkdownReport renders a markdown report for telemetry monitoring.
func (r *QuadtreeTelemetryRotary) FormatMarkdownReport() string {
	snap := r.GetSnapshot()
	ins := snap.InsertLatencyStats
	qry := snap.QueryLatencyStats

	return fmt.Sprintf(`# Spatial Quadtree Index Telemetry Report

| Metric | Value |
| :--- | :--- |
| **Total Insertions** | %d |
| **Total Range Queries** | %d |
| **Total Radius Queries** | %d |
| **Active Points Indexed** | %d |
| **Insertion Velocity** | %.2f points/sec |

### Operation Latency Breakdown (µs)
| Operation | Samples | Min | P50 (Median) | P90 | P99 | Max | Mean |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Insert** | %d | %.2f µs | %.2f µs | %.2f µs | %.2f µs | %.2f µs | %.2f µs |
| **Query** | %d | %.2f µs | %.2f µs | %.2f µs | %.2f µs | %.2f µs | %.2f µs |
`,
		snap.TotalInsertions, snap.TotalRangeQueries, snap.TotalRadiusQueries,
		snap.ActivePointsCount, snap.InsertionRatePerSec,
		ins.SampleCount, ins.MinMicroseconds, ins.P50Microseconds, ins.P90Microseconds, ins.P99Microseconds, ins.MaxMicroseconds, ins.MeanMicroseconds,
		qry.SampleCount, qry.MinMicroseconds, qry.P50Microseconds, qry.P90Microseconds, qry.P99Microseconds, qry.MaxMicroseconds, qry.MeanMicroseconds,
	)
}
