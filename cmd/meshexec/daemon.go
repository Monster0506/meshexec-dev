package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	core "github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/agent"
	"github.com/monster0506/meshexec/internal/config"
	"github.com/monster0506/meshexec/internal/discovery"
	"github.com/monster0506/meshexec/internal/executor"
	"github.com/monster0506/meshexec/internal/mesh"
	"github.com/monster0506/meshexec/internal/targeting"
	"github.com/spf13/cobra"
)

var daemonForeground bool

// Allow tests to stub mesh/agent builders
var newMeshNodeForDaemon = func(cfg *core.Config) (core.MeshNode, error) { return mesh.NewNodeFromConfig(cfg) }

// waitForSignal can be overridden in tests to avoid blocking on OS signals
var waitForSignal = func() os.Signal {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	return <-sigCh
}

// runDaemon encapsulates the main lifecycle so it can be tested if needed
func runDaemon(cmd *cobra.Command) error {
	// Load configuration
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
	cfg, err := manager.Load()
	if err != nil {
		me := core.NewConfigError("invalid_config", "failed to load configuration", map[string]interface{}{"error": err.Error()})
		fmt.Fprintln(os.Stderr, core.FormatUserError(me))
		return err
	}

	// On Windows, default to sidecar unless user overrides
	if runtime.GOOS == "windows" && os.Getenv("MESHEXEC_BLE_IMPL") == "" {
		_ = os.Setenv("MESHEXEC_BLE_IMPL", "sidecar")
	}

	if logger != nil {
		logger.Info("daemon: building mesh node", nil)
	}
	// Build mesh node
	node, err := newMeshNodeForDaemon(cfg)
	if err != nil {
		if logger != nil {
			logger.Error("daemon: mesh init failed", err, nil)
		}
		return fmt.Errorf("mesh init: %w", err)
	}

	// Build agent with default executor and evaluator
	exec := executor.NewDefaultCommandExecutor(cfg, logger)
	tgt := targeting.NewEvaluatorWithLevel(logLevel)
	dev := core.DeviceInfo{
		Name: cfg.Device.Name,
		Role: cfg.Device.Role,
		OS:   cfg.Device.OS,
		Arch: cfg.Device.Arch,
		Tags: cfg.Device.Tags,
	}
	ag := agent.New(node, nil, exec, tgt, dev, logger)

	// Start services
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if logger != nil {
		logger.Info("daemon: starting node and agent", nil)
	}
	if err := node.Start(ctx); err != nil {
		if logger != nil {
			logger.Error("daemon: mesh start failed", err, nil)
		}
		return fmt.Errorf("mesh start: %w", err)
	}
	if err := ag.Start(ctx); err != nil {
		_ = node.Stop()
		if logger != nil {
			logger.Error("daemon: agent start failed", err, nil)
		}
		return fmt.Errorf("agent start: %w", err)
	}

	if logger != nil {
		logger.Info("daemon started", map[string]interface{}{"device": cfg.Device.Name})
		// Add subscriptions to log inbound command/result messages for observability
		cmdCh := node.Subscribe(core.MessageTypeCommand)
		resCh := node.Subscribe(core.MessageTypeResult)
		go func() {
			for m := range cmdCh {
				logger.Info("daemon: received command", map[string]interface{}{"id": m.ID, "from": m.Sender, "ttl": m.TTL, "cmd": m.Command})
			}
		}()
		go func() {
			for m := range resCh {
				logger.Info("daemon: received result", map[string]interface{}{"id": m.ID, "from": m.Sender, "ttl": m.TTL})
			}
		}()
		// Start simple TCP listener for POC/discovery port and mDNS advertise
		go func(port int) {
			// Normalize port; 0 means ephemeral. We'll advertise the actual bound port below.
			if port < 0 {
				port = 0
			}
			addr := fmt.Sprintf(":%d", port)
			if logger != nil {
				logger.Info("daemon: starting tcp listener", map[string]interface{}{"addr": addr})
			}
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				if logger != nil {
					logger.Warn("tcp listen failed", map[string]interface{}{"addr": addr, "error": err.Error()})
				}
				return
			}
			// Resolve actual port in case of :0
			actualPort := port
			if ta, ok := ln.Addr().(*net.TCPAddr); ok {
				actualPort = ta.Port
			}
			if logger != nil {
				logger.Info("daemon: tcp listener ready", map[string]interface{}{"addr": ln.Addr().String(), "port": actualPort})
			}
			// mDNS advertise
			adv, err := discovery.StartAdvertiser(cfg.Device.Name, actualPort, map[string]string{
				"role": cfg.Device.Role,
				"os":   cfg.Device.OS,
				"arch": cfg.Device.Arch,
			})
			if err == nil {
				if logger != nil {
					logger.Info("daemon: mdns advertiser started", map[string]interface{}{"name": cfg.Device.Name, "port": port})
				}
				go func() { <-ctx.Done(); adv.Stop() }()
			}
			go func() { <-ctx.Done(); _ = ln.Close() }()
			for {
				c, err := ln.Accept()
				if err != nil {
					select {
					case <-ctx.Done():
						return
					default:
					}
					if logger != nil {
						logger.Warn("daemon: accept error", map[string]interface{}{"error": err.Error()})
					}
					continue
				}
				go func(conn net.Conn) {
					if logger != nil {
						logger.Info("daemon: connection accepted", map[string]interface{}{"remote": conn.RemoteAddr().String()})
					}
					defer func() { _ = conn.Close() }()
					// Read one line; support raw text or JSON {"cmd":"..."}
					r := bufio.NewReader(conn)
					line, _ := r.ReadString('\n')
					if logger != nil {
						logger.Debug("daemon: received line", map[string]interface{}{"bytes": len(line)})
					}
					cmdStr := ""
					var payload struct {
						Cmd string `json:"cmd"`
					}
					if json.Unmarshal([]byte(line), &payload) == nil && payload.Cmd != "" {
						cmdStr = payload.Cmd
					} else {
						cmdStr = line
					}
					cmdStr = strings.TrimSpace(cmdStr)
					if cmdStr == "" {
						return
					}
					if logger != nil {
						logger.Info("daemon: executing", map[string]interface{}{"cmd": cmdStr})
					}
					// Execute command via executor
					ctxExec, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()
					res, err := exec.Execute(ctxExec, cmdStr)
					if err != nil && res != nil && res.Stderr == "" {
						res.Stderr = err.Error()
					}
					if res != nil && logger != nil {
						logger.Debug("daemon: exec result", map[string]interface{}{"code": res.ExitCode, "stdout_len": len(res.Stdout), "stderr_len": len(res.Stderr)})
					}
					if res != nil {
						if res.ExitCode == 0 && res.Status == "" {
							res.Status = "success"
						}
						if res.ExitCode != 0 && res.Status == "" {
							res.Status = "failed"
						}
					}
					_ = json.NewEncoder(conn).Encode(map[string]interface{}{
						"ok":     true,
						"result": res,
					})
					if logger != nil {
						logger.Info("daemon: response sent", map[string]interface{}{"remote": conn.RemoteAddr().String()})
					}
				}(c)
			}
		}(cfg.Network.TCPPort)
	}

	// Wait for termination signal (test seam)
	_ = waitForSignal()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = ag.Stop()
	_ = node.Stop()
	select {
	case <-shutdownCtx.Done():
	default:
	}
	if logger != nil {
		logger.Info("daemon stopped", nil)
	}
	return nil
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run the MeshExec agent daemon",
	Long:  "Starts the local agent that listens for mesh commands and executes them securely.",
	RunE:  func(cmd *cobra.Command, args []string) error { return runDaemon(cmd) },
}

func init() {
	daemonCmd.Flags().BoolVar(&daemonForeground, "foreground", true, "Run in foreground")
	rootCmd.AddCommand(daemonCmd)
}
