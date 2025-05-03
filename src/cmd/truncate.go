package cmd

import (
	"fmt"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

// truncateCmd represents the truncate command
var truncateCmd = &cobra.Command{
	Use:   "truncate [PROCESS]",
	Short: "Truncate the logs for a running or stopped process",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		err := getClient().TruncateProcessLogs(name)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to truncate logs for process %s", name)

		}
		fmt.Printf("Process %s logs truncated\n", name)
	},
}

func init() {
	logsCmd.AddCommand(truncateCmd)
}
