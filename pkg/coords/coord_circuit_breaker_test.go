package coords

import (
	"errors"
	"testing"
)

func TestCoordCircuitBreaker(t *testing.T) {
	cb := NewCoordCircuitBreaker(2)
	boom := errors.New("fail")
	_ = cb.Call(func() error { return boom })
	_ = cb.Call(func() error { return boom })
	if err := cb.Call(func() error { return nil }); err != ErrCoordCircuitOpen {
		t.Fatalf("expected circuit open, got %v", err)
	}
}
