package executor

import (
	"runtime"
	"strings"
	"testing"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

func newTestConfig() *internal.Config {
	cfg := internal.DefaultConfig()
	cfg.Safety.MaxCommandLength = 64
	cfg.Safety.SafeMode = true
	return cfg
}

func getChecker(cfg *internal.Config) *SafetyChecker {
	return NewSafetyChecker(cfg, logging.NewLogger("none"))
}

func TestSafety_AllowsBenign(t *testing.T) {
	cfg := newTestConfig()
	chk := getChecker(cfg)
	for _, cmd := range []string{
		"echo hello",
		"whoami",
		"dir",
		"ls -la",
	} {
		if err := chk.ValidateCommand(cmd); err != nil {
			t.Fatalf("expected benign command to pass: %q, err=%v", cmd, err)
		}
	}
}

func TestSafety_BlocksDangerous_Defaults(t *testing.T) {
	cfg := newTestConfig()
	chk := getChecker(cfg)
	if runtime.GOOS == "windows" {
		for _, cmd := range []string{"del /s C:\\*", "format C:"} {
			if err := chk.ValidateCommand(cmd); err == nil {
				t.Fatalf("expected dangerous command to be blocked: %q", cmd)
			}
		}
	} else { // unix-like
		for _, cmd := range []string{"rm -rf /", "dd if=/dev/zero of=/dev/sda"} {
			if err := chk.ValidateCommand(cmd); err == nil {
				t.Fatalf("expected dangerous command to be blocked: %q", cmd)
			}
		}
	}
}

func TestSafety_NormalizesSpacing(t *testing.T) {
	cfg := newTestConfig()
	chk := getChecker(cfg)
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "del    /s    C:\\temp\\*"
	} else {
		cmd = "rm   -rf    /tmp"
	}
	if err := chk.ValidateCommand(cmd); err == nil {
		t.Fatalf("expected normalized dangerous command to be blocked: %q", cmd)
	}
}

func TestSafety_MaxLength(t *testing.T) {
	cfg := newTestConfig()
	cfg.Safety.MaxCommandLength = 10
	chk := getChecker(cfg)
	// longer than 10
	long := strings.Repeat("a", 11)
	if err := chk.ValidateCommand(long); err == nil {
		t.Fatalf("expected long command to be blocked by length")
	}
}

func TestSafety_SafeModeOff_Allows(t *testing.T) {
	cfg := newTestConfig()
	cfg.Safety.SafeMode = false
	chk := getChecker(cfg)
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "del /s C:\\*"
	} else {
		cmd = "rm -rf /"
	}
	if err := chk.ValidateCommand(cmd); err != nil {
		t.Fatalf("expected dangerous command allowed when safe mode off, got %v", err)
	}
}

func TestSafety_CustomPattern(t *testing.T) {
	cfg := newTestConfig()
	cfg.Safety.DangerousCommands = append(cfg.Safety.DangerousCommands, "shutdown")
	chk := getChecker(cfg)
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "shutdown /r /t 0"
	} else {
		cmd = "shutdown -h now"
	}
	if err := chk.ValidateCommand(cmd); err == nil {
		t.Fatalf("expected custom pattern to be blocked: %q", cmd)
	}
}

func TestSafety_NoFalsePositive_InnocentWords(t *testing.T) {
	cfg := newTestConfig()
	chk := getChecker(cfg)
	// should not match substrings inside larger words
	for _, cmd := range []string{
		"echo arm -rfile", // contains 'rm -rf' letters but not command
		"echo formatte",   // contains 'format' letters
	} {
		if err := chk.ValidateCommand(cmd); err != nil {
			t.Fatalf("unexpected block for innocuous: %q err=%v", cmd, err)
		}
	}
}

func TestSafety_ShellWrapper_SubcommandBlocked(t *testing.T) {
	cfg := newTestConfig()
	chk := getChecker(cfg)
	if runtime.GOOS == "windows" {
		if err := chk.ValidateCommand(`powershell -Command "Remove-Item -Recurse -Force C:\\*"`); err == nil {
			t.Fatalf("expected powershell dangerous command to be blocked")
		}
	} else {
		if err := chk.ValidateCommand(`sh -c "rm -rf /"`); err == nil {
			t.Fatalf("expected sh -c dangerous command to be blocked")
		}
	}
}
