package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/household"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	"github.com/hearth-ledger/hearth/internal/core/member"
	"github.com/hearth-ledger/hearth/internal/core/period"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
)

// Store is the PostgreSQL implementation of store.Store.
// TODO(phase-2): implement all methods against a real PostgreSQL database.
type Store struct{}

var errNotImplemented = fmt.Errorf("postgres store: not implemented (phase 2)")

func (s *Store) CreateHousehold(_ context.Context, _ household.Household) error {
	return errNotImplemented
}
func (s *Store) GetHousehold(_ context.Context, _ account.HouseholdID) (household.Household, error) {
	return household.Household{}, errNotImplemented
}
func (s *Store) CreateAccount(_ context.Context, _ account.Account) error {
	return errNotImplemented
}
func (s *Store) GetAccount(_ context.Context, _ account.AccountID) (account.Account, error) {
	return account.Account{}, errNotImplemented
}
func (s *Store) ListAccounts(_ context.Context, _ account.HouseholdID) ([]account.Account, error) {
	return nil, errNotImplemented
}
func (s *Store) CreateJournalEntry(_ context.Context, _ journal.JournalEntry) error {
	return errNotImplemented
}
func (s *Store) GetJournalEntry(_ context.Context, _ journal.EntryID) (journal.JournalEntry, error) {
	return journal.JournalEntry{}, errNotImplemented
}
func (s *Store) ListJournalEntries(_ context.Context, _ storeapi.JournalQuery) ([]journal.JournalEntry, error) {
	return nil, errNotImplemented
}
func (s *Store) GetAccountBalance(_ context.Context, _ account.AccountID, _ time.Time) (currency.Amount, error) {
	return currency.Amount{}, errNotImplemented
}
func (s *Store) CreateFiscalPeriod(_ context.Context, _ period.FiscalPeriod) error {
	return errNotImplemented
}
func (s *Store) LockFiscalPeriod(_ context.Context, _ period.PeriodID) error {
	return errNotImplemented
}
func (s *Store) CreateMember(_ context.Context, _ member.Member) error {
	return errNotImplemented
}
func (s *Store) GetMember(_ context.Context, _ member.MemberID) (member.Member, error) {
	return member.Member{}, errNotImplemented
}
func (s *Store) GetMemberByEmail(_ context.Context, _ account.HouseholdID, _ string) (member.Member, error) {
	return member.Member{}, errNotImplemented
}
func (s *Store) ListMembers(_ context.Context, _ account.HouseholdID) ([]member.Member, error) {
	return nil, errNotImplemented
}
func (s *Store) UpdateMemberRole(_ context.Context, _ member.MemberID, _ member.Role) error {
	return errNotImplemented
}
func (s *Store) CreateRefreshToken(_ context.Context, _ storeapi.RefreshToken) error {
	return errNotImplemented
}
func (s *Store) GetRefreshToken(_ context.Context, _ string) (storeapi.RefreshToken, error) {
	return storeapi.RefreshToken{}, errNotImplemented
}
func (s *Store) RevokeRefreshToken(_ context.Context, _ string) error {
	return errNotImplemented
}
func (s *Store) RevokeRefreshTokenFamily(_ context.Context, _ string) error {
	return errNotImplemented
}

// compile-time assertion that Store satisfies the store.Store interface.
var _ storeapi.Store = (*Store)(nil)
