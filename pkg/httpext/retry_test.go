package httpext

import (
	"testing"
	"time"
)

func TestBackoffConfig_DelayForAttempt(t *testing.T) {
	cfg := DefaultBackoffConfig() // Base: 100ms, Factor: 2.0, Max: 2s

	// Attempt 0
	d0 := cfg.DelayForAttempt(0)
	if d0 != 100*time.Millisecond {
		t.Fatalf("expected attempt 0 to return base delay 100ms, got %v", d0)
	}

	// Attempt 1 -> 100ms * 2^1 = 200ms
	d1 := cfg.DelayForAttempt(1)
	if d1 != 200*time.Millisecond {
		t.Fatalf("expected attempt 1 to return 200ms, got %v", d1)
	}

	// Attempt 2 -> 100ms * 2^2 = 400ms
	d2 := cfg.DelayForAttempt(2)
	if d2 != 400*time.Millisecond {
		t.Fatalf("expected attempt 2 to return 400ms, got %v", d2)
	}

	// High attempt -> capped at MaxDelay (2s)
	dHigh := cfg.DelayForAttempt(10)
	if dHigh != 2*time.Second {
		t.Fatalf("expected capped delay 2s, got %v", dHigh)
	}
}
