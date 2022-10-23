/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start [PROCESS]",
	Short: "Start a process",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		err := client.StartProcesses(pcAddress, port, name)
		if err != nil {
			log.Error().Msgf("Failed to start processes %s: %v", name, err)
			return
		}
		log.Info().Msgf("Process %s started", name)
	},
}

func init() {
	processCmd.AddCommand(startCmd)
}
