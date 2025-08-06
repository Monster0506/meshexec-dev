package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mechexec",
	Short: "MechExec CLI - Decentralized command execution over Bluetooth LE mesh",
	Long: `MechExec CLI allows multiple Bluetooth-enabled devices to form a self-healing mesh network 
to broadcast, relay, and execute shell commands securely and efficiently without Wi-Fi or central infrastructure.

Features:
- Decentralized shell command execution
- Bluetooth LE mesh network (no Wi-Fi)
- Role-based device discovery and targeting
- Secure message passing
- Low-latency, resilient command propagation
- TUI and CLI interfaces`,
	Version: "0.1.0",
}

func init() {
	// Add global flags here
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is $HOME/.mechexec/config.toml)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
} 