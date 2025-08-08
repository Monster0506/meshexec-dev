package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/monster0506/mechexec/internal/config"
	"github.com/monster0506/mechexec/internal/messages"
    "github.com/monster0506/mechexec/internal/executor"
	"github.com/spf13/cobra"
)

var (
	runTarget   string
	runDryRun   bool
	runWorkDir  string
	runTimeout  int
    runSafeMode bool
    runNoSign   bool
    runEncrypt  bool
    runFormat   string
)

var runCmd = &cobra.Command{
	Use:   "run [command] [args...]",
	Short: "Send a command to the mesh",
	Long:  "Run a shell command across the mesh targeting selected devices.",
	Args:  cobra.MinimumNArgs(1),
	Example: "mechexec run -t \"os=linux && role=worker\" -- echo hello\nmechexec run --target all -- uptime\nmechexec run --dry-run -t 'arch=arm' -- cat /proc/cpuinfo",
	Run: func(cmd *cobra.Command, args []string) {
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

		// Build the command and arguments
		command := args[0]
		cmdArgs := []string{}
		if len(args) > 1 {
			cmdArgs = args[1:]
		}

        // Safety validation (if enabled via flag or config)
        effectiveSafe := runSafeMode || cfg.Safety.SafeMode
        if effectiveSafe {
            full := command
            if len(cmdArgs) > 0 {
                full = full + " " + strings.Join(cmdArgs, " ")
            }
            checker := executor.NewSafetyChecker(cfg, logger)
            if err := checker.ValidateCommand(full); err != nil {
                if logger != nil {
                    logger.Warn("Command blocked by safety policy", map[string]interface{}{
                        "error": err.Error(),
                        "command": command,
                    })
                }
                fmt.Fprintf(os.Stderr, "Blocked by safety policy: %v\n", err)
                os.Exit(2)
            }
        }

        // Target evaluator integration pending – provide a placeholder message
        if logger != nil {
            logger.Info("Target evaluator integration not yet wired to CLI; proceeding with provided expression", map[string]interface{}{
                "target": runTarget,
            })
        }

        // Create a message to represent what would be sent
		mh := messages.NewMessageHandler()
		msg := mh.CreateCommandMessage(command, cmdArgs, []string{runTarget}, cfg.Device.Name, runWorkDir, runTimeout)

		if runDryRun {
			// Show dry-run information
			fmt.Println("Dry run: command dispatch preview")
			fmt.Printf("  Command: %s\n", command)
			if len(cmdArgs) > 0 {
				fmt.Printf("  Args   : %s\n", strings.Join(cmdArgs, " "))
			}
			fmt.Printf("  Target : %s\n", runTarget)
			if runWorkDir != "" {
				fmt.Printf("  Workdir: %s\n", runWorkDir)
			}
			fmt.Printf("  Timeout: %dms\n", runTimeout)
            fmt.Printf("  Safe   : %t\n", runSafeMode)
            fmt.Printf("  Sign   : %s\n", map[bool]string{true: "disabled", false: "enabled"}[runNoSign])
            fmt.Printf("  Encrypt: %t\n", runEncrypt)
            if runFormat != "" {
                fmt.Printf("  Format : %s\n", runFormat)
            }
			fmt.Printf("  Msg ID : %s\n", msg.ID)
			return
		}

		// Non-dry-run: execution pipeline not implemented yet
        if logger != nil {
            logger.Warn("Execution pipeline not implemented yet. Use --dry-run for now.", map[string]interface{}{
                "command": command,
                "args":    cmdArgs,
                "target":  runTarget,
                "safe":    runSafeMode,
                "no_sign": runNoSign,
                "encrypt": runEncrypt,
                "format":  runFormat,
            })
        }
		fmt.Fprintln(os.Stderr, "Execution is not implemented yet. Use --dry-run to preview.")
		os.Exit(3)
	},
}

func init() {
	runCmd.Flags().StringVarP(&runTarget, "target", "t", "all", "target expression (e.g., 'os=linux && role=worker')")
	runCmd.Flags().BoolVar(&runDryRun, "dry-run", false, "show what would be executed without sending")
	runCmd.Flags().StringVarP(&runWorkDir, "workdir", "w", "", "working directory for command execution")
	runCmd.Flags().IntVarP(&runTimeout, "timeout", "T", 30000, "command timeout in milliseconds")
    runCmd.Flags().BoolVar(&runSafeMode, "safe-mode", false, "enable safety filters for dangerous commands (stub)")
    runCmd.Flags().BoolVar(&runNoSign, "no-sign", false, "do not sign messages (stub)")
    runCmd.Flags().BoolVar(&runEncrypt, "encrypt", false, "encrypt command payloads (stub)")
    runCmd.Flags().StringVar(&runFormat, "format", "", "output format for results: text|json (stub)")

	rootCmd.AddCommand(runCmd)
}
