package httpext

import (
	"math"
	"time"
)

// BackoffConfig holds exponential backoff parameters for HTTP requests.
type BackoffConfig struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
	Factor    float64
	MaxTries  int
}

// DefaultBackoffConfig returns sensible retry parameters.
func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		BaseDelay: 100 * time.Millisecond,
		MaxDelay:  2 * time.Second,
		Factor:    2.0,
		MaxTries:  3,
	}
}

// DelayForAttempt calculates the backoff duration for a given attempt index.
func (c BackoffConfig) DelayForAttempt(attempt int) time.Duration {
	if attempt <= 0 {
		return c.BaseDelay
	}
	delay := float64(c.BaseDelay) * math.Pow(c.Factor, float64(attempt))
	if delay > float64(c.MaxDelay) {
		return c.MaxDelay
	}
	return time.Duration(delay)
}
