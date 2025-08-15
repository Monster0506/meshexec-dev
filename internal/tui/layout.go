package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Layout composes the banner, panes, and footer/status into a full view string.
type Layout struct {
	Theme Theme
}

// RenderRoot renders a simple placeholder application layout.
func (l Layout) RenderRoot(width, height int, banner string, pane Pane, footer string) string {
	bannerView := l.Theme.BannerText.Render(banner)
	footerView := l.Theme.StatusBar.Render(footer)
	paneHeight := height - lipgloss.Height(bannerView) - lipgloss.Height(footerView)
	if paneHeight < 3 {
		paneHeight = 3
	}
	content := pane.Render(width, paneHeight, l.Theme)
	return lipgloss.JoinVertical(lipgloss.Left, bannerView, content, footerView)
}
