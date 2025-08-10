package main

import (
	"fmt"
	"os"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/spf13/cobra"
)

var daemonForeground bool

// daemonCmd starts the local agent daemon. This is a stub until the agent and mesh layers are implemented.
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the MeshExec agent daemon",
	Long:  "Starts the local agent that will listen for mesh commands and execute them securely (stub).",
	RunE: func(cmd *cobra.Command, args []string) error {
		if logger != nil {
			logger.Info("Starting daemon (stub)", map[string]interface{}{
				"foreground": daemonForeground,
			})
		} else {
			fmt.Println("Starting daemon (stub)")
		}

		// Load configuration (if present) so we honor user/device settings early
		logLevel, _ := cmd.Root().PersistentFlags().GetString("log-level")
		verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
		if verbose {
			logLevel = "debug"
		}
		manager := config.NewManagerWithLevel(logLevel)
		cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
		if cfgPath != "" {
			manager.SetConfigPath(cfgPath)
		}
		if _, err := manager.Load(); err != nil {
			if logger != nil {
				logger.Warn("Proceeding without configuration (could not load)", map[string]interface{}{
					"error": err,
				})
			}
			fmt.Fprintln(os.Stderr, internal.FormatUserError(internal.NewConfigError("invalid_config", "failed to load configuration", map[string]interface{}{"error": err.Error()})))
		}

		fmt.Println("Agent daemon is not implemented yet. See tasks in tasks.md (section 8).")
		return nil
	},
}

func init() {
	daemonCmd.Flags().BoolVar(&daemonForeground, "foreground", true, "Run in foreground")
	rootCmd.AddCommand(daemonCmd)
}
