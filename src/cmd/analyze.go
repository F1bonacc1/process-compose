package cmd

import (
	"github.com/spf13/cobra"
)

// analyzeCmd represents the analyze command group. It has no Run of its own;
// Cobra auto-prints help when a parent command with no Run is invoked without
// a subcommand.
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze startup timing and dependency information",
	Long: `Analyze inspects a running process-compose instance to provide insight
into how long each process took to start up and how those times relate
through the dependency graph.

Available subcommands:
  critical-chain    Print the dependency chains ordered by startup time.`,
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
