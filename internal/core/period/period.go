package period

import (
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
)

// PeriodID is a UUID string identifying a fiscal period.
type PeriodID string

// FiscalPeriod defines a named accounting period for a household.
// Journal entries cannot be posted to a locked period.
type FiscalPeriod struct {
	ID          PeriodID
	HouseholdID account.HouseholdID
	Name        string
	StartDate   time.Time
	EndDate     time.Time
	LockedAt    *time.Time // nil means the period is open
	CreatedAt   time.Time
}

// IsLocked reports whether the period has been locked.
func (p FiscalPeriod) IsLocked() bool {
	return p.LockedAt != nil
}
