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

	// Extended semantic tokens
	Primary       lipgloss.Color
	Secondary     lipgloss.Color
	Border        lipgloss.Color
	SelectionBg   lipgloss.Color
	SelectionText lipgloss.Color
	ChipBg        lipgloss.Color
	ChipText      lipgloss.Color
	DangerBg      lipgloss.Color
	WarnBg        lipgloss.Color
	InfoBg        lipgloss.Color
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

		Primary:       lipgloss.Color("#58A6FF"),
		Secondary:     lipgloss.Color("#1F6FEB"),
		Border:        lipgloss.Color("#30363D"),
		SelectionBg:   lipgloss.Color("#1F6FEB"),
		SelectionText: lipgloss.Color("#FFFFFF"),
		ChipBg:        lipgloss.Color("#21262D"),
		ChipText:      lipgloss.Color("#C9D1D9"),
		DangerBg:      lipgloss.Color("#2B1313"),
		WarnBg:        lipgloss.Color("#2C2213"),
		InfoBg:        lipgloss.Color("#141A22"),
	}
}

// themeFor returns a themed palette by name. Supported: "dark" (default), "light", "hc" (high-contrast)
func themeFor(name string) theme {
	switch name {
	case "light":
		return theme{
			Background: lipgloss.Color("#FFFFFF"),
			Surface:    lipgloss.Color("#F6F8FA"),
			Muted:      lipgloss.Color("#57606A"),
			Text:       lipgloss.Color("#24292F"),
			AccentA:    lipgloss.Color("#0969DA"),
			AccentB:    lipgloss.Color("#D0D7DE"),
			Good:       lipgloss.Color("#1A7F37"),
			Warn:       lipgloss.Color("#9A6700"),
			Bad:        lipgloss.Color("#CF222E"),

			Primary:       lipgloss.Color("#0969DA"),
			Secondary:     lipgloss.Color("#218BFF"),
			Border:        lipgloss.Color("#D0D7DE"),
			SelectionBg:   lipgloss.Color("#218BFF"),
			SelectionText: lipgloss.Color("#FFFFFF"),
			ChipBg:        lipgloss.Color("#EAECEF"),
			ChipText:      lipgloss.Color("#24292F"),
			DangerBg:      lipgloss.Color("#FFE3E3"),
			WarnBg:        lipgloss.Color("#FFF5DA"),
			InfoBg:        lipgloss.Color("#E7F3FF"),
		}
	case "hc", "high-contrast":
		return theme{
			Background: lipgloss.Color("#000000"),
			Surface:    lipgloss.Color("#000000"),
			Muted:      lipgloss.Color("#AAAAAA"),
			Text:       lipgloss.Color("#FFFFFF"),
			AccentA:    lipgloss.Color("#FFFFFF"),
			AccentB:    lipgloss.Color("#FFFFFF"),
			Good:       lipgloss.Color("#00FF00"),
			Warn:       lipgloss.Color("#FFFF00"),
			Bad:        lipgloss.Color("#FF0000"),

			Primary:       lipgloss.Color("#FFFFFF"),
			Secondary:     lipgloss.Color("#FFFFFF"),
			Border:        lipgloss.Color("#FFFFFF"),
			SelectionBg:   lipgloss.Color("#FFFFFF"),
			SelectionText: lipgloss.Color("#000000"),
			ChipBg:        lipgloss.Color("#000000"),
			ChipText:      lipgloss.Color("#FFFFFF"),
			DangerBg:      lipgloss.Color("#330000"),
			WarnBg:        lipgloss.Color("#333300"),
			InfoBg:        lipgloss.Color("#001133"),
		}
	default:
		return defaultTheme()
	}
}
