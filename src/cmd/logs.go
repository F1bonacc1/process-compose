/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/rs/zerolog/log"
	"math"

	"github.com/spf13/cobra"
)

var (
	follow     bool
	tailLength int
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs [PROCESS]",
	Short: "Fetch the logs of a process",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		err := client.ReadProcessLogs(pcAddress, port, name, tailLength, follow)
		if err != nil {
			log.Error().Msgf("Failed to fetch logs for process %s: %v", name, err)
			return
		}
	},
}

func init() {
	processCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&tailLength, "tail", "n", math.MaxInt, "Number of lines to show from the end of the logs (default - all)")
}
