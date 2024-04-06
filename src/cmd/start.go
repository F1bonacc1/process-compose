package cmd

import (
	"fmt"
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
		err := getClient().StartProcess(name)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to start process %s", name)

		}
		fmt.Printf("Process %s started\n", name)
	},
}

func init() {
	processCmd.AddCommand(startCmd)
}
