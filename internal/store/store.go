package store

import (
	"context"
	"time"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/envelope"
	"github.com/hearth-ledger/hearth/internal/core/household"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	"github.com/hearth-ledger/hearth/internal/core/member"
	"github.com/hearth-ledger/hearth/internal/core/period"
)

// Store is the primary persistence interface. All methods require a context.
// Every implementation (SQLite, PostgreSQL) must satisfy this interface.
//
// Interface methods return typed errors from pkg/errors, not raw errors.
// When adding a new capability, define the method here first, then write
// tests, then implement in sqlite/ and postgres/.
type Store interface {
	// Household

	CreateHousehold(ctx context.Context, h household.Household) error
	GetHousehold(ctx context.Context, id account.HouseholdID) (household.Household, error)

	// Accounts

	CreateAccount(ctx context.Context, a account.Account) error
	GetAccount(ctx context.Context, id account.AccountID) (account.Account, error)
	ListAccounts(ctx context.Context, householdID account.HouseholdID) ([]account.Account, error)

	// Journal

	CreateJournalEntry(ctx context.Context, e journal.JournalEntry) error
	GetJournalEntry(ctx context.Context, id journal.EntryID) (journal.JournalEntry, error)
	ListJournalEntries(ctx context.Context, q JournalQuery) ([]journal.JournalEntry, error)

	// Balances (computed from postings)

	GetAccountBalance(ctx context.Context, id account.AccountID, asOf time.Time) (currency.Amount, error)

	// Fiscal periods

	CreateFiscalPeriod(ctx context.Context, p period.FiscalPeriod) error
	LockFiscalPeriod(ctx context.Context, id period.PeriodID) error
	ListLockedPeriods(ctx context.Context, householdID account.HouseholdID) ([]period.FiscalPeriod, error)

	// Members

	CreateMember(ctx context.Context, m member.Member) error
	GetMember(ctx context.Context, id member.MemberID) (member.Member, error)
	GetMemberByEmail(ctx context.Context, householdID account.HouseholdID, email string) (member.Member, error)
	ListMembers(ctx context.Context, householdID account.HouseholdID) ([]member.Member, error)
	UpdateMemberRole(ctx context.Context, id member.MemberID, role member.Role) error

	// Envelopes

	CreateEnvelope(ctx context.Context, e envelope.Envelope) error
	ListEnvelopes(ctx context.Context, householdID account.HouseholdID) ([]envelope.Envelope, error)
	CreateEnvelopeAllocation(ctx context.Context, a envelope.Allocation) error
	ListEnvelopeAllocations(ctx context.Context, envelopeID envelope.EnvelopeID) ([]envelope.Allocation, error)

	// Refresh tokens (server mode only; SQLite returns ErrNotImplemented)

	CreateRefreshToken(ctx context.Context, t RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeRefreshTokenFamily(ctx context.Context, familyID string) error
}

// RefreshToken represents a stored refresh token record.
// TokenHash is a bcrypt hash of the raw token — the raw value is never stored.
type RefreshToken struct {
	TokenHash   string
	FamilyID    string
	MemberID    member.MemberID
	HouseholdID account.HouseholdID
	IssuedAt    time.Time
	ExpiresAt   time.Time
	RevokedAt   *time.Time
}

// JournalQuery specifies filters for listing journal entries.
// Zero values mean "no filter" for that field.
type JournalQuery struct {
	HouseholdID     account.HouseholdID
	AccountID       account.AccountID // filter to entries that have a posting to this account
	After           time.Time         // inclusive lower bound on PostedAt
	Before          time.Time         // inclusive upper bound on PostedAt
	DescriptionLike string            // substring match on Description (case-insensitive)
	Limit           int               // 0 means no limit
	Offset          int
}
