package envelope

import (
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
)

// EnvelopeID is a UUID string that uniquely identifies an envelope.
type EnvelopeID string

// PeriodType describes how often an envelope resets.
type PeriodType string

const (
	PeriodMonthly   PeriodType = "monthly"
	PeriodQuarterly PeriodType = "quarterly"
	PeriodAnnual    PeriodType = "annual"
	PeriodOnce      PeriodType = "once"
)

// RolloverPolicy controls what happens to unspent funds at period end.
type RolloverPolicy string

const (
	RolloverZero  RolloverPolicy = "zero"  // unspent funds vanish
	RolloverCarry RolloverPolicy = "carry" // unspent funds carry to next period
	RolloverCap   RolloverPolicy = "cap"   // carry up to TargetAmount, then zero
)

// Envelope is a budget container. It is not an account — it is a view over
// spending categorised from the underlying journal entries.
type Envelope struct {
	ID             EnvelopeID
	HouseholdID    account.HouseholdID
	Name           string
	TargetAmount   currency.Amount
	PeriodType     PeriodType
	RolloverPolicy RolloverPolicy
	CreatedAt      time.Time
}

// AllocationID is a UUID string for an envelope allocation record.
type AllocationID string

// Allocation records how much money was allocated to an envelope for a period.
// Append-only — never mutated.
type Allocation struct {
	ID          AllocationID
	EnvelopeID  EnvelopeID
	PeriodStart time.Time
	Amount      currency.Amount
	CreatedAt   time.Time
}
