package coords

import (
	"errors"
	"math"
)

var (
	ErrInvalidDBSCANParams = errors.New("invalid DBSCAN parameters: epsilon must be positive and minPts >= 1")
	ErrEmptyPointSet       = errors.New("cannot cluster empty point set")
)

// PointCluster represents a labeled geospatial cluster.
type PointCluster struct {
	ClusterID int           `json:"cluster_id"`
	Points    []Coordinates `json:"points"`
	Centroid  Coordinates   `json:"centroid"`
	RadiusM   float64       `json:"radius_meters"`
}

// DBSCANResult contains the output partition of spatial clustering.
type DBSCANResult struct {
	Clusters   []PointCluster `json:"clusters"`
	Noise      []Coordinates  `json:"noise"`
	TotalNoise int            `json:"total_noise"`
}

// DBSCAN performs density-based spatial clustering of coordinates based on geodesic Haversine distance in meters.
func DBSCAN(points []Coordinates, epsilonM float64, minPts int) (*DBSCANResult, error) {
	if epsilonM <= 0 || minPts < 1 {
		return nil, ErrInvalidDBSCANParams
	}

	n := len(points)
	if n == 0 {
		return nil, ErrEmptyPointSet
	}

	// Validate coordinates
	for _, p := range points {
		if !p.IsValid() {
			return nil, ErrInvalidCoordinates
		}
	}

	const (
		unvisited = 0
		noise     = -1
	)

	visited := make([]bool, n)
	clusterAssignments := make([]int, n)

	currentClusterID := 1

	for i := 0; i < n; i++ {
		if visited[i] {
			continue
		}
		visited[i] = true

		neighbors := regionQuery(points, i, epsilonM)
		if len(neighbors) < minPts {
			clusterAssignments[i] = noise
		} else {
			// Expand cluster
			clusterAssignments[i] = currentClusterID
			expandCluster(points, i, neighbors, currentClusterID, epsilonM, minPts, visited, clusterAssignments)
			currentClusterID++
		}
	}

	// Group into results
	clusterMap := make(map[int][]Coordinates)
	var noisePoints []Coordinates

	for i, cID := range clusterAssignments {
		if cID == noise {
			noisePoints = append(noisePoints, points[i])
		} else if cID > 0 {
			clusterMap[cID] = append(clusterMap[cID], points[i])
		}
	}

	var clusters []PointCluster
	for cID, pts := range clusterMap {
		centroid := computeCentroid(pts)
		maxRadius := 0.0
		for _, pt := range pts {
			dKm, _ := HaversineDistance(centroid, pt)
			dMeters := dKm * 1000.0
			if dMeters > maxRadius {
				maxRadius = dMeters
			}
		}

		clusters = append(clusters, PointCluster{
			ClusterID: cID,
			Points:    pts,
			Centroid:  centroid,
			RadiusM:   math.Round(maxRadius*100) / 100,
		})
	}

	return &DBSCANResult{
		Clusters:   clusters,
		Noise:      noisePoints,
		TotalNoise: len(noisePoints),
	}, nil
}

func regionQuery(points []Coordinates, targetIdx int, epsilonM float64) []int {
	var neighbors []int
	target := points[targetIdx]

	for j, p := range points {
		dKm, err := HaversineDistance(target, p)
		if err == nil && dKm*1000.0 <= epsilonM {
			neighbors = append(neighbors, j)
		}
	}
	return neighbors
}

func expandCluster(
	points []Coordinates,
	pointIdx int,
	neighbors []int,
	clusterID int,
	epsilonM float64,
	minPts int,
	visited []bool,
	clusterAssignments []int,
) {
	queue := make([]int, len(neighbors))
	copy(queue, neighbors)

	for qIdx := 0; qIdx < len(queue); qIdx++ {
		neighborIdx := queue[qIdx]

		if !visited[neighborIdx] {
			visited[neighborIdx] = true
			subNeighbors := regionQuery(points, neighborIdx, epsilonM)
			if len(subNeighbors) >= minPts {
				// Append new neighbors to queue if not already present
				for _, sn := range subNeighbors {
					found := false
					for _, existing := range queue {
						if existing == sn {
							found = true
							break
						}
					}
					if !found {
						queue = append(queue, sn)
					}
				}
			}
		}

		if clusterAssignments[neighborIdx] <= 0 { // unassigned or noise
			clusterAssignments[neighborIdx] = clusterID
		}
	}
}

func computeCentroid(points []Coordinates) Coordinates {
	if len(points) == 0 {
		return Coordinates{}
	}

	sumLat := 0.0
	sumLon := 0.0
	for _, p := range points {
		sumLat += p.Latitude
		sumLon += p.Longitude
	}

	count := float64(len(points))
	return Coordinates{
		Latitude:  math.Round((sumLat/count)*1e6) / 1e6,
		Longitude: math.Round((sumLon/count)*1e6) / 1e6,
	}
}
