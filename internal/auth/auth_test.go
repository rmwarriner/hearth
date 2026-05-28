package auth_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/hearth-ledger/hearth/internal/auth"
	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/household"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	"github.com/hearth-ledger/hearth/internal/core/member"
	"github.com/hearth-ledger/hearth/internal/core/period"
	"github.com/hearth-ledger/hearth/internal/observability"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// testSecret is 32 bytes — minimum required by NewService.
var testSecret = []byte("test-secret-must-be-32-bytes-ok!")

func testConfig() auth.Config {
	return auth.Config{
		JWTSecret:       testSecret,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      bcrypt.MinCost,
	}
}

func newService(t *testing.T, store storeapi.Store) *auth.Service {
	t.Helper()
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
	svc, err := auth.NewService(store, testConfig(), logger)
	require.NoError(t, err)
	return svc
}

// ── fakeStore ────────────────────────────────────────────────────────────────

// fakeStore is a minimal in-memory Store used in auth tests.
// Only the methods called by auth.Service are non-panic stubs.
type fakeStore struct {
	members map[string]member.Member
	tokens  map[string]storeapi.RefreshToken
}

var _ storeapi.Store = (*fakeStore)(nil)

func newFakeStore() *fakeStore {
	return &fakeStore{
		members: make(map[string]member.Member),
		tokens:  make(map[string]storeapi.RefreshToken),
	}
}

func (f *fakeStore) addMember(m member.Member, plainPassword string) {
	hash, err := auth.HashPassword(plainPassword, bcrypt.MinCost)
	if err != nil {
		panic("addMember: " + err.Error())
	}
	m.PasswordHash = hash
	f.members[string(m.HouseholdID)+"/"+m.Email] = m
}

func (f *fakeStore) GetMemberByEmail(_ context.Context, hid account.HouseholdID, email string) (member.Member, error) {
	m, ok := f.members[string(hid)+"/"+email]
	if !ok {
		return member.Member{}, hearth.New(hearth.ErrMemberNotFound, "not found")
	}
	return m, nil
}

func (f *fakeStore) GetMember(_ context.Context, id member.MemberID) (member.Member, error) {
	for _, m := range f.members {
		if m.ID == id {
			return m, nil
		}
	}
	return member.Member{}, hearth.New(hearth.ErrMemberNotFound, "not found")
}

func (f *fakeStore) CreateRefreshToken(_ context.Context, t storeapi.RefreshToken) error {
	f.tokens[t.TokenHash] = t
	return nil
}

func (f *fakeStore) GetRefreshToken(_ context.Context, hash string) (storeapi.RefreshToken, error) {
	t, ok := f.tokens[hash]
	if !ok {
		return storeapi.RefreshToken{}, hearth.New(hearth.ErrTokenInvalid, "not found")
	}
	return t, nil
}

func (f *fakeStore) RevokeRefreshToken(_ context.Context, hash string) error {
	t, ok := f.tokens[hash]
	if !ok {
		return hearth.New(hearth.ErrTokenInvalid, "not found")
	}
	now := time.Now().UTC()
	t.RevokedAt = &now
	f.tokens[hash] = t
	return nil
}

// ForceToken directly overwrites a token record — used in tests to simulate expiry.
func (f *fakeStore) ForceToken(hash string, t storeapi.RefreshToken) {
	f.tokens[hash] = t
}

func (f *fakeStore) RevokeRefreshTokenFamily(_ context.Context, familyID string) error {
	now := time.Now().UTC()
	for k, t := range f.tokens {
		if t.FamilyID == familyID && t.RevokedAt == nil {
			t.RevokedAt = &now
			f.tokens[k] = t
		}
	}
	return nil
}

// Unused Store methods — panic so accidental calls are immediately visible.

func (f *fakeStore) CreateHousehold(_ context.Context, _ household.Household) error {
	panic("fakeStore: CreateHousehold not expected in auth tests")
}
func (f *fakeStore) GetHousehold(_ context.Context, _ account.HouseholdID) (household.Household, error) {
	panic("fakeStore: GetHousehold not expected in auth tests")
}
func (f *fakeStore) CreateAccount(_ context.Context, _ account.Account) error {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) GetAccount(_ context.Context, _ account.AccountID) (account.Account, error) {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) ListAccounts(_ context.Context, _ account.HouseholdID) ([]account.Account, error) {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) CreateJournalEntry(_ context.Context, _ journal.JournalEntry) error {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) GetJournalEntry(_ context.Context, _ journal.EntryID) (journal.JournalEntry, error) {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) ListJournalEntries(_ context.Context, _ storeapi.JournalQuery) ([]journal.JournalEntry, error) {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) GetAccountBalance(_ context.Context, _ account.AccountID, _ time.Time) (currency.Amount, error) {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) CreateFiscalPeriod(_ context.Context, _ period.FiscalPeriod) error {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) LockFiscalPeriod(_ context.Context, _ period.PeriodID) error {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) CreateMember(_ context.Context, _ member.Member) error {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) ListMembers(_ context.Context, _ account.HouseholdID) ([]member.Member, error) {
	panic("fakeStore: unexpected call")
}
func (f *fakeStore) UpdateMemberRole(_ context.Context, _ member.MemberID, _ member.Role) error {
	panic("fakeStore: unexpected call")
}

// ── Seed helper ───────────────────────────────────────────────────────────────

func seedMember(store *fakeStore) member.Member {
	m := member.Member{
		ID:          member.MemberID("mem-001"),
		HouseholdID: account.HouseholdID("hh-001"),
		DisplayName: "Alice",
		Email:       "alice@example.com",
		Role:        member.RoleOwner,
	}
	store.addMember(m, "correct-password")
	return m
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestNewService_RejectsShortSecret(t *testing.T) {
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug"})
	_, err := auth.NewService(newFakeStore(), auth.Config{
		JWTSecret:  []byte("too-short"),
		BcryptCost: bcrypt.MinCost,
	}, logger)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "32 bytes")
}

func TestLogin_HappyPath(t *testing.T) {
	store := newFakeStore()
	m := seedMember(store)
	svc := newService(t, store)

	pair, err := svc.Login(context.Background(), m.HouseholdID, m.Email, "correct-password")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Equal(t, int((15 * time.Minute).Seconds()), pair.ExpiresIn)

	claims, err := svc.ValidateAccessToken(pair.AccessToken)
	require.NoError(t, err)
	assert.Equal(t, string(m.ID), claims.MemberID)
	assert.Equal(t, string(m.HouseholdID), claims.HouseholdID)
	assert.Equal(t, string(m.Role), claims.Role)
}

func TestLogin_WrongPassword(t *testing.T) {
	store := newFakeStore()
	m := seedMember(store)
	svc := newService(t, store)

	_, err := svc.Login(context.Background(), m.HouseholdID, m.Email, "wrong-password")
	require.Error(t, err)
	var he *hearth.HearthError
	require.ErrorAs(t, err, &he)
	assert.Equal(t, hearth.ErrUnauthorized, he.Code)
}

func TestLogin_UnknownEmail_SameErrorAsWrongPassword(t *testing.T) {
	store := newFakeStore()
	seedMember(store)
	svc := newService(t, store)

	_, err := svc.Login(context.Background(), account.HouseholdID("hh-001"), "nobody@example.com", "password")
	require.Error(t, err)
	var he *hearth.HearthError
	require.ErrorAs(t, err, &he)
	assert.Equal(t, hearth.ErrUnauthorized, he.Code)
	// Must not distinguish between bad email and bad password (timing-safe UX).
	assert.Equal(t, "invalid credentials", he.Message)
}

func TestRefresh_TokenRotation(t *testing.T) {
	store := newFakeStore()
	m := seedMember(store)
	svc := newService(t, store)

	pair1, err := svc.Login(context.Background(), m.HouseholdID, m.Email, "correct-password")
	require.NoError(t, err)

	pair2, err := svc.Refresh(context.Background(), pair1.RefreshToken)
	require.NoError(t, err)
	assert.NotEmpty(t, pair2.AccessToken)
	assert.NotEmpty(t, pair2.RefreshToken)
	assert.NotEqual(t, pair1.RefreshToken, pair2.RefreshToken)

	// Old token must now be revoked.
	rec, err := store.GetRefreshToken(context.Background(), auth.HashToken(pair1.RefreshToken))
	require.NoError(t, err)
	assert.NotNil(t, rec.RevokedAt)
}

func TestRefresh_ReplayDetection(t *testing.T) {
	store := newFakeStore()
	m := seedMember(store)
	svc := newService(t, store)

	pair1, err := svc.Login(context.Background(), m.HouseholdID, m.Email, "correct-password")
	require.NoError(t, err)

	pair2, err := svc.Refresh(context.Background(), pair1.RefreshToken)
	require.NoError(t, err)

	// Presenting the already-used token triggers replay detection.
	_, err = svc.Refresh(context.Background(), pair1.RefreshToken)
	require.Error(t, err)
	var he *hearth.HearthError
	require.ErrorAs(t, err, &he)
	assert.Equal(t, hearth.ErrTokenRevoked, he.Code)

	// The replacement token (pair2) should also be revoked — entire family killed.
	rec, err := store.GetRefreshToken(context.Background(), auth.HashToken(pair2.RefreshToken))
	require.NoError(t, err)
	assert.NotNil(t, rec.RevokedAt, "entire token family should be revoked on replay")
}

func TestRefresh_ExpiredToken(t *testing.T) {
	store := newFakeStore()
	m := seedMember(store)
	svc := newService(t, store)

	pair, err := svc.Login(context.Background(), m.HouseholdID, m.Email, "correct-password")
	require.NoError(t, err)

	// Manually back-date the refresh token's expiry in the store.
	hash := auth.HashToken(pair.RefreshToken)
	rec, err := store.GetRefreshToken(context.Background(), hash)
	require.NoError(t, err)
	rec.ExpiresAt = time.Now().Add(-1 * time.Hour)
	store.ForceToken(hash, rec)

	_, err = svc.Refresh(context.Background(), pair.RefreshToken)
	require.Error(t, err)
	var he *hearth.HearthError
	require.ErrorAs(t, err, &he)
	assert.Equal(t, hearth.ErrTokenExpired, he.Code)
}

func TestLogout_RevokesToken(t *testing.T) {
	store := newFakeStore()
	m := seedMember(store)
	svc := newService(t, store)

	pair, err := svc.Login(context.Background(), m.HouseholdID, m.Email, "correct-password")
	require.NoError(t, err)

	require.NoError(t, svc.Logout(context.Background(), pair.RefreshToken))

	rec, err := store.GetRefreshToken(context.Background(), auth.HashToken(pair.RefreshToken))
	require.NoError(t, err)
	assert.NotNil(t, rec.RevokedAt)
}

func TestValidateAccessToken_ExpiredToken(t *testing.T) {
	store := newFakeStore()
	m := seedMember(store)

	// Use a 1ms TTL so the access token is expired by the time we validate.
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug"})
	svc, err := auth.NewService(store, auth.Config{
		JWTSecret:      testSecret,
		AccessTokenTTL: time.Millisecond,
		BcryptCost:     bcrypt.MinCost,
	}, logger)
	require.NoError(t, err)

	pair, loginErr := svc.Login(context.Background(), m.HouseholdID, m.Email, "correct-password")
	require.NoError(t, loginErr)

	time.Sleep(5 * time.Millisecond) // allow the 1ms access token to expire

	_, err = svc.ValidateAccessToken(pair.AccessToken)
	require.Error(t, err)
	var he *hearth.HearthError
	require.ErrorAs(t, err, &he)
	assert.Equal(t, hearth.ErrTokenExpired, he.Code)
}

func TestLogin_NoPIIInLogs(t *testing.T) {
	store := newFakeStore()
	m := seedMember(store)

	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug", Format: "json"})
	svc, err := auth.NewService(store, testConfig(), logger)
	require.NoError(t, err)

	_, err = svc.Login(context.Background(), m.HouseholdID, m.Email, "correct-password")
	require.NoError(t, err)

	logs := buf.String()
	assert.NotContains(t, logs, m.Email)
	assert.NotContains(t, logs, m.DisplayName)
}
