/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop [PROCESS]",
	Short: "Stop a running process",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		err := client.StopProcesses(pcAddress, port, name)
		if err != nil {
			log.Error().Msgf("Failed to stop processes %s: %v", name, err)
			return
		}
		log.Info().Msgf("Process %s stopped", name)
	},
}

func init() {
	processCmd.AddCommand(stopCmd)
}
