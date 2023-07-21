package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/spf13/cobra"
)

// restartCmd represents the restart command
var restartCmd = &cobra.Command{
	Use:   "restart [PROCESS]",
	Short: "Restart a process",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		err := client.RestartProcess(pcAddress, port, name)
		if err != nil {
			logFatal(err, "failed to restart process %s", name)
		}
		fmt.Printf("Process %s restarted\n", name)
	},
}

func init() {
	processCmd.AddCommand(restartCmd)
}
