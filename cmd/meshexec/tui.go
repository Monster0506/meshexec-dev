package main

import (
	"context"
	"fmt"
	"time"

	"github.com/monster0506/meshexec/internal/config"
	"github.com/monster0506/meshexec/internal/discovery"
	"github.com/monster0506/meshexec/internal/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the MeshExec terminal UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use root logging
		if logger != nil {
			logger.Info("Starting TUI with initial view", map[string]interface{}{"view": tuiView})
		}
		ui := tui.NewManager(logger)

		// Load configuration (silent unless verbose)
		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		if verbose {
			logLevel = "debug"
		}
		cfgMgr := config.NewManagerWithLevel(logLevel)
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if cfgPath != "" {
			cfgMgr.SetConfigPath(cfgPath)
		}
		if _, err := cfgMgr.Load(); err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// mDNS based peer listing for TUI
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if logger != nil {
			logger.Info("tui: mDNS discovery", nil)
		}

		// Subscribe to peer updates and push snapshots into the UI
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-subCtx.Done():
					return
				case <-ticker.C:
					c, cc := context.WithTimeout(context.Background(), 1500*time.Millisecond)
					peers, _ := discovery.Discover(c, 1500*time.Millisecond)
					cc()
					ui.UpdatePeers(peers)
				}
			}
		}()

		// Start TUI
		return ui.StartTUI(ctx, tui.WithInitialView(tuiView))
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	tuiCmd.Flags().StringVar(&tuiView, "view", "overview", "initial view: peers|results|overview")
	// removed allow-sim; BLE disabled
}

var tuiView string

// BLE simulation flag removed
