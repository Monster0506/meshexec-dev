package tui

import (
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/charmbracelet/bubbles/help"
    "github.com/charmbracelet/bubbles/key"
    list "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/progress"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/lipgloss"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/monster0506/mechexec/internal"
    "github.com/monster0506/mechexec/internal/logging"
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

func (k keyMap) ShortHelp() []key.Binding { return []key.Binding{k.PrevTab, k.NextTab, k.Quit} }
func (k keyMap) FullHelp() [][]key.Binding { return [][]key.Binding{{k.PrevTab, k.NextTab, k.Quit}} }

// list item for peers
type peerItem internal.PeerInfo

func (p peerItem) Title() string       { return fmt.Sprintf("%s (%s)", p.Name, p.Role) }
func (p peerItem) Description() string { return fmt.Sprintf("%s • %s/%s • RSSI %d", p.Address, p.OS, p.Arch, p.SignalStrength) }
func (p peerItem) FilterValue() string { return strings.ToLower(p.Name + " " + p.Role + " " + p.OS + " " + p.Arch) }

type model struct {
    logger *logging.Logger

    width  int
    height int

    tab    viewTab
    keys   keyMap
    help   help.Model
    theme  theme

    // Peers
    peerList list.Model

    // Results
    results     *internal.ExecutionResults
    progressBar progress.Model
    resultFilter textinput.Model

    // Commands (placeholder)
    input textinput.Model
}

func newModel(logger *logging.Logger) model {
    items := []list.Item{}
    // Custom delegate with professional colors
    del := list.NewDefaultDelegate()
    del.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(defaultTheme().AccentA).Bold(true)
    del.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(defaultTheme().Text)
    del.Styles.NormalTitle = lipgloss.NewStyle().Foreground(defaultTheme().Text)
    del.Styles.NormalDesc = lipgloss.NewStyle().Foreground(defaultTheme().Muted)
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

    pb := progress.New(progress.WithGradient("#58A6FF", "#1F6FEB"))

    ti := textinput.New()
    ti.Placeholder = "Type a command (e.g., echo hello)"
    ti.CharLimit = 256
    ti.Prompt = "> "
    ti.Focus()

    rf := textinput.New()
    rf.Placeholder = "Filter results (device, status, output)"
    rf.CharLimit = 80
    rf.Prompt = "Filter: "

    return model{
        logger:      logger,
        tab:         tabPeers,
        keys:        km,
        help:        help.New(),
        theme:       defaultTheme(),
        peerList:    lst,
        progressBar: pb,
        input:       ti,
        resultFilter: rf,
    }
}

// newModelWithInitialView constructs a model and selects initial tab based on view string.
// Supported values: "overview" (peers), "peers", "results", "commands".
func newModelWithInitialView(logger *logging.Logger, view string) model {
    m := newModel(logger)
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

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        maxWidth := 120
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
        m.peerList.SetSize(containerW-6, containerH-12)
        return m, nil
    case tea.KeyMsg:
        switch {
        case key.Matches(msg, m.keys.Quit):
            return m, tea.Quit
        case key.Matches(msg, m.keys.NextTab):
            m.tab = (m.tab + 1) % 3
            // manage focus
            if m.tab == tabCommands {
                m.input.Focus()
                m.resultFilter.Blur()
            } else if m.tab == tabResults {
                m.resultFilter.Focus()
                m.input.Blur()
            } else {
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
            if m.tab == tabCommands {
                m.input.Focus()
                m.resultFilter.Blur()
            } else if m.tab == tabResults {
                m.resultFilter.Focus()
                m.input.Blur()
            } else {
                m.input.Blur()
                m.resultFilter.Blur()
            }
            return m, nil
        default:
            // direct numeric tab selection
            switch msg.String() {
            case "1":
                m.tab = tabPeers
                m.input.Blur(); m.resultFilter.Blur()
                return m, nil
            case "2":
                m.tab = tabResults
                m.resultFilter.Focus(); m.input.Blur()
                return m, nil
            case "3":
                m.tab = tabCommands
                m.input.Focus(); m.resultFilter.Blur()
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
                    Summary: internal.ResultSummary{TotalDevices: 1, Successful: 1, Failed: 0, Timeout: 0, AverageDuration: 500},
                    Timestamp: time.Now(),
                }
                // Update results and switch to results tab
                m.results = res
                m.tab = tabResults
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
        return m, nil
    case resultsUpdateMsg:
        m.results = msg.Results
        return m, nil
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

var (
    titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0FF")).Bold(true)
    tabActive  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E1FF")).Bold(true)
    tabNormal  = lipgloss.NewStyle().Foreground(lipgloss.Color("#666"))
    boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).BorderForeground(lipgloss.Color("#4D4DFF"))
)

func (m model) View() string {
    // Construct styles
    bg := lipgloss.NewStyle().Background(m.theme.Background).Foreground(m.theme.Text)
    container := lipgloss.NewStyle().
        Width(min(120, max(40, m.width-4))).
        Height(max(20, m.height-4)).
        Border(lipgloss.NormalBorder()).
        BorderForeground(m.theme.AccentB).
        Padding(1, 3).
        Background(m.theme.Surface)

    headerStyle := lipgloss.NewStyle().Foreground(m.theme.Text).Bold(true)
    subtitleStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
    tabActiveStyle := lipgloss.NewStyle().Foreground(m.theme.AccentA).Bold(true)
    tabNormalStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)

    header := headerStyle.Render("MechExec Dashboard") + "\n" + subtitleStyle.Render("Decentralized command execution over BLE mesh")
    tabs := []string{
        choose(m.tab == tabPeers, tabActiveStyle.Render("Peers"), tabNormalStyle.Render("Peers")),
        choose(m.tab == tabResults, tabActiveStyle.Render("Results"), tabNormalStyle.Render("Results")),
        choose(m.tab == tabCommands, tabActiveStyle.Render("Commands"), tabNormalStyle.Render("Commands")),
    }
    tabBar := strings.Join(tabs, "  ")

    body := ""
    switch m.tab {
    case tabPeers:
        body = m.renderPeers()
    case tabResults:
        body = m.renderResults()
    case tabCommands:
        body = m.renderCommands()
    }

    spacer := "\n"
    panel := container.Render(lipgloss.JoinVertical(lipgloss.Top, header, spacer, tabBar, spacer, body, spacer, m.help.View(m.keys)))

    // Center the panel
    centered := lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel, lipgloss.WithWhitespaceForeground(m.theme.Background))
    return bg.Render(centered)
}

func (m model) renderResults() string {
    if m.results == nil {
        return "No results yet. Execute a command to see aggregated output."
    }

    // Summary
    sum := m.results.Summary
    pct := 0.0
    if sum.TotalDevices > 0 {
        pct = float64(sum.Successful+sum.Failed+sum.Timeout) / float64(sum.TotalDevices)
    }
    bar := m.progressBar.ViewAs(pct)

    var b strings.Builder
    b.WriteString(titleStyle.Render(fmt.Sprintf("Command: %s", m.results.Command)))
    b.WriteString("\n")
    b.WriteString(bar)
    b.WriteString("\n")
    b.WriteString(fmt.Sprintf("Total: %d  OK: %d  Failed: %d  Timeout: %d  Avg: %dms\n\n",
        sum.TotalDevices, sum.Successful, sum.Failed, sum.Timeout, sum.AverageDuration))

    // Filter + sort results
    results := filterResults(append([]internal.ExecutionResult(nil), m.results.Results...), strings.ToLower(strings.TrimSpace(m.resultFilter.Value())))
    sort.Slice(results, func(i, j int) bool { return results[i].Device < results[j].Device })
    for _, r := range results {
        status := r.Status
        if status == "" && r.ExitCode == 0 {
            status = "ok"
        }
        duration := time.Duration(r.Duration) * time.Millisecond
        line := fmt.Sprintf("%-18s  %-6s  code=%-2d  %7s  stdout=%s", r.Device, status, r.ExitCode, duration, truncate(r.Stdout, 60))
        b.WriteString(line)
        b.WriteString("\n")
        if r.Stderr != "" {
            b.WriteString("  err: ")
            b.WriteString(truncate(r.Stderr, 80))
            b.WriteString("\n")
        }
    }
    return lipgloss.JoinVertical(lipgloss.Top, m.resultFilter.View(), b.String())
}

func (m model) renderCommands() string {
    hint := lipgloss.NewStyle().Foreground(m.theme.Muted).Render("Press Enter to simulate execution (demo)")
    return lipgloss.JoinVertical(lipgloss.Top, m.input.View(), hint)
}

func (m model) renderPeers() string {
    return m.peerList.View()
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

