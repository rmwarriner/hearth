package accounts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

type loadedMsg struct {
	rows []accountRow
	err  error
}

type accountRow struct {
	acct    account.Account
	balance currency.Amount
}

// Model is the Accounts screen model.
type Model struct {
	store       store.Store
	householdID account.HouseholdID
	width       int

	loading bool
	spinner spinner.Model
	rows    []accountRow
	cursor  int
	err     string
}

// New constructs an accounts model.
func New(s store.Store, householdID account.HouseholdID) Model {
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
			m.rows = msg.rows
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadData())
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.loading {
		return "\n  " + m.spinner.View() + " Loading accounts…"
	}
	if m.err != "" {
		return styles.ErrorText.Render("\n  Error loading accounts: " + m.err)
	}
	if len(m.rows) == 0 {
		return styles.MutedText.Render("\n  No accounts found. Use the CLI to create accounts.")
	}

	var b strings.Builder
	b.WriteString("\n")

	header := fmt.Sprintf("  %-30s  %-12s  %-8s  %15s", "Name", "Type", "Currency", "Balance")
	b.WriteString(styles.TableHeader.Render(header) + "\n")
	b.WriteString(styles.MutedText.Render("  "+strings.Repeat("─", 70)) + "\n")

	for i, row := range m.rows {
		a := row.acct
		bal := row.balance
		line := fmt.Sprintf("  %-30s  %-12s  %-8s  %15s",
			truncate(a.Name, 30),
			string(a.Type),
			string(a.Currency),
			bal.Value.StringFixed(2),
		)
		if i == m.cursor {
			b.WriteString(styles.TableSelectedRow.Render(line) + "\n")
		} else {
			b.WriteString(styles.TableRow.Render(line) + "\n")
		}
	}

	b.WriteString("\n" + styles.MutedText.Render("  ↑/↓ navigate  r refresh") + "\n")
	return b.String()
}

func (m Model) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		accts, err := m.store.ListAccounts(ctx, m.householdID)
		if err != nil {
			return loadedMsg{err: err}
		}

		rows := make([]accountRow, 0, len(accts))
		for _, a := range accts {
			bal, err := m.store.GetAccountBalance(ctx, a.ID, time.Now())
			if err != nil {
				bal = currency.Amount{Currency: a.Currency}
			}
			rows = append(rows, accountRow{acct: a, balance: bal})
		}
		return loadedMsg{rows: rows}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
