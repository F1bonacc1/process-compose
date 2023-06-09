package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"math"
	"os"
	"os/signal"
	"time"
)

var (
	follow     bool
	tailLength int
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
		err := logger.ReadProcessLogs(pcAddress, port, name, tailLength, follow, os.Stdout)
		if err != nil {
			log.Error().Msgf("Failed to fetch logs for process %s: %v", name, err)
			return
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

	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVarP(&tailLength, "tail", "n", math.MaxInt, "Number of lines to show from the end of the logs")
}
