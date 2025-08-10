package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/ble"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/spf13/cobra"
)

var joinCmd = &cobra.Command{
	Use:   "join",
	Short: "Join the mesh network",
	Run: func(cmd *cobra.Command, args []string) {
		cfgMgr := config.NewManager()
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if cfgPath != "" {
			cfgMgr.SetConfigPath(cfgPath)
		}
		cfg, err := cfgMgr.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Initialize BLE
		transport, err := ble.NewWithLogger(&cfg.Network, logger)
		if err != nil {
			fmt.Fprintf(os.Stderr, "BLE init error: %v\n", err)
			os.Exit(1)
		}
		mgr := ble.NewManager(transport, logger)

		// Try to advertise; Windows backend will return error (unsupported)
		advCtx, advCancel := context.WithCancel(context.Background())
		defer advCancel()
		if err := transport.Advertise(advCtx, []byte("meshexec")); err != nil {
			if logger != nil {
				logger.Warn("Advertising not available; continuing with scanning only", map[string]interface{}{"error": err.Error()})
			}
		}

		// Start discovery
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := mgr.StartDiscovery(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Discovery error: %v\n", err)
			os.Exit(1)
		}

		// Stream updates
		updates := mgr.Subscribe(ctx)
		if logger != nil {
			logger.Info("Joined mesh; streaming peer updates (Ctrl-C to exit)", nil)
		} else {
			fmt.Fprintln(os.Stderr, "Joined mesh; streaming peer updates (Ctrl-C to exit)")
		}
		for p := range updates {
			fmt.Printf("peer: name=%s addr=%s rssi=%d connected=%v\n", p.Name, p.Address, p.SignalStrength, p.Connected)
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List nearby BLE devices",
	Run: func(cmd *cobra.Command, args []string) {
		timeout := time.Duration(listTimeoutMs) * time.Millisecond
		if timeout <= 0 {
			timeout = 5000 * time.Millisecond
		}

		cfgMgr := config.NewManager()
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if cfgPath != "" {
			cfgMgr.SetConfigPath(cfgPath)
		}
		cfg, err := cfgMgr.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		transport, err := ble.NewWithLogger(&cfg.Network, logger)
		if err != nil {
			fmt.Fprintf(os.Stderr, "BLE init error: %v\n", err)
			os.Exit(1)
		}
		mgr := ble.NewManager(transport, logger)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := mgr.StartDiscovery(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Discovery error: %v\n", err)
			os.Exit(1)
		}
		<-ctx.Done()

		peers := mgr.ListPeers()
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
			fmt.Printf("- %s  %s  RSSI=%d\n", p.Address, p.Name, p.SignalStrength)
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
		cfgMgr := config.NewManager()
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if cfgPath != "" {
			cfgMgr.SetConfigPath(cfgPath)
		}
		cfg, err := cfgMgr.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Initialize BLE and manager
		transport, err := ble.NewWithLogger(&cfg.Network, logger)
		if err != nil {
			fmt.Fprintf(os.Stderr, "BLE init error: %v\n", err)
			os.Exit(1)
		}
		mgr := ble.NewManager(transport, logger)

		// Perform a brief discovery scan
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		if err := mgr.StartDiscovery(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Discovery error: %v\n", err)
			os.Exit(1)
		}
		<-ctx.Done()

		// Collect peers and apply optional time filter
		peers := mgr.ListPeers()
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
				Name: cfg.Device.Name,
				Role: cfg.Device.Role,
				OS:   cfg.Device.OS,
				Arch: cfg.Device.Arch,
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

	// join flags (stubs)
	joinCmd.Flags().BoolVar(&joinForeground, "foreground", false, "run in foreground and stream logs (stub)")
	joinCmd.Flags().IntVar(&joinScanInterval, "scan-interval", 1000, "scan interval in ms (stub)")
	joinCmd.Flags().IntVar(&joinAdvertiseInterval, "advertise-interval", 1000, "advertise interval in ms (stub)")

	// list flags
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output peers as JSON")
	listCmd.Flags().IntVar(&listTimeoutMs, "timeout", 5000, "scan timeout in ms")

	// status flags
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output status as JSON")
	statusCmd.Flags().StringVar(&statusSince, "since", "", "filter peers seen within duration, e.g. 10m")
	statusCmd.Flags().IntVar(&statusTimeoutMs, "timeout", 2000, "scan timeout in ms")
}
