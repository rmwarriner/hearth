package middleware

import (
	"encoding/json"
	"net/http"

	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

type errorBody struct {
	Code    string   `json:"code"`
	Message string   `json:"message"`
	Hints   []string `json:"hints,omitempty"`
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorBody{Code: code, Message: message}); err != nil {
		// Best-effort: if we can't write the error body the status code was
		// already sent, so there's nothing more we can do.
		_ = err
	}
}

func respondAuthError(w http.ResponseWriter, err error) {
	var he *hearth.HearthError
	if ok := isHearthError(err, &he); ok {
		status := ErrorCodeToStatus(he.Code)
		respondError(w, status, string(he.Code), he.Message)
		return
	}
	respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication failed")
}

func isHearthError(err error, out **hearth.HearthError) bool {
	if he, ok := err.(*hearth.HearthError); ok {
		*out = he
		return true
	}
	return false
}

// ErrorCodeToStatus maps a HearthError code to an HTTP status code.
// Exported so handlers can reuse the same mapping.
func ErrorCodeToStatus(code hearth.ErrorCode) int {
	switch code {
	case hearth.ErrUnauthorized, hearth.ErrTokenInvalid, hearth.ErrTokenExpired, hearth.ErrTokenRevoked:
		return http.StatusUnauthorized
	case hearth.ErrForbidden:
		return http.StatusForbidden
	case hearth.ErrNotFound, hearth.ErrAccountNotFound, hearth.ErrHouseholdNotFound, hearth.ErrMemberNotFound:
		return http.StatusNotFound
	case hearth.ErrConflict:
		return http.StatusConflict
	case hearth.ErrInvalidRequest, hearth.ErrGAAPBalance, hearth.ErrGAAPMinPostings, hearth.ErrGAAPLockedPeriod:
		return http.StatusUnprocessableEntity
	case hearth.ErrRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
