package cryptoext

import (
	"crypto/sha256"
	"errors"
)

// KDFConfig defines options for deriving cryptographic keys from passwords.
type KDFConfig struct {
	SaltLength int
	Iterations int
	KeyLength  int
}

// DefaultKDFConfig returns recommended parameters.
func DefaultKDFConfig() KDFConfig {
	return KDFConfig{
		SaltLength: 16,
		Iterations: 100000,
		KeyLength:  32,
	}
}

// ValidateKDFConfig checks for secure parameter bounds.
func ValidateKDFConfig(c KDFConfig) error {
	if c.SaltLength < 16 {
		return errors.New("salt length must be at least 16 bytes")
	}
	if c.Iterations < 10000 {
		return errors.New("iteration count must be at least 10,000")
	}
	if c.KeyLength < 16 || c.KeyLength > sha256.Size {
		return errors.New("key length out of supported range")
	}
	return nil
}
