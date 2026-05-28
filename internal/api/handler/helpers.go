package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/hearth-ledger/hearth/internal/api/middleware"
	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/gaap"
	"github.com/hearth-ledger/hearth/internal/core/period"
)

// newID returns a 16-byte random hex string suitable for use as an entity ID.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// parseDecimal parses a decimal string. Returns an error if the string is not
// a valid decimal representation.
func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

// ptrStr dereferences a *string, returning "" if nil.
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// claimsFromRequest extracts the member ID from JWT claims injected by
// Authenticate middleware, returning "" if claims are absent.
func claimsFromRequest(r *http.Request) string {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		return ""
	}
	return claims.MemberID
}

// timeNowUTC returns the current UTC time.
func timeNowUTC() time.Time {
	return time.Now().UTC()
}

// buildValidationContext constructs a gaap.ValidationContext from store data.
func buildValidationContext(accounts []account.Account, periods []period.FiscalPeriod) gaap.ValidationContext {
	known := make(map[account.AccountID]account.HouseholdID, len(accounts))
	for _, a := range accounts {
		known[a.ID] = a.HouseholdID
	}
	locked := make([]gaap.LockedPeriod, 0, len(periods))
	for _, p := range periods {
		locked = append(locked, gaap.LockedPeriod{Start: p.StartDate, End: p.EndDate})
	}
	return gaap.ValidationContext{KnownAccounts: known, LockedPeriods: locked}
}
