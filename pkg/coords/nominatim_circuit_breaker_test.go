package coords

import "testing"

func TestNominatimCircuitBreaker(t *testing.T) {
	b := NewNominatimCircuitBreaker(2)
	b.RecordFailure()
	b.RecordFailure()
	if !b.IsOpen() {
		t.Fatal("expected circuit breaker to open")
	}
}
