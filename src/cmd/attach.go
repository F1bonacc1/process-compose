package cmd

import (
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/tui"
	"github.com/spf13/cobra"
	"time"
)

// attachCmd represents the attach command
var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach the Process Compose TUI Remotely to a Running Process Compose Server",
	Run: func(cmd *cobra.Command, args []string) {
		pcClient := client.NewClient(*pcFlags.Address, *pcFlags.PortNum, *pcFlags.LogLength)
		tui.SetupTui(pcClient, time.Duration(*pcFlags.RefreshRate)*time.Second)
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.Flags().IntVarP(pcFlags.RefreshRate, "ref-rate", "r", *pcFlags.RefreshRate, "TUI refresh rate in seconds")
	attachCmd.Flags().StringVarP(pcFlags.Address, "address", "a", *pcFlags.Address, "address of the target process compose server")
	attachCmd.Flags().IntVarP(pcFlags.LogLength, "log-length", "l", *pcFlags.LogLength, "log length to display in TUI")
}
