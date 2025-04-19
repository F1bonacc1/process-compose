package cmd

import (
	"errors"
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
	Args: func(cmd *cobra.Command, args []string) error {
		if *pcFlags.Namespace != "" {
			return nil
		}
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			return errors.New("requires at least one process name argument when --namespace is not set")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		if *pcFlags.Namespace != "" {
			processes := strings.Split(name, ",")
			states, err := getClient().GetRemoteProcessesState()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to list processes")
			}
			for _, state := range states.States {
				if state.Namespace == *pcFlags.Namespace {
					processes = append(processes, state.Name)
				}
			}
			if len(processes) == 0 {
				log.Fatal().Msgf("No processes in namespace %s", *pcFlags.Namespace)
			}
			name = strings.Join(processes, ",")
		} else {
			// validate nonempty process list
			if name == "" {
				log.Fatal().Msg("No processes specified")
			}
		}

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
	logsCmd.Flags().StringVarP(pcFlags.Namespace, "namespace", "N", *pcFlags.Namespace, "Logs all the processes in the given namespace")
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
