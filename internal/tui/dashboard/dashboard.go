package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

// loadedMsg carries the result of the async data load.
type loadedMsg struct {
	householdName string
	netWorth      currency.Amount
	recentLines   []string
	envelopeCount int
	err           error
}

// Model is the Dashboard screen model.
type Model struct {
	store       storeapi.Store
	householdID account.HouseholdID
	width       int

	loading       bool
	spinner       spinner.Model
	householdName string
	netWorth      currency.Amount
	recentLines   []string
	envelopeCount int
	err           string
}

// New constructs a dashboard model.
func New(s storeapi.Store, householdID account.HouseholdID) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	return Model{
		store:       s,
		householdID: householdID,
		loading:     true,
		spinner:     sp,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadData())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case loadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err.Error()
		} else {
			m.householdName = msg.householdName
			m.netWorth = msg.netWorth
			m.recentLines = msg.recentLines
			m.envelopeCount = msg.envelopeCount
		}
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return "\n  " + m.spinner.View() + " Loading dashboard…"
	}
	if m.err != "" {
		return styles.ErrorText.Render("\n  Error loading dashboard: " + m.err)
	}

	var b strings.Builder
	title := styles.BoldText.Render(fmt.Sprintf("Hearth · %s", m.householdName))
	b.WriteString("\n  " + title + "\n\n")

	nw := m.netWorth
	fmt.Fprintf(&b, "  Net Worth: %s %s\n\n", nw.Currency, nw.Value.StringFixed(2))

	if len(m.recentLines) > 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(styles.Primary).Render("  Recent Activity") + "\n")
		for _, line := range m.recentLines {
			b.WriteString("  " + line + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(styles.MutedText.Render(fmt.Sprintf("  Envelopes: %d active", m.envelopeCount)) + "\n")
	return b.String()
}

func (m Model) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		hh, err := m.store.GetHousehold(ctx, m.householdID)
		if err != nil {
			return loadedMsg{err: err}
		}

		accts, err := m.store.ListAccounts(ctx, m.householdID)
		if err != nil {
			return loadedMsg{err: err}
		}

		// Compute net worth from all asset accounts minus liabilities.
		// Use zero time to get current balance.
		netWorth := currency.Amount{Currency: hh.BaseCurrency}
		for _, a := range accts {
			bal, err := m.store.GetAccountBalance(ctx, a.ID, time.Now())
			if err != nil {
				continue
			}
			switch a.Type {
			case account.Asset:
				netWorth.Value = netWorth.Value.Add(bal.Value)
			case account.Liability:
				netWorth.Value = netWorth.Value.Sub(bal.Value)
			}
		}

		entries, err := m.store.ListJournalEntries(ctx, storeapi.JournalQuery{
			HouseholdID: m.householdID,
			Limit:       5,
		})
		if err != nil {
			return loadedMsg{err: err}
		}

		var recent []string
		for _, e := range entries {
			line := fmt.Sprintf("%-12s  %s", e.PostedAt.Format("2006-01-02"), e.Description)
			recent = append(recent, line)
		}

		envs, err := m.store.ListEnvelopes(ctx, m.householdID)
		if err != nil {
			return loadedMsg{err: err}
		}

		return loadedMsg{
			householdName: hh.Name,
			netWorth:      netWorth,
			recentLines:   recent,
			envelopeCount: len(envs),
		}
	}
}
