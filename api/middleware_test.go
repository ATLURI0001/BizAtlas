package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestKeyAuthHeaderParsingAndContext(t *testing.T) {
	okHandler := KeyAuth(func(ctx context.Context, key string) (int, string, error) {
		if key == "secret-123" {
			return 42, "ci-bot", nil
		}
		return 0, "", errors.New("bad key")
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := KeyFromContext(r.Context())
		if info == nil || info.ID != 42 || info.Name != "ci-bot" {
			t.Fatalf("expected KeyInfo in context, got %+v", info)
		}
		w.WriteHeader(http.StatusOK)
	}))

	// Case-insensitive Bearer with extra surrounding whitespace
	req := httptest.NewRequest(http.MethodGet, "/v1/ping", nil)
	req.Header.Set("Authorization", "  bearer   secret-123  ")
	rr := httptest.NewRecorder()
	okHandler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for bearer token, got %d", rr.Code)
	}

	// Blank Authorization falls back to X-API-Key, blank Bearer fails
	reqBlank := httptest.NewRequest(http.MethodGet, "/v1/ping", nil)
	reqBlank.Header.Set("Authorization", "Bearer    ")
	rrBlank := httptest.NewRecorder()
	okHandler.ServeHTTP(rrBlank, reqBlank)
	if rrBlank.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for empty bearer token, got %d", rrBlank.Code)
	}

	// Nil validator returns 500 rather than panicking
	nilHandler := KeyAuth(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	reqNil := httptest.NewRequest(http.MethodGet, "/v1/ping", nil)
	reqNil.Header.Set("X-API-Key", "secret-123")
	rrNil := httptest.NewRecorder()
	nilHandler.ServeHTTP(rrNil, reqNil)
	if rrNil.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 when validateKey is nil, got %d", rrNil.Code)
	}
}
