package coords

import (
	"errors"
	"time"
)

var ErrProxyCircuitOpen = errors.New("proxy pool circuit open")

// ProxyPoolCircuitBreaker guards proxy pool requests against repeated failures.
type ProxyPoolCircuitBreaker struct {
	threshold int
	cooldown  time.Duration
	failures  int
	openUntil time.Time
}

func NewProxyPoolCircuitBreaker(threshold int, cooldown time.Duration) *ProxyPoolCircuitBreaker {
	return &ProxyPoolCircuitBreaker{threshold: threshold, cooldown: cooldown}
}

func (cb *ProxyPoolCircuitBreaker) Call(fn func() error) error {
	if time.Now().Before(cb.openUntil) {
		return ErrProxyCircuitOpen
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
