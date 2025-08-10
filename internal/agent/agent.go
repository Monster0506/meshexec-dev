package agent

import (
	"context"
	"sync"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
	"github.com/monster0506/meshexec/internal/messages"
)

// Agent implements core.Agent
type Agent struct {
	mesh       core.MeshNode
	security   SignVerify
	exec       core.CommandExecutor
	targetEval core.TargetEvaluator
	device     core.DeviceInfo
	logger     *logging.Logger

	cancel context.CancelFunc
	mu     sync.Mutex
	run    bool
}

type SignVerify interface {
	SignMeshMessage(msg *core.MeshMessage) error
	VerifyMeshMessage(msg *core.MeshMessage) error
}

func New(mesh core.MeshNode, security SignVerify, exec core.CommandExecutor, target core.TargetEvaluator, device core.DeviceInfo, logger *logging.Logger) *Agent {
	if logger == nil {
		logger = logging.NewLogger("info")
	}
	return &Agent{mesh: mesh, security: security, exec: exec, targetEval: target, device: device, logger: logger}
}

func (a *Agent) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.run {
		a.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.run = true
	a.mu.Unlock()

	cmdCh := a.mesh.Subscribe(core.MessageTypeCommand)
	go func() {
		for {
			select {
			case <-runCtx.Done():
				return
			case msg, ok := <-cmdCh:
				if !ok {
					return
				}
				_ = a.ProcessCommand(msg)
			}
		}
	}()
	return nil
}

func (a *Agent) Stop() error {
	a.mu.Lock()
	if !a.run {
		a.mu.Unlock()
		return nil
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.run = false
	a.mu.Unlock()
	return nil
}

func (a *Agent) ProcessCommand(msg *core.MeshMessage) error {
	if msg == nil {
		return core.NewExecutionError("nil_message", "received nil command message", nil)
	}

	// verify signature if available
	if a.security != nil {
		if err := a.security.VerifyMeshMessage(msg); err != nil {
			if a.logger != nil {
				a.logger.Warn("Rejected command due to invalid signature", map[string]interface{}{"id": msg.ID, "error": err.Error()})
			}
			return core.NewSecurityError("signature_invalid", "invalid message signature", map[string]interface{}{"id": msg.ID})
		}
	}

	// target matching
	if a.targetEval != nil {
		// Treat empty target as broadcast
		matched := true
		if len(msg.Target) > 0 {
			// Combine simple target list as OR of tokens (device names or tags)
			// For now, evaluate against device name equality or delegate to evaluator on first token
			expr := msg.Target[0]
			ok, err := a.targetEval.Evaluate(expr, &a.device)
			if err != nil {
				return core.NewTargetingError("evaluation_failed", "failed to evaluate target expression", map[string]interface{}{"expr": expr, "error": err.Error()})
			}
			matched = ok
		}
		if !matched {
			if a.logger != nil {
				a.logger.Debug("Command not applicable to this device", map[string]interface{}{"id": msg.ID})
			}
			return nil
		}
	}

	// Safety validation and execution
	var execRes *core.ExecutionResult
	var execErr error
	if a.exec != nil {
		cmdLine := buildCommandLine(msg.Command, msg.Payload)
		if err := a.exec.ValidateCommand(cmdLine); err != nil {
			return err
		}
		timeout := 30 * time.Second
		ectx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		execRes, execErr = a.exec.Execute(ectx, cmdLine)
	}
	if execRes == nil {
		execRes = &core.ExecutionResult{Status: "failed", ExitCode: -1, Stderr: "no executor"}
	}
	if execErr != nil {
		execRes.Status = "failed"
		if execRes.Stderr == "" {
			execRes.Stderr = execErr.Error()
		}
	} else {
		if execRes.Status == "" {
			if execRes.ExitCode == 0 {
				execRes.Status = "success"
			} else {
				execRes.Status = "failed"
			}
		}
	}
	execRes.Device = a.device.Name

	// Create and sign result
	mh := messages.NewMessageHandlerWithLevel("none")
	result := mh.CreateResultMessage(msg.ID, *execRes, a.device.Name)
	if a.security != nil {
		_ = a.security.SignMeshMessage(&result.MeshMessage)
	}
	// Publish result
	return a.mesh.SendMessage(&result.MeshMessage)
}

func (a *Agent) ExecuteCommand(cmd string) (*core.ExecutionResult, error) {
	if a.exec == nil {
		return &core.ExecutionResult{Status: "failed", ExitCode: -1, Stderr: "no executor"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return a.exec.Execute(ctx, cmd)
}

func (a *Agent) ValidateCommand(msg *core.MeshMessage) error {
	if a.exec == nil {
		return nil
	}
	cmdLine := buildCommandLine(msg.Command, msg.Payload)
	return a.exec.ValidateCommand(cmdLine)
}

func buildCommandLine(command string, payload []byte) string {
	if command == "" {
		return ""
	}
	return command
}
