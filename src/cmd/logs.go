package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"time"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs [PROCESS]",
	Short: "Fetch the logs of a process",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		logger := client.LogClient{
			Format: "%s\n",
		}
		err := logger.ReadProcessLogs(*pcFlags.Address, *pcFlags.PortNum, name, *pcFlags.LogTailLength, *pcFlags.LogFollow, os.Stdout)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to fetch logs for process %s", name)
		}
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		select {
		case <-interrupt:
			fmt.Println("interrupt")
			_ = logger.CloseChannel()
			time.Sleep(time.Second)
		}

	},
}

func init() {
	processCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolVarP(pcFlags.LogFollow, "follow", "f", *pcFlags.LogFollow, "Follow log output")
	logsCmd.Flags().IntVarP(pcFlags.LogTailLength, "tail", "n", *pcFlags.LogTailLength, "Number of lines to show from the end of the logs")
}
