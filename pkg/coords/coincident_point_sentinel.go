package coords

import (
	"errors"
	"fmt"
	"math"
	"sync"
)

var (
	// ErrSingularityThresholdExceeded is returned when too many points occupy identical coordinates.
	ErrSingularityThresholdExceeded = errors.New("coordinate singularity threshold exceeded; potential infinite spatial subdivision")
)

// CoincidentPointConfig defines detection thresholds and protection bounds.
type CoincidentPointConfig struct {
	EpsilonTolerance         float64 `json:"epsilon_tolerance"` // In degrees (~1e-7 ~ 1.1cm)
	MaxClusterDensity        int     `json:"max_cluster_density"`
	EnableJitterDeduplication bool    `json:"enable_jitter_deduplication"`
	JitterScale              float64 `json:"jitter_scale"`
}

// DefaultCoincidentPointConfig provides production safety limits.
func DefaultCoincidentPointConfig() CoincidentPointConfig {
	return CoincidentPointConfig{
		EpsilonTolerance:         1e-7,
		MaxClusterDensity:        100,
		EnableJitterDeduplication: false,
		JitterScale:              1e-8,
	}
}

// CoincidentCluster groups points sharing identical or near-identical coordinates.
type CoincidentCluster struct {
	Centroid Coordinates    `json:"centroid"`
	Points   []SpatialPoint `json:"points"`
	Count    int            `json:"count"`
}

// CoincidenceReport provides audit statistics on analyzed coordinates.
type CoincidenceReport struct {
	TotalPoints         int     `json:"total_points"`
	UniqueCoordinates   int     `json:"unique_coordinates"`
	CoincidentPoints    int     `json:"coincident_points"`
	ClusterCount        int     `json:"cluster_count"`
	MaxClusterSize      int     `json:"max_cluster_size"`
	SingularityDetected bool    `json:"singularity_detected"`
	SingularityLocation Coordinates `json:"singularity_location,omitempty"`
}

// CoincidentPointSentinel protects spatial trees from recursive stack overflow.
type CoincidentPointSentinel struct {
	mu     sync.RWMutex
	config CoincidentPointConfig
}

// NewCoincidentPointSentinel constructs a sentinel with given configuration.
func NewCoincidentPointSentinel(cfg CoincidentPointConfig) *CoincidentPointSentinel {
	if cfg.EpsilonTolerance <= 0 {
		cfg.EpsilonTolerance = 1e-7
	}
	if cfg.MaxClusterDensity <= 0 {
		cfg.MaxClusterDensity = 100
	}
	return &CoincidentPointSentinel{
		config: cfg,
	}
}

// AreCoincident evaluates whether two coordinates are within the epsilon tolerance.
func (s *CoincidentPointSentinel) AreCoincident(a, b Coordinates) bool {
	dLat := math.Abs(a.Latitude - b.Latitude)
	dLon := math.Abs(a.Longitude - b.Longitude)
	return dLat <= s.config.EpsilonTolerance && dLon <= s.config.EpsilonTolerance
}

// AnalyzePoints scans an ingress batch of points and builds a coincidence report.
func (s *CoincidentPointSentinel) AnalyzePoints(points []SpatialPoint) CoincidenceReport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := len(points)
	if n == 0 {
		return CoincidenceReport{}
	}

	clusters := s.clusterPointsLocked(points)
	uniqueCount := len(clusters)
	coincidentCount := 0
	maxCluster := 0
	singularityDetected := false
	var singLoc Coordinates

	for _, c := range clusters {
		if c.Count > maxCluster {
			maxCluster = c.Count
		}
		if c.Count > 1 {
			coincidentCount += c.Count
		}
		if c.Count > s.config.MaxClusterDensity {
			singularityDetected = true
			singLoc = c.Centroid
		}
	}

	return CoincidenceReport{
		TotalPoints:         n,
		UniqueCoordinates:   uniqueCount,
		CoincidentPoints:    coincidentCount,
		ClusterCount:        uniqueCount,
		MaxClusterSize:      maxCluster,
		SingularityDetected: singularityDetected,
		SingularityLocation: singLoc,
	}
}

// SanitizeForTreeInsertion returns points safe for tree insertion, optionally applying micro-jitter
// or collapsing duplicate points to avoid unbounded tree depth recursion.
func (s *CoincidentPointSentinel) SanitizeForTreeInsertion(points []SpatialPoint) ([]SpatialPoint, CoincidenceReport, error) {
	report := s.AnalyzePoints(points)
	if report.SingularityDetected && !s.config.EnableJitterDeduplication {
		return nil, report, ErrSingularityThresholdExceeded
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.config.EnableJitterDeduplication || report.CoincidentPoints == 0 {
		// No modifications needed
		return points, report, nil
	}

	// Apply deterministic micro-jitter to coincident points
	sanitized := make([]SpatialPoint, 0, len(points))
	coordCounts := make(map[string]int)

	for _, p := range points {
		key := coordGridKey(p.Coord, s.config.EpsilonTolerance)
		count := coordCounts[key]
		coordCounts[key]++

		if count > 0 {
			// Apply deterministic spiral jitter
			angle := float64(count) * (2.0 * math.Pi / 8.0)
			radius := s.config.JitterScale * float64((count/8)+1)
			jitteredLat := p.Coord.Latitude + (radius * math.Sin(angle))
			jitteredLon := p.Coord.Longitude + (radius * math.Cos(angle))

			sanitized = append(sanitized, SpatialPoint{
				ID: p.ID,
				Coord: Coordinates{
					Latitude:  jitteredLat,
					Longitude: jitteredLon,
				},
				Metadata: p.Metadata,
			})
		} else {
			sanitized = append(sanitized, p)
		}
	}

	return sanitized, report, nil
}

func (s *CoincidentPointSentinel) clusterPointsLocked(points []SpatialPoint) []CoincidentCluster {
	clusters := make([]CoincidentCluster, 0, len(points))

	for _, p := range points {
		found := false
		for i := range clusters {
			if s.AreCoincident(clusters[i].Centroid, p.Coord) {
				clusters[i].Points = append(clusters[i].Points, p)
				clusters[i].Count++
				found = true
				break
			}
		}
		if !found {
			clusters = append(clusters, CoincidentCluster{
				Centroid: p.Coord,
				Points:   []SpatialPoint{p},
				Count:    1,
			})
		}
	}

	return clusters
}

func coordGridKey(c Coordinates, eps float64) string {
	latBin := int64(math.Round(c.Latitude / eps))
	lonBin := int64(math.Round(c.Longitude / eps))
	return fmt.Sprintf("%d:%d", latBin, lonBin)
}
