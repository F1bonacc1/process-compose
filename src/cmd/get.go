package cmd

import (
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get [PROCESS]",
	Short: "Get a process state",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
        state, _, err := getClient().GetProcessState(name)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to get process state")
		}
		states := types.ProcessesState{States: []types.ProcessState{*state}}
		if *pcFlags.OutputFormat == "" {
			*pcFlags.OutputFormat = "wide"
		}
		//pretty print state
		printStates(&states)
	},
}

func init() {
	processCmd.AddCommand(getCmd)

	getCmd.Flags().StringVarP(pcFlags.OutputFormat, "output", "o", *pcFlags.OutputFormat, "Output format. One of: (json, wide (default))")
}
