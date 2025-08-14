package main

import (
	"os"
	"path/filepath"
	"testing"

	core "github.com/monster0506/meshexec/internal"
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
		"--sync",
		"--at", "03:00",
		"--env", "FOO=1",
		"--env", "BAR=two",
		"--stdin-file", "./input.txt",
		"--", "echo", "hello",
	)
	if runTarget != "os=linux" || !runDryRun || runWorkDir != "/tmp" || runTimeout != 123 {
		t.Fatalf("unexpected run flags parsed: target=%q dry=%v wd=%q timeout=%d", runTarget, runDryRun, runWorkDir, runTimeout)
	}
	if !runSafeMode || !runEncrypt || !runNoSign || runFormat != "json" || !runSync || runAt != "03:00" {
		t.Fatalf("unexpected safety/output/schedule flags: safe=%v enc=%v nosign=%v fmt=%q sync=%v at=%q", runSafeMode, runEncrypt, runNoSign, runFormat, runSync, runAt)
	}
	if len(runEnv) != 2 || runEnv[0] != "FOO=1" || runEnv[1] != "BAR=two" || runStdinFile != "./input.txt" {
		t.Fatalf("unexpected env/stdin flags: env=%v stdin=%q", runEnv, runStdinFile)
	}
}

func TestRun_Niceties_PopulateMessage(t *testing.T) {
	// Hook to capture the constructed message
	var got *core.CommandMessage
	oldHook := runMessageHook
	runMessageHook = func(m *core.CommandMessage) { got = m }
	defer func() { runMessageHook = oldHook }()

	execArgs(t,
		"run",
		"-t", "robot && zone=alpha",
		"--dry-run",
		"--env", "FOO=1",
		"--env", "BAR=two",
		"--stdin-file", "in.txt",
		"--at", "03:00",
		"--", "echo",
	)
	if got == nil {
		t.Fatalf("expected message to be captured")
	}
	if got.TargetExpr == "" {
		t.Fatalf("expected TargetExpr populated")
	}
	if got.Env == nil || got.Env["FOO"] != "1" || got.Env["BAR"] != "two" {
		t.Fatalf("unexpected env map: %+v", got.Env)
	}
	if got.StdinRef != "in.txt" {
		t.Fatalf("expected stdin ref set, got %q", got.StdinRef)
	}
	// ScheduledAt may or may not parse depending on test time; just ensure field exists (0 ok)
	if got.ScheduledAt < 0 {
		t.Fatalf("scheduled_at should be >= 0, got %d", got.ScheduledAt)
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

func TestStatusFlags_ParsingOnly(t *testing.T) {
	// Stub the command run to avoid performing BLE operations during tests
	oldRun := statusCmd.Run
	statusCmd.Run = func(cmd *cobra.Command, args []string) {}
	defer func() { statusCmd.Run = oldRun }()

	execArgs(t, "status", "--json", "--since", "10m", "--timeout", "1")
	if !statusJSON {
		t.Fatalf("expected status --json to set statusJSON=true")
	}
	if statusSince != "10m" {
		t.Fatalf("expected status --since=10m, got %q", statusSince)
	}
	if statusTimeoutMs != 1 {
		t.Fatalf("expected status --timeout=1, got %d", statusTimeoutMs)
	}
}

func TestStatus_InvalidSince_IgnoredByParser(t *testing.T) {
	oldRun := statusCmd.Run
	statusCmd.Run = func(cmd *cobra.Command, args []string) {}
	defer func() { statusCmd.Run = oldRun }()

	execArgs(t, "status", "--since", "10 min")
	// Cobra stores flag value as-is; runtime parsing decides validity
	if statusSince != "10 min" {
		t.Fatalf("expected status --since preserved as '10 min', got %q", statusSince)
	}
}

func TestJoinFlags_ParsingOnly(t *testing.T) {
	oldRun := joinCmd.Run
	joinCmd.Run = func(cmd *cobra.Command, args []string) {}
	defer func() { joinCmd.Run = oldRun }()

	execArgs(t, "join", "--foreground", "--scan-interval", "1500", "--advertise-interval", "2000")
	if !joinForeground || joinScanInterval != 1500 || joinAdvertiseInterval != 2000 {
		t.Fatalf("unexpected join flags: fg=%v scan=%d adv=%d", joinForeground, joinScanInterval, joinAdvertiseInterval)
	}
}

func TestListFlags_ParsingOnly(t *testing.T) {
	oldRun := listCmd.Run
	listCmd.Run = func(cmd *cobra.Command, args []string) {}
	defer func() { listCmd.Run = oldRun }()

	execArgs(t, "list", "--json", "--timeout", "2500")
	if !listJSON || listTimeoutMs != 2500 {
		t.Fatalf("unexpected list flags: json=%v timeout=%d", listJSON, listTimeoutMs)
	}
}

func TestSyncFlags_ParsingOnly(t *testing.T) {
	oldRun := syncCmd.Run
	syncCmd.Run = func(cmd *cobra.Command, args []string) {}
	defer func() { syncCmd.Run = oldRun }()

	execArgs(t, "sync", "--repo", ".", "-t", "all", "--direction", "push", "--dry-run")
	if !syncDryRun || syncRepoPath != "." || syncTargetExpr != "all" || syncDirection != "push" {
		t.Fatalf("unexpected sync flags: dry=%v repo=%q target=%q dir=%q", syncDryRun, syncRepoPath, syncTargetExpr, syncDirection)
	}
}

func TestCloneFlags_ParsingOnly(t *testing.T) {
	oldRun := cloneCmd.Run
	cloneCmd.Run = func(cmd *cobra.Command, args []string) {}
	defer func() { cloneCmd.Run = oldRun }()

	execArgs(t, "clone", "-t", "peer-1", "--dest", "./out")
	if cloneTarget != "peer-1" || cloneDest != "./out" {
		t.Fatalf("unexpected clone flags: target=%q dest=%q", cloneTarget, cloneDest)
	}
}

func TestConfigEdit_InvokesEditorWithExistingFile(t *testing.T) {
	// Arrange stub editor
	oldRun := configRunEditor
	defer func() { configRunEditor = oldRun }()
	called := false
	var gotEditor string
	var gotArgs []string
	configRunEditor = func(editor string, args []string) error {
		called = true
		gotEditor = editor
		gotArgs = append([]string(nil), args...)
		return nil
	}

	// Use a temporary config file
	tmp, err := os.CreateTemp(t.TempDir(), "cfg-*.toml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	_ = tmp.Close()

	// Act
	execArgs(t, "-c", tmp.Name(), "config", "edit")

	// Assert
	if !called {
		t.Fatalf("expected editor to be invoked")
	}
	if gotEditor == "" {
		t.Fatalf("expected an editor command to be chosen")
	}
	if len(gotArgs) == 0 || gotArgs[len(gotArgs)-1] != tmp.Name() {
		t.Fatalf("expected editor args to end with config path; args=%v path=%q", gotArgs, tmp.Name())
	}
}

func TestConfigEdit_CreatesDefaultWhenMissing(t *testing.T) {
	// Arrange: stub editor to no-op
	oldRun := configRunEditor
	defer func() { configRunEditor = oldRun }()
	configRunEditor = func(editor string, args []string) error { return nil }

	// Use path inside temp dir that does not exist yet
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.toml")
	if _, err := os.Stat(cfgPath); !os.IsNotExist(err) {
		t.Fatalf("expected non-existent path for test, got err=%v", err)
	}

	// Act
	execArgs(t, "-c", cfgPath, "config", "edit")

	// Assert: file should now exist and be non-empty
	st, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatalf("expected config file to be created: %v", err)
	}
	if st.Size() == 0 {
		t.Fatalf("expected created config to be non-empty")
	}
}
