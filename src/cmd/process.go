package cmd

import (
	"github.com/spf13/cobra"
)

// processCmd represents the process command
var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Execute operations on available processes",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(processCmd)
	processCmd.PersistentFlags().StringVarP(&pcAddress, "address", "a", "localhost", "address of a running process compose server")
}
