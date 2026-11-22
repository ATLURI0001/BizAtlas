package coords

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
)

const base32Alphabet = "0123456789bcdefghjkmnpqrstuvwxyz"

var (
	base32Lookup = map[byte]int{}
	bits         = []int{16, 8, 4, 2, 1}

	ErrInvalidPrecision = errors.New("geohash precision must be between 1 and 12")
	ErrInvalidGeohash   = errors.New("invalid geohash string or unsupported character")
)

func init() {
	for i := 0; i < len(base32Alphabet); i++ {
		base32Lookup[base32Alphabet[i]] = i
	}
}

// GeoPoint represents a location with metadata stored in the spatial index.
type GeoPoint struct {
	ID        string  `json:"id"`
	Lat       float64 `json:"lat"`
	Lon       float64 `json:"lon"`
	Geohash   string  `json:"geohash"`
	Metadata  string  `json:"metadata,omitempty"`
}

// EncodeGeohash converts latitude and longitude into an interleaved base32 Morton code.
func EncodeGeohash(lat, lon float64, precision int) (string, error) {
	if lat < -90.0 || lat > 90.0 || lon < -180.0 || lon > 180.0 {
		return "", ErrInvalidCoordinates
	}
	if precision < 1 || precision > 12 {
		return "", ErrInvalidPrecision
	}

	latRange := [2]float64{-90.0, 90.0}
	lonRange := [2]float64{-180.0, 180.0}

	var sb strings.Builder
	sb.Grow(precision)

	isEven := true
	bitIndex := 0
	ch := 0

	for sb.Len() < precision {
		if isEven {
			mid := (lonRange[0] + lonRange[1]) / 2.0
			if lon >= mid {
				ch |= bits[bitIndex]
				lonRange[0] = mid
			} else {
				lonRange[1] = mid
			}
		} else {
			mid := (latRange[0] + latRange[1]) / 2.0
			if lat >= mid {
				ch |= bits[bitIndex]
				latRange[0] = mid
			} else {
				latRange[1] = mid
			}
		}

		isEven = !isEven
		if bitIndex < 4 {
			bitIndex++
		} else {
			sb.WriteByte(base32Alphabet[ch])
			bitIndex = 0
			ch = 0
		}
	}

	return sb.String(), nil
}

// DecodeGeohash decodes a geohash into its center latitude, longitude, and bounding box extents.
func DecodeGeohash(hash string) (lat, lon float64, bbox BoundingBox, err error) {
	clean := strings.ToLower(strings.TrimSpace(hash))
	if len(clean) == 0 || len(clean) > 12 {
		return 0, 0, BoundingBox{}, ErrInvalidGeohash
	}

	latRange := [2]float64{-90.0, 90.0}
	lonRange := [2]float64{-180.0, 180.0}
	isEven := true

	for i := 0; i < len(clean); i++ {
		val, exists := base32Lookup[clean[i]]
		if !exists {
			return 0, 0, BoundingBox{}, fmt.Errorf("%w: '%c'", ErrInvalidGeohash, clean[i])
		}

		for maskIdx := 0; maskIdx < 5; maskIdx++ {
			mask := bits[maskIdx]
			if isEven {
				mid := (lonRange[0] + lonRange[1]) / 2.0
				if val&mask != 0 {
					lonRange[0] = mid
				} else {
					lonRange[1] = mid
				}
			} else {
				mid := (latRange[0] + latRange[1]) / 2.0
				if val&mask != 0 {
					latRange[0] = mid
				} else {
					latRange[1] = mid
				}
			}
			isEven = !isEven
		}
	}

	lat = (latRange[0] + latRange[1]) / 2.0
	lon = (lonRange[0] + lonRange[1]) / 2.0
	bbox = BoundingBox{
		MinLat: latRange[0],
		MaxLat: latRange[1],
		MinLon: lonRange[0],
		MaxLon: lonRange[1],
	}
	return lat, lon, bbox, nil
}

// GetGeohashNeighbors calculates the 8 adjacent surrounding geohash cells to eliminate border edge query loss.
func GetGeohashNeighbors(hash string) ([]string, error) {
	lat, lon, bbox, err := DecodeGeohash(hash)
	if err != nil {
		return nil, err
	}

	latDelta := (bbox.MaxLat - bbox.MinLat)
	lonDelta := (bbox.MaxLon - bbox.MinLon)
	prec := len(hash)

	offsets := [][2]float64{
		{latDelta, 0},         // North
		{-latDelta, 0},        // South
		{0, lonDelta},         // East
		{0, -lonDelta},        // West
		{latDelta, lonDelta},   // North-East
		{latDelta, -lonDelta},  // North-West
		{-latDelta, lonDelta},  // South-East
		{-latDelta, -lonDelta}, // South-West
	}

	neighbors := make([]string, 0, 8)
	seen := map[string]struct{}{}

	for _, off := range offsets {
		nLat := math.Max(-90.0, math.Min(90.0, lat+off[0]))
		nLon := lon + off[1]
		if nLon > 180.0 {
			nLon -= 360.0
		} else if nLon < -180.0 {
			nLon += 360.0
		}

		nhash, err := EncodeGeohash(nLat, nLon, prec)
		if err == nil {
			if _, exists := seen[nhash]; !exists && nhash != hash {
				seen[nhash] = struct{}{}
				neighbors = append(neighbors, nhash)
			}
		}
	}

	return neighbors, nil
}

// GeohashSpatialIndex provides fast radius and bounding box spatial lookups via geohash bucket partitioning.
type GeohashSpatialIndex struct {
	mu        sync.RWMutex
	precision int
	points    map[string]GeoPoint
	grid      map[string]map[string]struct{}
}

// NewGeohashSpatialIndex initializes a spatial index with an indexing precision (default 6 ~1.2km cells).
func NewGeohashSpatialIndex(precision int) *GeohashSpatialIndex {
	if precision < 1 || precision > 12 {
		precision = 6
	}
	return &GeohashSpatialIndex{
		precision: precision,
		points:    make(map[string]GeoPoint),
		grid:      make(map[string]map[string]struct{}),
	}
}

// Insert adds a spatial point into the index.
func (idx *GeohashSpatialIndex) Insert(point GeoPoint) error {
	if point.ID == "" {
		return errors.New("point ID cannot be empty")
	}

	hash, err := EncodeGeohash(point.Lat, point.Lon, idx.precision)
	if err != nil {
		return err
	}
	point.Geohash = hash

	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.points[point.ID] = point
	if _, exists := idx.grid[hash]; !exists {
		idx.grid[hash] = make(map[string]struct{})
	}
	idx.grid[hash][point.ID] = struct{}{}
	return nil
}

// SearchRadius retrieves all points within radiusKm of the origin (lat, lon).
func (idx *GeohashSpatialIndex) SearchRadius(originLat, originLon float64, radiusKm float64) ([]GeoPoint, error) {
	if radiusKm <= 0 {
		return nil, errors.New("search radius must be > 0")
	}

	centerHash, err := EncodeGeohash(originLat, originLon, idx.precision)
	if err != nil {
		return nil, err
	}

	neighbors, err := GetGeohashNeighbors(centerHash)
	if err != nil {
		return nil, err
	}

	candidateHashes := append([]string{centerHash}, neighbors...)

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var results []GeoPoint
	origin := Coordinates{Latitude: originLat, Longitude: originLon}

	seenIDs := make(map[string]struct{})
	for _, h := range candidateHashes {
		pointIDs, exists := idx.grid[h]
		if !exists {
			continue
		}

		for id := range pointIDs {
			if _, seen := seenIDs[id]; seen {
				continue
			}
			seenIDs[id] = struct{}{}

			pt := idx.points[id]
			dist, distErr := HaversineDistance(origin, Coordinates{Latitude: pt.Lat, Longitude: pt.Lon})
			if distErr == nil && dist <= radiusKm {
				results = append(results, pt)
			}
		}
	}

	return results, nil
}
