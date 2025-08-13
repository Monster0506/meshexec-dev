package main

import (
	"fmt"
	"os"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage MeshExec CLI configuration files and settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		if verbose {
			logLevel = "debug"
		}
		manager := config.NewManagerWithLevel(logLevel)

		// Get config path from global flags
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if configPath != "" {
			manager.SetConfigPath(configPath)
		}

		cfg, err := manager.Load()
		if err != nil {
			me := internal.NewConfigError("invalid_config", "failed to load configuration", map[string]interface{}{"error": err.Error()})
			fmt.Fprintln(os.Stderr, internal.FormatUserError(me))
			os.Exit(1)
		}

		fmt.Printf("Configuration loaded from: %s\n", manager.GetConfigPath())
		fmt.Printf("Device Name: %s\n", cfg.Device.Name)
		fmt.Printf("Device Role: %s\n", cfg.Device.Role)
		fmt.Printf("Device OS: %s\n", cfg.Device.OS)
		fmt.Printf("Device Arch: %s\n", cfg.Device.Arch)
		fmt.Printf("Network TCP Port: %d\n", cfg.Network.TCPPort)
		fmt.Printf("Network TTL: %d\n", cfg.Network.TTL)
		fmt.Printf("Network Max Peers: %d\n", cfg.Network.MaxPeers)
		fmt.Printf("Safety Mode: %t\n", cfg.Safety.SafeMode)
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		if verbose {
			logLevel = "debug"
		}
		manager := config.NewManagerWithLevel(logLevel)

		// Get config path from global flags
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if configPath != "" {
			manager.SetConfigPath(configPath)
		}

		err := manager.CreateDefaultConfig()
		if err != nil {
			me := internal.NewConfigError("create_failed", "failed to create default configuration", map[string]interface{}{"error": err.Error()})
			fmt.Fprintln(os.Stderr, internal.FormatUserError(me))
			os.Exit(1)
		}

		fmt.Printf("Default configuration created at: %s\n", manager.GetConfigPath())
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open configuration in default editor",
	Run: func(cmd *cobra.Command, args []string) {
		// Stub implementation
		fmt.Fprintln(os.Stderr, "Config edit is not implemented yet.")
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		if verbose {
			logLevel = "debug"
		}
		manager := config.NewManagerWithLevel(logLevel)
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if configPath != "" {
			manager.SetConfigPath(configPath)
		}
		if _, err := manager.Load(); err != nil {
			me := internal.NewConfigError("invalid_config", "configuration invalid", map[string]interface{}{"error": err.Error()})
			fmt.Fprintln(os.Stderr, internal.FormatUserError(me))
			os.Exit(1)
		}
		fmt.Println("Configuration is valid.")
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}
