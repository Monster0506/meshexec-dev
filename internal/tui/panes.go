package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// PaneID enumerates available panes in the app.
type PaneID int

const (
	PaneOverview PaneID = iota
	PanePeers
	PaneResults
	PaneCommands
)

// Pane is a minimal contract for a content area.
type Pane interface {
	Title() string
	Render(width, height int, theme Theme) string
}

// BasicPane is a simple placeholder implementation.
type BasicPane struct {
	name string
	body string
}

func NewBasicPane(name, body string) *BasicPane { return &BasicPane{name: name, body: body} }

func (p *BasicPane) Title() string { return p.name }

func (p *BasicPane) Render(width, height int, theme Theme) string {
	box := lipgloss.NewStyle().Width(width).Height(height).Border(lipgloss.NormalBorder()).BorderForeground(theme.Secondary).Padding(0, 1)
	title := theme.PaneTitle.Render(p.Title())
	body := theme.AppBackground.Foreground(theme.Foreground).Render(p.body)
	return box.Render(fmt.Sprintf("%s\n%s", title, body))
}
