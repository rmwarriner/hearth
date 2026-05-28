package common

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

// RenderErrorPanel renders a dismissable error message box.
func RenderErrorPanel(msg string, width int) string {
	maxW := width - 8
	if maxW < 20 {
		maxW = 20
	}
	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Error).
		Foreground(styles.Error).
		Width(maxW).
		Padding(1, 2).
		Render("Error: " + msg + "\n\nPress Esc or Enter to dismiss")
	return panel
}
