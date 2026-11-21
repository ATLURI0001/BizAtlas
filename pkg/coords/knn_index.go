package coords

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
)

var (
	ErrInvalidK = errors.New("knn_index: k must be greater than zero")
)

// SpatialPoint represents a geospatial entity with coordinates and optional metadata.
type SpatialPoint struct {
	ID       string            `json:"id"`
	Coord    Coordinates       `json:"coord"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NeighborResult represents a nearest neighbor match with distance in kilometers.
type NeighborResult struct {
	Point      SpatialPoint `json:"point"`
	DistanceKm float64      `json:"distance_km"`
}

type gridCellKey struct {
	latIdx int
	lonIdx int
}

// SpatialGridIndex provides high-performance spatial partitioning for k-NN queries.
type SpatialGridIndex struct {
	cellSizeDeg float64
	mu          sync.RWMutex
	cells       map[gridCellKey][]SpatialPoint
	totalPoints int
}

// NewSpatialGridIndex initializes a grid index with specified cell size in degrees.
func NewSpatialGridIndex(cellSizeDeg float64) *SpatialGridIndex {
	if cellSizeDeg <= 0.0 || math.IsNaN(cellSizeDeg) {
		cellSizeDeg = 0.05 // default ~5.5km per cell
	}
	return &SpatialGridIndex{
		cellSizeDeg: cellSizeDeg,
		cells:       make(map[gridCellKey][]SpatialPoint),
	}
}

func (idx *SpatialGridIndex) cellKey(c Coordinates) gridCellKey {
	return gridCellKey{
		latIdx: int(math.Floor(c.Latitude / idx.cellSizeDeg)),
		lonIdx: int(math.Floor(c.Longitude / idx.cellSizeDeg)),
	}
}

// Insert adds a single point to the spatial grid.
func (idx *SpatialGridIndex) Insert(p SpatialPoint) error {
	if !p.Coord.IsValid() {
		return ErrInvalidCoordinates
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	k := idx.cellKey(p.Coord)
	idx.cells[k] = append(idx.cells[k], p)
	idx.totalPoints++
	return nil
}

// BatchInsert inserts multiple points in bulk.
func (idx *SpatialGridIndex) BatchInsert(points []SpatialPoint) error {
	for _, p := range points {
		if !p.Coord.IsValid() {
			return fmt.Errorf("%w: invalid point id %s", ErrInvalidCoordinates, p.ID)
		}
	}

	idx.mu.Lock()
	defer idx.mu.Unlock()

	for _, p := range points {
		k := idx.cellKey(p.Coord)
		idx.cells[k] = append(idx.cells[k], p)
		idx.totalPoints++
	}
	return nil
}

// Count returns the total number of indexed points.
func (idx *SpatialGridIndex) Count() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.totalPoints
}

// RangeSearch finds all points within radiusKm of origin.
func (idx *SpatialGridIndex) RangeSearch(origin Coordinates, radiusKm float64) ([]NeighborResult, error) {
	if !origin.IsValid() {
		return nil, ErrInvalidCoordinates
	}
	if radiusKm <= 0.0 || math.IsNaN(radiusKm) {
		return nil, ErrInvalidRadius
	}

	// Approximate bounding degree span
	latDeltaDeg := (radiusKm / EarthRadiusKm) * (180.0 / math.Pi)
	cosLat := math.Cos(degToRad(origin.Latitude))
	if math.Abs(cosLat) < 1e-6 {
		cosLat = 1e-6
	}
	lonDeltaDeg := latDeltaDeg / cosLat

	minLat := math.Max(-90.0, origin.Latitude-latDeltaDeg)
	maxLat := math.Min(90.0, origin.Latitude+latDeltaDeg)
	minLon := math.Max(-180.0, origin.Longitude-lonDeltaDeg)
	maxLon := math.Min(180.0, origin.Longitude+lonDeltaDeg)

	minLatIdx := int(math.Floor(minLat / idx.cellSizeDeg))
	maxLatIdx := int(math.Floor(maxLat / idx.cellSizeDeg))
	minLonIdx := int(math.Floor(minLon / idx.cellSizeDeg))
	maxLonIdx := int(math.Floor(maxLon / idx.cellSizeDeg))

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var results []NeighborResult
	for latI := minLatIdx; latI <= maxLatIdx; latI++ {
		for lonI := minLonIdx; lonI <= maxLonIdx; lonI++ {
			pts, ok := idx.cells[gridCellKey{latIdx: latI, lonIdx: lonI}]
			if !ok {
				continue
			}
			for _, p := range pts {
				dKm, err := HaversineDistance(origin, p.Coord)
				if err == nil && dKm <= radiusKm {
					results = append(results, NeighborResult{
						Point:      p,
						DistanceKm: dKm,
					})
				}
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DistanceKm < results[j].DistanceKm
	})

	return results, nil
}

// NearestK finds the k-nearest points within an optional maxRadiusKm constraint (0 for unbounded).
func (idx *SpatialGridIndex) NearestK(origin Coordinates, k int, maxRadiusKm float64) ([]NeighborResult, error) {
	if !origin.IsValid() {
		return nil, ErrInvalidCoordinates
	}
	if k <= 0 {
		return nil, ErrInvalidK
	}

	effectiveRadius := maxRadiusKm
	if effectiveRadius <= 0.0 {
		// Earth half circumference as upper bound (~20,000 km)
		effectiveRadius = EarthRadiusKm * math.Pi
	}

	matches, err := idx.RangeSearch(origin, effectiveRadius)
	if err != nil {
		return nil, err
	}

	if len(matches) > k {
		matches = matches[:k]
	}

	return matches, nil
}
