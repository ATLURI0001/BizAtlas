package coords

import (
	"errors"
	"testing"
	"time"
)

func TestProxyPoolCircuitBreaker(t *testing.T) {
	cb := NewProxyPoolCircuitBreaker(2, 5*time.Second)
	fail := func() error { return errors.New("proxy timeout") }
	_ = cb.Call(fail)
	_ = cb.Call(fail)
	if err := cb.Call(func() error { return nil }); err != ErrProxyCircuitOpen {
		t.Fatalf("expected ErrProxyCircuitOpen, got %v", err)
	}
}
