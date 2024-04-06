package cmd

import (
	"fmt"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List available processes",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		processNames, err := getClient().GetProcessesName()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to list processes")
		}
		for _, proc := range processNames {
			fmt.Println(proc)
		}
	},
}

func init() {
	processCmd.AddCommand(listCmd)
}
