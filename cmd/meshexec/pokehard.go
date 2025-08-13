//go:build dev

package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/monster0506/meshexec/internal/config"
	"github.com/spf13/cobra"
)

// pokehard sends a hardcoded text payload to a hardcoded BLE address via the sidecar
var pokeHardCmd = &cobra.Command{
	Use:   "pokehard",
	Short: "Send a hardcoded BLE write to a specific address (dev tool)",
	Run: func(cmd *cobra.Command, args []string) {
		// Load config for UUIDs
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
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			os.Exit(1)
		}

		// Hardcoded target and payload
		addr := "6c:1d:65:37:a6:20"
		text := "hello from pokehard"
		payloadB64 := base64.StdEncoding.EncodeToString([]byte(text))

		// Compose request for sidecar
		req := map[string]interface{}{
			"action": "central_write_to",
			"params": map[string]interface{}{
				"service_uuid":        cfg.Network.ServiceUUID,
				"characteristic_uuid": cfg.Network.CharacteristicUUID,
				"value_b64":           payloadB64,
				"addresses":           []string{addr},
			},
		}

		// Connect to sidecar
		sidecarAddr := os.Getenv("MESHEXEC_SIDECAR_ADDR")
		if sidecarAddr == "" {
			sidecarAddr = "127.0.0.1:8765"
		}
		if logger != nil {
			logger.Info("pokehard: sending", map[string]interface{}{"addr": addr, "sidecar": sidecarAddr})
		}
		conn, err := net.DialTimeout("tcp", sidecarAddr, 2*time.Second)
		if err != nil {
			fmt.Fprintf(os.Stderr, "connect sidecar: %v\n", err)
			os.Exit(2)
		}
		defer conn.Close()

		enc := json.NewEncoder(conn)
		if err := enc.Encode(req); err != nil {
			fmt.Fprintf(os.Stderr, "send request: %v\n", err)
			os.Exit(3)
		}
		// Read one-line response
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "read response: %v\n", err)
			os.Exit(4)
		}
		fmt.Print(line)
	},
}

func init() {
	rootCmd.AddCommand(pokeHardCmd)
}
