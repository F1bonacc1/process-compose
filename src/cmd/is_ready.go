package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	// default const polling inteval
	defaultPollingInterval = 1 * time.Second
)

// `projectIsReadyCmd` represents the `project is-ready` command
var projectIsReadyCmd = &cobra.Command{
	Use:   "is-ready",
	Short: "Check if Process Compose project is ready (or wait for it to be ready)",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()

		if *pcFlags.WaitReady {
			start := time.Now()
			for {
				notReady := checkProjectReady(client)
				if notReady == nil {
					break
				}
				log.Warn().Msgf("%s", notReady.Error())
				time.Sleep(defaultPollingInterval)
			}

			elapsed := time.Since(start)
			log.Info().Msgf("Project is ready after waiting for %s", elapsed.Round(10*time.Millisecond))
		} else {
			notReady := checkProjectReady(client)
			if notReady != nil {
				log.Fatal().Err(notReady).Send()
			}
		}
	},
}

type ProcessStateReason struct {
	State  types.ProcessState
	Reason string
}

func (p *ProcessStateReason) String() string {
	if p.Reason == "" {
		return p.State.Name
	} else {
		return fmt.Sprintf("%s (%s)", p.State.Name, p.Reason)
	}
}

func checkProjectReady(client *client.PcClient) error {
	states, err := client.GetProcessesState()
	if err != nil {
		return err
	}

	var notReady []ProcessStateReason = make([]ProcessStateReason, 0, len(states.States))
	for _, state := range states.States {
		isReady, reason := state.IsReadyReason()
		stateReason := ProcessStateReason{
			State:  state,
			Reason: reason,
		}
		if !isReady {
			notReady = append(notReady, stateReason)
		}
	}

	if len(notReady) == 0 {
		log.Info().Msgf("%d processes are ready", len(states.States))
		return nil
	} else {
		var explanations []string = make([]string, 0, len(notReady))
		for _, process := range notReady {
			explanations = append(explanations, process.String())
		}
		return fmt.Errorf("Processes are not ready:\n• %s", strings.Join(explanations, "\n• "))
	}
}

// processIsReadyCmd represents the `process is-ready` command
var processIsReadyCmd = &cobra.Command{
	Use:   "is-ready [PROCESS_NAME]",
	Short: "Check if a specific process is ready (or wait for it to be ready).",
	Long:  "Check if a specific process is ready (or wait for it to be ready). If process does not exist or disabled, an error is returned. If process was removed or disabled after check started, it will not be detected.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		processName := args[0]
		client := getClient()
		if *pcFlags.WaitReady {
			start := time.Now()
			for {
				cannotBeReady, err := checkProcessReady(client, processName)
				if cannotBeReady {
					log.Fatal().Err(err).Send()
				}
				if err == nil {
					break
				}
				log.Info().Msgf("%s", err.Error())
				time.Sleep(defaultPollingInterval)
			}

			elapsed := time.Since(start)
			log.Info().Msgf("Process '%s' is ready after waiting for %s", processName, elapsed.Round(10*time.Millisecond))
		} else {
			_, err := checkProcessReady(client, processName)
			if err != nil {
				log.Fatal().Err(err).Send()
			}
		}
	},
}

// Return true if process cannot be ready (does not exist in configuration, disabled).
// If true error not nil.
func checkProcessReady(client *client.PcClient, processName string) (bool, error) {
	state, doesNotExist, err := client.GetProcessState(processName)

	if err != nil {
		if doesNotExist {
			return true, err
		}

		return false, err
	}

	err = state.CannotBeReady()
	if err != nil {
		return true, err
	}

	isReady, reason := state.IsReadyReason()
	if !isReady {
		stateReason := ProcessStateReason{
			State:  *state,
			Reason: reason,
		}
		return false, fmt.Errorf("Process '%s' is not ready: %s", processName, stateReason.String())
	}

	log.Info().Msgf("Process '%s' is ready", processName)
	return false, nil
}

func init() {
	projectCmd.AddCommand(projectIsReadyCmd)
	projectIsReadyCmd.Flags().BoolVar(pcFlags.WaitReady, "wait", false, "Wait for the project to be ready instead of exiting with an error")

	processCmd.AddCommand(processIsReadyCmd)
	processIsReadyCmd.Flags().BoolVar(pcFlags.WaitReady, "wait", false, "Wait for the process to be ready instead of exiting with an error")
}
