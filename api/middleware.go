package api

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const apiKeyContextKey contextKey = "api_key"

// KeyInfo represents minimal API key information stored in context.
type KeyInfo struct {
	ID   int
	Name string
}

// ValidateKeyFunc is a function that validates an API key and returns key info.
type ValidateKeyFunc func(ctx context.Context, key string) (keyID int, keyName string, err error)

func extractAPIKey(r *http.Request) string {
	if authHeader := strings.TrimSpace(r.Header.Get("Authorization")); authHeader != "" {
		if len(authHeader) >= 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
			return strings.TrimSpace(authHeader[7:])
		}
		return authHeader
	}
	return strings.TrimSpace(r.Header.Get("X-API-Key"))
}

// KeyAuth middleware validates API keys from Authorization or X-API-Key headers.
func KeyAuth(validateKey ValidateKeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if validateKey == nil {
				http.Error(w, `{"error": "key validator unavailable"}`, http.StatusInternalServerError)
				return
			}

			key := extractAPIKey(r)
			if key == "" {
				http.Error(w, `{"error": "missing api key"}`, http.StatusUnauthorized)
				return
			}

			keyID, keyName, err := validateKey(r.Context(), key)
			if err != nil {
				http.Error(w, `{"error": "invalid api key"}`, http.StatusUnauthorized)
				return
			}

			apiKeyInfo := &KeyInfo{ID: keyID, Name: keyName}
			ctx := context.WithValue(r.Context(), apiKeyContextKey, apiKeyInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// KeyFromContext retrieves the API key info from context.
func KeyFromContext(ctx context.Context) *KeyInfo {
	if key, ok := ctx.Value(apiKeyContextKey).(*KeyInfo); ok {
		return key
	}

	return nil
}
