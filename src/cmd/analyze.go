package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// analyzeCmd represents the analyze command group.
var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze startup timing and dependency information",
	Long: `Analyze inspects a running process-compose instance to provide insight
into how long each process took to start up and how those times relate
through the dependency graph.

Available subcommands:
  critical-chain    Print the dependency chains ordered by startup time.`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(0)
		}
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
