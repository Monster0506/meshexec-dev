//go:build dev

package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/agent"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/monster0506/meshexec/internal/executor"
	"github.com/monster0506/meshexec/internal/mesh"
	"github.com/monster0506/meshexec/internal/messages"
	"github.com/monster0506/meshexec/internal/targeting"
	"github.com/spf13/cobra"
)

var (
	smokeCmdString string
	smokeTarget    string
	smokeTimeoutMs int
	smokeImplHint  string
)

var smokeCmd = &cobra.Command{
	Use:    "smoke",
	Short:  "Run a local end-to-end smoke test (requires WinBLE sidecar on Windows)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		// Configure logging
		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		if verbose {
			logLevel = "debug"
		}

		// Load config
		cfgMgr := config.NewManagerWithLevel(logLevel)
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if cfgPath != "" {
			cfgMgr.SetConfigPath(cfgPath)
		}
		cfg, err := cfgMgr.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
			os.Exit(1)
		}

		// Hint transport implementation
		impl := smokeImplHint
		if impl == "" {
			if runtime.GOOS == "windows" {
				impl = "sidecar"
			} else {
				impl = "sim"
			}
		}
		_ = os.Setenv("MESHEXEC_BLE_IMPL", impl)
		if logger != nil {
			logger.Info("smoke: using BLE implementation", map[string]interface{}{"impl": impl})
		}

		// Build mesh node
		node, err := mesh.NewNodeFromConfig(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to create mesh node: %v\n", err)
			os.Exit(2)
		}

		// Build agent dependencies
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
			fmt.Fprintf(os.Stderr, "node start error: %v\n", err)
			os.Exit(3)
		}
		if err := ag.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "agent start error: %v\n", err)
			_ = node.Stop()
			os.Exit(4)
		}

		// Subscribe for results before sending
		resCh := node.Subscribe(core.MessageTypeResult)

		// Create and send the command message
		mh := messages.NewMessageHandlerWithLevel(logLevel)
		if smokeTarget == "" {
			smokeTarget = "all"
		}
		cmdMsg := mh.CreateCommandMessage(smokeCmdString, nil, []string{smokeTarget}, cfg.Device.Name, "", smokeTimeoutMs)
		if logger != nil {
			logger.Info("smoke: sending command", map[string]interface{}{"id": cmdMsg.ID, "cmd": smokeCmdString, "target": smokeTarget})
		}
		if err := node.SendMessage(&cmdMsg.MeshMessage); err != nil {
			fmt.Fprintf(os.Stderr, "send command error: %v\n", err)
			_ = ag.Stop()
			_ = node.Stop()
			os.Exit(5)
		}

		// Await result
		timeout := time.Duration(smokeTimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		var got *core.MeshMessage
		select {
		case m := <-resCh:
			got = m
		case <-time.After(timeout):
			fmt.Fprintln(os.Stderr, "timed out waiting for result")
			_ = ag.Stop()
			_ = node.Stop()
			os.Exit(6)
		}

		// Print result summary
		fmt.Printf("Result received: id=%s type=%s sender=%s ttl=%d\n", got.ID, got.Type, got.Sender, got.TTL)
		// In this smoke path, stdout/stderr/exitcode are carried inside ResultMessage payload (handled by higher layers),
		// but MeshMessage schema does not include them; we just confirm receipt.

		// Cleanup
		_ = ag.Stop()
		_ = node.Stop()
	},
}

func init() {
	smokeCmd.Flags().StringVar(&smokeCmdString, "command", "echo smoke", "command to execute on this node during the smoke test")
	smokeCmd.Flags().StringVar(&smokeTarget, "target", "all", "target expression for the smoke test")
	smokeCmd.Flags().IntVar(&smokeTimeoutMs, "timeout", 5000, "timeout in milliseconds to wait for a result")
	smokeCmd.Flags().StringVar(&smokeImplHint, "ble-impl", "", "hint BLE implementation: sidecar|sim|native (default: sidecar on Windows, sim otherwise)")

	rootCmd.AddCommand(smokeCmd)
}
