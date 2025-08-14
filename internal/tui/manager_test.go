package tui

import (
    "bytes"
    "context"
    "io"
    "testing"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/monster0506/meshexec/internal"
    "github.com/monster0506/meshexec/internal/logging"
)

func TestManager_ConstructorsAndOptions_Basics(t *testing.T) {
	d := defaultOptions()
	if d.initialView != "" {
		t.Fatalf("unexpected default initialView: %q", d.initialView)
	}

	opt := WithInitialView("results")
	opt(&d)
	if d.initialView != "results" {
		t.Fatalf("WithInitialView did not apply")
	}

	m := NewManager(logging.NewLogger("none"))
	if m == nil {
		t.Fatalf("NewManager returned nil")
	}

	// UpdateResults and UpdatePeers are safe no-ops before StartTUI
	m.UpdateResults(&internal.ExecutionResults{CommandID: "c1"})
	m.UpdatePeers([]internal.PeerInfo{{Name: "p1"}})
}

func TestManager_StartTUI_ImmediateCancel_Basics(t *testing.T) {
	m := NewManager(logging.NewLogger("none"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// Should return quickly; tolerate transient bubbletea error values
    _ = m.StartTUI(
        ctx,
        WithInitialView("peers"),
        // Avoid blocking on console input and disable output in CI
        WithProgramOptions(tea.WithInput(bytes.NewReader(nil)), tea.WithOutput(io.Discard)),
    )

	// Now send updates after program has exited; should be safe no-ops
	m.UpdateResults(&internal.ExecutionResults{CommandID: "c2"})
	m.UpdatePeers([]internal.PeerInfo{{Name: "p2"}})

	time.Sleep(10 * time.Millisecond)
}
