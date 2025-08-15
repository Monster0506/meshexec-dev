package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Popup is a centered window overlay with a title and body.
type Popup struct {
	Title string
	Body  string
}

func (p Popup) Render(totalWidth, totalHeight int, theme Theme) string {
	maxWidth := totalWidth * 3 / 4
	if maxWidth < 20 {
		maxWidth = totalWidth
	}
	content := lipgloss.JoinVertical(lipgloss.Left, theme.PaneTitle.Render(p.Title), p.Body)
	inner := lipgloss.NewStyle().Width(maxWidth).Align(lipgloss.Center).Render(content)
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(theme.Accent).Padding(1, 2)
	modal := box.Render(inner)

	// Center within the screen area by applying margins
	hMargin := (totalWidth - lipgloss.Width(modal)) / 2
	vMargin := (totalHeight - lipgloss.Height(modal)) / 3
	if hMargin < 0 {
		hMargin = 0
	}
	if vMargin < 0 {
		vMargin = 0
	}
	return lipgloss.NewStyle().Margin(vMargin, hMargin).Render(modal)
}
