package styles

import "github.com/charmbracelet/lipgloss"

var (
	Primary   = lipgloss.Color("#7C9CCA")
	Secondary = lipgloss.Color("#9EC6A0")
	Muted     = lipgloss.Color("#6C7A89")
	Error     = lipgloss.Color("#E06C75")
	Success   = lipgloss.Color("#98C379")
	Warning   = lipgloss.Color("#E5C07B")
)

var (
	ActiveTab = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			Padding(0, 1)

	InactiveTab = lipgloss.NewStyle().
			Foreground(Muted).
			Padding(0, 1)

	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#2C3140")).
			Foreground(lipgloss.Color("#CDD6F4")).
			Padding(0, 1)

	Border = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(0, 1)

	TableHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	TableRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4"))

	TableSelectedRow = lipgloss.NewStyle().
				Background(lipgloss.Color("#3A3F52")).
				Foreground(lipgloss.Color("#CDD6F4"))

	ErrorText = lipgloss.NewStyle().Foreground(Error)
	MutedText = lipgloss.NewStyle().Foreground(Muted)
	BoldText  = lipgloss.NewStyle().Bold(true)
)
