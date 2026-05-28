package common

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/hearth-ledger/hearth/internal/tui/styles"
)

// RenderStatusBar returns the full-width footer bar.
// Left: AI tier indicator. Centre: screen name. Right: shortcut hints.
func RenderStatusBar(width int, screenName string) string {
	left := styles.StatusBar.Render("[AI: OFF]")
	centre := styles.StatusBar.Render(screenName)
	right := styles.StatusBar.Render("? help  q quit")

	leftWidth := lipgloss.Width(left)
	centreWidth := lipgloss.Width(centre)
	rightWidth := lipgloss.Width(right)

	totalPad := width - leftWidth - centreWidth - rightWidth
	if totalPad < 0 {
		totalPad = 0
	}
	leftPad := totalPad / 2
	rightPad := totalPad - leftPad

	bar := left +
		styles.StatusBar.Width(leftPad).Render("") +
		centre +
		styles.StatusBar.Width(rightPad).Render("") +
		right

	return styles.StatusBar.Width(width).Render(bar)
}
