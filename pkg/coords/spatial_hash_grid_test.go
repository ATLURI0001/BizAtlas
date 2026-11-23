package coords

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func TestSpatialHashGrid_BasicOperations(t *testing.T) {
	grid, err := NewSpatialHashGrid(0.01) // ~1.1km cell
	if err != nil {
		t.Fatalf("unexpected error creating grid: %v", err)
	}

	item1 := SpatialItem{
		ID:    "loc-1",
		Coord: Coordinates{Latitude: 37.7749, Longitude: -122.4194}, // San Francisco
		Data:  "SF Downtown",
	}
	item2 := SpatialItem{
		ID:    "loc-2",
		Coord: Coordinates{Latitude: 37.8044, Longitude: -122.2712}, // Oakland
		Data:  "Oakland Center",
	}

	if err := grid.Insert(item1); err != nil {
		t.Fatalf("failed to insert item1: %v", err)
	}
	if err := grid.Insert(item2); err != nil {
		t.Fatalf("failed to insert item2: %v", err)
	}

	if grid.ItemCount() != 2 {
		t.Fatalf("expected 2 items, got %d", grid.ItemCount())
	}

	// Query bounding box containing only SF
	sfBBox := BoundingBox{
		MinLat: 37.70,
		MaxLat: 37.79,
		MinLon: -122.50,
		MaxLon: -122.35,
	}
	results := grid.QueryBBox(sfBBox)
	if len(results) != 1 || results[0].ID != "loc-1" {
		t.Fatalf("expected 1 result (loc-1), got %d (%v)", len(results), results)
	}

	// Query radius from SF (within 500m)
	radiusResults, err := grid.QueryRadius(item1.Coord, 500.0)
	if err != nil {
		t.Fatalf("unexpected error in QueryRadius: %v", err)
	}
	if len(radiusResults) != 1 || radiusResults[0].ID != "loc-1" {
		t.Fatalf("expected loc-1 within 500m, got %d results", len(radiusResults))
	}

	// Update item1 location
	item1Updated := SpatialItem{
		ID:    "loc-1",
		Coord: Coordinates{Latitude: 37.7750, Longitude: -122.4195},
		Data:  "Updated SF",
	}
	if err := grid.Insert(item1Updated); err != nil {
		t.Fatalf("failed to update item1: %v", err)
	}
	if grid.ItemCount() != 2 {
		t.Fatalf("expected still 2 items after update, got %d", grid.ItemCount())
	}

	// Clear grid
	grid.Clear()
	if grid.ItemCount() != 0 || grid.BucketCount() != 0 {
		t.Fatalf("expected grid to be empty after Clear()")
	}
}

func TestSpatialHashGrid_ValidationErrors(t *testing.T) {
	_, err := NewSpatialHashGrid(0.0)
	if err != ErrInvalidCellSize {
		t.Fatalf("expected ErrInvalidCellSize, got %v", err)
	}

	_, err = NewSpatialHashGrid(15.0)
	if err != ErrInvalidCellSize {
		t.Fatalf("expected ErrInvalidCellSize for oversized cell, got %v", err)
	}

	grid, _ := NewSpatialHashGrid(0.05)
	if err := grid.Insert(SpatialItem{ID: ""}); err != ErrEmptyItemID {
		t.Fatalf("expected ErrEmptyItemID, got %v", err)
	}

	if err := grid.Insert(SpatialItem{ID: "bad-coord", Coord: Coordinates{Latitude: 100.0, Longitude: 0.0}}); err != ErrInvalidCoordinates {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
	}
}

func TestPolygonsIntersect(t *testing.T) {
	// Poly A: Square [0, 0] to [2, 2]
	polyA, _ := NewPolygon([]Coordinates{
		{Latitude: 0, Longitude: 0},
		{Latitude: 2, Longitude: 0},
		{Latitude: 2, Longitude: 2},
		{Latitude: 0, Longitude: 2},
	})

	// Poly B: Overlapping square [1, 1] to [3, 3]
	polyB, _ := NewPolygon([]Coordinates{
		{Latitude: 1, Longitude: 1},
		{Latitude: 3, Longitude: 1},
		{Latitude: 3, Longitude: 3},
		{Latitude: 1, Longitude: 3},
	})

	// Poly C: Disjoint square [10, 10] to [12, 12]
	polyC, _ := NewPolygon([]Coordinates{
		{Latitude: 10, Longitude: 10},
		{Latitude: 12, Longitude: 10},
		{Latitude: 12, Longitude: 12},
		{Latitude: 10, Longitude: 12},
	})

	// Poly D: Fully enclosed inside Poly A [0.5, 0.5] to [1.5, 1.5]
	polyD, _ := NewPolygon([]Coordinates{
		{Latitude: 0.5, Longitude: 0.5},
		{Latitude: 1.5, Longitude: 0.5},
		{Latitude: 1.5, Longitude: 1.5},
		{Latitude: 0.5, Longitude: 1.5},
	})

	// Overlapping
	intersectAB, err := PolygonsIntersect(polyA, polyB)
	if err != nil || !intersectAB {
		t.Fatalf("expected polyA and polyB to intersect, got %v (err: %v)", intersectAB, err)
	}

	// Disjoint (broad-phase AABB rejection)
	intersectAC, err := PolygonsIntersect(polyA, polyC)
	if err != nil || intersectAC {
		t.Fatalf("expected polyA and polyC to be disjoint, got %v (err: %v)", intersectAC, err)
	}

	// Enclosed containment
	intersectAD, err := PolygonsIntersect(polyA, polyD)
	if err != nil || !intersectAD {
		t.Fatalf("expected polyA and polyD to intersect (containment), got %v", intersectAD)
	}
}

func TestSpatialHashGrid_Concurrency(t *testing.T) {
	grid, _ := NewSpatialHashGrid(0.02)
	var wg sync.WaitGroup

	// Concurrent inserters
	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				item := SpatialItem{
					ID:    fmt.Sprintf("w%d-item-%d", workerID, i),
					Coord: Coordinates{Latitude: 37.0 + float64(i)*0.001, Longitude: -122.0 + float64(workerID)*0.001},
				}
				_ = grid.Insert(item)
			}
		}(w)
	}

	// Concurrent readers
	for r := 0; r < 10; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				_ = grid.QueryBBox(BoundingBox{MinLat: 37.0, MaxLat: 37.05, MinLon: -122.05, MaxLon: -121.95})
			}
		}()
	}

	wg.Wait()
	if grid.ItemCount() != 500 {
		t.Fatalf("expected 500 items after concurrent insertion, got %d", grid.ItemCount())
	}
}

// -----------------------------------------------------------------------
// BENCHMARKS (Style D)
// -----------------------------------------------------------------------

func BenchmarkSpatialHashGrid_Insert(b *testing.B) {
	grid, _ := NewSpatialHashGrid(0.01)
	r := rand.New(rand.NewSource(42))

	items := make([]SpatialItem, b.N)
	for i := 0; i < b.N; i++ {
		items[i] = SpatialItem{
			ID: fmt.Sprintf("bench-item-%d", i),
			Coord: Coordinates{
				Latitude:  37.0 + r.Float64()*1.0,
				Longitude: -122.0 + r.Float64()*1.0,
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = grid.Insert(items[i])
	}
}

func BenchmarkSpatialHashGrid_QueryBBox(b *testing.B) {
	grid, _ := NewSpatialHashGrid(0.01)
	r := rand.New(rand.NewSource(42))

	// Pre-populate with 10,000 coordinates
	for i := 0; i < 10000; i++ {
		_ = grid.Insert(SpatialItem{
			ID: fmt.Sprintf("init-%d", i),
			Coord: Coordinates{
				Latitude:  37.0 + r.Float64()*1.0,
				Longitude: -122.0 + r.Float64()*1.0,
			},
		})
	}

	queryBox := BoundingBox{
		MinLat: 37.4,
		MaxLat: 37.6,
		MinLon: -122.3,
		MaxLon: -122.1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res := grid.QueryBBox(queryBox)
		_ = res
	}
}

func BenchmarkPolygonsIntersect_BroadRejection(b *testing.B) {
	polyA, _ := NewPolygon([]Coordinates{
		{Latitude: 0, Longitude: 0},
		{Latitude: 1, Longitude: 0},
		{Latitude: 1, Longitude: 1},
		{Latitude: 0, Longitude: 1},
	})
	polyB, _ := NewPolygon([]Coordinates{
		{Latitude: 10, Longitude: 10},
		{Latitude: 11, Longitude: 10},
		{Latitude: 11, Longitude: 11},
		{Latitude: 10, Longitude: 11},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = PolygonsIntersect(polyA, polyB)
	}
}

func BenchmarkPolygonsIntersect_NarrowIntersection(b *testing.B) {
	polyA, _ := NewPolygon([]Coordinates{
		{Latitude: 0, Longitude: 0},
		{Latitude: 2, Longitude: 0},
		{Latitude: 2, Longitude: 2},
		{Latitude: 0, Longitude: 2},
	})
	polyB, _ := NewPolygon([]Coordinates{
		{Latitude: 1, Longitude: 1},
		{Latitude: 3, Longitude: 1},
		{Latitude: 3, Longitude: 3},
		{Latitude: 1, Longitude: 3},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = PolygonsIntersect(polyA, polyB)
	}
}
