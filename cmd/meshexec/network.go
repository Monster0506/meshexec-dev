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
        transport, err := ble.New(&cfg.Network)
        if err != nil {
            fmt.Fprintf(os.Stderr, "BLE init error: %v\n", err)
            os.Exit(1)
        }
        mgr := ble.NewManager(transport)

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

        transport, err := ble.New(&cfg.Network)
        if err != nil {
            fmt.Fprintf(os.Stderr, "BLE init error: %v\n", err)
            os.Exit(1)
        }
        mgr := ble.NewManager(transport)

        ctx, cancel := context.WithTimeout(context.Background(), timeout)
        defer cancel()
        if err := mgr.StartDiscovery(ctx); err != nil {
            fmt.Fprintf(os.Stderr, "Discovery error: %v\n", err)
            os.Exit(1)
        }
        <-ctx.Done()

        peers := mgr.ListPeers()
        if listJSON {
            type out struct{ Peers []internal.PeerInfo `json:"peers"` }
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
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = ctx
		if logger != nil {
            logger.Info("Status command (stub)", map[string]interface{}{"json": statusJSON, "since": statusSince})
		}
        if statusJSON {
            fmt.Println("{\"status\": \"not_implemented\"}")
            return
        }
        if statusSince != "" {
            fmt.Printf("Status since %s: (not implemented)\n", statusSince)
            return
        }
        fmt.Println("Status: (not implemented)")
	},
}

var (
    joinForeground       bool
    joinScanInterval     int
    joinAdvertiseInterval int
    listJSON             bool
    listTimeoutMs        int
    statusJSON           bool
    statusSince          string
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
    statusCmd.Flags().BoolVar(&statusJSON, "json", false, "output status as JSON (stub)")
    statusCmd.Flags().StringVar(&statusSince, "since", "", "filter results newer than duration, e.g. 10m (stub)")
}
