package coords

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrClusterSpecNil          = errors.New("cluster_contract: cluster spec cannot be nil")
	ErrInvalidMetricType       = errors.New("cluster_contract: unsupported distance metric type")
	ErrClusterRadiusOutOfRange = errors.New("cluster_contract: max radius out of physiological bounds [0.001km, 1000km]")
	ErrMinPointsBelowThreshold = errors.New("cluster_contract: min points must be >= 1")
	ErrEmptyClusterPoints      = errors.New("cluster_contract: cluster points list cannot be empty")
	ErrCoordinateBoundaryBreach= errors.New("cluster_contract: point coordinates out of WGS84 range")
)

type ClusterMetricType string

const (
	ClusterMetricHaversine ClusterMetricType = "HAVERSINE"
	ClusterMetricEuclidean ClusterMetricType = "EUCLIDEAN"
	ClusterMetricManhattan ClusterMetricType = "MANHATTAN"
)

const (
	MinClusterRadiusKm = 0.001  // 1 meter
	MaxClusterRadiusKm = 1000.0 // 1,000 kilometers
)

// ClusterBoundingBox defines the enclosing geographic bounding envelope.
type ClusterBoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLon float64 `json:"max_lon"`
}

// ClusterSpecV2 represents a normalized, validated spatial cluster contract schema.
type ClusterSpecV2 struct {
	Version      string             `json:"version"`
	ClusterID    string             `json:"cluster_id"`
	Metric       ClusterMetricType  `json:"metric"`
	MaxRadiusKm  float64            `json:"max_radius_km"`
	PointCount   int                `json:"point_count"`
	Centroid     Coordinates        `json:"centroid"`
	BoundingBox  ClusterBoundingBox `json:"bounding_box"`
	MaxSpreadKm  float64            `json:"max_spread_km"`
	Points       []Coordinates      `json:"points"`
}

// ValidateAndSanitizeCluster validates spatial coordinates, computes enclosing bbox,
// calculates geographic centroid, and enforces distance metric invariants.
func ValidateAndSanitizeCluster(
	clusterID string,
	metric ClusterMetricType,
	maxRadiusKm float64,
	points []Coordinates,
) (*ClusterSpecV2, error) {
	if clusterID == "" {
		return nil, errors.New("cluster_contract: clusterID cannot be empty")
	}

	if metric != ClusterMetricHaversine && metric != ClusterMetricEuclidean && metric != ClusterMetricManhattan {
		return nil, fmt.Errorf("%w: %s", ErrInvalidMetricType, metric)
	}

	if maxRadiusKm < MinClusterRadiusKm || maxRadiusKm > MaxClusterRadiusKm {
		return nil, fmt.Errorf("%w: got %f", ErrClusterRadiusOutOfRange, maxRadiusKm)
	}

	if len(points) == 0 {
		return nil, ErrEmptyClusterPoints
	}

	minLat := 90.0
	maxLat := -90.0
	minLon := 180.0
	maxLon := -180.0

	var sumLat, sumLon float64

	sanitizedPoints := make([]Coordinates, len(points))
	for i, p := range points {
		if !p.IsValid() {
			return nil, fmt.Errorf("%w: lat=%f, lon=%f", ErrCoordinateBoundaryBreach, p.Latitude, p.Longitude)
		}
		sanitizedPoints[i] = p
		sumLat += p.Latitude
		sumLon += p.Longitude

		if p.Latitude < minLat {
			minLat = p.Latitude
		}
		if p.Latitude > maxLat {
			maxLat = p.Latitude
		}
		if p.Longitude < minLon {
			minLon = p.Longitude
		}
		if p.Longitude > maxLon {
			maxLon = p.Longitude
		}
	}

	n := float64(len(points))
	centroid := Coordinates{
		Latitude:  math.Round((sumLat/n)*1000000) / 1000000,
		Longitude: math.Round((sumLon/n)*1000000) / 1000000,
	}

	// Compute maximum spread from centroid
	maxSpreadKm := 0.0
	for _, p := range sanitizedPoints {
		distKm, _ := HaversineDistance(centroid, p)
		if distKm > maxSpreadKm {
			maxSpreadKm = distKm
		}
	}

	return &ClusterSpecV2{
		Version:     "bizatlas/cluster-v2",
		ClusterID:   clusterID,
		Metric:      metric,
		MaxRadiusKm: maxRadiusKm,
		PointCount:  len(sanitizedPoints),
		Centroid:    centroid,
		BoundingBox: ClusterBoundingBox{
			MinLat: minLat,
			MaxLat: maxLat,
			MinLon: minLon,
			MaxLon: maxLon,
		},
		MaxSpreadKm: math.Round(maxSpreadKm*1000) / 1000,
		Points:      sanitizedPoints,
	}, nil
}

// MigrateClusterSpecV1ToV2 upgrades legacy v1 metric meters parameters to v2 schema.
func MigrateClusterSpecV1ToV2(v1ID string, epsilonMeters float64, points []Coordinates) (*ClusterSpecV2, error) {
	radiusKm := epsilonMeters / 1000.0
	return ValidateAndSanitizeCluster(v1ID, ClusterMetricHaversine, radiusKm, points)
}
