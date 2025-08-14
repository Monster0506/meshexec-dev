package tui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	list "github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// messages pushed into the UI
type resultsUpdateMsg struct{ Results *internal.ExecutionResults }
type peersUpdateMsg struct{ Peers []internal.PeerInfo }

type viewTab int

const (
	tabPeers viewTab = iota
	tabResults
	tabCommands
)

// key map for help
type keyMap struct {
	NextTab key.Binding
	PrevTab key.Binding
	Quit    key.Binding
}

func (k keyMap) ShortHelp() []key.Binding  { return []key.Binding{k.PrevTab, k.NextTab, k.Quit} }
func (k keyMap) FullHelp() [][]key.Binding { return [][]key.Binding{{k.PrevTab, k.NextTab, k.Quit}} }

// list item for peers
type peerItem internal.PeerInfo

func (p peerItem) Title() string { return fmt.Sprintf("%s (%s)", p.Name, p.Role) }
func (p peerItem) Description() string {
	return fmt.Sprintf("%s • %s/%s • RSSI %d", p.Address, p.OS, p.Arch, p.SignalStrength)
}
func (p peerItem) FilterValue() string {
	return strings.ToLower(p.Name + " " + p.Role + " " + p.OS + " " + p.Arch)
}

type model struct {
	logger *logging.Logger

	width  int
	height int

	tab      viewTab
	keys     keyMap
	help     help.Model
	theme    theme
	useEmoji bool

	// cached styles
	styles uiStyles

	// Peers
	peerList list.Model
	// peers detail needs width awareness

	// Results
	results      *internal.ExecutionResults
	progressBar  progress.Model
	resultFilter textinput.Model
	resultsTable table.Model
	sortBy       string // device|status|duration

	// Commands (placeholder)
	input      textinput.Model
	suggList   list.Model
	cmdHistory []string

	// Toasts
	lastToast   string
	lastToastAt time.Time
}

type uiStyles struct {
	bg                lipgloss.Style
	container         lipgloss.Style
	header            lipgloss.Style
	subtitle          lipgloss.Style
	tabActive         lipgloss.Style
	tabNormal         lipgloss.Style
	divider           lipgloss.Style
	footer            lipgloss.Style
	badge             lipgloss.Style
	chipOk            lipgloss.Style
	chipFail          lipgloss.Style
	chipWarn          lipgloss.Style
	selection         lipgloss.Style
	listItemTitle     lipgloss.Style
	listItemDesc      lipgloss.Style
	listSelectedTitle lipgloss.Style
	listSelectedDesc  lipgloss.Style
	detailCard        lipgloss.Style
	skeleton          lipgloss.Style
}

func buildStyles(th theme) uiStyles {
	return uiStyles{
		bg:                lipgloss.NewStyle().Background(th.Background).Foreground(th.Text),
		container:         lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(th.Border).Padding(1, 3).Background(th.Surface),
		header:            lipgloss.NewStyle().Foreground(th.Text).Bold(true),
		subtitle:          lipgloss.NewStyle().Foreground(th.Muted),
		tabActive:         lipgloss.NewStyle().Foreground(th.Primary).Bold(true).Underline(true),
		tabNormal:         lipgloss.NewStyle().Foreground(th.Muted),
		divider:           lipgloss.NewStyle().Foreground(th.Border),
		footer:            lipgloss.NewStyle().Foreground(th.Muted),
		badge:             lipgloss.NewStyle().Foreground(th.ChipText).Background(th.ChipBg).Padding(0, 1).Bold(true),
		chipOk:            lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.Good).Padding(0, 1),
		chipFail:          lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.Bad).Padding(0, 1),
		chipWarn:          lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.Warn).Padding(0, 1),
		selection:         lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.SelectionBg),
		listItemTitle:     lipgloss.NewStyle().Foreground(th.Text),
		listItemDesc:      lipgloss.NewStyle().Foreground(th.Muted),
		listSelectedTitle: lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.SelectionBg).Bold(true),
		listSelectedDesc:  lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.SelectionBg),
		detailCard:        lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(th.Border).Padding(1).Background(th.Surface),
		skeleton:          lipgloss.NewStyle().Foreground(th.Muted).Background(th.InfoBg),
	}
}

func newModel(logger *logging.Logger, th theme, useEmoji bool) model {
	items := []list.Item{}
	// Custom delegate with professional colors
	del := list.NewDefaultDelegate()
	del.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.SelectionBg).Bold(true)
	del.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.SelectionBg)
	del.Styles.NormalTitle = lipgloss.NewStyle().Foreground(th.Text)
	del.Styles.NormalDesc = lipgloss.NewStyle().Foreground(th.Muted)
	lst := list.New(items, del, 0, 0)
	lst.Title = "Peers"
	lst.SetShowHelp(false)
	lst.SetShowFilter(true)
	lst.SetStatusBarItemName("peer", "peers")

	km := keyMap{
		NextTab: key.NewBinding(key.WithKeys("right", "]", "2", "3"), key.WithHelp("→ 2/3", "next tab")),
		PrevTab: key.NewBinding(key.WithKeys("left", "[", "1", "2"), key.WithHelp("← 1/2", "prev tab")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}

	pb := progress.New(progress.WithGradient(string(th.Primary), string(th.Secondary)))

	ti := textinput.New()
	ti.Placeholder = "Type a command (e.g., echo hello)"
	ti.CharLimit = 256
	ti.Prompt = "> "
	ti.Focus()

	rf := textinput.New()
	rf.Placeholder = "Filter results (device, status, output)"
	rf.CharLimit = 80
	rf.Prompt = "Filter: "

	// Suggestions list for commands
	suggItems := []list.Item{
		list.Item(peerItem{Address: "", Name: "uptime", Role: "cmd"}),
		list.Item(peerItem{Address: "", Name: "df -h", Role: "cmd"}),
		list.Item(peerItem{Address: "", Name: "whoami", Role: "cmd"}),
		list.Item(peerItem{Address: "", Name: "hostname", Role: "cmd"}),
		list.Item(peerItem{Address: "", Name: "date", Role: "cmd"}),
		list.Item(peerItem{Address: "", Name: "echo hello", Role: "cmd"}),
	}
	suggDel := list.NewDefaultDelegate()
	suggDel.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.SelectionBg).Bold(true)
	suggDel.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(th.SelectionText).Background(th.SelectionBg)
	suggDel.Styles.NormalTitle = lipgloss.NewStyle().Foreground(th.Text)
	suggDel.Styles.NormalDesc = lipgloss.NewStyle().Foreground(th.Muted)
	sl := list.New(suggItems, suggDel, 0, 0)
	sl.Title = "Suggestions"
	sl.SetShowHelp(false)
	sl.SetShowFilter(false)
	sl.SetStatusBarItemName("suggestion", "suggestions")

	// Results table
	columns := []table.Column{
		{Title: "Device", Width: 18},
		{Title: "Status", Width: 8},
		{Title: "Code", Width: 4},
		{Title: "Duration", Width: 9},
		{Title: "Stdout", Width: 60},
	}
	resultsTable := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}), table.WithFocused(true))
	resultsTable.SetStyles(tableDefaultStyles(th))

	return model{
		logger:       logger,
		tab:          tabPeers,
		keys:         km,
		help:         help.New(),
		theme:        th,
		useEmoji:     useEmoji,
		styles:       buildStyles(th),
		peerList:     lst,
		progressBar:  pb,
		input:        ti,
		resultFilter: rf,
		resultsTable: resultsTable,
		sortBy:       "device",
		suggList:     sl,
	}
}

// newModelWithInitialView constructs a model and selects initial tab based on view string.
// Supported values: "overview" (peers), "peers", "results", "commands".
func newModelWithInitialView(logger *logging.Logger, th theme, useEmoji bool, view string) model {
	m := newModel(logger, th, useEmoji)
	switch strings.ToLower(strings.TrimSpace(view)) {
	case "peers", "overview", "":
		m.tab = tabPeers
		m.input.Blur()
		m.resultFilter.Blur()
	case "results":
		m.tab = tabResults
		m.resultFilter.Focus()
		m.input.Blur()
	case "commands":
		m.tab = tabCommands
		m.input.Focus()
		m.resultFilter.Blur()
	default:
		// Fallback to peers
		m.tab = tabPeers
		m.input.Blur()
		m.resultFilter.Blur()
	}
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return t })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		maxWidth := 140
		containerW := m.width - 6
		if containerW > maxWidth {
			containerW = maxWidth
		}
		containerH := m.height - 6
		if containerW < 40 {
			containerW = 40
		}
		if containerH < 20 {
			containerH = 20
		}
		// calculate columns for peers view
		leftCol := int(math.Round(float64(containerW-6) * 0.45))
		if leftCol < 30 {
			leftCol = containerW - 10
		}
		m.peerList.SetSize(leftCol, containerH-14)
		m.suggList.SetSize(containerW-8, 8)
		m.resultsTable.SetHeight(containerH - 16)
		// width of stdout column adjusts to containerW
		cols := m.resultsTable.Columns()
		if len(cols) == 5 {
			used := cols[0].Width + cols[1].Width + cols[2].Width + cols[3].Width
			cols[4].Width = max(20, containerW-10-used)
			m.resultsTable.SetColumns(cols)
		}
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.NextTab):
			m.tab = (m.tab + 1) % 3
			// manage focus
			switch m.tab {
			case tabCommands:
				m.input.Focus()
				m.resultFilter.Blur()
			case tabResults:
				m.resultFilter.Focus()
				m.input.Blur()
			default:
				m.input.Blur()
				m.resultFilter.Blur()
			}
			return m, nil
		case key.Matches(msg, m.keys.PrevTab):
			if m.tab == 0 {
				m.tab = 2
			} else {
				m.tab--
			}
			// manage focus
			switch m.tab {
			case tabCommands:
				m.input.Focus()
				m.resultFilter.Blur()
			case tabResults:
				m.resultFilter.Focus()
				m.input.Blur()
			default:
				m.input.Blur()
				m.resultFilter.Blur()
			}
			return m, nil
		default:
			// direct numeric tab selection
			switch msg.String() {
			case "1":
				m.tab = tabPeers
				m.input.Blur()
				m.resultFilter.Blur()
				return m, nil
			case "2":
				m.tab = tabResults
				m.resultFilter.Focus()
				m.input.Blur()
				return m, nil
			case "3":
				m.tab = tabCommands
				m.input.Focus()
				m.resultFilter.Blur()
				return m, nil
			}
		}
		// Additional key handling per tab
		if m.tab == tabCommands && (msg.Type == tea.KeyEnter) {
			// Simulate execution producing results
			cmd := strings.TrimSpace(m.input.Value())
			if cmd != "" {
				res := &internal.ExecutionResults{
					CommandID: "local-" + time.Now().Format("150405"),
					Command:   cmd,
					Target:    "demo",
					Results: []internal.ExecutionResult{
						{ID: "x1", Device: "local", ExitCode: 0, Stdout: cmd + " - ok", Duration: 500, Status: "ok"},
					},
					Summary:   internal.ResultSummary{TotalDevices: 1, Successful: 1, Failed: 0, Timeout: 0, AverageDuration: 500},
					Timestamp: time.Now(),
				}
				// Update results and switch to results tab
				m.results = res
				m.cmdHistory = append([]string{cmd}, m.cmdHistory...)
				m.toast("Executed: " + cmd)
				m.tab = tabResults
			}
			return m, nil
		}
		if m.tab == tabResults {
			switch msg.String() {
			case "d":
				m.sortBy = "device"
			case "s":
				m.sortBy = "status"
			case "u":
				m.sortBy = "duration"
			}
			return m, nil
		}
	case peersUpdateMsg:
		// Sort peers for consistent display
		peers := append([]internal.PeerInfo(nil), msg.Peers...)
		sort.Slice(peers, func(i, j int) bool { return peers[i].Name < peers[j].Name })
		items := make([]list.Item, 0, len(peers))
		for _, p := range peers {
			items = append(items, peerItem(p))
		}
		m.peerList.SetItems(items)
		if len(items) > 0 && m.peerList.Index() == -1 {
			m.peerList.Select(0)
		}
		return m, nil
	case resultsUpdateMsg:
		m.results = msg.Results
		return m, nil
	case time.Time:
		// periodic tick to refresh animations/toasts
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg { return t })
	}

	// Delegate updates to components
	if m.tab == tabPeers {
		var cmd tea.Cmd
		m.peerList, cmd = m.peerList.Update(msg)
		return m, cmd
	}
	if m.tab == tabCommands {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	if m.tab == tabResults {
		var cmd tea.Cmd
		m.resultFilter, cmd = m.resultFilter.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	styles := m.styles
	container := styles.container.
		Width(min(140, max(40, m.width-4))).
		Height(max(22, m.height-4))

	// Header
	title := "MeshExec Dashboard"
	subtitle := "Decentralized command execution over BLE mesh"
	if m.useEmoji {
		title = "🕸️ " + title
	}
	header := styles.header.Render(title) + "\n" + styles.subtitle.Render(subtitle)

	// Tabs
	tabs := []string{
		choose(m.tab == tabPeers, styles.tabActive.Render("Peers"), styles.tabNormal.Render("Peers")),
		choose(m.tab == tabResults, styles.tabActive.Render("Results"), styles.tabNormal.Render("Results")),
		choose(m.tab == tabCommands, styles.tabActive.Render("Commands"), styles.tabNormal.Render("Commands")),
	}
	tabBar := strings.Join(tabs, "  ")

	// Body
	body := ""
	switch m.tab {
	case tabPeers:
		body = m.renderPeers()
	case tabResults:
		body = m.renderResults()
	case tabCommands:
		body = m.renderCommands()
	}

	// Footer
	footerLeft := styles.footer.Render(fmt.Sprintf("view:%s", m.currentViewName()))
	footerRight := m.help.View(m.keys)
	footer := lipgloss.JoinHorizontal(lipgloss.Top, footerLeft, "  ", footerRight)

	// Toast
	toast := m.renderToast()

	panel := container.Render(lipgloss.JoinVertical(lipgloss.Top, header, "", tabBar, "", body, "", footer, toast))
	centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel, lipgloss.WithWhitespaceForeground(m.theme.Background))
	return styles.bg.Render(centered)
}

func (m model) renderResults() string {
	if m.results == nil {
		return m.styles.subtitle.Render("No results yet. Execute a command to see aggregated output.")
	}

	// Summary
	sum := m.results.Summary
	pct := 0.0
	if sum.TotalDevices > 0 {
		pct = float64(sum.Successful+sum.Failed+sum.Timeout) / float64(sum.TotalDevices)
	}
	// progress bar rendered below

	// Filter + sort results
	results := filterResults(append([]internal.ExecutionResult(nil), m.results.Results...), strings.ToLower(strings.TrimSpace(m.resultFilter.Value())))
	switch m.sortBy {
	case "status":
		sort.Slice(results, func(i, j int) bool { return results[i].Status < results[j].Status })
	case "duration":
		sort.Slice(results, func(i, j int) bool { return results[i].Duration < results[j].Duration })
	default:
		sort.Slice(results, func(i, j int) bool { return results[i].Device < results[j].Device })
	}

	rows := make([]table.Row, 0, len(results))
	for _, r := range results {
		status := r.Status
		if status == "" && r.ExitCode == 0 {
			status = "ok"
		}
		chip := m.styles.chipOk.Render("OK")
		switch strings.ToLower(status) {
		case "ok", "success":
			chip = m.styles.chipOk.Render("OK")
		case "timeout":
			chip = m.styles.chipWarn.Render("TMO")
		default:
			if r.ExitCode != 0 {
				chip = m.styles.chipFail.Render("ERR")
			}
		}
		duration := time.Duration(r.Duration) * time.Millisecond
		rows = append(rows, table.Row{
			r.Device,
			chip,
			fmt.Sprintf("%d", r.ExitCode),
			fmt.Sprintf("%7s", duration),
			truncate(r.Stdout, 120),
		})
	}
	m.resultsTable.SetRows(rows)
	summary := fmt.Sprintf("Total: %d  OK: %d  Failed: %d  Timeout: %d  Avg: %dms",
		sum.TotalDevices, sum.Successful, sum.Failed, sum.Timeout, sum.AverageDuration)
	return lipgloss.JoinVertical(lipgloss.Top, m.resultFilter.View(), m.progressBar.ViewAs(pct), summary, "", m.resultsTable.View())
}

func (m model) renderCommands() string {
	hint := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Press Enter to simulate execution (demo)")
	return lipgloss.JoinVertical(lipgloss.Top, m.input.View(), m.suggList.View(), hint)
}

func (m model) renderPeers() string {
	w := min(140, max(40, m.width-4)) - 6
	left := int(math.Round(float64(w) * 0.45))
	if left < 30 {
		left = w - 10
	}
	right := w - left - 2
	// Detail panel
	detail := m.renderPeerDetail(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(left).Render(m.peerList.View()), lipgloss.NewStyle().Width(2).Render(""), lipgloss.NewStyle().Width(right).Render(detail))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func choose[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// filterResults returns results that contain the query in device, status, stdout or stderr fields
func filterResults(results []internal.ExecutionResult, query string) []internal.ExecutionResult {
	if query == "" {
		return results
	}
	out := make([]internal.ExecutionResult, 0, len(results))
	for _, r := range results {
		if strings.Contains(strings.ToLower(r.Device), query) ||
			strings.Contains(strings.ToLower(r.Status), query) ||
			strings.Contains(strings.ToLower(r.Stdout), query) ||
			strings.Contains(strings.ToLower(r.Stderr), query) {
			out = append(out, r)
		}
	}
	return out
}

func (m *model) renderPeerDetail(width int) string {
	if len(m.peerList.Items()) == 0 {
		// skeleton state
		return m.styles.skeleton.Width(width).Height(8).Render("Discovering peers…")
	}
	it := m.peerList.SelectedItem()
	if it == nil {
		return m.styles.subtitle.Render("Select a peer to view details")
	}
	p, ok := it.(peerItem)
	if !ok {
		return ""
	}
	name := m.styles.header.Render(p.Name) + " " + m.styles.badge.Render(p.Role)
	addr := fmt.Sprintf("%s • %s/%s", p.Address, p.OS, p.Arch)
	sig := signalBar(p.SignalStrength)
	lastSeen := fmt.Sprintf("Last seen: %s", relativeTime(p.LastSeen))
	content := lipgloss.JoinVertical(lipgloss.Top, name, addr, sig, lastSeen)
	return m.styles.detailCard.Width(width).Render(content)
}

func signalBar(rssi int) string {
	// Map RSSI rough ranges to 0-5 bars
	bars := 0
	switch {
	case rssi >= -50:
		bars = 5
	case rssi >= -60:
		bars = 4
	case rssi >= -70:
		bars = 3
	case rssi >= -80:
		bars = 2
	case rssi >= -90:
		bars = 1
	default:
		bars = 0
	}
	full := strings.Repeat("▮", bars)
	empty := strings.Repeat("▯", 5-bars)
	return fmt.Sprintf("Signal: %s%s (%ddBm)", full, empty, rssi)
}

func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func (m *model) toast(msg string) {
	m.lastToast = msg
	m.lastToastAt = time.Now()
}

func (m model) renderToast() string {
	if m.lastToast == "" {
		return ""
	}
	// expire after 5s visually
	if time.Since(m.lastToastAt) > 5*time.Second {
		return ""
	}
	box := lipgloss.NewStyle().Foreground(m.theme.SelectionText).Background(m.theme.SelectionBg).Padding(0, 1)
	return "\n" + box.Render(m.lastToast)
}

func tableDefaultStyles(th theme) table.Styles {
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(th.Border).
		BorderBottom(true).
		Bold(true).
		Foreground(th.Text)
	s.Selected = s.Selected.
		Foreground(th.SelectionText).
		Background(th.SelectionBg)
	s.Cell = s.Cell.Foreground(th.Text)
	return s
}

func (m model) currentViewName() string {
	switch m.tab {
	case tabPeers:
		return "peers"
	case tabResults:
		return "results"
	case tabCommands:
		return "commands"
	default:
		return ""
	}
}
