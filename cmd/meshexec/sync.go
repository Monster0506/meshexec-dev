package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	syncRepoPath   string
	syncTargetExpr string
	syncDirection  string
	syncDryRun     bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize files or repositories between mesh peers",
	Run: func(cmd *cobra.Command, args []string) {
		if logger != nil {
			logger.Info("sync: invoked", map[string]interface{}{
				"repo": syncRepoPath, "target": syncTargetExpr, "direction": syncDirection, "dry_run": syncDryRun,
			})
		}
		// Stub: only parse and echo arguments for now
		if syncDryRun {
			fmt.Println("Dry run: sync preview")
			if syncRepoPath != "" {
				fmt.Printf("  Repo   : %s\n", syncRepoPath)
			}
			fmt.Printf("  Target : %s\n", syncTargetExpr)
			if syncDirection != "" {
				fmt.Printf("  Dir    : %s\n", syncDirection)
			}
			return
		}
		// Not implemented yet
		if logger != nil {
			logger.Warn("sync: not implemented", nil)
		}
		fmt.Fprintln(os.Stderr, "Sync is not implemented yet. Use --dry-run to preview.")
		os.Exit(3)
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringVar(&syncRepoPath, "repo", "", "path to a Git repository to share with peers (stub)")
	syncCmd.Flags().StringVarP(&syncTargetExpr, "target", "t", "all", "target devices to sync with (stub)")
	syncCmd.Flags().StringVar(&syncDirection, "direction", "both", "sync direction: push|pull|both (stub)")
	syncCmd.Flags().BoolVar(&syncDryRun, "dry-run", false, "show what would be synced without transferring (stub)")
}
