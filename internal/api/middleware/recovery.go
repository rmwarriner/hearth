package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/rs/zerolog/log"
)

// Recovery catches panics, logs the stack trace at error level, and returns
// a 500 Internal Server Error response. It never echoes the panic value in
// the response body.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Ctx(r.Context()).Error().
					Bytes("stack", debug.Stack()).
					Str("operation", "panic_recovery").
					Msgf("panic: %v", rec)
				respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
