package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/monster0506/meshexec/internal/ble"
	"github.com/monster0506/meshexec/internal/config"
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
		cfg, err := cfgMgr.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Initialize BLE transport (no simulation unless explicitly allowed)
		tr, err := ble.NewWithLogger(&cfg.Network, logger)
		if err != nil {
			return err
		}
		// Disallow simulated transport by default
		if !tuiAllowSim {
			if _, isSim := tr.(*ble.Transport); isSim {
				return errors.New("no real BLE transport available (simulation disabled); ensure hardware/backend present or pass --allow-sim")
			}
		}

		// Create BLE manager and start discovery
		mgr := ble.NewManager(tr, logger)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := mgr.StartDiscovery(ctx); err != nil {
			return err
		}
		if logger != nil {
			logger.Info("tui: discovery started", nil)
		}

		// Subscribe to peer updates and push snapshots into the UI
		subCtx, subCancel := context.WithCancel(ctx)
		defer subCancel()
		updates := mgr.Subscribe(subCtx)
		go func() {
			// Debounce bursts to reduce UI churn
			var last time.Time
			for range updates {
				now := time.Now()
				if now.Sub(last) < 200*time.Millisecond {
					continue
				}
				last = now
				peers := mgr.ListPeers()
				ui.UpdatePeers(peers)
			}
		}()

		// Start TUI
		return ui.StartTUI(ctx, tui.WithInitialView(tuiView))
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	tuiCmd.Flags().StringVar(&tuiView, "view", "overview", "initial view: peers|results|overview")
	tuiCmd.Flags().BoolVar(&tuiAllowSim, "allow-sim", false, "allow simulated BLE transport if native is unavailable")
}

var tuiView string
var tuiAllowSim bool
