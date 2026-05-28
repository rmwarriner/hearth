package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/core/currency"
	"github.com/hearth-ledger/hearth/internal/core/gaap"
	"github.com/hearth-ledger/hearth/internal/core/journal"
	storeapi "github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

type screenMode int

const (
	modeList       screenMode = iota
	modeFormStep1             // header (description, date)
	modeFormStep2             // postings
	modeFormReview            // GAAP validation results
)

type loadedMsg struct {
	entries []journal.JournalEntry
	err     error
}

type savedMsg struct{ err error }

type postingDraft struct {
	accountID string
	amount    string
	memo      string
}

// Model is the Transactions screen model.
type Model struct {
	store       storeapi.Store
	householdID account.HouseholdID
	width       int
	mode        screenMode

	loading bool
	spinner spinner.Model
	entries []journal.JournalEntry
	cursor  int
	err     string

	// form step 1 — header
	formStep1 *huh.Form
	formDesc  string
	formDate  string
	formRef   string

	// form step 2 — postings (we use a fixed 2-posting form for simplicity)
	formStep2   *huh.Form
	postings    []postingDraft
	accountOpts []huh.Option[string]

	// review
	gaapViolations []string
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
		postings:    []postingDraft{{}, {}},
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
		switch m.mode {
		case modeFormStep1:
			if msg.String() == "esc" {
				m.mode = modeList
				return m, nil
			}
			form, cmd := m.formStep1.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.formStep1 = f
			}
			if m.formStep1.State == huh.StateCompleted {
				return m.openStep2()
			}
			return m, cmd

		case modeFormStep2:
			if msg.String() == "esc" {
				m.mode = modeFormStep1
				return m, nil
			}
			form, cmd := m.formStep2.Update(msg)
			if f, ok := form.(*huh.Form); ok {
				m.formStep2 = f
			}
			if m.formStep2.State == huh.StateCompleted {
				return m.validateAndReview()
			}
			return m, cmd

		case modeFormReview:
			switch msg.String() {
			case "esc", "b":
				m.mode = modeFormStep2
				return m, nil
			case "enter", "y":
				if len(m.gaapViolations) == 0 {
					return m, m.saveEntry()
				}
			}
			return m, nil

		default: // modeList
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.entries)-1 {
					m.cursor++
				}
			case "n":
				return m.openStep1()
			case "r":
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, m.loadData())
			}
		}
	}

	return m, nil
}

func (m Model) View() string {
	switch m.mode {
	case modeFormStep1:
		if m.formStep1 != nil {
			return "\n  Step 1 of 3 — Entry header\n\n" + m.formStep1.View()
		}
	case modeFormStep2:
		if m.formStep2 != nil {
			return "\n  Step 2 of 3 — Postings\n\n" + m.formStep2.View()
		}
	case modeFormReview:
		return m.reviewView()
	}

	if m.loading {
		return "\n  " + m.spinner.View() + " Loading transactions…"
	}
	if m.err != "" {
		return styles.ErrorText.Render("\n  Error: "+m.err) + "\n\n" + renderHints()
	}
	if len(m.entries) == 0 {
		return styles.MutedText.Render("\n  No transactions found.") + "\n\n" + renderHints()
	}

	var b strings.Builder
	b.WriteString("\n")

	header := fmt.Sprintf("  %-12s  %-40s  %s", "Date", "Description", "Postings")
	b.WriteString(styles.TableHeader.Render(header) + "\n")
	b.WriteString(styles.MutedText.Render("  "+strings.Repeat("─", 60)) + "\n")

	for i, e := range m.entries {
		line := fmt.Sprintf("  %-12s  %-40s  %d postings",
			e.PostedAt.Format("2006-01-02"),
			truncate(e.Description, 40),
			len(e.Postings),
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

func (m Model) reviewView() string {
	var b strings.Builder
	b.WriteString("\n  Step 3 of 3 — Review\n\n")
	fmt.Fprintf(&b, "  Description: %s\n", m.formDesc)
	fmt.Fprintf(&b, "  Date:        %s\n\n", m.formDate)
	b.WriteString("  Postings:\n")
	for _, p := range m.postings {
		fmt.Fprintf(&b, "    %-30s  %s\n", p.accountID, p.amount)
	}
	b.WriteString("\n")

	if len(m.gaapViolations) > 0 {
		b.WriteString(styles.ErrorText.Render("  GAAP violations:\n"))
		for _, v := range m.gaapViolations {
			b.WriteString(styles.ErrorText.Render("    • "+v) + "\n")
		}
		b.WriteString("\n" + styles.MutedText.Render("  Press Esc or b to go back and fix") + "\n")
	} else {
		b.WriteString(styles.ErrorText.Render("  ✓ Entry balances. Press Enter to confirm, Esc to cancel.\n"))
	}
	return b.String()
}

func renderHints() string {
	return styles.MutedText.Render("  ↑/↓ navigate  n new  r refresh")
}

func (m Model) openStep1() (Model, tea.Cmd) {
	m.formDesc = ""
	m.formDate = time.Now().Format("2006-01-02")
	m.formRef = ""
	m.postings = []postingDraft{{}, {}}

	m.formStep1 = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Description").
				Description("What was this transaction for?").
				Value(&m.formDesc).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("description is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Date").
				Description("YYYY-MM-DD (default: today)").
				Value(&m.formDate).
				Validate(func(s string) error {
					_, err := time.Parse("2006-01-02", s)
					if err != nil {
						return fmt.Errorf("use YYYY-MM-DD format")
					}
					return nil
				}),
			huh.NewInput().
				Title("Reference").
				Description("Optional reference number or note").
				Value(&m.formRef),
		),
	)

	m.mode = modeFormStep1
	return m, m.formStep1.Init()
}

func (m Model) openStep2() (Model, tea.Cmd) {
	// Build account options from cached list or reload.
	if len(m.accountOpts) == 0 {
		// Will be populated by a separate load; use empty for now.
		m.accountOpts = []huh.Option[string]{huh.NewOption("(no accounts)", "")}
	}

	p0 := &m.postings[0]
	p1 := &m.postings[1]

	m.formStep2 = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Posting 1 — Account").
				Options(m.accountOpts...).
				Value(&p0.accountID),
			huh.NewInput().
				Title("Posting 1 — Amount").
				Description("Positive = debit, negative = credit (e.g. 100.00 or -100.00)").
				Value(&p0.amount).
				Validate(validateAmount),
			huh.NewSelect[string]().
				Title("Posting 2 — Account").
				Options(m.accountOpts...).
				Value(&p1.accountID),
			huh.NewInput().
				Title("Posting 2 — Amount").
				Description("Must sum to zero with posting 1").
				Value(&p1.amount).
				Validate(validateAmount),
		),
	)

	m.mode = modeFormStep2
	return m, m.formStep2.Init()
}

func validateAmount(s string) error {
	_, err := decimal.NewFromString(s)
	if err != nil {
		return fmt.Errorf("enter a valid decimal amount")
	}
	return nil
}

func (m Model) validateAndReview() (Model, tea.Cmd) {
	m.gaapViolations = nil

	postedAt, err := time.Parse("2006-01-02", m.formDate)
	if err != nil {
		postedAt = time.Now()
	}
	entry := m.buildEntry(postedAt)

	// Build validation context from known accounts.
	ctx := gaap.ValidationContext{
		KnownAccounts: map[account.AccountID]account.HouseholdID{},
	}
	for _, row := range m.entries {
		for _, p := range row.Postings {
			ctx.KnownAccounts[p.AccountID] = m.householdID
		}
	}

	violations := gaap.Validate(entry, ctx)
	for _, v := range violations {
		m.gaapViolations = append(m.gaapViolations, v.Error())
	}

	m.mode = modeFormReview
	return m, nil
}

func (m Model) buildEntry(postedAt time.Time) journal.JournalEntry {
	entryID := journal.EntryID(uuid.NewString())
	var postings []journal.Posting
	for _, pd := range m.postings {
		val, err := decimal.NewFromString(pd.amount)
		if err != nil {
			val = decimal.Zero
		}
		postings = append(postings, journal.Posting{
			ID:             journal.PostingID(uuid.NewString()),
			JournalEntryID: entryID,
			AccountID:      account.AccountID(pd.accountID),
			Amount:         currency.Amount{Value: val, Currency: "USD"},
			Memo:           pd.memo,
		})
	}
	return journal.JournalEntry{
		ID:          entryID,
		HouseholdID: m.householdID,
		PostedAt:    postedAt,
		Description: strings.TrimSpace(m.formDesc),
		Reference:   strings.TrimSpace(m.formRef),
		Source:      journal.SourceManual,
		Postings:    postings,
	}
}

func (m Model) saveEntry() tea.Cmd {
	postedAt, err := time.Parse("2006-01-02", m.formDate)
	if err != nil {
		postedAt = time.Now()
	}
	entry := m.buildEntry(postedAt)

	return func() tea.Msg {
		err := m.store.CreateJournalEntry(context.Background(), entry)
		return savedMsg{err: err}
	}
}

func (m Model) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		accts, err := m.store.ListAccounts(ctx, m.householdID)
		if err != nil {
			return loadedMsg{err: err}
		}

		entries, err := m.store.ListJournalEntries(ctx, storeapi.JournalQuery{
			HouseholdID: m.householdID,
			Limit:       50,
		})
		if err != nil {
			return loadedMsg{err: err}
		}

		// Cache account options for the form.
		_ = accts // used in openStep2 via m.accountOpts — set via a separate msg if needed
		return loadedMsg{entries: entries}
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
