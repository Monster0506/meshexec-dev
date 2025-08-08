package main

import (
    "context"
    "time"

    "github.com/monster0506/mechexec/internal"
    "github.com/monster0506/mechexec/internal/tui"
    "github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
    Use:   "tui",
    Short: "Launch the MechExec terminal UI",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Use root logging
        ui := tui.NewManager(logger)

        // For now, seed with some demo data until real integrations are implemented
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()

        // Simulate periodic peer updates in the background
        go func() {
            peers := []internal.PeerInfo{
                {ID: "1", Name: "alpha", Address: "A1:B2:C3:D4", Role: "leader", OS: "windows", Arch: "amd64", SignalStrength: -45, Connected: true, LastSeen: time.Now()},
                {ID: "2", Name: "bravo", Address: "E5:F6:G7:H8", Role: "worker", OS: "linux", Arch: "arm64", SignalStrength: -60, Connected: true, LastSeen: time.Now()},
                {ID: "3", Name: "charlie", Address: "I9:J0:K1:L2", Role: "worker", OS: "darwin", Arch: "arm64", SignalStrength: -72, Connected: false, LastSeen: time.Now().Add(-2 * time.Minute)},
            }
            ticker := time.NewTicker(3 * time.Second)
            defer ticker.Stop()
            for {
                select {
                case <-ctx.Done():
                    return
                case <-ticker.C:
                    ui.UpdatePeers(peers)
                }
            }
        }()

        // Simulate results arriving later
        go func() {
            time.Sleep(5 * time.Second)
            res := &internal.ExecutionResults{
                CommandID: "demo-1",
                Command:   "echo hello",
                Target:    "role=worker",
                Results: []internal.ExecutionResult{
                    {ID: "r1", Device: "alpha", ExitCode: 0, Stdout: "hello", Duration: 1200, Status: "ok"},
                    {ID: "r2", Device: "bravo", ExitCode: 1, Stdout: "", Stderr: "permission denied", Duration: 900, Status: "failed"},
                },
                Summary: internal.ResultSummary{TotalDevices: 2, Successful: 1, Failed: 1, Timeout: 0, AverageDuration: 1050},
                Timestamp: time.Now(),
            }
            ui.UpdateResults(res)
        }()

        return ui.StartTUI(ctx)
    },
}

func init() {
    rootCmd.AddCommand(tuiCmd)
}

