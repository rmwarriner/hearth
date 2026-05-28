package app_test

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/envelope"
	"github.com/hearth-ledger/hearth/internal/core/household"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	"github.com/hearth-ledger/hearth/internal/core/member"
	"github.com/hearth-ledger/hearth/internal/core/period"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/app"
	hearth "github.com/hearth-ledger/hearth/pkg/errors"
)

// ── fakeStore ──────────────────────────────────────────────────────────────────

type fakeStore struct {
	hh       household.Household
	accounts []account.Account
}

var _ storeapi.Store = (*fakeStore)(nil)

func (f *fakeStore) GetHousehold(_ context.Context, _ account.HouseholdID) (household.Household, error) {
	return f.hh, nil
}
func (f *fakeStore) CreateHousehold(_ context.Context, _ household.Household) error { return nil }
func (f *fakeStore) CreateAccount(_ context.Context, _ account.Account) error       { return nil }
func (f *fakeStore) GetAccount(_ context.Context, _ account.AccountID) (account.Account, error) {
	return account.Account{}, hearth.New(hearth.ErrAccountNotFound, "not found")
}
func (f *fakeStore) ListAccounts(_ context.Context, _ account.HouseholdID) ([]account.Account, error) {
	return f.accounts, nil
}
func (f *fakeStore) CreateJournalEntry(_ context.Context, _ journal.JournalEntry) error { return nil }
func (f *fakeStore) GetJournalEntry(_ context.Context, _ journal.EntryID) (journal.JournalEntry, error) {
	return journal.JournalEntry{}, hearth.New(hearth.ErrNotFound, "not found")
}
func (f *fakeStore) ListJournalEntries(_ context.Context, _ storeapi.JournalQuery) ([]journal.JournalEntry, error) {
	return nil, nil
}
func (f *fakeStore) GetAccountBalance(_ context.Context, _ account.AccountID, _ time.Time) (currency.Amount, error) {
	return currency.Amount{}, nil
}
func (f *fakeStore) CreateFiscalPeriod(_ context.Context, _ period.FiscalPeriod) error { return nil }
func (f *fakeStore) LockFiscalPeriod(_ context.Context, _ period.PeriodID) error       { return nil }
func (f *fakeStore) ListLockedPeriods(_ context.Context, _ account.HouseholdID) ([]period.FiscalPeriod, error) {
	return nil, nil
}
func (f *fakeStore) CreateMember(_ context.Context, _ member.Member) error { return nil }
func (f *fakeStore) GetMember(_ context.Context, _ member.MemberID) (member.Member, error) {
	return member.Member{}, hearth.New(hearth.ErrMemberNotFound, "not found")
}
func (f *fakeStore) GetMemberByEmail(_ context.Context, _ account.HouseholdID, _ string) (member.Member, error) {
	return member.Member{}, hearth.New(hearth.ErrMemberNotFound, "not found")
}
func (f *fakeStore) ListMembers(_ context.Context, _ account.HouseholdID) ([]member.Member, error) {
	return nil, nil
}
func (f *fakeStore) UpdateMemberRole(_ context.Context, _ member.MemberID, _ member.Role) error {
	return nil
}
func (f *fakeStore) CreateEnvelope(_ context.Context, _ envelope.Envelope) error { return nil }
func (f *fakeStore) ListEnvelopes(_ context.Context, _ account.HouseholdID) ([]envelope.Envelope, error) {
	return nil, nil
}
func (f *fakeStore) CreateEnvelopeAllocation(_ context.Context, _ envelope.Allocation) error {
	return nil
}
func (f *fakeStore) ListEnvelopeAllocations(_ context.Context, _ envelope.EnvelopeID) ([]envelope.Allocation, error) {
	return nil, nil
}
func (f *fakeStore) CreateRefreshToken(_ context.Context, _ storeapi.RefreshToken) error { return nil }
func (f *fakeStore) GetRefreshToken(_ context.Context, _ string) (storeapi.RefreshToken, error) {
	return storeapi.RefreshToken{}, hearth.New(hearth.ErrNotFound, "not found")
}
func (f *fakeStore) RevokeRefreshToken(_ context.Context, _ string) error       { return nil }
func (f *fakeStore) RevokeRefreshTokenFamily(_ context.Context, _ string) error { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

func newTestApp() app.App {
	s := &fakeStore{
		hh: household.Household{
			ID: "hh-1", Name: "Test Household", BaseCurrency: "USD",
		},
	}
	return app.New(s, "hh-1")
}

func updateWithSize(m tea.Model) tea.Model {
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return updated
}

func sendKey(m tea.Model, key string) (tea.Model, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
}

func sendSpecialKey(m tea.Model, key tea.KeyType) (tea.Model, tea.Cmd) {
	return m.Update(tea.KeyMsg{Type: key})
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestApp_InitialView_ShowsTabBar(t *testing.T) {
	m := updateWithSize(newTestApp())
	view := m.View()
	assert.Contains(t, view, "[1] Dashboard")
	assert.Contains(t, view, "[2] Accounts")
	assert.Contains(t, view, "[3] Transactions")
	assert.Contains(t, view, "[4] Envelopes")
}

func TestApp_StatusBar_ShowsAIIndicator(t *testing.T) {
	m := updateWithSize(newTestApp())
	assert.Contains(t, m.View(), "[AI: OFF]")
}

func TestApp_TabSwitch_NumericKey2(t *testing.T) {
	m := updateWithSize(newTestApp())
	m, _ = sendKey(m, "2")
	assert.Contains(t, m.View(), "Accounts")
}

func TestApp_TabSwitch_NumericKey3(t *testing.T) {
	m := updateWithSize(newTestApp())
	m, _ = sendKey(m, "3")
	assert.Contains(t, m.View(), "Transactions")
}

func TestApp_TabSwitch_NumericKey4(t *testing.T) {
	m := updateWithSize(newTestApp())
	m, _ = sendKey(m, "4")
	assert.Contains(t, m.View(), "Envelopes")
}

func TestApp_Quit_Key(t *testing.T) {
	m := updateWithSize(newTestApp())
	_, cmd := sendKey(m, "q")
	require.NotNil(t, cmd)
	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit)
}

func TestApp_TabKey_CyclesForward(t *testing.T) {
	m := updateWithSize(newTestApp())
	m, _ = sendSpecialKey(m, tea.KeyTab)
	assert.Contains(t, m.View(), "Accounts")
}

func TestApp_ErrorOverlay_ShownOnSetError(t *testing.T) {
	m := updateWithSize(newTestApp().SetError("something went wrong"))
	assert.Contains(t, m.View(), "something went wrong")
}

func TestApp_ErrorOverlay_DismissedWithEsc(t *testing.T) {
	m := updateWithSize(newTestApp().SetError("something went wrong"))
	m, _ = sendSpecialKey(m, tea.KeyEsc)
	assert.NotContains(t, m.View(), "something went wrong")
}

func TestApp_TeatestModel_StartsAndQuits(t *testing.T) {
	m := newTestApp()
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}
