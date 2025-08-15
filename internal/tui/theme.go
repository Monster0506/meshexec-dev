package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines colors and styles used across the TUI.
type Theme struct {
	// Base colors
	Primary    lipgloss.Color
	Secondary  lipgloss.Color
	Accent     lipgloss.Color
	Danger     lipgloss.Color
	Warning    lipgloss.Color
	Success    lipgloss.Color
	Foreground lipgloss.Color
	Background lipgloss.Color

	// Styles
	AppBackground lipgloss.Style
	BannerText    lipgloss.Style
	PaneBorder    lipgloss.Style
	PaneTitle     lipgloss.Style
	PopupBorder   lipgloss.Style
	PopupBody     lipgloss.Style
	StatusBar     lipgloss.Style
}

// ThemeType represents different theme variants
type ThemeType string

const (
	ThemeDark         ThemeType = "dark"
	ThemeLight        ThemeType = "light"
	ThemeHighContrast ThemeType = "hc"
	ThemeOcean        ThemeType = "ocean"
	ThemeForest       ThemeType = "forest"
	ThemeSunset       ThemeType = "sunset"
	ThemeCyberpunk    ThemeType = "cyberpunk"
	ThemeRetro        ThemeType = "retro"
	ThemeMonokai      ThemeType = "monokai"
	ThemeNord         ThemeType = "nord"
	ThemeDracula      ThemeType = "dracula"
	ThemeSolarized    ThemeType = "solarized"
	ThemeGruvbox      ThemeType = "gruvbox"
	ThemeTokyo        ThemeType = "tokyo"
	ThemeCandy        ThemeType = "candy"
)

// defaultTheme returns a tasteful dark theme as a starting point.
func defaultTheme() Theme {
	return getTheme(ThemeDark)
}

// getTheme returns a theme based on the theme type
func getTheme(themeType ThemeType) Theme {
	switch themeType {
	case ThemeDark:
		return darkTheme()
	case ThemeLight:
		return lightTheme()
	case ThemeHighContrast:
		return highContrastTheme()
	case ThemeOcean:
		return oceanTheme()
	case ThemeForest:
		return forestTheme()
	case ThemeSunset:
		return sunsetTheme()
	case ThemeCyberpunk:
		return cyberpunkTheme()
	case ThemeRetro:
		return retroTheme()
	case ThemeMonokai:
		return monokaiTheme()
	case ThemeNord:
		return nordTheme()
	case ThemeDracula:
		return draculaTheme()
	case ThemeSolarized:
		return solarizedTheme()
	case ThemeGruvbox:
		return gruvboxTheme()
	case ThemeTokyo:
		return tokyoTheme()
	case ThemeCandy:
		return candyTheme()
	default:
		return darkTheme()
	}
}

// darkTheme - Professional dark theme with blue accents
func darkTheme() Theme {
	fg := lipgloss.Color("#E6E6E6")
	bg := lipgloss.Color("#0E0E10")
	primary := lipgloss.Color("#6C9AFF")
	secondary := lipgloss.Color("#8A8F98")
	accent := lipgloss.Color("#A78BFA")
	danger := lipgloss.Color("#FF6C6B")
	warn := lipgloss.Color("#FBBF24")
	success := lipgloss.Color("#34D399")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(base.Secondary)
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Accent)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#16161A"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#1F1F23")).Foreground(base.Secondary)

	return base
}

// lightTheme - Clean light theme for bright environments
func lightTheme() Theme {
	fg := lipgloss.Color("#2D3748")
	bg := lipgloss.Color("#F7FAFC")
	primary := lipgloss.Color("#3182CE")
	secondary := lipgloss.Color("#718096")
	accent := lipgloss.Color("#805AD5")
	danger := lipgloss.Color("#E53E3E")
	warn := lipgloss.Color("#D69E2E")
	success := lipgloss.Color("#38A169")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#E2E8F0"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Accent)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#FFFFFF"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#EDF2F7")).Foreground(base.Secondary)

	return base
}

// highContrastTheme - Accessibility-focused theme
func highContrastTheme() Theme {
	fg := lipgloss.Color("#FFFFFF")
	bg := lipgloss.Color("#000000")
	primary := lipgloss.Color("#00FFFF")
	secondary := lipgloss.Color("#C0C0C0")
	accent := lipgloss.Color("#FFFF00")
	danger := lipgloss.Color("#FF0000")
	warn := lipgloss.Color("#FFA500")
	success := lipgloss.Color("#00FF00")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(base.Secondary).Border(lipgloss.ThickBorder())
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Accent).Border(lipgloss.ThickBorder())
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#111111"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#222222")).Foreground(base.Secondary)

	return base
}

// oceanTheme - Deep blue ocean-inspired theme
func oceanTheme() Theme {
	fg := lipgloss.Color("#E6F3FF")
	bg := lipgloss.Color("#0A1929")
	primary := lipgloss.Color("#4FC3F7")
	secondary := lipgloss.Color("#81C784")
	accent := lipgloss.Color("#FFB74D")
	danger := lipgloss.Color("#F44336")
	warn := lipgloss.Color("#FF9800")
	success := lipgloss.Color("#4CAF50")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#1E3A5F"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Secondary).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Accent)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#102027"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#1E3A5F")).Foreground(base.Secondary)

	return base
}

// forestTheme - Nature-inspired green theme
func forestTheme() Theme {
	fg := lipgloss.Color("#E8F5E8")
	bg := lipgloss.Color("#1B2F1B")
	primary := lipgloss.Color("#66BB6A")
	secondary := lipgloss.Color("#81C784")
	accent := lipgloss.Color("#FFB74D")
	danger := lipgloss.Color("#EF5350")
	warn := lipgloss.Color("#FFA726")
	success := lipgloss.Color("#4CAF50")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#2E4A2E"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Secondary).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Accent)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#243324"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#2E4A2E")).Foreground(base.Secondary)

	return base
}

// sunsetTheme - Warm orange and purple gradient theme
func sunsetTheme() Theme {
	fg := lipgloss.Color("#FFF8E1")
	bg := lipgloss.Color("#2D1B69")
	primary := lipgloss.Color("#FF7043")
	secondary := lipgloss.Color("#AB47BC")
	accent := lipgloss.Color("#FFD54F")
	danger := lipgloss.Color("#D32F2F")
	warn := lipgloss.Color("#FF8F00")
	success := lipgloss.Color("#388E3C")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#4A148C"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Secondary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#3F1F5F"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#4A148C")).Foreground(base.Secondary)

	return base
}

// cyberpunkTheme - Neon cyberpunk aesthetic
func cyberpunkTheme() Theme {
	fg := lipgloss.Color("#00FF41")
	bg := lipgloss.Color("#0A0A0A")
	primary := lipgloss.Color("#FF0080")
	secondary := lipgloss.Color("#00FFFF")
	accent := lipgloss.Color("#FFFF00")
	danger := lipgloss.Color("#FF0000")
	warn := lipgloss.Color("#FF8000")
	success := lipgloss.Color("#00FF00")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(base.Secondary)
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#1A1A1A"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#2A2A2A")).Foreground(base.Secondary)

	return base
}

// retroTheme - 80s retro computing aesthetic
func retroTheme() Theme {
	fg := lipgloss.Color("#00FF00")
	bg := lipgloss.Color("#000000")
	primary := lipgloss.Color("#FF00FF")
	secondary := lipgloss.Color("#00FFFF")
	accent := lipgloss.Color("#FFFF00")
	danger := lipgloss.Color("#FF0000")
	warn := lipgloss.Color("#FF8000")
	success := lipgloss.Color("#00FF00")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(base.Secondary)
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#111111"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#222222")).Foreground(base.Secondary)

	return base
}

// monokaiTheme - Classic Monokai color scheme
func monokaiTheme() Theme {
	fg := lipgloss.Color("#F8F8F2")
	bg := lipgloss.Color("#272822")
	primary := lipgloss.Color("#F92672")
	secondary := lipgloss.Color("#75715E")
	accent := lipgloss.Color("#A6E22E")
	danger := lipgloss.Color("#F92672")
	warn := lipgloss.Color("#FD971F")
	success := lipgloss.Color("#A6E22E")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#3E3D32"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#3E3D32"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#3E3D32")).Foreground(base.Secondary)

	return base
}

// nordTheme - Arctic-inspired Nord color scheme
func nordTheme() Theme {
	fg := lipgloss.Color("#ECEFF4")
	bg := lipgloss.Color("#2E3440")
	primary := lipgloss.Color("#88C0D0")
	secondary := lipgloss.Color("#81A1C1")
	accent := lipgloss.Color("#B48EAD")
	danger := lipgloss.Color("#BF616A")
	warn := lipgloss.Color("#EBCB8B")
	success := lipgloss.Color("#A3BE8C")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#3B4252"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#3B4252"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#3B4252")).Foreground(base.Secondary)

	return base
}

// draculaTheme - Dracula color scheme
func draculaTheme() Theme {
	fg := lipgloss.Color("#F8F8F2")
	bg := lipgloss.Color("#282A36")
	primary := lipgloss.Color("#BD93F9")
	secondary := lipgloss.Color("#6272A4")
	accent := lipgloss.Color("#FF79C6")
	danger := lipgloss.Color("#FF5555")
	warn := lipgloss.Color("#FFB86C")
	success := lipgloss.Color("#50FA7B")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#44475A"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#44475A"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#44475A")).Foreground(base.Secondary)

	return base
}

// solarizedTheme - Solarized dark color scheme
func solarizedTheme() Theme {
	fg := lipgloss.Color("#839496")
	bg := lipgloss.Color("#002B36")
	primary := lipgloss.Color("#268BD2")
	secondary := lipgloss.Color("#586E75")
	accent := lipgloss.Color("#D33682")
	danger := lipgloss.Color("#DC322F")
	warn := lipgloss.Color("#CB4B16")
	success := lipgloss.Color("#859900")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#073642"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#073642"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#073642")).Foreground(base.Secondary)

	return base
}

// gruvboxTheme - Gruvbox dark color scheme
func gruvboxTheme() Theme {
	fg := lipgloss.Color("#EBDBB2")
	bg := lipgloss.Color("#282828")
	primary := lipgloss.Color("#83A598")
	secondary := lipgloss.Color("#928374")
	accent := lipgloss.Color("#D3869B")
	danger := lipgloss.Color("#FB4934")
	warn := lipgloss.Color("#FE8019")
	success := lipgloss.Color("#B8BB26")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#3C3836"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#3C3836"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#3C3836")).Foreground(base.Secondary)

	return base
}

// tokyoTheme - Tokyo Night inspired theme
func tokyoTheme() Theme {
	fg := lipgloss.Color("#A9B1D6")
	bg := lipgloss.Color("#1A1B26")
	primary := lipgloss.Color("#7AA2F7")
	secondary := lipgloss.Color("#565A6E")
	accent := lipgloss.Color("#BB9AF7")
	danger := lipgloss.Color("#F7768E")
	warn := lipgloss.Color("#E0AF68")
	success := lipgloss.Color("#9ECE6A")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#24283B"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#24283B"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#24283B")).Foreground(base.Secondary)

	return base
}

// candyTheme - Sweet pastel candy theme
func candyTheme() Theme {
	fg := lipgloss.Color("#2D1B69")
	bg := lipgloss.Color("#FFE5F1")
	primary := lipgloss.Color("#FF69B4")
	secondary := lipgloss.Color("#87CEEB")
	accent := lipgloss.Color("#DDA0DD")
	danger := lipgloss.Color("#FF6B6B")
	warn := lipgloss.Color("#FFB347")
	success := lipgloss.Color("#98FB98")

	base := Theme{
		Primary:    primary,
		Secondary:  secondary,
		Accent:     accent,
		Danger:     danger,
		Warning:    warn,
		Success:    success,
		Foreground: fg,
		Background: bg,
	}

	base.AppBackground = lipgloss.NewStyle().Background(base.Background)
	base.BannerText = lipgloss.NewStyle().Foreground(base.Primary).Bold(true)
	base.PaneBorder = lipgloss.NewStyle().BorderForeground(lipgloss.Color("#FFC0CB"))
	base.PaneTitle = lipgloss.NewStyle().Foreground(base.Accent).Bold(true)
	base.PopupBorder = lipgloss.NewStyle().BorderForeground(base.Primary)
	base.PopupBody = lipgloss.NewStyle().Foreground(base.Foreground).Background(lipgloss.Color("#FFF0F5"))
	base.StatusBar = lipgloss.NewStyle().Background(lipgloss.Color("#FFC0CB")).Foreground(base.Secondary)

	return base
}
