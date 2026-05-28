package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// VerifyHousehold checks that the {householdId} URL parameter matches the
// household ID embedded in the JWT claims. Returns 403 on mismatch.
// Must run after Authenticate (depends on claimsKey being set).
func VerifyHousehold(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlHID := chi.URLParam(r, "householdId")
		claims := ClaimsFromContext(r.Context())
		if claims == nil {
			respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
			return
		}
		if urlHID != claims.HouseholdID {
			respondError(w, http.StatusForbidden, "FORBIDDEN", "you do not have access to this household")
			return
		}
		ctx := context.WithValue(r.Context(), householdIDKey, urlHID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
