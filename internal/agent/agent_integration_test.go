package agent

import (
	"context"
	"testing"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
	"github.com/monster0506/meshexec/internal/mesh"
	"github.com/monster0506/meshexec/internal/messages"
	"github.com/monster0506/meshexec/internal/targeting"
)

// mockExecutorIntegration is a lightweight executor for integration tests
type mockExecutorIntegration struct{}

func (e mockExecutorIntegration) Execute(ctx context.Context, cmd string) (*core.ExecutionResult, error) {
	return &core.ExecutionResult{ExitCode: 0, Stdout: "ok", Status: "success"}, nil
}
func (e mockExecutorIntegration) ValidateCommand(cmd string) error { return nil }

func TestAgent_Integration_EndToEnd_CommandToResult(t *testing.T) {
	logger := logging.NewLogger("none")
	cfg := core.DefaultConfig()

	// Set a deterministic local device
	device := core.DeviceInfo{Name: "dev-int", OS: "windows", Arch: "amd64", Role: "worker"}

	// Mesh node without BLE transport (local subscriptions only)
	node := mesh.NewNode(nil, &cfg.Network, core.PeerInfo{ID: device.Name, Name: device.Name})

	// Agent wired to the node
	tgt := targeting.NewEvaluatorWithLevel("none")
	ag := New(node, nil, mockExecutorIntegration{}, tgt, device, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := node.Start(ctx); err != nil {
		t.Fatalf("node start error: %v", err)
	}
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("agent start error: %v", err)
	}

	// Subscribe for results
	results := node.Subscribe(core.MessageTypeResult)

	// Send a command into the mesh; agent should consume and publish a result
	mh := messages.NewMessageHandlerWithLevel("none")
	cmd := mh.CreateCommandMessage("echo", nil, []string{"all"}, "cli", "", 5)
	if err := node.SendMessage(&cmd.MeshMessage); err != nil {
		t.Fatalf("send command error: %v", err)
	}

	select {
	case res := <-results:
		if res == nil || res.Type != core.MessageTypeResult {
			t.Fatalf("unexpected result message: %+v", res)
		}
		if res.Sender != device.Name {
			t.Fatalf("expected result sender %q, got %q", device.Name, res.Sender)
		}
		if res.TTL <= 0 {
			t.Fatalf("expected positive TTL on result, got %d", res.TTL)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for result message")
	}

	if err := ag.Stop(); err != nil {
		t.Fatalf("agent stop error: %v", err)
	}
	if err := node.Stop(); err != nil {
		t.Fatalf("node stop error: %v", err)
	}
}

func TestAgent_Integration_TargetMismatch_NoResult(t *testing.T) {
	logger := logging.NewLogger("none")
	cfg := core.DefaultConfig()

	device := core.DeviceInfo{Name: "dev-int2", OS: "windows", Arch: "amd64", Role: "worker"}
	node := mesh.NewNode(nil, &cfg.Network, core.PeerInfo{ID: device.Name, Name: device.Name})

	tgt := targeting.NewEvaluatorWithLevel("none")
	ag := New(node, nil, mockExecutorIntegration{}, tgt, device, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := node.Start(ctx); err != nil {
		t.Fatalf("node start error: %v", err)
	}
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("agent start error: %v", err)
	}

	results := node.Subscribe(core.MessageTypeResult)

	mh := messages.NewMessageHandlerWithLevel("none")
	// Target that should not match the device
	cmd := mh.CreateCommandMessage("echo", nil, []string{"os=linux"}, "cli", "", 5)
	if err := node.SendMessage(&cmd.MeshMessage); err != nil {
		t.Fatalf("send command error: %v", err)
	}

	select {
	case <-results:
		t.Fatal("unexpected result published for non-matching target")
	case <-time.After(500 * time.Millisecond):
		// expected: no result
	}

	_ = ag.Stop()
	_ = node.Stop()
}
