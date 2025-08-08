package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func execArgs(t *testing.T, args ...string) *cobra.Command {
	t.Helper()
	rootCmd.SetArgs(args)
	cmd, err := rootCmd.ExecuteC()
	if err != nil {
		t.Fatalf("execute failed: %v (args=%v)", err, args)
	}
	return cmd
}

func TestRootFlags_ParseDefaultsAndVerbose(t *testing.T) {
	// Default log level should be none
	execArgs(t, "help")
	lvl, _ := rootCmd.PersistentFlags().GetString("log-level")
	if lvl != "none" {
		t.Fatalf("expected default log-level=none, got %s", lvl)
	}
	// Verbose toggles flag parse
	execArgs(t, "-v", "help")
	vb, _ := rootCmd.PersistentFlags().GetBool("verbose")
	if !vb {
		t.Fatalf("expected verbose flag to be true")
	}
}

func TestRunFlags_ParsingOnly(t *testing.T) {
	// Use --dry-run to avoid exiting in Run
	execArgs(t,
		"run",
		"-t", "os=linux",
		"--dry-run",
		"--workdir", "/tmp",
		"--timeout", "123",
		"--safe-mode",
		"--format", "json",
		"--encrypt",
		"--no-sign",
		"--", "echo", "hello",
	)
	if runTarget != "os=linux" || !runDryRun || runWorkDir != "/tmp" || runTimeout != 123 {
		t.Fatalf("unexpected run flags parsed: target=%q dry=%v wd=%q timeout=%d", runTarget, runDryRun, runWorkDir, runTimeout)
	}
	if !runSafeMode || !runEncrypt || !runNoSign || runFormat != "json" {
		t.Fatalf("unexpected safety/output flags: safe=%v enc=%v nosign=%v fmt=%q", runSafeMode, runEncrypt, runNoSign, runFormat)
	}
}

func TestTUIFlags_ViewParsing(t *testing.T) {
	old := tuiCmd.RunE
	tuiCmd.RunE = func(cmd *cobra.Command, args []string) error { return nil }
	defer func() { tuiCmd.RunE = old }()

	execArgs(t, "tui", "--view", "results")
	if tuiView != "results" {
		t.Fatalf("expected tui --view=results, got %q", tuiView)
	}
}
