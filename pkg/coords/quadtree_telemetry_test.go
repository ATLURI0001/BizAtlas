package coords

import (
	"strings"
	"testing"
	"time"
)

func TestQuadtreeTelemetryRotary_Basic(t *testing.T) {
	rect := GlobalEarthRect()
	tree := NewSpatialQuadtree(rect, 4, 8)
	rotary := NewQuadtreeTelemetryRotary(tree, 100)

	// Initial snapshot
	snapInit := rotary.GetSnapshot()
	if snapInit.TotalInsertions != 0 {
		t.Fatalf("expected 0 insertions, got %d", snapInit.TotalInsertions)
	}

	// Insert points with monitored wrapper
	p1 := SpatialPoint{
		ID:    "pt-1",
		Coord: Coordinates{Latitude: 37.7749, Longitude: -122.4194},
	}
	p2 := SpatialPoint{
		ID:    "pt-2",
		Coord: Coordinates{Latitude: 37.7833, Longitude: -122.4167},
	}

	ok, err := rotary.MonitoredInsert(p1)
	if err != nil || !ok {
		t.Fatalf("insert p1 failed: %v", err)
	}

	ok, err = rotary.MonitoredInsert(p2)
	if err != nil || !ok {
		t.Fatalf("insert p2 failed: %v", err)
	}

	snapIns := rotary.GetSnapshot()
	if snapIns.TotalInsertions != 2 {
		t.Fatalf("expected 2 insertions, got %d", snapIns.TotalInsertions)
	}
	if snapIns.ActivePointsCount != 2 {
		t.Fatalf("expected 2 active points, got %d", snapIns.ActivePointsCount)
	}
	if snapIns.InsertLatencyStats.SampleCount != 2 {
		t.Fatalf("expected 2 latency samples, got %d", snapIns.InsertLatencyStats.SampleCount)
	}

	// Range query
	queryBox := GeoRect{
		MinLat: 37.70,
		MaxLat: 37.80,
		MinLon: -122.50,
		MaxLon: -122.40,
	}
	pts := rotary.MonitoredQueryRange(queryBox)
	if len(pts) != 2 {
		t.Fatalf("expected 2 points in query box, got %d", len(pts))
	}

	// Radius query
	center := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	radPts, err := rotary.MonitoredQueryRadius(center, 10.0) // 10 km
	if err != nil || len(radPts) == 0 {
		t.Fatalf("radius query failed: %v, got %d pts", err, len(radPts))
	}

	snapQuery := rotary.GetSnapshot()
	if snapQuery.TotalRangeQueries != 1 {
		t.Fatalf("expected 1 range query, got %d", snapQuery.TotalRangeQueries)
	}
	if snapQuery.TotalRadiusQueries != 1 {
		t.Fatalf("expected 1 radius query, got %d", snapQuery.TotalRadiusQueries)
	}

	// Markdown report
	report := rotary.FormatMarkdownReport()
	if !strings.Contains(report, "# Spatial Quadtree Index Telemetry Report") {
		t.Fatalf("report missing expected header")
	}
	if !strings.Contains(report, "**Total Insertions** | 2") {
		t.Fatalf("report missing total insertions")
	}
}

func TestQuadtreeTelemetryRotary_CapacityEviction(t *testing.T) {
	rect := GlobalEarthRect()
	tree := NewSpatialQuadtree(rect, 4, 8)
	rotary := NewQuadtreeTelemetryRotary(tree, 10) // small capacity

	for i := 0; i < 25; i++ {
		rotary.RecordInsert(time.Duration(i+1) * time.Microsecond)
	}

	snap := rotary.GetSnapshot()
	if snap.TotalInsertions != 25 {
		t.Fatalf("expected 25 total recorded, got %d", snap.TotalInsertions)
	}
	if snap.InsertLatencyStats.SampleCount != 10 {
		t.Fatalf("expected ring buffer capacity of 10, got %d", snap.InsertLatencyStats.SampleCount)
	}
}
