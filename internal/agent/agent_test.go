package agent

import (
	"context"
	"testing"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

type mockMesh struct {
	sent []*core.MeshMessage
	subs map[core.MessageType]chan *core.MeshMessage
}

func (m *mockMesh) Start(ctx context.Context) error         { return nil }
func (m *mockMesh) Stop() error                             { return nil }
func (m *mockMesh) SendMessage(msg *core.MeshMessage) error { m.sent = append(m.sent, msg); return nil }
func (m *mockMesh) Subscribe(t core.MessageType) <-chan *core.MeshMessage {
	if m.subs == nil {
		m.subs = make(map[core.MessageType]chan *core.MeshMessage)
	}
	ch := make(chan *core.MeshMessage, 16)
	m.subs[t] = ch
	return ch
}
func (m *mockMesh) GetPeers() []core.PeerInfo { return nil }

type allowAllTarget struct{}

func (a allowAllTarget) Evaluate(expression string, device *core.DeviceInfo) (bool, error) {
	return true, nil
}
func (a allowAllTarget) Parse(expression string) (*core.TargetAST, error) {
	return &core.TargetAST{Type: "literal", Value: expression}, nil
}

type denyTarget struct{}

func (d denyTarget) Evaluate(expression string, device *core.DeviceInfo) (bool, error) {
	return false, nil
}
func (d denyTarget) Parse(expression string) (*core.TargetAST, error) {
	return &core.TargetAST{Type: "literal", Value: expression}, nil
}

type mockExecutor struct{ err error }

func (e mockExecutor) Execute(ctx context.Context, cmd string) (*core.ExecutionResult, error) {
	if e.err != nil {
		return &core.ExecutionResult{ExitCode: 1, Stderr: e.err.Error()}, e.err
	}
	return &core.ExecutionResult{ExitCode: 0, Stdout: "ok"}, nil
}
func (e mockExecutor) ValidateCommand(cmd string) error { return nil }

func TestAgent_ProcessCommand_Success(t *testing.T) {
	mesh := &mockMesh{}
	device := core.DeviceInfo{Name: "dev1"}
	ag := New(mesh, nil, mockExecutor{}, allowAllTarget{}, device, logging.NewLogger("none"))

	cmd := &core.MeshMessage{ID: "1", TTL: 5, Sender: "cli", Type: core.MessageTypeCommand, Command: "echo", Timestamp: time.Now().Unix()}
	if err := ag.ProcessCommand(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mesh.sent) != 1 || mesh.sent[0].Type != core.MessageTypeResult {
		t.Fatalf("expected one result message, got %d", len(mesh.sent))
	}
}

func TestAgent_ProcessCommand_TargetMismatch(t *testing.T) {
	mesh := &mockMesh{}
	device := core.DeviceInfo{Name: "dev1"}
	ag := New(mesh, nil, mockExecutor{}, denyTarget{}, device, logging.NewLogger("none"))
	cmd := &core.MeshMessage{ID: "1", TTL: 5, Sender: "cli", Type: core.MessageTypeCommand, Target: []string{"role=worker"}, Command: "echo", Timestamp: time.Now().Unix()}
	if err := ag.ProcessCommand(cmd); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mesh.sent) != 0 {
		t.Fatalf("expected no result publish on mismatch")
	}
}

func TestAgent_ProcessCommand_InvalidSignature_NoSecurityOK(t *testing.T) {
	mesh := &mockMesh{}
	device := core.DeviceInfo{Name: "dev1"}
	ag := New(mesh, nil, mockExecutor{}, allowAllTarget{}, device, logging.NewLogger("none"))
	cmd := &core.MeshMessage{ID: "1", TTL: 5, Sender: "cli", Type: core.MessageTypeCommand, Command: "echo", Timestamp: time.Now().Unix()}
	if err := ag.ProcessCommand(cmd); err != nil {
		t.Fatalf("unexpected error without security: %v", err)
	}
}

func TestAgent_StartStop_Idempotent(t *testing.T) {
	mesh := &mockMesh{}
	device := core.DeviceInfo{Name: "dev1"}
	ag := New(mesh, nil, mockExecutor{}, allowAllTarget{}, device, logging.NewLogger("none"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("start error: %v", err)
	}
	if err := ag.Start(ctx); err != nil {
		t.Fatalf("start idempotent error: %v", err)
	}
	if err := ag.Stop(); err != nil {
		t.Fatalf("stop error: %v", err)
	}
	if err := ag.Stop(); err != nil {
		t.Fatalf("stop idempotent error: %v", err)
	}
}
