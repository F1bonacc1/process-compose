package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"strings"
	"time"
)

// logsCmd represents the logs command
var logsCmd = &cobra.Command{
	Use:   "logs [PROCESS]",
	Short: "Fetch the logs of a process(es). For multiple processes, separate them with a comma (proc1,proc2)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		printProcessName := !*pcFlags.IsRawLogOutput && len(strings.Split(name, ",")) > 1
		ct := pclog.NewColorTracker()
		logger := getLogClient()
		fn := func(message api.LogMessage) {
			if printProcessName {
				fmt.Printf("[%s\t] %s\n", ct.GetColor(message.ProcessName)(message.ProcessName), message.Message)
			} else {
				fmt.Printf("%s\n", message.Message)
			}
		}
		done, err := logger.ReadProcessLogs(name, *pcFlags.LogTailLength, *pcFlags.LogFollow, fn)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to fetch logs for process %s", name)
		}
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		select {
		case <-interrupt:
			_ = logger.CloseChannel()
			time.Sleep(time.Second)
		case <-done:
			_ = logger.CloseChannel()
			time.Sleep(time.Second)
		}

	},
}

func init() {
	processCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolVarP(pcFlags.LogFollow, "follow", "f", *pcFlags.LogFollow, "Follow log output")
	logsCmd.Flags().BoolVar(pcFlags.IsRawLogOutput, "raw-log", *pcFlags.IsRawLogOutput, "If set, don't format the multi process log output to include the process name")
	logsCmd.Flags().IntVarP(pcFlags.LogTailLength, "tail", "n", *pcFlags.LogTailLength, "Number of lines to show from the end of the logs")
}

func getLogClient() *client.LogClient {
	var lc *client.LogClient
	if *pcFlags.IsUnixSocket {
		lc = client.NewLogClient("unix", *pcFlags.UnixSocketPath)
	} else {
		address := fmt.Sprintf("%s:%d", *pcFlags.Address, *pcFlags.PortNum)
		lc = client.NewLogClient(address, "")
	}
	return lc
}
