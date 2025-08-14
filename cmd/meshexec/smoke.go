//go:build dev

package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
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
    Short:  "Run a local end-to-end smoke test",
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

        _ = runtime.GOOS // no BLE implementation needed

        // Build mesh node (local-only)
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

		// Print rich result if payload contains a serialized ResultMessage
		printed := false
		if len(got.Payload) > 0 {
			mh := messages.NewMessageHandlerWithLevel(logLevel)
			if v, err := mh.DeserializeMessage(got.Payload); err == nil {
				if res, ok := v.(*core.ResultMessage); ok {
					r := res.Result
					fmt.Printf("Result: status=%s code=%d device=%s\n", r.Status, r.ExitCode, r.Device)
					if s := strings.TrimSpace(r.Stdout); s != "" {
						fmt.Println("stdout:")
						fmt.Println(s)
					}
					if s := strings.TrimSpace(r.Stderr); s != "" {
						fmt.Println("stderr:")
						fmt.Println(s)
					}
					printed = true
				}
			}
		}
		if !printed {
			fmt.Printf("Result received: id=%s type=%s sender=%s ttl=%d\n", got.ID, got.Type, got.Sender, got.TTL)
		}

		// Cleanup
		_ = ag.Stop()
		_ = node.Stop()
	},
}

func init() {
	smokeCmd.Flags().StringVar(&smokeCmdString, "command", "echo smoke", "command to execute on this node during the smoke test")
	smokeCmd.Flags().StringVar(&smokeTarget, "target", "all", "target expression for the smoke test")
	smokeCmd.Flags().IntVar(&smokeTimeoutMs, "timeout", 5000, "timeout in milliseconds to wait for a result")
    // removed BLE impl hint

	rootCmd.AddCommand(smokeCmd)
}
