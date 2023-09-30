package cmd

import (
	"github.com/spf13/cobra"
)

// processCmd represents the process command
var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Execute operations on the available processes",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	rootCmd.AddCommand(processCmd)
	processCmd.PersistentFlags().StringVarP(pcFlags.Address, "address", "a", *pcFlags.Address, "address of the target process compose server")
}
