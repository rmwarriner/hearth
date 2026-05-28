package app

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hearth-ledger/hearth/internal/core/account"
	"github.com/hearth-ledger/hearth/internal/store"
	"github.com/hearth-ledger/hearth/internal/tui/accounts"
	"github.com/hearth-ledger/hearth/internal/tui/common"
	"github.com/hearth-ledger/hearth/internal/tui/dashboard"
	"github.com/hearth-ledger/hearth/internal/tui/envelopes"
	"github.com/hearth-ledger/hearth/internal/tui/styles"
	"github.com/hearth-ledger/hearth/internal/tui/transactions"
)

var tabNames = []string{"Dashboard", "Accounts", "Transactions", "Envelopes"}

// App is the root bubbletea model. It owns the tab bar and status bar chrome
// and delegates all screen logic to the active child model.
type App struct {
	store       store.Store
	householdID account.HouseholdID
	width       int
	height      int
	activeTab   int
	tabs        []tea.Model
	errMsg      string // non-empty when an error overlay is shown
}

// New constructs a new App model ready to run.
func New(s store.Store, householdID account.HouseholdID) App {
	return App{
		store:       s,
		householdID: householdID,
		activeTab:   0,
		tabs: []tea.Model{
			dashboard.New(s, householdID),
			accounts.New(s, householdID),
			transactions.New(s, householdID),
			envelopes.New(s, householdID),
		},
	}
}

// Start launches the TUI program. It blocks until the user quits.
func Start(s store.Store, householdID account.HouseholdID) error {
	p := tea.NewProgram(New(s, householdID), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (a App) Init() tea.Cmd {
	if len(a.tabs) == 0 {
		return nil
	}
	return a.tabs[a.activeTab].Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = m.Width
		a.height = m.Height
		// Propagate size to all child models.
		for i, tab := range a.tabs {
			updated, _ := tab.Update(msg)
			a.tabs[i] = updated
		}
		return a, nil

	case tea.KeyMsg:
		// Dismiss error overlay first if one is shown.
		if a.errMsg != "" {
			switch m.String() {
			case "esc", "enter":
				a.errMsg = ""
				return a, nil
			}
			return a, nil
		}

		switch m.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "1", "2", "3", "4":
			idx := int(m.Runes[0] - '1')
			return a.switchTab(idx)
		case "tab":
			return a.switchTab((a.activeTab + 1) % len(a.tabs))
		case "shift+tab":
			return a.switchTab((a.activeTab + len(a.tabs) - 1) % len(a.tabs))
		}
	}

	// Delegate to active child.
	if len(a.tabs) > 0 {
		updated, cmd := a.tabs[a.activeTab].Update(msg)
		a.tabs[a.activeTab] = updated
		return a, cmd
	}
	return a, nil
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading…"
	}

	tabBar := a.renderTabBar()
	statusBar := common.RenderStatusBar(a.width, tabNames[a.activeTab])

	// Content area height = total - tab bar - status bar.
	contentHeight := a.height - lipgloss.Height(tabBar) - lipgloss.Height(statusBar)
	if contentHeight < 0 {
		contentHeight = 0
	}

	var content string
	if len(a.tabs) > 0 {
		content = a.tabs[a.activeTab].View()
	}
	// Pad/trim content to fill the available height.
	content = lipgloss.NewStyle().Height(contentHeight).Width(a.width).Render(content)

	view := tabBar + "\n" + content + "\n" + statusBar

	if a.errMsg != "" {
		overlay := common.RenderErrorPanel(a.errMsg, a.width)
		// Centre the overlay vertically and horizontally.
		ow := lipgloss.Width(overlay)
		oh := lipgloss.Height(overlay)
		padLeft := (a.width - ow) / 2
		padTop := (a.height - oh) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		if padTop < 0 {
			padTop = 0
		}
		_ = padTop // full overlay positioning requires more complex rendering; show inline for now
		view += "\n" + lipgloss.NewStyle().PaddingLeft(padLeft).Render(overlay)
	}

	return view
}

func (a App) renderTabBar() string {
	var tabs []string
	for i, name := range tabNames {
		label := fmt.Sprintf("[%d] %s", i+1, name)
		if i == a.activeTab {
			tabs = append(tabs, styles.ActiveTab.Render(label))
		} else {
			tabs = append(tabs, styles.InactiveTab.Render(label))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	return lipgloss.NewStyle().
		Width(a.width).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.Muted).
		Render(bar)
}

func (a App) switchTab(idx int) (App, tea.Cmd) {
	if idx < 0 || idx >= len(a.tabs) {
		return a, nil
	}
	a.activeTab = idx
	return a, a.tabs[idx].Init()
}
