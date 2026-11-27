package coords

import (
	"errors"
	"time"
)

var ErrGeocoderCircuitOpen = errors.New("geocoder circuit breaker open")

// GeocoderCircuitBreaker guards geocoding API calls against repeated failures.
type GeocoderCircuitBreaker struct {
	threshold int
	cooldown  time.Duration
	failures  int
	openUntil time.Time
}

func NewGeocoderCircuitBreaker(threshold int, cooldown time.Duration) *GeocoderCircuitBreaker {
	return &GeocoderCircuitBreaker{threshold: threshold, cooldown: cooldown}
}

func (cb *GeocoderCircuitBreaker) Call(fn func() error) error {
	if time.Now().Before(cb.openUntil) {
		return ErrGeocoderCircuitOpen
	}
	if err := fn(); err != nil {
		cb.failures++
		if cb.failures >= cb.threshold {
			cb.openUntil = time.Now().Add(cb.cooldown)
			cb.failures = 0
		}
		return err
	}
	cb.failures = 0
	return nil
}
