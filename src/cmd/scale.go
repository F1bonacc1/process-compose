package cmd

import (
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/rs/zerolog/log"
	"strconv"

	"github.com/spf13/cobra"
)

// scaleCmd represents the scale command
var scaleCmd = &cobra.Command{
	Use:   "scale [PROCESS] [COUNT]",
	Short: "Scale a process to a given count",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		count, err := strconv.Atoi(args[1])
		if err != nil {
			logFatal(err, "second argument must be an integer")
		}
		err = client.ScaleProcess(*pcFlags.Address, *pcFlags.PortNum, name, count)
		if err != nil {
			logFatal(err, "failed to scale process %s", name)
		}
		log.Info().Msgf("Process %s scaled to %s", name, args[1])
	},
}

func init() {
	processCmd.AddCommand(scaleCmd)
}
