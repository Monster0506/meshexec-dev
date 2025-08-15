package tui

import (
	"context"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// Manager is a Bubble Tea based implementation of a terminal UI for MeshExec
type Manager struct {
	program *tea.Program
	logger  *logging.Logger

	mu sync.Mutex
}

// NewManager constructs a new UI Manager
func NewManager(logger *logging.Logger) *Manager {
	return &Manager{logger: logger}
}

// Options for starting the TUI
type options struct {
	initialView    string
	themeName      string
	useEmoji       bool
	programOptions []tea.ProgramOption
}

type Option func(*options)

func defaultOptions() options {
	return options{initialView: "", themeName: "dark", useEmoji: true, programOptions: []tea.ProgramOption{tea.WithAltScreen()}}
}

// WithInitialView sets the initial view key (e.g., "overview", "peers", "results")
func WithInitialView(view string) Option { return func(o *options) { o.initialView = view } }

// WithTheme selects a theme variant: "dark", "light", "hc", "ocean", "forest", "sunset", "cyberpunk", "retro", "monokai", "nord", "dracula", "solarized", "gruvbox", "tokyo", "candy"
func WithTheme(name string) Option { return func(o *options) { o.themeName = name } }

// WithEmoji toggles emoji/micro-icons usage
func WithEmoji(enabled bool) Option { return func(o *options) { o.useEmoji = enabled } }

// WithProgramOptions forwards options to bubbletea's NewProgram (useful for tests like WithoutInput/WithoutRenderer)
func WithProgramOptions(opts ...tea.ProgramOption) Option {
	return func(o *options) { o.programOptions = append(o.programOptions, opts...) }
}

// StartTUI launches the Bubble Tea program and blocks until it exits
func (m *Manager) StartTUI(ctx context.Context, opts ...Option) error {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(&cfg)
	}

	// Validate initial view early to provide clear feedback
	if cfg.initialView != "" {
		switch cfg.initialView {
		case "overview", "peers", "results", "commands":
			// ok
		default:
			if m.logger != nil {
				m.logger.Warn("Unknown initial view; falling back to overview", map[string]interface{}{"view": cfg.initialView})
			}
			cfg.initialView = "overview"
		}
	}

	theme := getTheme(ThemeType(cfg.themeName))
	var modelInstance model
	if cfg.initialView != "" {
		modelInstance = newModelWithInitialView(m.logger, theme, cfg.useEmoji, cfg.initialView)
	} else {
		modelInstance = newModel(m.logger, theme, cfg.useEmoji)
	}
	m.mu.Lock()
	// Always include context; allow callers to provide additional tea.ProgramOption values
	progOpts := append([]tea.ProgramOption{tea.WithContext(ctx)}, cfg.programOptions...)
	m.program = tea.NewProgram(&modelInstance, progOpts...)
	m.mu.Unlock()

	// Watch for context cancellation and request quit
	go func() {
		<-ctx.Done()
		if m.logger != nil {
			m.logger.Debug("TUI context cancelled; quitting program", nil)
		}
		m.mu.Lock()
		if m.program != nil {
			m.program.Quit()
		}
		m.mu.Unlock()
	}()

	if m.logger != nil {
		m.logger.Info("Starting TUI", map[string]interface{}{"initial_view": cfg.initialView})
	}
	_, err := m.program.Run()
	if err != nil {
		if m.logger != nil {
			m.logger.Error("TUI exited with error", err, nil)
		}
		return err
	}
	if m.logger != nil {
		m.logger.Info("TUI exited cleanly", nil)
	}
	return nil
}

// UpdateResults sends new execution results into the TUI
func (m *Manager) UpdateResults(results *internal.ExecutionResults) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.program != nil && results != nil {
		if m.logger != nil {
			m.logger.Debug("Updating TUI results", map[string]interface{}{
				"command_id":    results.CommandID,
				"results_count": len(results.Results),
			})
		}
		m.program.Send(resultsUpdateMsg{Results: results})
	}
}

// UpdatePeers sends an updated peer list into the TUI
func (m *Manager) UpdatePeers(peers []internal.PeerInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.program != nil {
		if m.logger != nil {
			m.logger.Debug("Updating TUI peers", map[string]interface{}{"count": len(peers)})
		}
		m.program.Send(peersUpdateMsg{Peers: peers})
	}
}
