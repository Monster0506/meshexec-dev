package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/monster0506/meshexec/internal/executor"
	"github.com/monster0506/meshexec/internal/messages"
	"github.com/spf13/cobra"
)

var (
	runTarget    string
	runDryRun    bool
	runWorkDir   string
	runTimeout   int
	runSafeMode  bool
	runNoSign    bool
	runEncrypt   bool
	runFormat    string
	runSync      bool
	runAt        string
	runEnv       []string
	runStdinFile string
)

var runCmd = &cobra.Command{
	Use:     "run [command] [args...]",
	Short:   "Send a command to the mesh",
	Long:    "Run a shell command across the mesh targeting selected devices.",
	Args:    cobra.MinimumNArgs(1),
	Example: "meshexec run -t \"os=linux && role=worker\" -- echo hello\nmeshexec run --target all -- uptime\nmeshexec run --dry-run -t 'arch=arm' -- cat /proc/cpuinfo",
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
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
			me := internal.NewConfigError("invalid_config", "failed to load configuration", map[string]interface{}{"error": err.Error()})
			fmt.Fprintln(os.Stderr, internal.FormatUserError(me))
			os.Exit(1)
		}

		// Log invocation
		if logger != nil {
			logger.Info("run: starting command dispatch", map[string]interface{}{
				"target": runTarget, "dry_run": runDryRun, "workdir": runWorkDir, "timeout_ms": runTimeout,
				"safe_mode": runSafeMode, "no_sign": runNoSign, "encrypt": runEncrypt, "format": runFormat,
				"sync": runSync, "at": runAt, "env_count": len(runEnv), "stdin_file": runStdinFile,
			})
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
						"error":   err.Error(),
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
		// Fill niceties when present (schema supports omitempty)
		msg.TargetExpr = runTarget
		if len(runEnv) > 0 {
			msg.Env = make(map[string]string, len(runEnv))
			for _, kv := range runEnv {
				if kv == "" {
					continue
				}
				if eq := strings.IndexByte(kv, '='); eq > 0 {
					k := kv[:eq]
					v := kv[eq+1:]
					msg.Env[k] = v
				}
			}
		}
		if runAt != "" {
			// Defer parsing to backend; leave as string in CLI, but also store planned epoch if parseable
			if d, err := time.Parse("15:04", runAt); err == nil {
				// Today at HH:MM; backend may reinterpret
				now := time.Now()
				when := time.Date(now.Year(), now.Month(), now.Day(), d.Hour(), d.Minute(), 0, 0, now.Location())
				if when.Before(now) {
					when = when.Add(24 * time.Hour)
				}
				msg.ScheduledAt = when.Unix()
			}
		}
		if runStdinFile != "" {
			msg.StdinRef = runStdinFile
		}

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
			if runSync {
				fmt.Printf("  Sync   : %t\n", runSync)
			}
			if runAt != "" {
				fmt.Printf("  At     : %s\n", runAt)
			}
			if len(runEnv) > 0 {
				fmt.Printf("  Env    : %s\n", strings.Join(runEnv, ", "))
			}
			if runStdinFile != "" {
				fmt.Printf("  Stdin  : %s\n", runStdinFile)
			}
			if runFormat != "" {
				fmt.Printf("  Format : %s\n", runFormat)
			}
			fmt.Printf("  Msg ID : %s\n", msg.ID)
			if logger != nil {
				logger.Info("run: dry-run complete", map[string]interface{}{"msg_id": msg.ID})
			}
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
	runCmd.Flags().BoolVar(&runSync, "sync", false, "ensure synchronized execution start across targets (stub)")
	runCmd.Flags().StringVar(&runAt, "at", "", "schedule execution at a specific time (e.g., '03:00' or '+5m') (stub)")
	runCmd.Flags().StringArrayVar(&runEnv, "env", nil, "environment variables in KEY=VAL form (repeatable) (stub)")
	runCmd.Flags().StringVar(&runStdinFile, "stdin-file", "", "file path to send as stdin to the command (stub)")

	rootCmd.AddCommand(runCmd)
}
