package coords

import (
	"errors"
	"math"
	"sync"
)

var (
	// ErrInvalidCellSize indicates cell size must be positive and within reasonable geographic bounds.
	ErrInvalidCellSize = errors.New("coords: cell size must be greater than 0 and less than or equal to 10 degrees")
	// ErrEmptyItemID indicates an item ID cannot be empty.
	ErrEmptyItemID = errors.New("coords: spatial item ID cannot be empty")
)

// SpatialItem represents an indexed point entity with coordinates and optional metadata.
type SpatialItem struct {
	ID    string
	Coord Coordinates
	Data  any
}

// Contains checks if the given coordinate lies inside the bounding box.
func (bb BoundingBox) Contains(c Coordinates) bool {
	return c.Latitude >= bb.MinLat && c.Latitude <= bb.MaxLat &&
		c.Longitude >= bb.MinLon && c.Longitude <= bb.MaxLon
}

// Intersects checks if two bounding boxes overlap.
func (bb BoundingBox) Intersects(other BoundingBox) bool {
	return !(bb.MaxLat < other.MinLat || bb.MinLat > other.MaxLat ||
		bb.MaxLon < other.MinLon || bb.MinLon > other.MaxLon)
}

// SpatialHashGrid provides fast O(1) amortized spatial partitioning and range lookups.
type SpatialHashGrid struct {
	mu       sync.RWMutex
	cellSize float64
	grid     map[int64][]SpatialItem
	items    map[string]Coordinates
}

// NewSpatialHashGrid initializes an empty spatial hash grid with the specified cell size in degrees.
func NewSpatialHashGrid(cellSizeDegrees float64) (*SpatialHashGrid, error) {
	if cellSizeDegrees <= 0.0 || cellSizeDegrees > 10.0 {
		return nil, ErrInvalidCellSize
	}

	return &SpatialHashGrid{
		cellSize: cellSizeDegrees,
		grid:     make(map[int64][]SpatialItem),
		items:    make(map[string]Coordinates),
	}, nil
}

func (shg *SpatialHashGrid) cellKey(lat, lon float64) int64 {
	cx := int32(math.Floor(lon / shg.cellSize))
	cy := int32(math.Floor(lat / shg.cellSize))
	return (int64(cx) << 32) | (int64(cy) & 0xffffffff)
}

// Insert adds an item to the spatial grid.
func (shg *SpatialHashGrid) Insert(item SpatialItem) error {
	if item.ID == "" {
		return ErrEmptyItemID
	}
	if !item.Coord.IsValid() {
		return ErrInvalidCoordinates
	}

	key := shg.cellKey(item.Coord.Latitude, item.Coord.Longitude)

	shg.mu.Lock()
	defer shg.mu.Unlock()

	// If item already exists, remove previous placement
	if prevCoord, exists := shg.items[item.ID]; exists {
		prevKey := shg.cellKey(prevCoord.Latitude, prevCoord.Longitude)
		shg.removeLocked(prevKey, item.ID)
	}

	shg.grid[key] = append(shg.grid[key], item)
	shg.items[item.ID] = item.Coord
	return nil
}

func (shg *SpatialHashGrid) removeLocked(key int64, itemID string) {
	bucket := shg.grid[key]
	for i, it := range bucket {
		if it.ID == itemID {
			shg.grid[key] = append(bucket[:i], bucket[i+1:]...)
			if len(shg.grid[key]) == 0 {
				delete(shg.grid, key)
			}
			break
		}
	}
}

// QueryBBox retrieves all items located within the given bounding box.
func (shg *SpatialHashGrid) QueryBBox(bb BoundingBox) []SpatialItem {
	minLat := math.Min(bb.MinLat, bb.MaxLat)
	maxLat := math.Max(bb.MinLat, bb.MaxLat)
	minLon := math.Min(bb.MinLon, bb.MaxLon)
	maxLon := math.Max(bb.MinLon, bb.MaxLon)

	minCX := int32(math.Floor(minLon / shg.cellSize))
	maxCX := int32(math.Floor(maxLon / shg.cellSize))
	minCY := int32(math.Floor(minLat / shg.cellSize))
	maxCY := int32(math.Floor(maxLat / shg.cellSize))

	shg.mu.RLock()
	defer shg.mu.RUnlock()

	var result []SpatialItem
	seen := make(map[string]struct{})

	for cx := minCX; cx <= maxCX; cx++ {
		for cy := minCY; cy <= maxCY; cy++ {
			key := (int64(cx) << 32) | (int64(cy) & 0xffffffff)
			bucket, ok := shg.grid[key]
			if !ok {
				continue
			}

			for _, it := range bucket {
				if _, already := seen[it.ID]; already {
					continue
				}
				if it.Coord.Latitude >= minLat && it.Coord.Latitude <= maxLat &&
					it.Coord.Longitude >= minLon && it.Coord.Longitude <= maxLon {
					seen[it.ID] = struct{}{}
					result = append(result, it)
				}
			}
		}
	}

	return result
}

// QueryRadius finds all items within radiusMeters from the center coordinate.
func (shg *SpatialHashGrid) QueryRadius(center Coordinates, radiusMeters float64) ([]SpatialItem, error) {
	if !center.IsValid() {
		return nil, ErrInvalidCoordinates
	}
	if radiusMeters < 0 {
		return nil, errors.New("coords: radius cannot be negative")
	}

	// Approximate bounding box in degrees (1 degree lat ~= 111,320m)
	latDelta := radiusMeters / 111320.0
	lonDelta := radiusMeters / (111320.0 * math.Max(0.01, math.Cos(center.Latitude*math.Pi/180.0)))

	bb := BoundingBox{
		MinLat: center.Latitude - latDelta,
		MaxLat: center.Latitude + latDelta,
		MinLon: center.Longitude - lonDelta,
		MaxLon: center.Longitude + lonDelta,
	}

	candidates := shg.QueryBBox(bb)
	var filtered []SpatialItem

	for _, cand := range candidates {
		distKm, err := HaversineDistance(center, cand.Coord)
		if err == nil && distKm*1000.0 <= radiusMeters {
			filtered = append(filtered, cand)
		}
	}

	return filtered, nil
}

// ItemCount returns the total number of items indexed.
func (shg *SpatialHashGrid) ItemCount() int {
	shg.mu.RLock()
	defer shg.mu.RUnlock()
	return len(shg.items)
}

// BucketCount returns the number of active hash buckets.
func (shg *SpatialHashGrid) BucketCount() int {
	shg.mu.RLock()
	defer shg.mu.RUnlock()
	return len(shg.grid)
}

// Clear removes all indexed items and resets the grid.
func (shg *SpatialHashGrid) Clear() {
	shg.mu.Lock()
	defer shg.mu.Unlock()
	shg.grid = make(map[int64][]SpatialItem)
	shg.items = make(map[string]Coordinates)
}

// PolygonBoundingBox calculates the axis-aligned bounding box of a Polygon.
func PolygonBoundingBox(poly *Polygon) BoundingBox {
	if len(poly.Vertices) == 0 {
		return BoundingBox{}
	}

	minLat, maxLat := poly.Vertices[0].Latitude, poly.Vertices[0].Latitude
	minLon, maxLon := poly.Vertices[0].Longitude, poly.Vertices[0].Longitude

	for _, v := range poly.Vertices[1:] {
		if v.Latitude < minLat {
			minLat = v.Latitude
		}
		if v.Latitude > maxLat {
			maxLat = v.Latitude
		}
		if v.Longitude < minLon {
			minLon = v.Longitude
		}
		if v.Longitude > maxLon {
			maxLon = v.Longitude
		}
	}

	return BoundingBox{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLon: minLon,
		MaxLon: maxLon,
	}
}

// PolygonsIntersect determines if two polygons overlap, utilizing broad-phase AABB rejection
// and narrow-phase ray-casting and segment intersection tests.
func PolygonsIntersect(polyA, polyB *Polygon) (bool, error) {
	if polyA == nil || polyB == nil || len(polyA.Vertices) < 3 || len(polyB.Vertices) < 3 {
		return false, ErrDegeneratePolygon
	}

	// 1. Broad phase: AABB check
	bbA := PolygonBoundingBox(polyA)
	bbB := PolygonBoundingBox(polyB)
	if !bbA.Intersects(bbB) {
		return false, nil
	}

	// 2. Narrow phase: Check vertex containment (A inside B or B inside A)
	for _, v := range polyA.Vertices {
		inside, err := polyB.ContainsPoint(v)
		if err != nil {
			return false, err
		}
		if inside {
			return true, nil
		}
	}

	for _, v := range polyB.Vertices {
		inside, err := polyA.ContainsPoint(v)
		if err != nil {
			return false, err
		}
		if inside {
			return true, nil
		}
	}

	// 3. Narrow phase: Edge-edge intersection testing
	for i := 0; i < len(polyA.Vertices); i++ {
		a1 := polyA.Vertices[i]
		a2 := polyA.Vertices[(i+1)%len(polyA.Vertices)]

		for j := 0; j < len(polyB.Vertices); j++ {
			b1 := polyB.Vertices[j]
			b2 := polyB.Vertices[(j+1)%len(polyB.Vertices)]

			if segmentsIntersect(a1, a2, b1, b2) {
				return true, nil
			}
		}
	}

	return false, nil
}

func segmentsIntersect(p1, p2, p3, p4 Coordinates) bool {
	ccw := func(a, b, c Coordinates) float64 {
		return (b.Longitude-a.Longitude)*(c.Latitude-a.Latitude) -
			(b.Latitude-a.Latitude)*(c.Longitude-a.Longitude)
	}

	d1 := ccw(p3, p4, p1)
	d2 := ccw(p3, p4, p2)
	d3 := ccw(p1, p2, p3)
	d4 := ccw(p1, p2, p4)

	return ((d1 > 0 && d2 < 0) || (d1 < 0 && d2 > 0)) &&
		((d3 > 0 && d4 < 0) || (d3 < 0 && d4 > 0))
}
