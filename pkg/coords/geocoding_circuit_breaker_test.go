package coords

import (
	"errors"
	"testing"
	"time"
)

func TestGeocodingCircuitBreaker_Nominal(t *testing.T) {
	primary := func(c Coordinates) (string, error) {
		return "100 Market St, San Francisco, CA", nil
	}
	fallback := func(c Coordinates) (string, error) {
		return "Cached Downtown SF", nil
	}

	cb, err := NewGeocodingCircuitBreaker(3, 50*time.Millisecond, 2, primary, fallback)
	if err != nil {
		t.Fatalf("failed to init circuit breaker: %v", err)
	}

	pt := Coordinates{Latitude: 37.7749, Longitude: -122.4194}
	res, err := cb.Resolve(pt, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Source != SourcePrimaryAPI {
		t.Fatalf("expected primary source, got %s", res.Source)
	}
	if res.CircuitState != CircuitStateClosed {
		t.Fatalf("expected CLOSED state, got %s", res.CircuitState)
	}
	if res.FallbackUsed {
		t.Fatal("expected FallbackUsed to be false")
	}
}

func TestGeocodingCircuitBreaker_TripToOpenAndFallback(t *testing.T) {
	failPrimary := func(c Coordinates) (string, error) {
		return "", errors.New("upstream HTTP 429 rate limit")
	}
	cachedFallback := func(c Coordinates) (string, error) {
		return "Cached District 4", nil
	}

	cb, _ := NewGeocodingCircuitBreaker(2, 100*time.Millisecond, 2, failPrimary, cachedFallback)
	pt := Coordinates{Latitude: 40.7128, Longitude: -74.0060}
	now := time.Now()

	// 1st failure: state still CLOSED, fallback engaged
	res1, _ := cb.Resolve(pt, now)
	if !res1.FallbackUsed || res1.Source != SourceOfflineCache {
		t.Fatalf("expected offline cache fallback on 1st fail, got %s", res1.Source)
	}
	state, trips, _, _ := cb.GetMetrics()
	if state != CircuitStateClosed || trips != 0 {
		t.Fatalf("expected CLOSED after 1 fail, got %s", state)
	}

	// 2nd failure: trips to OPEN!
	res2, _ := cb.Resolve(pt, now.Add(10*time.Millisecond))
	if !res2.FallbackUsed {
		t.Fatal("expected fallback used on trip")
	}
	state2, trips2, _, _ := cb.GetMetrics()
	if state2 != CircuitStateOpen || trips2 != 1 {
		t.Fatalf("expected OPEN state after 2 fails, got %s (trips=%d)", state2, trips2)
	}
}

func TestGeocodingCircuitBreaker_HalfOpenRecovery(t *testing.T) {
	primaryHealthy := false
	primary := func(c Coordinates) (string, error) {
		if !primaryHealthy {
			return "", errors.New("temporary 503")
		}
		return "Recovered Address", nil
	}
	fallback := func(c Coordinates) (string, error) {
		return "Offline Centroid", nil
	}

	cb, _ := NewGeocodingCircuitBreaker(1, 50*time.Millisecond, 2, primary, fallback)
	pt := Coordinates{Latitude: 51.5074, Longitude: -0.1278}
	t0 := time.Now()

	// 1 fail trips to OPEN
	cb.Resolve(pt, t0)
	state, _, _, _ := cb.GetMetrics()
	if state != CircuitStateOpen {
		t.Fatalf("expected OPEN state, got %s", state)
	}

	// Time advances beyond recovery timeout (60ms > 50ms) and primary heals
	primaryHealthy = true
	tRecovery := t0.Add(60 * time.Millisecond)

	// 1st canary probe: transitions to HALF_OPEN, succeeds
	resProbe1, _ := cb.Resolve(pt, tRecovery)
	if resProbe1.CircuitState != CircuitStateHalfOpen {
		t.Fatalf("expected HALF_OPEN state on probe, got %s", resProbe1.CircuitState)
	}

	// 2nd canary probe: succeeds -> returns to CLOSED!
	resProbe2, _ := cb.Resolve(pt, tRecovery.Add(10*time.Millisecond))
	if resProbe2.Source != SourcePrimaryAPI {
		t.Fatalf("expected primary source on recovery, got %s", resProbe2.Source)
	}
	stateEnd, _, _, _ := cb.GetMetrics()
	if stateEnd != CircuitStateClosed {
		t.Fatalf("expected circuit to close after 2 successes, got %s", stateEnd)
	}
}

func TestGeocodingCircuitBreaker_InvalidCoordinates(t *testing.T) {
	cb, _ := NewGeocodingCircuitBreaker(3, time.Second, 2, nil, nil)
	badPt := Coordinates{Latitude: 95.0, Longitude: 200.0} // Out of bounds

	_, err := cb.Resolve(badPt, time.Now())
	if err == nil {
		t.Fatal("expected error on invalid coordinates")
	}
	if !errors.Is(err, ErrInvalidCoordinates) {
		t.Fatalf("expected ErrInvalidCoordinates, got %v", err)
	}
}
