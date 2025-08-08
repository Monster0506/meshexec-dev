package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/monster0506/mechexec/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Manage MechExec CLI configuration files and settings.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		manager := config.NewManager()
		
		// Get config path from global flags
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if configPath != "" {
			manager.SetConfigPath(configPath)
		}
		
		cfg, err := manager.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Configuration loaded from: %s\n", manager.GetConfigPath())
		fmt.Printf("Device Name: %s\n", cfg.Device.Name)
		fmt.Printf("Device Role: %s\n", cfg.Device.Role)
		fmt.Printf("Device OS: %s\n", cfg.Device.OS)
		fmt.Printf("Device Arch: %s\n", cfg.Device.Arch)
		fmt.Printf("Network TTL: %d\n", cfg.Network.TTL)
		fmt.Printf("Network Max Peers: %d\n", cfg.Network.MaxPeers)
		fmt.Printf("Safety Mode: %t\n", cfg.Safety.SafeMode)
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		manager := config.NewManager()
		
		// Get config path from global flags
		configPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if configPath != "" {
			manager.SetConfigPath(configPath)
		}
		
		err := manager.CreateDefaultConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
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
        manager := config.NewManager()
        configPath, _ := cmd.Root().PersistentFlags().GetString("config")
        if configPath != "" {
            manager.SetConfigPath(configPath)
        }
        if _, err := manager.Load(); err != nil {
            fmt.Fprintf(os.Stderr, "Configuration invalid: %v\n", err)
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