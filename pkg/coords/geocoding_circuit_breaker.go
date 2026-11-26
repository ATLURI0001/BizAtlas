package coords

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Circuit state definitions.
const (
	CircuitStateClosed   = "CLOSED"
	CircuitStateOpen     = "OPEN"
	CircuitStateHalfOpen = "HALF_OPEN"
)

// Fallback sources.
const (
	SourcePrimaryAPI   = "PRIMARY_API"
	SourceOfflineCache = "OFFLINE_CACHE"
	SourceCentroid     = "FALLBACK_CENTROID"
)

// GeocodingResult represents the output of a reverse geocoding resolution.
type GeocodingResult struct {
	Coords       Coordinates   `json:"coordinates"`
	Address      string        `json:"address"`
	Source       string        `json:"source"`
	CircuitState string        `json:"circuit_state"`
	Latency      time.Duration `json:"latency"`
	FallbackUsed bool          `json:"fallback_used"`
}

// GeocodingProviderFunc defines the function signature for resolving coordinates to an address.
type GeocodingProviderFunc func(coords Coordinates) (string, error)

// GeocodingCircuitBreaker defends reverse geocoding requests against upstream outages and latency spikes.
type GeocodingCircuitBreaker struct {
	mu                       sync.RWMutex
	failureThreshold         int
	recoveryTimeout          time.Duration
	halfOpenSuccessThreshold int
	primaryProvider          GeocodingProviderFunc
	fallbackProvider         GeocodingProviderFunc

	state               string
	consecutiveFailures int
	consecutiveSuccess  int
	lastStateChange     time.Time
	tripCount           int
	totalEvaluated      int
	fallbackInvocations int
}

// NewGeocodingCircuitBreaker initializes a thread-safe geocoding circuit breaker.
func NewGeocodingCircuitBreaker(
	failureThreshold int,
	recoveryTimeout time.Duration,
	halfOpenSuccessThreshold int,
	primary GeocodingProviderFunc,
	fallback GeocodingProviderFunc,
) (*GeocodingCircuitBreaker, error) {
	if failureThreshold < 1 {
		return nil, errors.New("failureThreshold must be >= 1")
	}
	if recoveryTimeout <= 0 {
		return nil, errors.New("recoveryTimeout must be positive")
	}
	if halfOpenSuccessThreshold < 1 {
		return nil, errors.New("halfOpenSuccessThreshold must be >= 1")
	}

	return &GeocodingCircuitBreaker{
		failureThreshold:         failureThreshold,
		recoveryTimeout:          recoveryTimeout,
		halfOpenSuccessThreshold: halfOpenSuccessThreshold,
		primaryProvider:          primary,
		fallbackProvider:         fallback,
		state:                    CircuitStateClosed,
		consecutiveFailures:      0,
		consecutiveSuccess:       0,
		lastStateChange:          time.Now(),
		tripCount:                0,
		totalEvaluated:           0,
		fallbackInvocations:      0,
	}, nil
}

// Resolve processes reverse geocoding with circuit breaker resilience and automatic fallback.
func (cb *GeocodingCircuitBreaker) Resolve(coords Coordinates, now time.Time) (GeocodingResult, error) {
	if !coords.IsValid() {
		return GeocodingResult{}, fmt.Errorf("%w: %s", ErrInvalidCoordinates, coords.String())
	}

	cb.mu.Lock()
	cb.totalEvaluated++

	// Check timeout transition from OPEN to HALF_OPEN
	if cb.state == CircuitStateOpen && now.Sub(cb.lastStateChange) >= cb.recoveryTimeout {
		cb.state = CircuitStateHalfOpen
		cb.consecutiveSuccess = 0
		cb.lastStateChange = now
	}

	currentState := cb.state
	primary := cb.primaryProvider
	fallback := cb.fallbackProvider
	cb.mu.Unlock()

	tStart := time.Now()

	// If CLOSED or HALF_OPEN: try primary
	if currentState == CircuitStateClosed || currentState == CircuitStateHalfOpen {
		if primary != nil {
			addr, err := primary(coords)
			dur := time.Since(tStart)
			if err == nil && addr != "" {
				cb.recordPrimarySuccess(now)
				return GeocodingResult{
					Coords:       coords,
					Address:      addr,
					Source:       SourcePrimaryAPI,
					CircuitState: currentState,
					Latency:      dur,
					FallbackUsed: false,
				}, nil
			}
		}

		// Primary failed
		cb.recordPrimaryFailure(now)
	}

	// OPEN or Primary Failed -> Execute Fallback
	cb.mu.Lock()
	cb.fallbackInvocations++
	finalState := cb.state
	cb.mu.Unlock()

	dur := time.Since(tStart)
	if fallback != nil {
		fAddr, fErr := fallback(coords)
		if fErr == nil && fAddr != "" {
			return GeocodingResult{
				Coords:       coords,
				Address:      fAddr,
				Source:       SourceOfflineCache,
				CircuitState: finalState,
				Latency:      dur,
				FallbackUsed: true,
			}, nil
		}
	}

	// Conservative default centroid
	return GeocodingResult{
		Coords:       coords,
		Address:      fmt.Sprintf("Centroid (%.4f, %.4f)", coords.Latitude, coords.Longitude),
		Source:       SourceCentroid,
		CircuitState: finalState,
		Latency:      dur,
		FallbackUsed: true,
	}, nil
}

func (cb *GeocodingCircuitBreaker) recordPrimarySuccess(now time.Time) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures = 0
	if cb.state == CircuitStateHalfOpen {
		cb.consecutiveSuccess++
		if cb.consecutiveSuccess >= cb.halfOpenSuccessThreshold {
			cb.state = CircuitStateClosed
			cb.consecutiveSuccess = 0
			cb.lastStateChange = now
		}
	}
}

func (cb *GeocodingCircuitBreaker) recordPrimaryFailure(now time.Time) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveFailures++
	if cb.state == CircuitStateHalfOpen {
		// Immediate re-trip on single failure during half-open canary probe
		cb.trip(now)
	} else if cb.state == CircuitStateClosed && cb.consecutiveFailures >= cb.failureThreshold {
		cb.trip(now)
	}
}

func (cb *GeocodingCircuitBreaker) trip(now time.Time) {
	cb.state = CircuitStateOpen
	cb.tripCount++
	cb.lastStateChange = now
	cb.consecutiveSuccess = 0
}

// GetMetrics returns snapshot of circuit metrics.
func (cb *GeocodingCircuitBreaker) GetMetrics() (string, int, int, int) {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state, cb.tripCount, cb.totalEvaluated, cb.fallbackInvocations
}

// Reset clears the circuit state back to CLOSED.
func (cb *GeocodingCircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitStateClosed
	cb.consecutiveFailures = 0
	cb.consecutiveSuccess = 0
	cb.lastStateChange = time.Now()
}
