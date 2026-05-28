package envelopes

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/shopspring/decimal"

	"github.com/google/uuid"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/envelope"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

type screenMode int

const (
	modeList screenMode = iota
	modeCreate
)

type loadedMsg struct {
	envelopes []envelope.Envelope
	err       error
}

type savedMsg struct{ err error }

// Model is the Envelopes screen model.
type Model struct {
	store       storeapi.Store
	householdID account.HouseholdID
	width       int
	mode        screenMode

	loading   bool
	spinner   spinner.Model
	envelopes []envelope.Envelope
	cursor    int
	err       string

	// create form fields
	form         *huh.Form
	formName     string
	formTarget   string
	formCurrency string
	formPeriod   string
	formRollover string
}

// New constructs an envelopes model.
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
			m.envelopes = msg.envelopes
			m.err = ""
		}
		return m, nil

	case savedMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.mode = modeList
			return m, nil
		}
		m.mode = modeList
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, m.loadData())

	case tea.KeyMsg:
		if m.mode == modeCreate {
			if msg.String() == "esc" {
				m.mode = modeList
				return m, nil
			}
			// Let huh form handle the key.
			form, cmd := m.form.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.form = f
			}
			if m.form.State == huh.StateCompleted {
				return m, m.saveEnvelope()
			}
			return m, cmd
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.envelopes)-1 {
				m.cursor++
			}
		case "n":
			return m.openCreateForm()
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadData())
		}
	}

	if m.mode == modeCreate && m.form != nil {
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		if m.form.State == huh.StateCompleted {
			return m, m.saveEnvelope()
		}
		return m, cmd
	}

	return m, nil
}

func (m Model) View() string {
	if m.mode == modeCreate && m.form != nil {
		return "\n" + m.form.View()
	}

	if m.loading {
		return "\n  " + m.spinner.View() + " Loading envelopes…"
	}
	if m.err != "" {
		return styles.ErrorText.Render("\n  Error: "+m.err) + "\n\n" + renderHints()
	}
	if len(m.envelopes) == 0 {
		return styles.MutedText.Render("\n  No envelopes found.") + "\n\n" + renderHints()
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

	b.WriteString("\n" + renderHints() + "\n")
	return b.String()
}

func renderHints() string {
	return styles.MutedText.Render("  ↑/↓ navigate  n new  r refresh")
}

func (m Model) openCreateForm() (Model, tea.Cmd) {
	m.formName = ""
	m.formTarget = ""
	m.formCurrency = "USD"
	m.formPeriod = string(envelope.PeriodMonthly)
	m.formRollover = string(envelope.RolloverZero)

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("A short label for this budget envelope").
				Value(&m.formName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Target amount").
				Description("Monthly/periodic budget target (e.g. 500.00)").
				Value(&m.formTarget).
				Validate(func(s string) error {
					v, err := decimal.NewFromString(s)
					if err != nil || v.IsNegative() {
						return fmt.Errorf("enter a positive decimal amount")
					}
					return nil
				}),
			huh.NewInput().
				Title("Currency").
				Description("ISO 4217 currency code (e.g. USD)").
				Value(&m.formCurrency).
				Validate(func(s string) error {
					if len(strings.TrimSpace(s)) == 0 {
						return fmt.Errorf("currency is required")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Period type").
				Options(
					huh.NewOption("Monthly", string(envelope.PeriodMonthly)),
					huh.NewOption("Quarterly", string(envelope.PeriodQuarterly)),
					huh.NewOption("Annual", string(envelope.PeriodAnnual)),
					huh.NewOption("One-time", string(envelope.PeriodOnce)),
				).
				Value(&m.formPeriod),
			huh.NewSelect[string]().
				Title("Rollover policy").
				Description("What happens to unspent funds at period end").
				Options(
					huh.NewOption("Zero (unspent funds reset)", string(envelope.RolloverZero)),
					huh.NewOption("Carry forward", string(envelope.RolloverCarry)),
					huh.NewOption("Cap at target then zero", string(envelope.RolloverCap)),
				).
				Value(&m.formRollover),
		),
	)

	m.mode = modeCreate
	return m, m.form.Init()
}

func (m Model) saveEnvelope() tea.Cmd {
	return func() tea.Msg {
		val, err := decimal.NewFromString(m.formTarget)
		if err != nil {
			val = decimal.Zero
		}
		e := envelope.Envelope{
			ID:          envelope.EnvelopeID(uuid.NewString()),
			HouseholdID: m.householdID,
			Name:        strings.TrimSpace(m.formName),
			TargetAmount: currency.Amount{
				Value:    val,
				Currency: currency.Currency(strings.ToUpper(strings.TrimSpace(m.formCurrency))),
			},
			PeriodType:     envelope.PeriodType(m.formPeriod),
			RolloverPolicy: envelope.RolloverPolicy(m.formRollover),
		}
		createErr := m.store.CreateEnvelope(context.Background(), e)
		return savedMsg{err: createErr}
	}
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
