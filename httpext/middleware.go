package httpext

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/gosom/google-maps-scraper/log"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// LoggingMiddleware logs HTTP requests with timing, status, and client info.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestID := strings.TrimSpace(r.Header.Get("CF-Ray"))
		if requestID == "" {
			requestID = strings.TrimSpace(r.Header.Get("X-Request-ID"))
		}
		if requestID == "" {
			requestID = strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
		}
		if requestID == "" {
			requestID = generateRequestID()
		}

		ctx := setContextRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		clientIP := strings.TrimSpace(r.Header.Get("CF-Connecting-IP"))
		if clientIP == "" {
			if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
				clientIP = strings.TrimSpace(strings.Split(xff, ",")[0])
			}
		}
		if clientIP == "" {
			clientIP = strings.TrimSpace(r.Header.Get("X-Real-IP"))
		}
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			log.Info("request",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", clientIP,
				"user_agent", r.UserAgent(),
				"status", ww.Status(),
				"duration_ms", float64(time.Since(start).Nanoseconds())/1e6,
				"bytes", ww.BytesWritten(),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)

	return hex.EncodeToString(b)
}

func setContextRequestID(ctx context.Context, value string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, requestIDKey, value)
}

// RequestIDFromContext retrieves the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}

	return ""
}
