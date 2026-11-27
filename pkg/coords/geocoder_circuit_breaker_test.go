package coords

import (
	"errors"
	"testing"
	"time"
)

func TestGeocoderCircuitBreaker(t *testing.T) {
	cb := NewGeocoderCircuitBreaker(2, 5*time.Second)
	fail := func() error { return errors.New("geocode 429") }
	_ = cb.Call(fail)
	_ = cb.Call(fail)
	if err := cb.Call(func() error { return nil }); err != ErrGeocoderCircuitOpen {
		t.Fatalf("expected ErrGeocoderCircuitOpen, got %v", err)
	}
}
