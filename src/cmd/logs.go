package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
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
			processes := []string{}
			if name != "" {
				processes = strings.Split(name, ",")
			}
			states, err := getClient().GetRemoteProcessesState()
			if err != nil {
				log.Fatal().Err(err).Msg("failed to list processes")
			}
			for _, state := range states.States {
				if state.Namespace.Contains(*pcFlags.Namespace) {
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

		notifyInteractiveProcesses(name)

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

// notifyInteractiveProcesses prints a notice for any requested process that is
// interactive. Interactive processes use a PTY and render their output in the
// terminal pane; that output is not captured into the line log buffer, so
// `process logs` has nothing to show for them. Emitting an explicit notice
// keeps an empty result from looking like a bug (issue #511). It does not skip
// streaming, so any error/probe lines still captured for the process come
// through as before.
func notifyInteractiveProcesses(names string) {
	c := getClient()
	for pName := range strings.SplitSeq(names, ",") {
		pName = strings.TrimSpace(pName)
		if pName == "" {
			continue
		}
		info, err := c.GetProcessInfo(pName)
		if err != nil {
			// Couldn't fetch the process config (e.g. unknown process);
			// let the log subscription surface that error as before.
			continue
		}
		if info.IsInteractive {
			// Write to stderr (not the zerolog logger, which the CLI routes to
			// a log file): the user running `process logs` needs to see this on
			// their terminal, while stdout stays clean for piped log output.
			fmt.Fprintf(os.Stderr, "process %q is interactive: 'process logs' has nothing to display for it\n", pName)
		}
	}
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
