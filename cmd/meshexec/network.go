package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/monster0506/meshexec/internal/discovery"
	"github.com/spf13/cobra"
)

var (
	configNewManagerWithLevel = config.NewManagerWithLevel
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join the mesh network (mDNS stream)",
	Run: func(cmd *cobra.Command, args []string) {
		_ = configNewManagerWithLevel // retain for possible future use
		// Stream mDNS snapshots periodically
		if logger != nil {
			logger.Info("Streaming mDNS discovery (Ctrl-C to exit)", nil)
		}
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			if logger != nil {
				logger.Debug("join: discovering peers", map[string]interface{}{"timeout_ms": 3000})
			}
			peers, _ := discovery.Discover(ctx, 3*time.Second)
			cancel()
			if logger != nil {
				logger.Info("join: discovered peers", map[string]interface{}{"count": len(peers)})
			}
			fmt.Printf("Peers (%d):\n", len(peers))
			for _, p := range peers {
				fmt.Printf("- %s  %s  %s  %s\n", p.Address, p.Name, p.OS, p.Role)
			}
			time.Sleep(3 * time.Second)
		}
	},
}

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover devices via mDNS",
	Run: func(cmd *cobra.Command, args []string) {
		timeout := time.Duration(statusTimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 2000 * time.Millisecond
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if logger != nil {
			logger.Debug("list: mDNS discover", map[string]interface{}{"timeout_ms": timeout.Milliseconds()})
		}
		peers, _ := discovery.Discover(ctx, timeout)
		if logger != nil {
			logger.Info("list: discovered peers", map[string]interface{}{"count": len(peers)})
		}
		if listJSON {
			type out struct {
				Peers []internal.PeerInfo `json:"peers"`
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(out{Peers: peers})
			return
		}
		if len(peers) == 0 {
			fmt.Println("No devices found")
			return
		}
		fmt.Println("Devices found (mDNS):")
		for _, p := range peers {
			fmt.Printf("- %s  %s  %s  %s\n", p.Address, p.Name, p.OS, p.Role)
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List devices (mDNS)",
	Run: func(cmd *cobra.Command, args []string) {
		timeout := time.Duration(listTimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 5000 * time.Millisecond
		}

		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		if verbose {
			logLevel = "debug"
		}
		cfgMgr := configNewManagerWithLevel(logLevel)
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if cfgPath != "" {
			cfgMgr.SetConfigPath(cfgPath)
		}
		_, _ = cfgMgr.Load() // config not used in mDNS-only join

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		peers, _ := discovery.Discover(ctx, timeout)
		if listJSON {
			type out struct {
				Peers []internal.PeerInfo `json:"peers"`
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(out{Peers: peers})
			return
		}
		if len(peers) == 0 {
			fmt.Println("No devices found")
			return
		}
		fmt.Println("Devices found:")
		for _, p := range peers {
			fmt.Printf("- %s  %s  %s  %s\n", p.Address, p.Name, p.OS, p.Role)
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show network status",
	Run: func(cmd *cobra.Command, args []string) {
		// Resolve timeout
		timeout := time.Duration(statusTimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 2 * time.Second
		}

		// Optional since filter
		var sinceCutoff time.Time
		if statusSince != "" {
			if d, err := time.ParseDuration(statusSince); err == nil {
				sinceCutoff = time.Now().Add(-d)
			} else if logger != nil {
				logger.Warn("Invalid --since duration; ignoring", map[string]interface{}{"since": statusSince, "error": err.Error()})
			}
		}

		// Load configuration
		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		if verbose {
			logLevel = "debug"
		}
		cfgMgr := configNewManagerWithLevel(logLevel)
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if cfgPath != "" {
			cfgMgr.SetConfigPath(cfgPath)
		}
		cfg, err := cfgMgr.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// mDNS discovery only
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		peers, _ := discovery.Discover(ctx, timeout)
		// Apply optional time filter
		if !sinceCutoff.IsZero() {
			filtered := make([]internal.PeerInfo, 0, len(peers))
			for _, p := range peers {
				if p.LastSeen.After(sinceCutoff) {
					filtered = append(filtered, p)
				}
			}
			peers = filtered
		}

		// Build status object
		status := internal.NetworkStatus{
			LocalNode: internal.PeerInfo{
				Name:    cfg.Device.Name,
				Role:    cfg.Device.Role,
				OS:      cfg.Device.OS,
				Arch:    cfg.Device.Arch,
				Address: fmt.Sprintf(":%d", cfg.Network.TCPPort),
			},
			Peers:          peers,
			Routes:         map[string][]string{},
			Updated:        time.Now(),
			TotalPeers:     len(peers),
			ConnectedPeers: 0,
		}
		for _, p := range peers {
			if p.Connected {
				status.ConnectedPeers++
			}
		}

		// Output
		if statusJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(status)
			return
		}

		// Human-readable output
		if !sinceCutoff.IsZero() {
			fmt.Printf("Mesh status (since %s): %d peers (%d connected)\n", statusSince, status.TotalPeers, status.ConnectedPeers)
		} else {
			fmt.Printf("Mesh status: %d peers (%d connected)\n", status.TotalPeers, status.ConnectedPeers)
		}
		if len(status.Peers) == 0 {
			fmt.Println("No peers discovered")
			return
		}
		for _, p := range status.Peers {
			age := time.Since(p.LastSeen).Truncate(time.Second)
			fmt.Printf("- %s  %s  RSSI=%d  seen %s ago  connected=%v\n", p.Address, p.Name, p.SignalStrength, age, p.Connected)
		}
	},
}

var (
	joinForeground        bool
	joinScanInterval      int
	joinAdvertiseInterval int
	listJSON              bool
	listTimeoutMs         int
	statusJSON            bool
	statusSince           string
	statusTimeoutMs       int
)

func init() {
	rootCmd.AddCommand(joinCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(discoverCmd)

	// join flags (stubs)
	joinCmd.Flags().BoolVar(&joinForeground, "foreground", false, "run in foreground and stream logs (stub)")
	joinCmd.Flags().IntVar(&joinScanInterval, "scan-interval", 1000, "scan interval in ms (stub)")
	joinCmd.Flags().IntVar(&joinAdvertiseInterval, "advertise-interval", 1000, "advertise interval in ms (stub)")

	// list flags
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output peers as JSON")
	listCmd.Flags().IntVar(&listTimeoutMs, "timeout", 5000, "scan timeout in ms")

	// discover flags reuse list/status flags
	discoverCmd.Flags().BoolVar(&listJSON, "json", false, "output peers as JSON")
	discoverCmd.Flags().IntVar(&statusTimeoutMs, "timeout", 2000, "discover timeout in ms")

	// status flags
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output status as JSON")
	statusCmd.Flags().StringVar(&statusSince, "since", "", "filter peers seen within duration, e.g. 10m")
	statusCmd.Flags().IntVar(&statusTimeoutMs, "timeout", 2000, "scan timeout in ms")
}
