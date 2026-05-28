package accounts

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

type screenMode int

const (
	modeList screenMode = iota
	modeCreate
)

type loadedMsg struct {
	rows []accountRow
	err  error
}

type savedMsg struct{ err error }

type accountRow struct {
	acct    account.Account
	balance currency.Amount
}

// Model is the Accounts screen model.
type Model struct {
	store       storeapi.Store
	householdID account.HouseholdID
	width       int
	mode        screenMode

	loading bool
	spinner spinner.Model
	rows    []accountRow
	cursor  int
	err     string

	// create form fields
	form         *huh.Form
	formName     string
	formType     string
	formCurrency string
}

// New constructs an accounts model.
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
			m.rows = msg.rows
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
			form, cmd := m.form.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.form = f
			}
			if m.form.State == huh.StateCompleted {
				return m, m.saveAccount()
			}
			return m, cmd
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
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
			return m, m.saveAccount()
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
		return "\n  " + m.spinner.View() + " Loading accounts…"
	}
	if m.err != "" {
		return styles.ErrorText.Render("\n  Error: "+m.err) + "\n\n" + renderHints()
	}
	if len(m.rows) == 0 {
		return styles.MutedText.Render("\n  No accounts found.") + "\n\n" + renderHints()
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

	b.WriteString("\n" + renderHints() + "\n")
	return b.String()
}

func renderHints() string {
	return styles.MutedText.Render("  ↑/↓ navigate  n new  r refresh")
}

func (m Model) openCreateForm() (Model, tea.Cmd) {
	m.formName = ""
	m.formType = string(account.Asset)
	m.formCurrency = "USD"

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Account name").
				Description("A descriptive name for this account").
				Value(&m.formName).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Account type").
				Options(
					huh.NewOption("Asset", string(account.Asset)),
					huh.NewOption("Liability", string(account.Liability)),
					huh.NewOption("Equity", string(account.Equity)),
					huh.NewOption("Income", string(account.Income)),
					huh.NewOption("Expense", string(account.Expense)),
				).
				Value(&m.formType),
			huh.NewInput().
				Title("Currency").
				Description("ISO 4217 currency code (e.g. USD)").
				Value(&m.formCurrency).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("currency is required")
					}
					return nil
				}),
		),
	)

	m.mode = modeCreate
	return m, m.form.Init()
}

func (m Model) saveAccount() tea.Cmd {
	return func() tea.Msg {
		a := account.Account{
			ID:          account.AccountID(uuid.NewString()),
			HouseholdID: m.householdID,
			Name:        strings.TrimSpace(m.formName),
			Type:        account.AccountType(m.formType),
			Currency:    currency.Currency(strings.ToUpper(strings.TrimSpace(m.formCurrency))),
		}
		if err := a.Validate(); err != nil {
			return savedMsg{err: err}
		}
		return savedMsg{err: m.store.CreateAccount(context.Background(), a)}
	}
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
