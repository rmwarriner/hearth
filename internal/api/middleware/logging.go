package middleware

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

// RequestLogger logs method, path, status, duration, and request ID for
// every request. It never logs the request body, query parameters, or
// response body to avoid capturing PII.
func RequestLogger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			// Attach the logger to the context so handlers can use log.Ctx(ctx).
			ctx := logger.WithContext(r.Context())
			next.ServeHTTP(rw, r.WithContext(ctx))

			reqID := ""
			if id, ok := hlog.IDFromRequest(r); ok {
				reqID = id.String()
			}
			logger.Info().
				Str("request_id", reqID).
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", rw.status).
				Dur("duration_ms", time.Since(start)).
				Msg("request")
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
