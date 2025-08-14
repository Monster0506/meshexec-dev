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
		// enable discovery logging
		discovery.SetLogger(logger)

		// Subscribe to peer updates and push snapshots into the UI, sequentially
		go func() {
			interval := 2 * time.Second
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				start := time.Now()
				c, cc := context.WithTimeout(ctx, 4*time.Second)
				peers, err := discovery.Discover(c, 3500*time.Millisecond)
				cc()
				if err != nil && logger != nil {
					logger.Debug("tui: discovery error", map[string]interface{}{"error": err.Error()})
				}
				ui.UpdatePeers(peers)
				// maintain roughly the desired interval
				elapsed := time.Since(start)
				if remaining := interval - elapsed; remaining > 0 {
					select {
					case <-ctx.Done():
						return
					case <-time.After(remaining):
					}
				}
			}
		}()

		// Start TUI
		return ui.StartTUI(ctx, tui.WithInitialView(tuiView), tui.WithTheme(tuiTheme), tui.WithEmoji(!tuiNoEmoji))
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	tuiCmd.Flags().StringVar(&tuiView, "view", "overview", "initial view: peers|results|overview")
	tuiCmd.Flags().StringVar(&tuiTheme, "theme", "dark", "theme: dark|light|hc")
	tuiCmd.Flags().BoolVar(&tuiNoEmoji, "no-emoji", false, "disable emoji/icons in the TUI")
	// removed allow-sim; BLE disabled
}

var tuiView string
var tuiTheme string
var tuiNoEmoji bool

// BLE simulation flag removed
