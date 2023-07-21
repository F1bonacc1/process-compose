package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop [PROCESS...]",
	Short: "Stop a running process",
	Run: func(cmd *cobra.Command, args []string) {
		stopped, err := client.StopProcesses(pcAddress, port, args)
		if err != nil {
			logFatal(err, "failed to stop processes %v", args)
		}
		fmt.Printf("Processes %v stopped\n", stopped)
	},
}

func init() {
	processCmd.AddCommand(stopCmd)
}
