package main

import (
	"fmt"
	"os"

	"github.com/monster0506/mechexec/internal/logging"
	"github.com/spf13/cobra"
)

var logger *logging.Logger

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
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	// Add global flags here
	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default is $HOME/.mechexec/config.toml)")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	
	// Initialize logging after flags are defined
	cobra.OnInitialize(initializeLogging)
}

func initializeLogging() {
	// Get log level from flags
	logLevel, _ := rootCmd.PersistentFlags().GetString("log-level")
	verbose, _ := rootCmd.PersistentFlags().GetBool("verbose")
	
	// Override log level if verbose flag is set
	if verbose {
		logLevel = "debug"
	}
	
	// Initialize logger
	logger = logging.NewLogger(logLevel)
	
	// Log startup information
	logger.Info("MechExec CLI starting", map[string]interface{}{
		"version":   rootCmd.Version,
		"log_level": logLevel,
		"verbose":   verbose,
	})
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		if logger != nil {
			logger.Error("CLI execution failed", err, nil)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}
