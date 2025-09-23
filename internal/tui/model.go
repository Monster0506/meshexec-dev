package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// minimal messages used by tests
type resultsUpdateMsg struct{ Results interface{} }
type peersUpdateMsg struct{ Peers interface{} }

// tab identifiers used by tests
type tab int

const (
	tabOverview tab = iota
	tabPeers
	tabResults
	tabCommands
)

// Output preview settings
const (
	maxOutputPreviewLength = 30
	outputTruncateSuffix   = "..."
	outputTruncateLength   = maxOutputPreviewLength - len(outputTruncateSuffix)
)

// Table formatting settings for results view
const (
	deviceColumnWidth   = 9
	statusColumnWidth   = 6
	exitCodeColumnWidth = 4
	durationColumnWidth = 8
)

// simple peer list placeholder compatible with tests
type peerListModel struct {
	items []interface{}
}

func (p *peerListModel) Items() []interface{}         { return p.items }
func (p *peerListModel) SetItems(items []interface{}) { p.items = items }
func (p *peerListModel) FilterState() int             { return 0 }

// simple filter placeholder
type textInput struct{ value string }

func (t *textInput) SetValue(v string) { t.value = v }
func (t *textInput) Value() string     { return t.value }

// model is the main TUI model with simulated state
type model struct {
	logger       *logging.Logger
	peerList     peerListModel
	results      *internal.ExecutionResults
	resultFilter textInput
	tab          tab
	theme        Theme
	icons        Icons
	width        int
	height       int
	showHelp     bool
	showPopup    bool
	popupContent string
	popupTitle   string
	// Theme picker state
	showThemePicker bool
	selectedTheme   int
	// Simulated data
	simulatedPeers   []internal.PeerInfo
	simulatedResults []*internal.ExecutionResults
	lastUpdate       time.Time
}

// Available themes for the picker
var availableThemes = []ThemeType{
	ThemeDark,
	ThemeLight,
	ThemeHighContrast,
	ThemeOcean,
	ThemeForest,
	ThemeSunset,
	ThemeCyberpunk,
	ThemeRetro,
	ThemeMonokai,
	ThemeNord,
	ThemeDracula,
	ThemeSolarized,
	ThemeGruvbox,
	ThemeTokyo,
	ThemeCandy,
}

func newModel(logger *logging.Logger, theme Theme, useEmoji bool) model {
	m := model{
		logger: logger,
		theme:  theme,
		icons:  defaultIcons(useEmoji),
		tab:    tabOverview,
		width:  80,
		height: 24,
	}
	m.initSimulatedData()
	return m
}

func newModelWithInitialView(logger *logging.Logger, theme Theme, useEmoji bool, view string) model {
	m := newModel(logger, theme, useEmoji)
	switch view {
	case "overview":
		m.tab = tabOverview
	case "peers":
		m.tab = tabPeers
	case "results":
		m.tab = tabResults
	case "commands":
		m.tab = tabCommands
	default:
		m.tab = tabOverview
	}
	return m
}

func (m *model) initSimulatedData() {
	// Simulate peer discovery
	m.simulatedPeers = []internal.PeerInfo{
		{ID: "alpha-01", Name: "alpha-node", Address: "192.168.1.10", Role: "worker", OS: "Linux", Arch: "x86_64", Connected: true, LastSeen: time.Now().Add(-2 * time.Second)},
		{ID: "beta-02", Name: "beta-node", Address: "192.168.1.11", Role: "worker", OS: "Linux", Arch: "x86_64", Connected: true, LastSeen: time.Now().Add(-5 * time.Second)},
		{ID: "gamma-03", Name: "gamma-node", Address: "192.168.1.12", Role: "controller", OS: "Windows", Arch: "x86_64", Connected: false, LastSeen: time.Now().Add(-1 * time.Minute)},
		{ID: "delta-04", Name: "delta-node", Address: "192.168.1.13", Role: "worker", OS: "macOS", Arch: "arm64", Connected: true, LastSeen: time.Now().Add(-10 * time.Second)},
		{ID: "epsilon-05", Name: "epsilon-node", Address: "192.168.1.14", Role: "worker", OS: "Linux", Arch: "x86_64", Connected: true, LastSeen: time.Now().Add(-15 * time.Second)},
	}

	// Simulate command results
	m.simulatedResults = []*internal.ExecutionResults{
		{
			CommandID: "cmd-001",
			Command:   "ls -la",
			Target:    "all",
			Results: []internal.ExecutionResult{
				{Device: "alpha-node", Status: "ok", ExitCode: 0, Duration: 150, Stdout: "total 24\ndrwxr-xr-x 2 user user 4096 Jan 15 10:00 ."},
				{Device: "beta-node", Status: "ok", ExitCode: 0, Duration: 180, Stdout: "total 24\ndrwxr-xr-x 2 user user 4096 Jan 15 10:00 ."},
				{Device: "delta-node", Status: "ok", ExitCode: 0, Duration: 200, Stdout: "total 24\ndrwxr-xr-x 2 user user 4096 Jan 15 10:00 ."},
			},
			Timestamp: time.Now().Add(-2 * time.Second),
		},
		{
			CommandID: "cmd-002",
			Command:   "df -h",
			Target:    "all",
			Results: []internal.ExecutionResult{
				{Device: "alpha-node", Status: "ok", ExitCode: 0, Duration: 120, Stdout: "Filesystem Size Used Avail Use% Mounted on\n/dev/sda1 100G 45G 50G 47% /"},
				{Device: "beta-node", Status: "failed", ExitCode: 1, Duration: 150, Stderr: "df: cannot access '/proc/mounts': No such file or directory"},
				{Device: "delta-node", Status: "ok", ExitCode: 0, Duration: 180, Stdout: "Filesystem Size Used Avail Use% Mounted on\n/dev/disk1 500G 200G 300G 40% /"},
			},
			Timestamp: time.Now().Add(-1 * time.Minute),
		},
	}

	m.lastUpdate = time.Now()
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.KeyMsg:
		if m.showThemePicker {
			return m.handleThemePickerKeys(v)
		}
		return m.handleMainKeys(v)
	case tea.WindowSizeMsg:
		m.width = v.Width
		m.height = v.Height
	case peersUpdateMsg:
		if v.Peers != nil {
			if peers, ok := v.Peers.([]internal.PeerInfo); ok {
				items := make([]interface{}, 0, len(peers))
				for _, p := range peers {
					items = append(items, p)
				}
				m.peerList.SetItems(items)
			}
		}
	case resultsUpdateMsg:
		if v.Results != nil {
			if r, ok := v.Results.(*internal.ExecutionResults); ok {
				m.results = r
			}
		}
	}
	return m, nil
}

func (m *model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "ctrl+t":
		m.showThemePicker = true
		m.selectedTheme = 0
	case "h", "left":
		if m.tab > 0 {
			m.tab--
		}
	case "l", "right":
		if m.tab < tabCommands {
			m.tab++
		}
	case "?":
		m.showHelp = !m.showHelp
	case "enter":
		if m.tab == tabPeers {
			m.showPopup = true
			m.popupTitle = "Peer Information"
			m.popupContent = "Select a peer to view details"
		}
	case "esc":
		m.showPopup = false
		m.showHelp = false
	case "1":
		m.tab = tabOverview
	case "2":
		m.tab = tabPeers
	case "3":
		m.tab = tabResults
	case "4":
		m.tab = tabCommands
	}
	return m, nil
}

func (m *model) handleThemePickerKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.showThemePicker = false
	case "up", "k":
		if m.selectedTheme > 0 {
			m.selectedTheme--
		}
	case "down", "j":
		if m.selectedTheme < len(availableThemes)-1 {
			m.selectedTheme++
		}
	case "enter":
		// Apply the selected theme
		selectedThemeType := availableThemes[m.selectedTheme]
		m.theme = getTheme(selectedThemeType)
		m.showThemePicker = false
	}
	return m, nil
}

func (m *model) View() string {
	if m.showThemePicker {
		return m.renderThemePicker()
	}

	if m.showHelp {
		return m.renderHelp()
	}

	if m.showPopup {
		return m.renderWithPopup()
	}

	return m.renderMain()
}

func (m *model) renderThemePicker() string {
	// Create a preview of the current theme
	previewTheme := getTheme(availableThemes[m.selectedTheme])

	// Header
	header := previewTheme.BannerText.Render("Theme Picker - Press Enter to apply, Esc to cancel")

	// Theme list
	var themeList []string
	for i, themeType := range availableThemes {
		theme := getTheme(themeType)
		themeName := string(themeType)

		// Style the theme name based on selection
		var style lipgloss.Style
		if i == m.selectedTheme {
			style = lipgloss.NewStyle().Foreground(theme.Primary).Bold(true)
		} else {
			style = lipgloss.NewStyle().Foreground(theme.Secondary)
		}

		// Add selection indicator
		indicator := "  "
		if i == m.selectedTheme {
			indicator = "▶ "
		}

		themeList = append(themeList, style.Render(indicator+themeName))
	}

	// Create theme preview
	preview := fmt.Sprintf(`
 Theme Preview: %-45s
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  Primary:   %s                                                         │
│  Secondary: %s                                                         │
│  Accent:    %s                                                         │
│  Success:   %s                                                         │
│  Warning:   %s                                                         │
│  Danger:    %s                                                         │
│                                                                             │
│  Navigation: ↑/↓ to select, Enter to apply, Esc to cancel                   │
└─────────────────────────────────────────────────────────────────────────────┘`,
		availableThemes[m.selectedTheme],
		previewTheme.Primary,
		previewTheme.Secondary,
		previewTheme.Accent,
		previewTheme.Success,
		previewTheme.Warning,
		previewTheme.Danger)

	// Combine header, theme list, and preview
	content := strings.Join(themeList, "\n")

	// Center the content
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(previewTheme.Primary).
		Padding(1, 2).
		Width(m.width - 4).
		Height(m.height - 4)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
		box.Render(header+"\n\n"+content+"\n\n"+preview))
}

func (m *model) renderMain() string {
	// Header with banner
	header := m.renderHeader()

	// Tab navigation
	_ = m.renderTabs()

	// Main content area
	content := m.renderContent()

	// Footer/status bar
	footer := m.renderFooter()

	// Combine all sections
	layout := Layout{Theme: m.theme}
	return layout.RenderRoot(m.width, m.height, header, &BasicPane{name: "content", body: content}, footer)
}

func (m *model) renderHeader() string {
	banner := BannerASCII()
	return m.theme.BannerText.Render(banner)
}

func (m *model) renderTabs() string {
	tabNames := []string{"Overview", "Peers", "Results", "Commands"}
	tabStyle := lipgloss.NewStyle().Padding(0, 1).Margin(0, 1)
	activeStyle := tabStyle.Foreground(m.theme.Primary).Bold(true)
	inactiveStyle := tabStyle.Foreground(m.theme.Secondary)

	var tabs []string
	for i, name := range tabNames {
		if i == int(m.tab) {
			tabs = append(tabs, activeStyle.Render(fmt.Sprintf("[%s]", name)))
		} else {
			tabs = append(tabs, inactiveStyle.Render(fmt.Sprintf("[%s]", name)))
		}
	}

	status := fmt.Sprintf("Status: %s", m.getNetworkStatus())
	statusStyle := lipgloss.NewStyle().Foreground(m.theme.Success).Align(lipgloss.Right)

	tabRow := lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
	statusRow := statusStyle.Render(status)

	// Combine tabs and status on same line
	combined := lipgloss.JoinHorizontal(lipgloss.Left, tabRow, lipgloss.NewStyle().Width(m.width-lipgloss.Width(tabRow)-lipgloss.Width(statusRow)).Render(""), statusRow)

	return combined + "\n" + strings.Repeat("─", m.width)
}

func (m *model) renderContent() string {
	switch m.tab {
	case tabOverview:
		return m.renderOverview()
	case tabPeers:
		return m.renderPeers()
	case tabResults:
		return m.renderResults()
	case tabCommands:
		return m.renderCommands()
	default:
		return "Unknown tab"
	}
}

func (m *model) renderOverview() string {
	onlineCount := 0
	for _, peer := range m.simulatedPeers {
		if peer.Connected {
			onlineCount++
		}
	}

	overview := fmt.Sprintf(`
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│   Network   │  │   Status    │  │ Quick Cmd   │
│   Status    │  │   Summary   │  │   Input     │
│             │  │             │  │             │
│ Peers: %d    │  │ Online: %d   │  │ [cmd >]     │
│ Routes: 8   │  │ Offline: %d  │  │ [Run]       │
└─────────────┘  └─────────────┘  └─────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│ Recent Activity                                                     │
│ • Command 'ls -la' completed on %d devices (2s ago)                 │
│ • New peer 'alpha-node' discovered (5s ago)                         │
│ • Command 'df -h' failed on 'beta-node' (1m ago)                    │
└─────────────────────────────────────────────────────────────────────┘`,
		len(m.simulatedPeers), onlineCount, len(m.simulatedPeers)-onlineCount, len(m.simulatedResults[0].Results))

	return overview
}

func (m *model) renderPeers() string {
	onlineCount := 0
	for _, peer := range m.simulatedPeers {
		if peer.Connected {
			onlineCount++
		}
	}

	header := "Peers                                    [Filter: █] [Refresh] [Add]"
	tableHeader := "ID          │ Name        │ Status  │ OS      │ Last Seen │ Actions"
	separator := strings.Repeat("─", m.width-4)

	var rows []string
	for _, peer := range m.simulatedPeers {
		status := "Online"
		statusIcon := "●"
		if !peer.Connected {
			status = "Offline"
			statusIcon = "○"
		}

		lastSeen := time.Since(peer.LastSeen)
		lastSeenStr := "now"
		if lastSeen > time.Second {
			lastSeenStr = lastSeen.Round(time.Second).String() + " ago"
		}

		row := fmt.Sprintf("│ %-10s │ %-11s │ %-7s │ %-7s │ %-10s │ [Info]",
			peer.ID, peer.Name, statusIcon+" "+status, peer.OS, lastSeenStr)
		rows = append(rows, row)
	}

	summary := fmt.Sprintf("Total: %d peers | Online: %d | Offline: %d",
		len(m.simulatedPeers), onlineCount, len(m.simulatedPeers)-onlineCount)

	content := strings.Join(rows, "\n")

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s",
		header, separator, tableHeader, separator, content, separator, summary)
}

func (m *model) renderResults() string {
	if len(m.simulatedResults) == 0 {
		return "No command results available"
	}

	latest := m.simulatedResults[0]
	successCount := 0
	for _, result := range latest.Results {
		if result.Status == "ok" {
			successCount++
		}
	}

	header := "Results                                  [Filter: █] [Export] [Clear]"
	commandInfo := fmt.Sprintf("Command: '%s' | Target: '%s' | Status: Completed (%d/%d)",
		latest.Command, latest.Target, successCount, len(latest.Results))
	tableHeader := "Device     │ Status │ Exit │ Duration │ Output Preview"
	separator := strings.Repeat("─", m.width-4)

	var rows []string
	for _, result := range latest.Results {
		status := "OK"
		if result.Status != "ok" {
			status = "Err"
		}

		output := result.Stdout
		if output == "" {
			output = result.Stderr
		}
		if len(output) > maxOutputPreviewLength {
			output = output[:outputTruncateLength] + outputTruncateSuffix
		}

		row := fmt.Sprintf("│ %-*s │ %-*s │ %-*d │ %-*d │ %s",
			deviceColumnWidth, result.Device,
			statusColumnWidth, status,
			exitCodeColumnWidth, result.ExitCode,
			durationColumnWidth, result.Duration,
			output)
		rows = append(rows, row)
	}

	actions := "[View Full Output] [Download Results] [Rerun Command]"
	content := strings.Join(rows, "\n")

	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s\n%s",
		header, separator, commandInfo, separator, tableHeader, separator, content, separator, actions)
}

func (m *model) renderCommands() string {
	header := "Commands                               [New Command] [Templates] [History]"
	separator := strings.Repeat("─", m.width-4)

	commandInput := `
┌─────────────────────────────────────────────────────────────────────┐
│ Command Input                                                       │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ [Command: █]                                                    │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│ ┌─────────────────────────────────────────────────────────────────┐ │
│ │ [Target: █] [Timeout: 30s] [Work Dir: █] [Safe Mode: ☑]       │ │
│ └─────────────────────────────────────────────────────────────────┘ │
│                                                                     │
│ [Dry Run] [Execute] [Schedule] [Save Template]                     │
└─────────────────────────────────────────────────────────────────────┘`

	recentCommands := `
┌─────────────────────────────────────────────────────────────────────┐
│ Recent Commands                                                     │
│ • ls -la (2s ago) - 8 devices, 8 successful                       │
│ • df -h (1m ago) - 8 devices, 7 successful, 1 failed              │
│ • whoami (5m ago) - 8 devices, 8 successful                       │
└─────────────────────────────────────────────────────────────────────┘`

	return fmt.Sprintf("%s\n%s%s%s", header, separator, commandInput, recentCommands)
}

func (m *model) renderFooter() string {
	onlineCount := 0
	for _, peer := range m.simulatedPeers {
		if peer.Connected {
			onlineCount++
		}
	}

	lastUpdate := time.Since(m.lastUpdate).Round(time.Second)
	footer := fmt.Sprintf("Peers: %d | Commands: %d | Last Update: %s ago | Press ? for help | Ctrl+T for themes",
		onlineCount, len(m.simulatedResults), lastUpdate)

	return m.theme.StatusBar.Render(footer)
}

func (m *model) renderHelp() string {
	help := `
┌─────────────────────────────────────────────────────────────────────────────┐
│ MeshExec TUI Help                                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│ Navigation:                                                                 │
│   h/←  - Previous tab                                                       │
│   l/→  - Next tab                                                           │
│   1-4  - Jump to specific tab (1=Overview, 2=Peers, 3=Results, 4=Commands)  │
│                                                                             │
│ Actions:                                                                    │
│   Enter - Select/Open (context dependent)                                   │
│   Esc   - Close/Cancel popup                                                │
│   ?     - Toggle this help                                                  │
│   Ctrl+T - Open theme picker                                                │
│                                                                             │
│ Global:                                                                     │
│   q/Ctrl+C - Quit                                                           │
│                                                                             │
│ Press any key to return to main view...                                     │
└─────────────────────────────────────────────────────────────────────────────┘`

	return help
}

func (m *model) renderWithPopup() string {
	mainContent := m.renderMain()

	popup := Popup{
		Title: m.popupTitle,
		Body:  m.popupContent,
	}

	popupContent := popup.Render(m.width, m.height, m.theme)

	// Overlay popup on main content
	return lipgloss.JoinVertical(lipgloss.Left, mainContent, popupContent)
}

func (m *model) getNetworkStatus() string {
	onlineCount := 0
	for _, peer := range m.simulatedPeers {
		if peer.Connected {
			onlineCount++
		}
	}

	if onlineCount == len(m.simulatedPeers) {
		return "Online"
	} else if onlineCount > 0 {
		return "Partial"
	} else {
		return "Offline"
	}
}

// renderResultsFiltered applies simple filtering by substring to ExecutionResults
func (m *model) renderResultsFiltered() string {
	if m.results == nil {
		return ""
	}
	needle := strings.ToLower(m.resultFilter.Value())
	var b strings.Builder
	for _, r := range m.results.Results {
		row := r.Device + " " + r.Status + " " + r.Stdout + " " + r.Stderr
		if needle == "" || strings.Contains(strings.ToLower(row), needle) {
			b.WriteString(r.Device)
			b.WriteString("\n")
		}
	}
	return b.String()
}
