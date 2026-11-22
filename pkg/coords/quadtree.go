package coords

import (
	"errors"
	"math"
	"sync"
)

var (
	ErrPointOutOfBounds = errors.New("point coordinates lie outside quadtree boundary")
	ErrEmptyQuadtree    = errors.New("quadtree contains no points")
)

// GeoRect defines a 2D bounding rectangle in geographic coordinate space.
type GeoRect struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLon float64 `json:"max_lon"`
}

// Contains checks if a point lies within the closed rectangle boundary.
func (r GeoRect) Contains(c Coordinates) bool {
	return c.Latitude >= r.MinLat && c.Latitude <= r.MaxLat &&
		c.Longitude >= r.MinLon && c.Longitude <= r.MaxLon
}

// Intersects evaluates if two geographic bounding rectangles overlap.
func (r GeoRect) Intersects(other GeoRect) bool {
	return !(other.MinLon > r.MaxLon ||
		other.MaxLon < r.MinLon ||
		other.MinLat > r.MaxLat ||
		other.MaxLat < r.MinLat)
}

// GlobalEarthRect returns a bounding box covering the entire WGS84 coordinate plane.
func GlobalEarthRect() GeoRect {
	return GeoRect{
		MinLat: -90.0,
		MaxLat: 90.0,
		MinLon: -180.0,
		MaxLon: 180.0,
	}
}

// quadtreeNode represents a quadrant within the recursive space partition.
type quadtreeNode struct {
	boundary GeoRect
	capacity int
	depth    int
	maxDepth int
	points   []SpatialPoint
	divided  bool
	nw       *quadtreeNode
	ne       *quadtreeNode
	sw       *quadtreeNode
	se       *quadtreeNode
}

func newQuadtreeNode(boundary GeoRect, capacity, depth, maxDepth int) *quadtreeNode {
	return &quadtreeNode{
		boundary: boundary,
		capacity: capacity,
		depth:    depth,
		maxDepth: maxDepth,
		points:   make([]SpatialPoint, 0, capacity),
	}
}

func (n *quadtreeNode) subdivide() {
	midLat := (n.boundary.MinLat + n.boundary.MaxLat) / 2.0
	midLon := (n.boundary.MinLon + n.boundary.MaxLon) / 2.0
	nextDepth := n.depth + 1

	n.nw = newQuadtreeNode(GeoRect{MinLat: midLat, MaxLat: n.boundary.MaxLat, MinLon: n.boundary.MinLon, MaxLon: midLon}, n.capacity, nextDepth, n.maxDepth)
	n.ne = newQuadtreeNode(GeoRect{MinLat: midLat, MaxLat: n.boundary.MaxLat, MinLon: midLon, MaxLon: n.boundary.MaxLon}, n.capacity, nextDepth, n.maxDepth)
	n.sw = newQuadtreeNode(GeoRect{MinLat: n.boundary.MinLat, MaxLat: midLat, MinLon: n.boundary.MinLon, MaxLon: midLon}, n.capacity, nextDepth, n.maxDepth)
	n.se = newQuadtreeNode(GeoRect{MinLat: n.boundary.MinLat, MaxLat: midLat, MinLon: midLon, MaxLon: n.boundary.MaxLon}, n.capacity, nextDepth, n.maxDepth)

	n.divided = true

	// Redistribute points to children
	oldPoints := n.points
	n.points = make([]SpatialPoint, 0, n.capacity)
	for _, p := range oldPoints {
		n.insertChild(p)
	}
}

func (n *quadtreeNode) insertChild(p SpatialPoint) bool {
	if n.nw.insert(p) {
		return true
	}
	if n.ne.insert(p) {
		return true
	}
	if n.sw.insert(p) {
		return true
	}
	if n.se.insert(p) {
		return true
	}
	return false
}

func (n *quadtreeNode) insert(p SpatialPoint) bool {
	if !n.boundary.Contains(p.Coord) {
		return false
	}

	if len(n.points) < n.capacity || n.depth >= n.maxDepth {
		n.points = append(n.points, p)
		return true
	}

	if !n.divided {
		n.subdivide()
	}

	return n.insertChild(p)
}

func (n *quadtreeNode) queryRange(rect GeoRect, found *[]SpatialPoint) {
	if !n.boundary.Intersects(rect) {
		return
	}

	for _, p := range n.points {
		if rect.Contains(p.Coord) {
			*found = append(*found, p)
		}
	}

	if n.divided {
		n.nw.queryRange(rect, found)
		n.ne.queryRange(rect, found)
		n.sw.queryRange(rect, found)
		n.se.queryRange(rect, found)
	}
}

func (n *quadtreeNode) collectAll(found *[]SpatialPoint) {
	*found = append(*found, n.points...)
	if n.divided {
		n.nw.collectAll(found)
		n.ne.collectAll(found)
		n.sw.collectAll(found)
		n.se.collectAll(found)
	}
}

// SpatialQuadtree coordinates thread-safe hierarchical 2D spatial indexing.
type SpatialQuadtree struct {
	mu         sync.RWMutex
	root       *quadtreeNode
	capacity   int
	maxDepth   int
	totalCount int
}

// NewSpatialQuadtree constructs a spatial quadtree covering a defined boundary.
func NewSpatialQuadtree(boundary GeoRect, capacity, maxDepth int) *SpatialQuadtree {
	if capacity <= 0 {
		capacity = 8
	}
	if maxDepth <= 0 {
		maxDepth = 16
	}
	return &SpatialQuadtree{
		root:     newQuadtreeNode(boundary, capacity, 0, maxDepth),
		capacity: capacity,
		maxDepth: maxDepth,
	}
}

// Insert adds a geographic point into the quadtree index.
func (qt *SpatialQuadtree) Insert(p SpatialPoint) (bool, error) {
	if !p.Coord.IsValid() {
		return false, ErrInvalidCoordinates
	}

	qt.mu.Lock()
	defer qt.mu.Unlock()

	inserted := qt.root.insert(p)
	if !inserted {
		return false, ErrPointOutOfBounds
	}
	qt.totalCount++
	return true, nil
}

// QueryRange finds all points within a rectangular geographic bounding box.
func (qt *SpatialQuadtree) QueryRange(rect GeoRect) []SpatialPoint {
	qt.mu.RLock()
	defer qt.mu.RUnlock()

	var results []SpatialPoint
	qt.root.queryRange(rect, &results)
	return results
}

// QueryRadius finds all points located within radiusKm of the target coordinate center.
func (qt *SpatialQuadtree) QueryRadius(center Coordinates, radiusKm float64) ([]SpatialPoint, error) {
	if !center.IsValid() {
		return nil, ErrInvalidCoordinates
	}
	if radiusKm < 0.0 || math.IsNaN(radiusKm) {
		return nil, ErrInvalidRadius
	}

	minC, maxC, err := center.BoundingBox(radiusKm)
	if err != nil {
		return nil, err
	}

	bbox := GeoRect{
		MinLat: minC.Latitude,
		MaxLat: maxC.Latitude,
		MinLon: minC.Longitude,
		MaxLon: maxC.Longitude,
	}

	candidates := qt.QueryRange(bbox)

	var filtered []SpatialPoint
	for _, p := range candidates {
		d, err := HaversineDistance(center, p.Coord)
		if err == nil && d <= radiusKm {
			filtered = append(filtered, p)
		}
	}

	return filtered, nil
}

// NearestNeighbor finds the closest point to target coordinates using Haversine distance.
func (qt *SpatialQuadtree) NearestNeighbor(target Coordinates) (*SpatialPoint, float64, error) {
	if !target.IsValid() {
		return nil, 0.0, ErrInvalidCoordinates
	}

	qt.mu.RLock()
	defer qt.mu.RUnlock()

	if qt.totalCount == 0 {
		return nil, 0.0, ErrEmptyQuadtree
	}

	var allPoints []SpatialPoint
	qt.root.collectAll(&allPoints)

	var bestPoint *SpatialPoint
	minDist := math.MaxFloat64

	for i := range allPoints {
		p := &allPoints[i]
		d, err := HaversineDistance(target, p.Coord)
		if err == nil && d < minDist {
			minDist = d
			bestPoint = p
		}
	}

	if bestPoint == nil {
		return nil, 0.0, ErrEmptyQuadtree
	}

	return bestPoint, minDist, nil
}

// Count returns the number of indexed points.
func (qt *SpatialQuadtree) Count() int {
	qt.mu.RLock()
	defer qt.mu.RUnlock()
	return qt.totalCount
}

// Clear removes all points from the index.
func (qt *SpatialQuadtree) Clear() {
	qt.mu.Lock()
	defer qt.mu.Unlock()

	boundary := qt.root.boundary
	qt.root = newQuadtreeNode(boundary, qt.capacity, 0, qt.maxDepth)
	qt.totalCount = 0
}
