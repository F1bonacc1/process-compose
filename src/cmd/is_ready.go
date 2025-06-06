package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

// `isReadyCmd` represents the `project is-ready` command
var isReadyCmd = &cobra.Command{
	Use:   "is-ready",
	Short: "Check if Process Compose project is ready (or wait for it to be ready)",
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()

		if *pcFlags.WaitReady {
			start := time.Now()
			for {
				notReady := checkReady(client)
				if notReady == nil {
					break
				}
				log.Warn().Msgf("%s", notReady.Error())
				time.Sleep(1 * time.Second)
			}

			elapsed := time.Since(start)
			log.Info().Msgf("Project is ready after waiting for %s", elapsed.Round(10*time.Millisecond))
		} else {
			notReady := checkReady(client)
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

func checkReady(client *client.PcClient) error {
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

func init() {
	projectCmd.AddCommand(isReadyCmd)
	isReadyCmd.Flags().BoolVar(pcFlags.WaitReady, "wait", false, "Wait for the project to be ready instead of exiting with an error")
}
