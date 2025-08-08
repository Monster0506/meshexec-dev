package tui

import "github.com/charmbracelet/lipgloss"

type theme struct {
	Background lipgloss.Color
	Surface    lipgloss.Color
	Muted      lipgloss.Color
	Text       lipgloss.Color
	AccentA    lipgloss.Color
	AccentB    lipgloss.Color
	Good       lipgloss.Color
	Warn       lipgloss.Color
	Bad        lipgloss.Color
}

func defaultTheme() theme {
	return theme{
		Background: lipgloss.Color("#0D1117"),
		Surface:    lipgloss.Color("#161B22"),
		Muted:      lipgloss.Color("#8B949E"),
		Text:       lipgloss.Color("#C9D1D9"),
		AccentA:    lipgloss.Color("#58A6FF"),
		AccentB:    lipgloss.Color("#30363D"),
		Good:       lipgloss.Color("#3FB950"),
		Warn:       lipgloss.Color("#D29922"),
		Bad:        lipgloss.Color("#F85149"),
	}
}
