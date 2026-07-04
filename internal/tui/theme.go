package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds all styles used across the UI.
type Theme struct {
	User      lipgloss.Style
	Assistant lipgloss.Style
	Tool      lipgloss.Style
	ToolErr   lipgloss.Style
	Prompt    lipgloss.Style
	Border    lipgloss.Style
	Status    lipgloss.Style
	Title     lipgloss.Style
	Dim       lipgloss.Style
	Header    lipgloss.Style
	Panel     lipgloss.Style
	InputBox  lipgloss.Style
	Hint      lipgloss.Style
	Glow      lipgloss.Style
}

// DraculaTheme is now a premium dark enterprise terminal theme inspired by
// modern agent CLIs: minimal chrome, crisp borders, subtle green accent, and
// high-contrast prompt panels.
func DraculaTheme() Theme {
	var (
		bg      = lipgloss.Color("#0f1115")
		panel   = lipgloss.Color("#121417")
		border  = lipgloss.Color("#30343f")
		border2 = lipgloss.Color("#5e6472")
		accent  = lipgloss.Color("#7CFF9B")
		accent2 = lipgloss.Color("#B8FFCA")
		blue    = lipgloss.Color("#8AB4FF")
		red     = lipgloss.Color("#FF5C7A")
		muted   = lipgloss.Color("#777D8B")
		fg      = lipgloss.Color("#ECEFF4")
	)
	return Theme{
		User:      lipgloss.NewStyle().Foreground(accent2).Bold(true),
		Assistant: lipgloss.NewStyle().Foreground(fg),
		Tool:      lipgloss.NewStyle().Foreground(accent).Bold(true),
		ToolErr:   lipgloss.NewStyle().Foreground(red).Bold(true),
		Prompt:    lipgloss.NewStyle().Foreground(accent).Bold(true),
		Border:    lipgloss.NewStyle().Foreground(border2).BorderForeground(border2),
		Status:    lipgloss.NewStyle().Foreground(muted).Background(bg),
		Title:     lipgloss.NewStyle().Foreground(fg).Bold(true),
		Dim:       lipgloss.NewStyle().Foreground(muted),
		Header:    lipgloss.NewStyle().Foreground(fg).Background(lipgloss.Color("#191C23")).Padding(0, 1),
		Panel:     lipgloss.NewStyle().Foreground(fg).Background(panel).Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(0, 1),
		InputBox:  lipgloss.NewStyle().Foreground(fg).Background(bg).Border(lipgloss.NormalBorder()).BorderForeground(border2).Padding(0, 1),
		Hint:      lipgloss.NewStyle().Foreground(muted),
		Glow:      lipgloss.NewStyle().Foreground(blue).Bold(true),
	}
}
