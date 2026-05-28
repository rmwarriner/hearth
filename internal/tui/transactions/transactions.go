package transactions

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

type loadedMsg struct {
	entries []journal.JournalEntry
	err     error
}

// Model is the Transactions screen model.
type Model struct {
	store       storeapi.Store
	householdID account.HouseholdID
	width       int

	loading bool
	spinner spinner.Model
	entries []journal.JournalEntry
	cursor  int
	err     string
}

// New constructs a transactions model.
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
			m.entries = msg.entries
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
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
		return "\n  " + m.spinner.View() + " Loading transactions…"
	}
	if m.err != "" {
		return styles.ErrorText.Render("\n  Error loading transactions: " + m.err)
	}
	if len(m.entries) == 0 {
		return styles.MutedText.Render("\n  No transactions found.")
	}

	var b strings.Builder
	b.WriteString("\n")

	header := fmt.Sprintf("  %-12s  %-40s  %s", "Date", "Description", "Postings")
	b.WriteString(styles.TableHeader.Render(header) + "\n")
	b.WriteString(styles.MutedText.Render("  "+strings.Repeat("─", 70)) + "\n")

	for i, e := range m.entries {
		postingCount := fmt.Sprintf("%d postings", len(e.Postings))
		line := fmt.Sprintf("  %-12s  %-40s  %s",
			e.PostedAt.Format("2006-01-02"),
			truncate(e.Description, 40),
			postingCount,
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
		entries, err := m.store.ListJournalEntries(ctx, storeapi.JournalQuery{
			HouseholdID: m.householdID,
			Limit:       50,
		})
		if err != nil {
			return loadedMsg{err: err}
		}
		return loadedMsg{entries: entries}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
