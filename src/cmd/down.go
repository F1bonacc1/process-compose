package cmd

import (
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
)

// downCmd represents the down command
var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stops all the running processes and terminates the Process Compose",
	Run: func(cmd *cobra.Command, args []string) {
		pcClient := client.NewClient(*pcFlags.Address, *pcFlags.PortNum, *pcFlags.LogLength)
		err := pcClient.ShutDownProject()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to stop project")
		}
		log.Info().Msgf("Project stopped")
	},
}

func init() {
	rootCmd.AddCommand(downCmd)
	downCmd.Flags().StringVarP(pcFlags.Address, "address", "a", *pcFlags.Address, "address of the target process compose server")
}
