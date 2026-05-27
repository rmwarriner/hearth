package gaap_test

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/gaap"
	"github.com/hearth-ledger/hearth/internal/core/journal"
)

const (
	hhID   = account.HouseholdID("hh-1")
	accID1 = account.AccountID("acc-1")
	accID2 = account.AccountID("acc-2")
	accID3 = account.AccountID("acc-3")
)

var baseDate = time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

func posting(id journal.PostingID, accID account.AccountID, amount string, cur currency.Currency) journal.Posting {
	return journal.Posting{
		ID:        id,
		AccountID: accID,
		Amount: currency.Amount{
			Value:    decimal.RequireFromString(amount),
			Currency: cur,
		},
	}
}

func balancedEntry(postings ...journal.Posting) journal.JournalEntry {
	return journal.JournalEntry{
		ID:          "entry-1",
		HouseholdID: hhID,
		PostedAt:    baseDate,
		Description: "Test entry",
		Source:      journal.SourceManual,
		Postings:    postings,
	}
}

func defaultCtx() gaap.ValidationContext {
	return gaap.ValidationContext{
		KnownAccounts: map[account.AccountID]account.HouseholdID{
			accID1: hhID,
			accID2: hhID,
			accID3: hhID,
		},
	}
}

// ── ValidateMinimumPostings ────────────────────────────────────────────────

func TestValidate_MinimumPostings_TwoPostings_IsValid(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.Empty(t, errs)
}

func TestValidate_MinimumPostings_OnePosting_ReturnsInsufficientPostings(t *testing.T) {
	entry := balancedEntry(posting("p1", accID1, "50.00", "USD"))
	errs := gaap.Validate(entry, defaultCtx())
	require.NotEmpty(t, errs)
	assert.True(t, hasErr(errs, gaap.ErrInsufficientPostings), "expected ErrInsufficientPostings")
}

func TestValidate_MinimumPostings_ZeroPostings_ReturnsInsufficientPostings(t *testing.T) {
	entry := balancedEntry()
	errs := gaap.Validate(entry, defaultCtx())
	require.NotEmpty(t, errs)
	assert.True(t, hasErr(errs, gaap.ErrInsufficientPostings))
}

func TestValidate_MinimumPostings_ThreePostings_IsValid(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "100.00", "USD"),
		posting("p2", accID2, "-60.00", "USD"),
		posting("p3", accID3, "-40.00", "USD"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.Empty(t, errs)
}

// ── ValidateBalance ────────────────────────────────────────────────────────

func TestValidate_Balance_BalancedEntry_IsValid(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "100.00", "USD"),
		posting("p2", accID2, "-100.00", "USD"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.Empty(t, errs)
}

func TestValidate_Balance_UnbalancedEntry_ReturnsUnbalancedEntry(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "100.00", "USD"),
		posting("p2", accID2, "-90.00", "USD"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.True(t, hasErr(errs, gaap.ErrUnbalancedEntry))
}

func TestValidate_Balance_MultiCurrencyBalanced_IsValid(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "100.00", "USD"),
		posting("p2", accID2, "-100.00", "USD"),
		posting("p3", accID1, "85.00", "EUR"),
		posting("p4", accID2, "-85.00", "EUR"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.Empty(t, errs)
}

func TestValidate_Balance_MultiCurrencyUnbalanced_ReturnsUnbalancedEntry(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "100.00", "USD"),
		posting("p2", accID2, "-100.00", "USD"),
		posting("p3", accID1, "85.00", "EUR"),
		posting("p4", accID2, "-80.00", "EUR"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.True(t, hasErr(errs, gaap.ErrUnbalancedEntry))
}

// ── ValidateNonZeroAmounts ────────────────────────────────────────────────

func TestValidate_NonZeroAmounts_NonZeroPostings_IsValid(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.Empty(t, errs)
}

func TestValidate_NonZeroAmounts_ZeroPosting_ReturnsZeroAmountPosting(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "0.00", "USD"),
		posting("p2", accID2, "0.00", "USD"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.True(t, hasErr(errs, gaap.ErrZeroAmountPosting))
}

// ── ValidateHouseholdConsistency ──────────────────────────────────────────

func TestValidate_HouseholdConsistency_AllAccountsSameHousehold_IsValid(t *testing.T) {
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	errs := gaap.Validate(entry, defaultCtx())
	assert.Empty(t, errs)
}

func TestValidate_HouseholdConsistency_AccountFromOtherHousehold_ReturnsCrossHousehold(t *testing.T) {
	ctx := gaap.ValidationContext{
		KnownAccounts: map[account.AccountID]account.HouseholdID{
			accID1: hhID,
			accID2: "other-household",
		},
	}
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	errs := gaap.Validate(entry, ctx)
	assert.True(t, hasErr(errs, gaap.ErrCrossHouseholdEntry))
}

func TestValidate_HouseholdConsistency_UnknownAccount_ReturnsCrossHousehold(t *testing.T) {
	ctx := gaap.ValidationContext{
		KnownAccounts: map[account.AccountID]account.HouseholdID{
			accID1: hhID,
			// accID2 missing
		},
	}
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	errs := gaap.Validate(entry, ctx)
	assert.True(t, hasErr(errs, gaap.ErrCrossHouseholdEntry))
}

// ── ValidatePeriodNotLocked ───────────────────────────────────────────────

func TestValidate_PeriodNotLocked_OpenPeriod_IsValid(t *testing.T) {
	ctx := defaultCtx()
	// No locked periods at all
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	errs := gaap.Validate(entry, ctx)
	assert.Empty(t, errs)
}

func TestValidate_PeriodNotLocked_DateInLockedPeriod_ReturnsLockedPeriod(t *testing.T) {
	ctx := defaultCtx()
	ctx.LockedPeriods = []gaap.LockedPeriod{
		{
			Start: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2025, 6, 30, 23, 59, 59, 0, time.UTC),
		},
	}
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	// baseDate is June 15, 2025 — within the locked period
	errs := gaap.Validate(entry, ctx)
	assert.True(t, hasErr(errs, gaap.ErrLockedPeriod))
}

func TestValidate_PeriodNotLocked_DateOutsideLockedPeriod_IsValid(t *testing.T) {
	ctx := defaultCtx()
	ctx.LockedPeriods = []gaap.LockedPeriod{
		{
			Start: time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2025, 5, 31, 23, 59, 59, 0, time.UTC),
		},
	}
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	// baseDate is June 15 — outside May locked period
	errs := gaap.Validate(entry, ctx)
	assert.Empty(t, errs)
}

func TestValidate_PeriodNotLocked_DateOnPeriodBoundary_ReturnsLockedPeriod(t *testing.T) {
	ctx := defaultCtx()
	ctx.LockedPeriods = []gaap.LockedPeriod{
		{
			Start: baseDate,
			End:   baseDate.Add(24 * time.Hour),
		},
	}
	entry := balancedEntry(
		posting("p1", accID1, "50.00", "USD"),
		posting("p2", accID2, "-50.00", "USD"),
	)
	errs := gaap.Validate(entry, ctx)
	assert.True(t, hasErr(errs, gaap.ErrLockedPeriod))
}

// ── Validate (composite) ──────────────────────────────────────────────────

func TestValidate_MultipleViolations_ReturnsAllErrors(t *testing.T) {
	// Single posting (min postings violation) + amount is zero (zero amount violation)
	entry := journal.JournalEntry{
		ID:          "entry-bad",
		HouseholdID: hhID,
		PostedAt:    baseDate,
		Postings: []journal.Posting{
			posting("p1", accID1, "0.00", "USD"),
		},
	}
	errs := gaap.Validate(entry, defaultCtx())
	assert.True(t, hasErr(errs, gaap.ErrInsufficientPostings), "expected ErrInsufficientPostings")
	assert.True(t, hasErr(errs, gaap.ErrZeroAmountPosting), "expected ErrZeroAmountPosting")
	assert.GreaterOrEqual(t, len(errs), 2, "expected at least 2 errors")
}

// ── helpers ───────────────────────────────────────────────────────────────

func hasErr(errs []gaap.ValidationError, target error) bool {
	for _, e := range errs {
		if errors.Is(e, target) {
			return true
		}
	}
	return false
}
