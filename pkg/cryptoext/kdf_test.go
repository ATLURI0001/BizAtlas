package cryptoext

import "testing"

func TestValidateKDFConfig(t *testing.T) {
	def := DefaultKDFConfig()
	if err := ValidateKDFConfig(def); err != nil {
		t.Fatalf("expected default KDF config to be valid, got: %v", err)
	}

	// Short salt
	badSalt := def
	badSalt.SaltLength = 8
	if err := ValidateKDFConfig(badSalt); err == nil {
		t.Fatal("expected error for salt < 16 bytes")
	}

	// Low iterations
	badIter := def
	badIter.Iterations = 1000
	if err := ValidateKDFConfig(badIter); err == nil {
		t.Fatal("expected error for iterations < 10000")
	}

	// Bad key length
	badKey := def
	badKey.KeyLength = 8
	if err := ValidateKDFConfig(badKey); err == nil {
		t.Fatal("expected error for key length < 16")
	}
}
