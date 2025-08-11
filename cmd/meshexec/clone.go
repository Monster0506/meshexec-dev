package main

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
)

var (
    cloneTarget string
    cloneDest   string
)

var cloneCmd = &cobra.Command{
    Use:   "clone",
    Short: "Clone a repository or file set from a mesh peer",
    Run: func(cmd *cobra.Command, args []string) {
        // Stub: only parse and echo arguments for now
        fmt.Println("Clone preview (stub)")
        fmt.Printf("  Target : %s\n", cloneTarget)
        if cloneDest != "" {
            fmt.Printf("  Dest   : %s\n", cloneDest)
        }
        fmt.Fprintln(os.Stderr, "Clone is not implemented yet.")
        os.Exit(3)
    },
}

func init() {
    rootCmd.AddCommand(cloneCmd)
    cloneCmd.Flags().StringVarP(&cloneTarget, "target", "t", "auto", "specific peer to clone from (default: auto) (stub)")
    cloneCmd.Flags().StringVar(&cloneDest, "dest", ".", "destination directory (stub)")
}


