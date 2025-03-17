package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"os"
	"sort"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List available processes",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		states, err := getClient().GetRemoteProcessesState()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to list processes")
		}
		//sort states by name
		sort.Slice(states.States, func(i, j int) bool {
			return states.States[i].Name < states.States[j].Name
		})

		printStates(states)

	},
}

func printStates(states *types.ProcessesState) {
	switch *pcFlags.OutputFormat {
	case "json":
		b, err := json.MarshalIndent(states.States, "", "\t")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to marshal processes")
		}
		os.Stdout.Write(b)
	case "wide":
		app.PrintStatesAsTable(states.States)
	case "":
		for _, state := range states.States {
			fmt.Println(state.Name)
		}
	default:
		log.Fatal().Msgf("unknown output format %s", *pcFlags.OutputFormat)
	}
}

func init() {
	processCmd.AddCommand(listCmd)
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringVarP(pcFlags.OutputFormat, "output", "o", *pcFlags.OutputFormat, "Output format. One of: (json, wide)")
}
