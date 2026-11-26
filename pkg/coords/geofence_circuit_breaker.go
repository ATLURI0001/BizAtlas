package coords

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrGeofenceVertexExplosion = errors.New("geofence: vertex count exceeds safety threshold")
	ErrGeofenceCircuitOpen     = errors.New("geofence: circuit breaker is open, evaluation rejected")
	ErrGeofenceSelfIntersect   = errors.New("geofence: polygon vertices contain invalid geometry or collinear loops")
)

// CircuitState represents the operational mode of the geofence circuit breaker.
type CircuitState string

const (
	CircuitClosed   CircuitState = "CLOSED"
	CircuitOpen     CircuitState = "OPEN"
	CircuitHalfOpen CircuitState = "HALF_OPEN"
)

// GeofencePolygon defines a boundary of geographic coordinate vertices.
type GeofencePolygon struct {
	ID       string
	Vertices []Coordinates
}

// GeofenceCircuitBreaker guards spatial raycasting against algorithmic complexity attacks.
type GeofenceCircuitBreaker struct {
	mu                       sync.RWMutex
	maxVertices              int
	failureThreshold         int
	recoverySuccessThreshold int
	cooldownDuration         time.Duration
	state                    CircuitState
	consecutiveFailures      int
	consecutiveSuccesses     int
	tripTime                 time.Time
}

// NewGeofenceCircuitBreaker creates a circuit breaker for geofence queries.
func NewGeofenceCircuitBreaker(maxVertices int, failureThreshold int, cooldown time.Duration) *GeofenceCircuitBreaker {
	if maxVertices <= 0 {
		maxVertices = 5000
	}
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	if cooldown <= 0 {
		cooldown = 1 * time.Second
	}

	return &GeofenceCircuitBreaker{
		maxVertices:              maxVertices,
		failureThreshold:         failureThreshold,
		recoverySuccessThreshold: 3,
		cooldownDuration:         cooldown,
		state:                    CircuitClosed,
	}
}

// ContainsPoint evaluates if a point lies within a geofence using raycasting under circuit protection.
func (cb *GeofenceCircuitBreaker) ContainsPoint(poly GeofencePolygon, pt Coordinates) (bool, error) {
	cb.mu.Lock()
	now := time.Now()

	// Check if cooldown elapsed for OPEN state
	if cb.state == CircuitOpen && now.Sub(cb.tripTime) >= cb.cooldownDuration {
		cb.state = CircuitHalfOpen
		cb.consecutiveSuccesses = 0
	}

	if cb.state == CircuitOpen {
		cb.mu.Unlock()
		return false, ErrGeofenceCircuitOpen
	}
	cb.mu.Unlock()

	// Validate inputs
	if err := cb.validateInputs(poly, pt); err != nil {
		cb.recordFailure()
		return false, err
	}

	// Execute raycasting (Point-in-Polygon)
	inside := pointInPolygon(poly.Vertices, pt)
	cb.recordSuccess()
	return inside, nil
}

func (cb *GeofenceCircuitBreaker) validateInputs(poly GeofencePolygon, pt Coordinates) error {
	if pt.Latitude < -90 || pt.Latitude > 90 || pt.Longitude < -180 || pt.Longitude > 180 {
		return ErrInvalidCoordinates
	}

	n := len(poly.Vertices)
	if n < 3 {
		return ErrGeofenceSelfIntersect
	}
	if n > cb.maxVertices {
		return ErrGeofenceVertexExplosion
	}

	for _, v := range poly.Vertices {
		if v.Latitude < -90 || v.Latitude > 90 || v.Longitude < -180 || v.Longitude > 180 {
			return ErrInvalidCoordinates
		}
	}
	return nil
}

func (cb *GeofenceCircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures++
	cb.consecutiveSuccesses = 0

	if cb.state == CircuitHalfOpen || cb.consecutiveFailures >= cb.failureThreshold {
		cb.state = CircuitOpen
		cb.tripTime = time.Now()
	}
}

func (cb *GeofenceCircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == CircuitHalfOpen {
		cb.consecutiveSuccesses++
		if cb.consecutiveSuccesses >= cb.recoverySuccessThreshold {
			cb.state = CircuitClosed
			cb.consecutiveFailures = 0
		}
	} else {
		cb.consecutiveFailures = 0
	}
}

// pointInPolygon executes even-odd rule raycasting algorithm.
func pointInPolygon(vertices []Coordinates, pt Coordinates) bool {
	inside := false
	n := len(vertices)
	j := n - 1

	for i := 0; i < n; i++ {
		xi, yi := vertices[i].Longitude, vertices[i].Latitude
		xj, yj := vertices[j].Longitude, vertices[j].Latitude

		intersect := ((yi > pt.Latitude) != (yj > pt.Latitude)) &&
			(pt.Longitude < (xj-xi)*(pt.Latitude-yi)/(yj-yi)+xi)
		if intersect {
			inside = !inside
		}
		j = i
	}

	return inside
}
