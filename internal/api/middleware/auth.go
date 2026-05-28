package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/hearth-ledger/hearth/internal/auth"
)

type contextKey int

const (
	claimsKey contextKey = iota
	householdIDKey
)

// Authenticate extracts the Bearer token, validates it, and injects the
// *auth.Claims into the request context. Returns 401 on any failure.
func Authenticate(svc *auth.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or malformed Authorization header")
				return
			}
			token := strings.TrimPrefix(header, "Bearer ")
			claims, err := svc.ValidateAccessToken(token)
			if err != nil {
				respondAuthError(w, err)
				return
			}
			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ClaimsFromContext returns the *auth.Claims injected by Authenticate.
func ClaimsFromContext(ctx context.Context) *auth.Claims {
	c, ok := ctx.Value(claimsKey).(*auth.Claims)
	if !ok {
		return nil
	}
	return c
}

// HouseholdIDFromContext returns the verified household ID injected by VerifyHousehold.
func HouseholdIDFromContext(ctx context.Context) string {
	id, ok := ctx.Value(householdIDKey).(string)
	if !ok {
		return ""
	}
	return id
}
