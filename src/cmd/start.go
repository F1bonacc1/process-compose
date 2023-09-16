/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [PROCESS]",
	Short: "Start a process",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		err := client.StartProcess(*pcFlags.Address, *pcFlags.PortNum, name)
		if err != nil {
			logFatal(err, "failed to start process %s", name)
		}
		fmt.Printf("Process %s started\n", name)
	},
}

func init() {
	processCmd.AddCommand(startCmd)
}
