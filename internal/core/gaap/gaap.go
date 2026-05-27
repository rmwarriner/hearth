package gaap

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/journal"
)

// ValidationError wraps a single GAAP rule violation.
type ValidationError struct {
	Rule string
	Err  error
	Hint string
}

func (v ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s", v.Rule, v.Err.Error())
}

func (v ValidationError) Unwrap() error {
	return v.Err
}

// Sentinel error types for each GAAP rule.
var (
	ErrInsufficientPostings = fmt.Errorf("entry must have at least 2 postings")
	ErrUnbalancedEntry      = fmt.Errorf("postings do not sum to zero")
	ErrZeroAmountPosting    = fmt.Errorf("posting has a zero amount")
	ErrCrossHouseholdEntry  = fmt.Errorf("postings reference accounts from multiple households")
	ErrLockedPeriod         = fmt.Errorf("entry is posted to a locked fiscal period")
)

// LockedPeriod describes a locked fiscal period for validation purposes.
type LockedPeriod struct {
	Start time.Time
	End   time.Time
}

// ValidationContext carries the information the guard needs to check
// rules that require external data (household membership, locked periods).
type ValidationContext struct {
	// KnownAccounts maps AccountID to the HouseholdID that owns it.
	// The guard uses this to enforce household consistency.
	KnownAccounts map[account.AccountID]account.HouseholdID

	// LockedPeriods is the list of locked fiscal periods for the household.
	LockedPeriods []LockedPeriod
}

// Validate runs all GAAP rules against the entry and returns every violation
// found. An empty slice means the entry is valid.
func Validate(entry journal.JournalEntry, ctx ValidationContext) []ValidationError {
	var errs []ValidationError

	if e := ValidateMinimumPostings(entry); e != nil {
		errs = append(errs, *e)
	}
	if e := ValidateNonZeroAmounts(entry); e != nil {
		errs = append(errs, *e)
	}
	if e := ValidateBalance(entry); e != nil {
		errs = append(errs, *e)
	}
	if e := ValidateHouseholdConsistency(entry, ctx); e != nil {
		errs = append(errs, *e)
	}
	if e := ValidatePeriodNotLocked(entry, ctx); e != nil {
		errs = append(errs, *e)
	}

	return errs
}

// ValidateMinimumPostings checks that the entry has at least two postings.
func ValidateMinimumPostings(entry journal.JournalEntry) *ValidationError {
	if len(entry.Postings) >= 2 {
		return nil
	}
	return &ValidationError{
		Rule: "MinimumPostings",
		Err:  ErrInsufficientPostings,
		Hint: "Add at least two postings — one debit and one credit — so the entry balances.",
	}
}

// ValidateBalance checks that the sum of all posting amounts equals zero.
// For multi-currency entries, each currency must independently sum to zero.
func ValidateBalance(entry journal.JournalEntry) *ValidationError {
	totals := make(map[currency.Currency]decimal.Decimal)
	for _, p := range entry.Postings {
		totals[p.Amount.Currency] = totals[p.Amount.Currency].Add(p.Amount.Value)
	}
	for cur, total := range totals {
		if !total.IsZero() {
			return &ValidationError{
				Rule: "Balance",
				Err:  fmt.Errorf("%w: %s total is %s, expected 0", ErrUnbalancedEntry, cur, total.String()),
				Hint: "Adjust posting amounts so that debits equal credits for each currency.",
			}
		}
	}
	return nil
}

// ValidateNonZeroAmounts checks that no posting has a zero amount.
func ValidateNonZeroAmounts(entry journal.JournalEntry) *ValidationError {
	for _, p := range entry.Postings {
		if p.Amount.IsZero() {
			return &ValidationError{
				Rule: "NonZeroAmounts",
				Err:  fmt.Errorf("%w: posting %s has amount 0", ErrZeroAmountPosting, p.ID),
				Hint: "Remove zero-amount postings or assign a non-zero value.",
			}
		}
	}
	return nil
}

// ValidateHouseholdConsistency checks that all referenced accounts belong to
// the same household as the entry.
func ValidateHouseholdConsistency(entry journal.JournalEntry, ctx ValidationContext) *ValidationError {
	for _, p := range entry.Postings {
		ownerHH, ok := ctx.KnownAccounts[p.AccountID]
		if !ok {
			return &ValidationError{
				Rule: "HouseholdConsistency",
				Err:  fmt.Errorf("%w: account %s not found", ErrCrossHouseholdEntry, p.AccountID),
				Hint: "Ensure all accounts exist and belong to this household before recording the entry.",
			}
		}
		if ownerHH != entry.HouseholdID {
			return &ValidationError{
				Rule: "HouseholdConsistency",
				Err:  fmt.Errorf("%w: account %s belongs to household %s, not %s", ErrCrossHouseholdEntry, p.AccountID, ownerHH, entry.HouseholdID),
				Hint: "All accounts in an entry must belong to the same household.",
			}
		}
	}
	return nil
}

// ValidatePeriodNotLocked checks that the entry's posted date does not fall
// within a locked fiscal period.
func ValidatePeriodNotLocked(entry journal.JournalEntry, ctx ValidationContext) *ValidationError {
	for _, lp := range ctx.LockedPeriods {
		if !entry.PostedAt.Before(lp.Start) && !entry.PostedAt.After(lp.End) {
			return &ValidationError{
				Rule: "PeriodNotLocked",
				Err:  fmt.Errorf("%w: %s falls within locked period %s–%s", ErrLockedPeriod, entry.PostedAt.Format("2006-01-02"), lp.Start.Format("2006-01-02"), lp.End.Format("2006-01-02")),
				Hint: "Choose a date outside the locked period, or unlock the period before posting.",
			}
		}
	}
	return nil
}
