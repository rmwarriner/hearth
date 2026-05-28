package envelopes

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/envelope"
	"github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

type loadedMsg struct {
	envelopes []envelope.Envelope
	err       error
}

// Model is the Envelopes screen model.
type Model struct {
	store       store.Store
	householdID account.HouseholdID
	width       int

	loading   bool
	spinner   spinner.Model
	envelopes []envelope.Envelope
	cursor    int
	err       string
}

// New constructs an envelopes model.
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
			m.envelopes = msg.envelopes
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.envelopes)-1 {
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
		return "\n  " + m.spinner.View() + " Loading envelopes…"
	}
	if m.err != "" {
		return styles.ErrorText.Render("\n  Error loading envelopes: " + m.err)
	}
	if len(m.envelopes) == 0 {
		return styles.MutedText.Render("\n  No envelopes found.")
	}

	var b strings.Builder
	b.WriteString("\n")

	header := fmt.Sprintf("  %-30s  %-12s  %-12s  %15s", "Name", "Period", "Rollover", "Target")
	b.WriteString(styles.TableHeader.Render(header) + "\n")
	b.WriteString(styles.MutedText.Render("  "+strings.Repeat("─", 70)) + "\n")

	for i, e := range m.envelopes {
		line := fmt.Sprintf("  %-30s  %-12s  %-12s  %15s",
			truncate(e.Name, 30),
			string(e.PeriodType),
			string(e.RolloverPolicy),
			e.TargetAmount.Value.StringFixed(2),
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
		envs, err := m.store.ListEnvelopes(ctx, m.householdID)
		if err != nil {
			return loadedMsg{err: err}
		}
		return loadedMsg{envelopes: envs}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
