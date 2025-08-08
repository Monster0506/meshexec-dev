package executor

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// DefaultCommandExecutor executes commands via the system shell
// Windows: cmd.exe /C
// Unix-like: /bin/sh -c
// Safety validation is exposed via ValidateCommand; callers decide when to enforce.
type DefaultCommandExecutor struct {
	cfg    *internal.Config
	logger *logging.Logger
}

func NewDefaultCommandExecutor(cfg *internal.Config, logger *logging.Logger) *DefaultCommandExecutor {
	return &DefaultCommandExecutor{cfg: cfg, logger: logger}
}

func (e *DefaultCommandExecutor) shellAndArgs(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/C", command}
	}
	return "/bin/sh", []string{"-c", command}
}

// Execute runs the provided command string through the platform shell using the given context.
// It captures stdout/stderr, returns exit code and duration. If the context times out or is
// canceled, the underlying process is killed and the context error is returned.
func (e *DefaultCommandExecutor) Execute(ctx context.Context, command string) (*internal.ExecutionResult, error) {
	shell, args := e.shellAndArgs(command)

	start := time.Now()
	cmd := exec.CommandContext(ctx, shell, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	duration := time.Since(start)

	result := &internal.ExecutionResult{
		Stdout:   stdoutBuf.String(),
		Stderr:   stderrBuf.String(),
		ExitCode: 0,
		Duration: int64(duration / time.Millisecond),
	}

	if err != nil {
		// If the context was canceled or deadline exceeded, return the context error
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		// Process exited with non-zero exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		// Other exec errors (e.g., shell not found)
		return result, err
	}

	// Success path
	return result, nil
}

// ValidateCommand delegates to SafetyChecker created from current config.
func (e *DefaultCommandExecutor) ValidateCommand(cmd string) error {
	checker := NewSafetyChecker(e.cfg, e.logger)
	return checker.ValidateCommand(cmd)
}
