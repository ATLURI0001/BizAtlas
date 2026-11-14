package cryptoext_test

import (
	"strings"
	"testing"

	"github.com/gosom/google-maps-scraper/cryptoext"
)

func TestGenerateBackupCodes(t *testing.T) {
	t.Run("zero or negative count", func(t *testing.T) {
		codes, err := cryptoext.GenerateBackupCodes(0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(codes) != 0 {
			t.Fatalf("expected 0 codes, got %d", len(codes))
		}

		codesNeg, err := cryptoext.GenerateBackupCodes(-5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(codesNeg) != 0 {
			t.Fatalf("expected 0 codes for negative num, got %d", len(codesNeg))
		}
	})

	t.Run("fewer than 10 codes does not panic or out-of-bounds", func(t *testing.T) {
		codes, err := cryptoext.GenerateBackupCodes(4)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(codes) != 4 {
			t.Fatalf("expected 4 codes, got %d", len(codes))
		}
		for i, c := range codes {
			parts := strings.Split(c, "-")
			if len(parts) != 2 || len(parts[0]) != 4 || len(parts[1]) != 4 {
				t.Fatalf("invalid code format at %d: %q", i, c)
			}
		}
	})

	t.Run("more than 10 codes generates full slice", func(t *testing.T) {
		codes, err := cryptoext.GenerateBackupCodes(16)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(codes) != 16 {
			t.Fatalf("expected 16 codes, got %d", len(codes))
		}
		for i, c := range codes {
			if c == "" {
				t.Fatalf("empty code at index %d", i)
			}
		}
	})

	t.Run("HashBackupCodes produces comma separated hashes", func(t *testing.T) {
		codes := []string{"1111-2222", "3333-4444"}
		hashed := cryptoext.HashBackupCodes(codes)
		parts := strings.Split(hashed, ",")
		if len(parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(parts))
		}
		if len(parts[0]) != 64 || len(parts[1]) != 64 {
			t.Fatalf("hashes should be 64-char sha256 hex strings")
		}
	})
}

func TestRandomHexBounds(t *testing.T) {
	if s := cryptoext.GenerateRandomHex(0); s != "" {
		t.Fatalf("expected empty string for length 0, got %q", s)
	}
	if s := cryptoext.GenerateRandomHex(-10); s != "" {
		t.Fatalf("expected empty string for negative length, got %q", s)
	}
	if s := cryptoext.GenerateRandomHex(8); len(s) != 16 {
		t.Fatalf("expected 16 hex chars for 8 bytes, got %d", len(s))
	}

	str, err := cryptoext.GenerateRandomHexString(0)
	if err != nil || str != "" {
		t.Fatalf("expected empty string, got %q (err: %v)", str, err)
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, _, err := cryptoext.GenerateEncryptionKey()
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := "sensitive_production_payload_data_xyz_123"
	encrypted, err := cryptoext.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decrypted, err := cryptoext.Decrypt(encrypted, key)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("mismatched decrypted text: expected %q, got %q", plaintext, decrypted)
	}

	_, err = cryptoext.Decrypt("dG9vc2hvcnQ=", key)
	if err == nil {
		t.Fatalf("expected error on truncated ciphertext, got nil")
	}
}

func TestConstantTimeCompareAndPass(t *testing.T) {
	if !cryptoext.ConstantTimeCompare("secret123", "secret123") {
		t.Fatalf("expected constant time compare match")
	}
	if cryptoext.ConstantTimeCompare("secret123", "secret124") {
		t.Fatalf("expected constant time compare mismatch")
	}

	hash, err := cryptoext.HashPassword("my-pass-phrase")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	if !cryptoext.VerifyPassword("my-pass-phrase", hash) {
		t.Fatalf("password verification failed")
	}
	if cryptoext.VerifyPassword("wrong-pass", hash) {
		t.Fatalf("password verification should have failed on wrong password")
	}
}
