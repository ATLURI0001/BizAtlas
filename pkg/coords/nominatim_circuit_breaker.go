package coords

// NominatimCircuitBreaker trips after consecutive geocoder failures.
type NominatimCircuitBreaker struct {
	failures  int
	threshold int
	open      bool
}

func NewNominatimCircuitBreaker(threshold int) *NominatimCircuitBreaker {
	return &NominatimCircuitBreaker{threshold: threshold}
}

func (b *NominatimCircuitBreaker) RecordFailure() {
	b.failures++
	if b.failures >= b.threshold {
		b.open = true
	}
}

func (b *NominatimCircuitBreaker) IsOpen() bool {
	return b.open
}
