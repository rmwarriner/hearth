package middleware_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/hearth-ledger/hearth/internal/api/middleware"
	"github.com/hearth-ledger/hearth/internal/auth"
	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/envelope"
	"github.com/hearth-ledger/hearth/internal/core/household"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	"github.com/hearth-ledger/hearth/internal/core/member"
	"github.com/hearth-ledger/hearth/internal/core/period"
	"github.com/hearth-ledger/hearth/internal/observability"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

var testSecret = []byte("test-secret-must-be-32-bytes-ok!")

func newAuthService(t *testing.T) (*auth.Service, *fakeStore) {
	t.Helper()
	store := newFakeStore()
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug"})
	svc, err := auth.NewService(store, auth.Config{
		JWTSecret:       testSecret,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		BcryptCost:      bcrypt.MinCost,
	}, logger)
	require.NoError(t, err)
	return svc, store
}

func issueToken(t *testing.T, svc *auth.Service, store *fakeStore, hhID, email string) string {
	t.Helper()
	m := member.Member{
		ID:          member.MemberID("mem-" + email),
		HouseholdID: account.HouseholdID(hhID),
		DisplayName: "Test User",
		Email:       email,
		Role:        member.RoleMember,
	}
	hash, err := auth.HashPassword("password", bcrypt.MinCost)
	require.NoError(t, err)
	m.PasswordHash = hash
	store.members[hhID+"/"+email] = m

	pair, err := svc.Login(context.Background(), account.HouseholdID(hhID), email, "password") //nolint:contextcheck
	require.NoError(t, err)
	return pair.AccessToken
}

// ── Authenticate ─────────────────────────────────────────────────────────────

func TestAuthenticate_ValidToken_PassesThrough(t *testing.T) {
	svc, store := newAuthService(t)
	token := issueToken(t, svc, store, "hh-001", "user@example.com")

	handler := middleware.Authenticate(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.ClaimsFromContext(r.Context())
		assert.NotNil(t, claims)
		assert.Equal(t, "hh-001", claims.HouseholdID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestAuthenticate_MissingHeader_Returns401(t *testing.T) {
	svc, _ := newAuthService(t)
	handler := middleware.Authenticate(svc)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticate_InvalidToken_Returns401(t *testing.T) {
	svc, _ := newAuthService(t)
	handler := middleware.Authenticate(svc)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuthenticate_ExpiredToken_Returns401(t *testing.T) {
	store := newFakeStore()
	var buf bytes.Buffer
	logger := observability.NewLoggerWithWriter(&buf, observability.Config{Level: "debug"})
	// 1ms TTL so the access token is expired almost immediately.
	svc, err := auth.NewService(store, auth.Config{
		JWTSecret:      testSecret,
		AccessTokenTTL: time.Millisecond,
		BcryptCost:     bcrypt.MinCost,
	}, logger)
	require.NoError(t, err)

	token := issueToken(t, svc, store, "hh-001", "user@example.com")
	time.Sleep(5 * time.Millisecond)

	handler := middleware.Authenticate(svc)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

// ── VerifyHousehold ───────────────────────────────────────────────────────────

func TestVerifyHousehold_MatchingHousehold_PassesThrough(t *testing.T) {
	svc, store := newAuthService(t)
	token := issueToken(t, svc, store, "hh-001", "user@example.com")

	r := chi.NewRouter()
	r.Use(middleware.Authenticate(svc))
	r.Route("/households/{householdId}", func(r chi.Router) {
		r.Use(middleware.VerifyHousehold)
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			hid := middleware.HouseholdIDFromContext(r.Context())
			assert.Equal(t, "hh-001", hid)
			w.WriteHeader(http.StatusOK)
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/households/hh-001", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestVerifyHousehold_WrongHousehold_Returns403(t *testing.T) {
	svc, store := newAuthService(t)
	// Token claims hh-001 but URL targets hh-002.
	token := issueToken(t, svc, store, "hh-001", "user@example.com")

	r := chi.NewRouter()
	r.Use(middleware.Authenticate(svc))
	r.Route("/households/{householdId}", func(r chi.Router) {
		r.Use(middleware.VerifyHousehold)
		r.Get("/", okHandler().ServeHTTP)
	})

	req := httptest.NewRequest(http.MethodGet, "/households/hh-002", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusForbidden, rr.Code)
}

// ── Recovery ──────────────────────────────────────────────────────────────────

func TestRecovery_PanicHandler_Returns500(t *testing.T) {
	panic_handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})
	handler := middleware.Recovery(panic_handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

// ── fakeStore (mirrors auth_test fakeStore) ──────────────────────────────────

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
func (f *fakeStore) CreateHousehold(_ context.Context, _ household.Household) error {
	panic("unexpected")
}
func (f *fakeStore) GetHousehold(_ context.Context, _ account.HouseholdID) (household.Household, error) {
	panic("unexpected")
}
func (f *fakeStore) CreateAccount(_ context.Context, _ account.Account) error { panic("unexpected") }
func (f *fakeStore) GetAccount(_ context.Context, _ account.AccountID) (account.Account, error) {
	panic("unexpected")
}
func (f *fakeStore) ListAccounts(_ context.Context, _ account.HouseholdID) ([]account.Account, error) {
	panic("unexpected")
}
func (f *fakeStore) CreateJournalEntry(_ context.Context, _ journal.JournalEntry) error {
	panic("unexpected")
}
func (f *fakeStore) GetJournalEntry(_ context.Context, _ journal.EntryID) (journal.JournalEntry, error) {
	panic("unexpected")
}
func (f *fakeStore) ListJournalEntries(_ context.Context, _ storeapi.JournalQuery) ([]journal.JournalEntry, error) {
	panic("unexpected")
}
func (f *fakeStore) GetAccountBalance(_ context.Context, _ account.AccountID, _ time.Time) (currency.Amount, error) {
	panic("unexpected")
}
func (f *fakeStore) CreateFiscalPeriod(_ context.Context, _ period.FiscalPeriod) error {
	panic("unexpected")
}
func (f *fakeStore) LockFiscalPeriod(_ context.Context, _ period.PeriodID) error {
	panic("unexpected")
}
func (f *fakeStore) CreateMember(_ context.Context, _ member.Member) error { panic("unexpected") }
func (f *fakeStore) ListMembers(_ context.Context, _ account.HouseholdID) ([]member.Member, error) {
	panic("unexpected")
}
func (f *fakeStore) UpdateMemberRole(_ context.Context, _ member.MemberID, _ member.Role) error {
	panic("unexpected")
}
func (f *fakeStore) ListLockedPeriods(_ context.Context, _ account.HouseholdID) ([]period.FiscalPeriod, error) {
	panic("unexpected")
}
func (f *fakeStore) CreateEnvelope(_ context.Context, _ envelope.Envelope) error {
	panic("unexpected")
}
func (f *fakeStore) ListEnvelopes(_ context.Context, _ account.HouseholdID) ([]envelope.Envelope, error) {
	panic("unexpected")
}
func (f *fakeStore) CreateEnvelopeAllocation(_ context.Context, _ envelope.Allocation) error {
	panic("unexpected")
}
func (f *fakeStore) ListEnvelopeAllocations(_ context.Context, _ envelope.EnvelopeID) ([]envelope.Allocation, error) {
	panic("unexpected")
}
