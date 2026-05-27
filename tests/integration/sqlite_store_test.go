package integration_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/household"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	"github.com/hearth-ledger/hearth/internal/core/period"
	"github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/store/sqlite"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

func newTestStore(t *testing.T) *sqlite.Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	ctx := context.Background()

	db, err := sqlite.Open(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	return sqlite.New(db)
}

func seedHousehold(t *testing.T, s *sqlite.Store) {
	t.Helper()
	hh := household.Household{
		ID: "hh-1", Name: "Test Household", FiscalYearStart: 1, BaseCurrency: "USD",
	}
	require.NoError(t, s.CreateHousehold(context.Background(), hh))
}

func seedAccounts(t *testing.T, s *sqlite.Store) (account.Account, account.Account) {
	t.Helper()
	checking := account.Account{
		ID: "acc-checking", HouseholdID: "hh-1", Name: "Checking",
		Type: account.Asset, Currency: "USD",
	}
	groceries := account.Account{
		ID: "acc-groceries", HouseholdID: "hh-1", Name: "Groceries",
		Type: account.Expense, Currency: "USD",
	}
	require.NoError(t, s.CreateAccount(context.Background(), checking))
	require.NoError(t, s.CreateAccount(context.Background(), groceries))
	return checking, groceries
}

func makeBalancedEntry(checkingID, expenseID account.AccountID) journal.JournalEntry {
	return journal.JournalEntry{
		ID:          "entry-1",
		HouseholdID: "hh-1",
		PostedAt:    time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
		Description: "Grocery run",
		Source:      journal.SourceManual,
		CreatedBy:   "member-1",
		Postings: []journal.Posting{
			{
				ID:             "post-1",
				JournalEntryID: "entry-1",
				AccountID:      expenseID,
				Amount:         currency.Amount{Value: decimal.RequireFromString("50.00"), Currency: "USD"},
			},
			{
				ID:             "post-2",
				JournalEntryID: "entry-1",
				AccountID:      checkingID,
				Amount:         currency.Amount{Value: decimal.RequireFromString("-50.00"), Currency: "USD"},
			},
		},
	}
}

// ── Household ──────────────────────────────────────────────────────────────

func TestSQLiteStore_CreateHousehold_HappyPath(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s) // verifies no error
}

func TestSQLiteStore_GetHousehold_HappyPath(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)

	got, err := s.GetHousehold(context.Background(), "hh-1")
	require.NoError(t, err)
	assert.Equal(t, account.HouseholdID("hh-1"), got.ID)
	assert.Equal(t, "Test Household", got.Name)
	assert.Equal(t, currency.Currency("USD"), got.BaseCurrency)
}

func TestSQLiteStore_GetHousehold_NotFound_ReturnsHouseholdNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetHousehold(context.Background(), "no-such-id")
	require.Error(t, err)
	var he *hearth.HearthError
	require.ErrorAs(t, err, &he)
	assert.Equal(t, hearth.ErrHouseholdNotFound, he.Code)
}

// ── Accounts ───────────────────────────────────────────────────────────────

func TestSQLiteStore_CreateAndGetAccount_HappyPath(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	checking, _ := seedAccounts(t, s)

	got, err := s.GetAccount(context.Background(), checking.ID)
	require.NoError(t, err)
	assert.Equal(t, "Checking", got.Name)
	assert.Equal(t, account.Asset, got.Type)
}

func TestSQLiteStore_GetAccount_NotFound_ReturnsAccountNotFound(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetAccount(context.Background(), "no-such-account")
	require.Error(t, err)
	var he *hearth.HearthError
	require.ErrorAs(t, err, &he)
	assert.Equal(t, hearth.ErrAccountNotFound, he.Code)
}

func TestSQLiteStore_CreateAccount_UnknownHousehold_ReturnsForeignKeyError(t *testing.T) {
	s := newTestStore(t)
	a := account.Account{
		ID: "acc-orphan", HouseholdID: "no-such-household",
		Name: "Orphan", Type: account.Asset, Currency: "USD",
	}
	err := s.CreateAccount(context.Background(), a)
	require.Error(t, err)
	var he *hearth.HearthError
	assert.ErrorAs(t, err, &he)
}

func TestSQLiteStore_ListAccounts_HappyPath(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	seedAccounts(t, s)

	accounts, err := s.ListAccounts(context.Background(), "hh-1")
	require.NoError(t, err)
	assert.Len(t, accounts, 2)
}

func TestSQLiteStore_ListAccounts_EmptyHousehold_ReturnsEmpty(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.CreateHousehold(context.Background(), household.Household{
		ID: "hh-empty", Name: "Empty", FiscalYearStart: 1, BaseCurrency: "USD",
	}))

	accounts, err := s.ListAccounts(context.Background(), "hh-empty")
	require.NoError(t, err)
	assert.Empty(t, accounts)
}

// ── Journal Entries ────────────────────────────────────────────────────────

func TestSQLiteStore_CreateJournalEntry_HappyPath(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	checking, groceries := seedAccounts(t, s)

	entry := makeBalancedEntry(checking.ID, groceries.ID)
	require.NoError(t, s.CreateJournalEntry(context.Background(), entry))
}

func TestSQLiteStore_GetJournalEntry_HappyPath(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	checking, groceries := seedAccounts(t, s)

	entry := makeBalancedEntry(checking.ID, groceries.ID)
	require.NoError(t, s.CreateJournalEntry(context.Background(), entry))

	got, err := s.GetJournalEntry(context.Background(), entry.ID)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, got.ID)
	assert.Equal(t, "Grocery run", got.Description)
	assert.Len(t, got.Postings, 2)
}

func TestSQLiteStore_CreateJournalEntry_FailingPostingRollsBackEntireEntry(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	checking, _ := seedAccounts(t, s)

	entry := journal.JournalEntry{
		ID: "entry-rollback", HouseholdID: "hh-1",
		PostedAt: time.Now().UTC(), Description: "Should fail",
		Source: journal.SourceManual,
		Postings: []journal.Posting{
			{
				ID:        "post-ok",
				AccountID: checking.ID,
				Amount:    currency.Amount{Value: decimal.RequireFromString("50.00"), Currency: "USD"},
			},
			{
				ID:        "post-bad",
				AccountID: "acc-does-not-exist", // FK violation → transaction must roll back
				Amount:    currency.Amount{Value: decimal.RequireFromString("-50.00"), Currency: "USD"},
			},
		},
	}

	err := s.CreateJournalEntry(context.Background(), entry)
	require.Error(t, err)

	// The entry itself must not have been persisted
	_, err2 := s.GetJournalEntry(context.Background(), "entry-rollback")
	require.Error(t, err2)
}

func TestSQLiteStore_ListJournalEntries_FilterByAccount(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	checking, groceries := seedAccounts(t, s)

	entry := makeBalancedEntry(checking.ID, groceries.ID)
	require.NoError(t, s.CreateJournalEntry(context.Background(), entry))

	// Filter by groceries account — should return our entry
	entries, err := s.ListJournalEntries(context.Background(), store.JournalQuery{
		HouseholdID: "hh-1",
		AccountID:   groceries.ID,
	})
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	// Filter by a non-existent account — should return empty
	entries, err = s.ListJournalEntries(context.Background(), store.JournalQuery{
		HouseholdID: "hh-1",
		AccountID:   "acc-other",
	})
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestSQLiteStore_ListJournalEntries_FilterByDateRange(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	checking, groceries := seedAccounts(t, s)

	entry := makeBalancedEntry(checking.ID, groceries.ID)
	require.NoError(t, s.CreateJournalEntry(context.Background(), entry))

	// entry.PostedAt is 2025-06-15; query for June only
	entries, err := s.ListJournalEntries(context.Background(), store.JournalQuery{
		HouseholdID: "hh-1",
		After:       time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Before:      time.Date(2025, 6, 30, 23, 59, 59, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	// Query for July — should return nothing
	entries, err = s.ListJournalEntries(context.Background(), store.JournalQuery{
		HouseholdID: "hh-1",
		After:       time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Empty(t, entries)
}

// ── Fiscal Periods ─────────────────────────────────────────────────────────

func TestSQLiteStore_CreateFiscalPeriod_HappyPath(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.CreateHousehold(context.Background(), household.Household{
		ID: "hh-p", Name: "Period HH", FiscalYearStart: 1, BaseCurrency: "USD",
	}))
	p := period.FiscalPeriod{
		ID:          "period-1",
		HouseholdID: "hh-p",
		Name:        "2025-Q1",
		StartDate:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, s.CreateFiscalPeriod(context.Background(), p))
}

func TestSQLiteStore_LockFiscalPeriod_HappyPath(t *testing.T) {
	s := newTestStore(t)
	require.NoError(t, s.CreateHousehold(context.Background(), household.Household{
		ID: "hh-lock", Name: "Lock HH", FiscalYearStart: 1, BaseCurrency: "USD",
	}))
	p := period.FiscalPeriod{
		ID: "period-lock", HouseholdID: "hh-lock", Name: "2025-H1",
		StartDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, s.CreateFiscalPeriod(context.Background(), p))
	require.NoError(t, s.LockFiscalPeriod(context.Background(), "period-lock"))
}

// ── Balance ────────────────────────────────────────────────────────────────

func TestSQLiteStore_GetAccountBalance_AfterOneEntry(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	checking, groceries := seedAccounts(t, s)

	entry := makeBalancedEntry(checking.ID, groceries.ID)
	require.NoError(t, s.CreateJournalEntry(context.Background(), entry))

	bal, err := s.GetAccountBalance(context.Background(), groceries.ID, time.Now().UTC())
	require.NoError(t, err)
	assert.True(t, bal.Value.Equal(decimal.RequireFromString("50.00")))
	assert.Equal(t, currency.Currency("USD"), bal.Currency)
}

func TestSQLiteStore_GetAccountBalance_AsOfBeforeEntry_ReturnsZero(t *testing.T) {
	s := newTestStore(t)
	seedHousehold(t, s)
	checking, groceries := seedAccounts(t, s)

	entry := makeBalancedEntry(checking.ID, groceries.ID)
	require.NoError(t, s.CreateJournalEntry(context.Background(), entry))

	// entry posted 2025-06-15; query as of 2025-06-14
	asOf := time.Date(2025, 6, 14, 0, 0, 0, 0, time.UTC)
	bal, err := s.GetAccountBalance(context.Background(), groceries.ID, asOf)
	require.NoError(t, err)
	assert.True(t, bal.Value.IsZero())
}
