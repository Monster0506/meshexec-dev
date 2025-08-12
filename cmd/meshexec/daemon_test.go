package main

import (
	"context"
	"errors"
	"os"
	"testing"

	core "github.com/monster0506/meshexec/internal"
)

// mockMeshNode implements core.MeshNode for daemon tests
type mockMeshNode struct {
	started bool
}

func (m *mockMeshNode) Start(ctx context.Context) error         { m.started = true; return nil }
func (m *mockMeshNode) Stop() error                             { m.started = false; return nil }
func (m *mockMeshNode) SendMessage(msg *core.MeshMessage) error { return nil }
func (m *mockMeshNode) Subscribe(t core.MessageType) <-chan *core.MeshMessage {
	ch := make(chan *core.MeshMessage)
	close(ch)
	return ch
}
func (m *mockMeshNode) GetPeers() []core.PeerInfo { return nil }

func TestDaemon_StartsAndStops(t *testing.T) {
	// Stub builders and signal wait
	oldBuilder := newMeshNodeForDaemon
	oldWait := waitForSignal
	defer func() { newMeshNodeForDaemon = oldBuilder; waitForSignal = oldWait }()

	n := &mockMeshNode{}
	newMeshNodeForDaemon = func(cfg *core.Config) (core.MeshNode, error) { return n, nil }
	waitForSignal = func() os.Signal { var s os.Signal; return s }

	// Execute; runDaemon should return after waitForSignal
	if err := runDaemon(rootCmd); err != nil {
		t.Fatalf("daemon run returned error: %v", err)
	}
	if n.started {
		t.Fatalf("expected mesh to be stopped after shutdown")
	}
}

func TestDaemon_InitErrors(t *testing.T) {
	oldBuilder := newMeshNodeForDaemon
	oldWait := waitForSignal
	defer func() { newMeshNodeForDaemon = oldBuilder; waitForSignal = oldWait }()

	newMeshNodeForDaemon = func(cfg *core.Config) (core.MeshNode, error) { return nil, errors.New("boom") }
	waitForSignal = func() os.Signal { var s os.Signal; return s }

	if err := runDaemon(rootCmd); err == nil {
		t.Fatalf("expected error from mesh init failure")
	}
}
