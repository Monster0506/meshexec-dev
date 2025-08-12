package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/agent"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/monster0506/meshexec/internal/executor"
	"github.com/monster0506/meshexec/internal/mesh"
	"github.com/monster0506/meshexec/internal/targeting"
	"github.com/spf13/cobra"
)

var daemonForeground bool

// Allow tests to stub mesh/agent builders
var newMeshNodeForDaemon = func(cfg *core.Config) (core.MeshNode, error) { return mesh.NewNodeFromConfig(cfg) }

// waitForSignal can be overridden in tests to avoid blocking on OS signals
var waitForSignal = func() os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	return <-sigCh
}

// runDaemon encapsulates the main lifecycle so it can be tested if needed
func runDaemon(cmd *cobra.Command) error {
	// Load configuration
	logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	if verbose {
		logLevel = "debug"
	}
	manager := config.NewManagerWithLevel(logLevel)
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath != "" {
		manager.SetConfigPath(cfgPath)
	}
	cfg, err := manager.Load()
	if err != nil {
		me := core.NewConfigError("invalid_config", "failed to load configuration", map[string]interface{}{"error": err.Error()})
		fmt.Fprintln(os.Stderr, core.FormatUserError(me))
		return err
	}

	// On Windows, default to sidecar unless user overrides
	if runtime.GOOS == "windows" && os.Getenv("MESHEXEC_BLE_IMPL") == "" {
		_ = os.Setenv("MESHEXEC_BLE_IMPL", "sidecar")
	}

	// Build mesh node
	node, err := newMeshNodeForDaemon(cfg)
	if err != nil {
		return fmt.Errorf("mesh init: %w", err)
	}

	// Build agent with default executor and evaluator
	exec := executor.NewDefaultCommandExecutor(cfg, logger)
	tgt := targeting.NewEvaluatorWithLevel(logLevel)
	dev := core.DeviceInfo{
		Name: cfg.Device.Name,
		Role: cfg.Device.Role,
		OS:   cfg.Device.OS,
		Arch: cfg.Device.Arch,
		Tags: cfg.Device.Tags,
	}
	ag := agent.New(node, nil, exec, tgt, dev, logger)

	// Start services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := node.Start(ctx); err != nil {
		return fmt.Errorf("mesh start: %w", err)
	}
	if err := ag.Start(ctx); err != nil {
		_ = node.Stop()
		return fmt.Errorf("agent start: %w", err)
	}

	if logger != nil {
		logger.Info("daemon started", map[string]interface{}{"device": cfg.Device.Name})
	}

	// Wait for termination signal (test seam)
	_ = waitForSignal()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = ag.Stop()
	_ = node.Stop()
	select {
	case <-shutdownCtx.Done():
	default:
	}
	if logger != nil {
		logger.Info("daemon stopped", nil)
	}
	return nil
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the MeshExec agent daemon",
	Long:  "Starts the local agent that listens for mesh commands and executes them securely.",
	RunE:  func(cmd *cobra.Command, args []string) error { return runDaemon(cmd) },
}

func init() {
	daemonCmd.Flags().BoolVar(&daemonForeground, "foreground", true, "Run in foreground")
	rootCmd.AddCommand(daemonCmd)
}
