package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/monster0506/meshexec/internal/discovery"
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

// runMessageHook allows tests to inspect the constructed CommandMessage.
// In production, this remains nil.
var runMessageHook func(*core.CommandMessage)

// tcpDial is a test seam to avoid real network dials in unit tests.
var tcpDial = func(addr string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: 3 * time.Second}
	if timeout > 0 {
		d.Timeout = timeout
	}
	return d.Dial("tcp", addr)
}

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
			me := core.NewConfigError("invalid_config", "failed to load configuration", map[string]interface{}{"error": err.Error()})
			fmt.Fprintln(os.Stderr, core.FormatUserError(me))
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

        // CLI performs basic target filtering below

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

		if runMessageHook != nil {
			runMessageHook(msg)
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

		// Non-dry-run: mDNS discover peers and send over TCP
		if logger != nil {
			discovery.SetLogger(logger)
			logger.Debug("run: starting mDNS discovery", map[string]interface{}{"timeout_ms": 5000})
		}
		dctx, dcancel := context.WithTimeout(context.Background(), 5*time.Second)
		peers, derr := discovery.Discover(dctx, 5*time.Second)
		dcancel()
		if derr != nil {
			if logger != nil {
				logger.Warn("run: discovery error", map[string]interface{}{"error": derr.Error()})
			}
		}
		if logger != nil {
			logger.Info("run: discovered peers", map[string]interface{}{"count": len(peers)})
		}
		if len(peers) == 0 {
			fmt.Fprintln(os.Stderr, "No peers discovered via mDNS")
			os.Exit(6)
		}
		// Filter by basic target expression
		selected := make([]core.PeerInfo, 0, len(peers))
		if strings.TrimSpace(strings.ToLower(runTarget)) == "all" || runTarget == "" {
			selected = peers
		} else {
			want := map[string]string{}
			parts := strings.FieldsFunc(runTarget, func(r rune) bool { return r == '&' || r == '|' || r == ' ' })
			for _, pr := range parts {
				if eq := strings.IndexByte(pr, '='); eq > 0 {
					k := strings.ToLower(strings.TrimSpace(pr[:eq]))
					v := strings.Trim(strings.TrimSpace(pr[eq+1:]), "\"")
					want[k] = v
				}
			}
			for _, peer := range peers {
				ok := true
				for k, v := range want {
					switch k {
					case "name":
						ok = ok && strings.EqualFold(peer.Name, v)
					case "role":
						ok = ok && strings.EqualFold(peer.Role, v)
					case "os":
						ok = ok && strings.EqualFold(peer.OS, v)
					case "arch":
						ok = ok && strings.EqualFold(peer.Arch, v)
					default:
						if tv, exists := peer.Tags[k]; exists {
							ok = ok && strings.EqualFold(tv, v)
						} else {
							ok = false
						}
					}
					if !ok {
						break
					}
				}
				if ok {
					selected = append(selected, peer)
				}
			}
		}
		if logger != nil {
			logger.Info("run: peers after target filter", map[string]interface{}{"count": len(selected), "target": runTarget})
		}
		if len(selected) == 0 {
			fmt.Fprintln(os.Stderr, "No peers matched target expression")
			os.Exit(7)
		}
		for _, p := range selected {
			addr := p.Address
			if !strings.Contains(addr, ":") {
				addr = addr + fmt.Sprintf(":%d", cfg.Network.TCPPort)
			}
			if logger != nil {
				logger.Debug("run: dialing peer", map[string]interface{}{"peer": p.Name, "addr": addr})
			}
			res, err := sendCommandTCP(addr, command, time.Duration(runTimeout)*time.Millisecond)
			if err != nil {
				if logger != nil {
					logger.Warn("run: send failed", map[string]interface{}{"peer": p.Name, "addr": addr, "error": err.Error()})
				}
				fmt.Fprintf(os.Stderr, "send %s: %v\n", addr, err)
				continue
			}
			if logger != nil {
				logger.Info("run: got response", map[string]interface{}{"peer": p.Name, "status": res.Status, "code": res.ExitCode})
			}
			fmt.Printf("%s: status=%s code=%d\n", addr, res.Status, res.ExitCode)
			if s := strings.TrimSpace(res.Stdout); s != "" {
				fmt.Println("stdout:\n" + s)
			}
			if s := strings.TrimSpace(res.Stderr); s != "" {
				fmt.Println("stderr:\n" + s)
			}
		}
	},
}

func sendCommandTCP(addr, command string, timeout time.Duration) (*core.ExecutionResult, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	conn, err := tcpDial(addr, 3*time.Second)
	if err != nil {
		return nil, err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	enc := json.NewEncoder(conn)
	if err := enc.Encode(map[string]string{"cmd": command}); err != nil {
		return nil, err
	}
	var resp struct {
		Ok     bool                  `json:"ok"`
		Result *core.ExecutionResult `json:"result"`
	}
	dec := json.NewDecoder(bufio.NewReader(conn))
	if err := dec.Decode(&resp); err != nil {
		return nil, err
	}
	if !resp.Ok {
		return nil, fmt.Errorf("remote error")
	}
	if resp.Result == nil {
		r := &core.ExecutionResult{Status: "unknown"}
		return r, nil
	}
	return resp.Result, nil
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
