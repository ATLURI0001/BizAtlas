package coords

import "errors"

var ErrCoordCircuitOpen = errors.New("coord circuit open")

// CoordCircuitBreaker trips after maxFail consecutive failures.
type CoordCircuitBreaker struct {
	maxFail int
	fails   int
	open    bool
}

func NewCoordCircuitBreaker(maxFail int) *CoordCircuitBreaker {
	return &CoordCircuitBreaker{maxFail: maxFail}
}

func (c *CoordCircuitBreaker) Call(fn func() error) error {
	if c.open {
		return ErrCoordCircuitOpen
	}
	err := fn()
	if err != nil {
		c.fails++
		if c.fails >= c.maxFail {
			c.open = true
		}
		return err
	}
	c.fails = 0
	return nil
}
