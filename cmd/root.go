package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "uigraph",
	Short: "UiGraph CLI - Sync services and APIs to UiGraph Gateway",
	Long: `UiGraph CLI is a stateless, non-interactive tool designed for CI/CD environments.
It syncs service metadata and API specifications to the UiGraph Gateway backend.`,
}

// Execute runs the root command
func Execute(ctx context.Context) {
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Register subcommands
	rootCmd.AddCommand(syncCmd)
}
