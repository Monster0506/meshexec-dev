package tui

import (
    "context"
    "sync"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/monster0506/mechexec/internal"
    "github.com/monster0506/mechexec/internal/logging"
)

// Manager is a Bubble Tea based implementation of a terminal UI for MechExec
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
    initialView string
}

type Option func(*options)

func defaultOptions() options { return options{initialView: ""} }

// WithInitialView sets the initial view key (e.g., "overview", "peers", "results")
func WithInitialView(view string) Option { return func(o *options) { o.initialView = view } }

// StartTUI launches the Bubble Tea program and blocks until it exits
func (m *Manager) StartTUI(ctx context.Context, opts ...Option) error {
    cfg := defaultOptions()
    for _, opt := range opts {
        opt(&cfg)
    }

    model := newModel(m.logger)
    // Note: initialView option will be used when model supports it
    m.mu.Lock()
    m.program = tea.NewProgram(model, tea.WithContext(ctx))
    m.mu.Unlock()

    // Watch for context cancellation and request quit
    go func() {
        <-ctx.Done()
        m.mu.Lock()
        if m.program != nil {
            m.program.Quit()
        }
        m.mu.Unlock()
    }()

    if m.logger != nil {
        m.logger.Info("Starting TUI", nil)
    }
    _, err := m.program.Run()
    return err
}

// UpdateResults sends new execution results into the TUI
func (m *Manager) UpdateResults(results *internal.ExecutionResults) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.program != nil && results != nil {
        m.program.Send(resultsUpdateMsg{Results: results})
    }
}

// UpdatePeers sends an updated peer list into the TUI
func (m *Manager) UpdatePeers(peers []internal.PeerInfo) {
    m.mu.Lock()
    defer m.mu.Unlock()
    if m.program != nil {
        m.program.Send(peersUpdateMsg{Peers: peers})
    }
}

