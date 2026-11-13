package httpext_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gosom/google-maps-scraper/httpext"
)

func TestNew_NilRouter(t *testing.T) {
	srv, err := httpext.New(nil)
	require.Error(t, err)
	require.Nil(t, srv)
	require.Contains(t, err.Error(), "router cannot be nil")
}

func TestNew_ValidRouter(t *testing.T) {
	mux := http.NewServeMux()
	srv, err := httpext.New(mux, httpext.WithAddr(":9999"), nil)
	require.NoError(t, err)
	require.NotNil(t, srv)
}

func TestRequestIDFromContext_NilContext(t *testing.T) {
	require.Equal(t, "", httpext.RequestIDFromContext(nil))
}

func TestLoggingMiddleware_Headers(t *testing.T) {
	t.Run("CF-Ray priority", func(t *testing.T) {
		var capturedID string
		handler := httpext.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = httpext.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("CF-Ray", "ray-123")
		req.Header.Set("X-Request-ID", "req-456")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		require.Equal(t, "ray-123", capturedID)
	})

	t.Run("X-Request-ID fallback", func(t *testing.T) {
		var capturedID string
		handler := httpext.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = httpext.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "req-456")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		require.Equal(t, "req-456", capturedID)
	})

	t.Run("X-Correlation-ID fallback", func(t *testing.T) {
		var capturedID string
		handler := httpext.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = httpext.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Correlation-ID", "corr-789")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		require.Equal(t, "corr-789", capturedID)
	})

	t.Run("Generated request ID fallback", func(t *testing.T) {
		var capturedID string
		handler := httpext.LoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = httpext.RequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		require.NotEmpty(t, capturedID)
		require.Len(t, capturedID, 32)
	})
}
