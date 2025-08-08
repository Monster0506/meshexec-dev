package executor

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

func newExec() *DefaultCommandExecutor {
	return NewDefaultCommandExecutor(internal.DefaultConfig(), logging.NewLogger("none"))
}

func TestExecute_Success(t *testing.T) {
	exec := newExec()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := "echo hello"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c echo hello" // shell will be cmd.exe /C "cmd /c echo hello"; still works
	}
	res, err := exec.Execute(ctx, cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", res.ExitCode)
	}
	if res.Stdout == "" {
		t.Fatalf("expected stdout not empty")
	}
}

func TestExecute_NonZeroExit(t *testing.T) {
	exec := newExec()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := "sh -c 'exit 3'"
	if runtime.GOOS == "windows" {
		cmd = "cmd /c exit 5"
	}
	res, err := exec.Execute(ctx, cmd)
	if err != nil { // non-zero exit should not be error, it's captured in ExitCode
		t.Fatalf("unexpected exec error: %v", err)
	}
	if runtime.GOOS == "windows" {
		if res.ExitCode == 0 {
			t.Fatalf("expected non-zero exit code on windows, got 0")
		}
	} else if res.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code on unix, got 0")
	}
}

func TestExecute_Timeout(t *testing.T) {
	exec := newExec()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cmd := "sh -c 'sleep 2'"
	if runtime.GOOS == "windows" {
		cmd = "ping 127.0.0.1 -n 3 >NUL" // ~2 seconds
	}
	_, err := exec.Execute(ctx, cmd)
	if err == nil {
		t.Fatalf("expected timeout/cancel error, got nil")
	}
}
