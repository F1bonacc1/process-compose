package cmd

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var (
	stopVerboseOutput = false
)

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop [PROCESS...]",
	Short: "Stop running processes",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		stopped, err := getClient().StopProcesses(args)
		if err != nil {
			log.Fatal().Err(err).Msgf("failed to stop processes %v", args)
		}

		if stopVerboseOutput {
			status, exitCode := prepareVerboseOutput(stopped, args)
			fmt.Print(status)
			os.Exit(exitCode)
		} else {
			status, exitCode := prepareConciseOutput(stopped, args)
			fmt.Print(status)
			os.Exit(exitCode)
		}
	},
}

func prepareVerboseOutput(stopped map[string]string, processes []string) (output string, exitCode int) {
	for _, name := range processes {
		status, ok := stopped[name]
		if !ok {
			log.Error().Msgf("Process %s does not exist", name)
			output += fmt.Sprintf("%s Unknown status for process %s\n", color.RedString("✘"), name)
			exitCode = 1
			continue
		}
		if status == "ok" {
			output += fmt.Sprintf("%s Successfully stopped %s\n", color.GreenString("✓"), name)
		} else {
			output += fmt.Sprintf("%s Failed to stop %s: %s\n", color.RedString("✘"), name, status)
			exitCode = 1
		}
	}
	return output, exitCode
}

func prepareConciseOutput(stopped map[string]string, processes []string) (string, int) {
	failed := make([]string, 0)
	pass := make([]string, 0)
	for _, name := range processes {
		if stopped[name] != "ok" {
			failed = append(failed, name)
		} else {
			pass = append(pass, name)
		}
	}
	if len(pass) == 0 && len(failed) != 0 {
		return fmt.Sprintf("Failed to stop: %s\n",
			"'"+strings.Join(failed, `', '`)+`'`), 1
	} else if len(failed) != 0 && len(pass) != 0 {
		return fmt.Sprintf("Successfully stopped %s but encountered failures for: %s\n",
			"'"+strings.Join(pass, `', '`)+`'`,
			"'"+strings.Join(failed, `', '`)+`'`), 1
	}
	return fmt.Sprintf("Successfully stopped: %s\n", "'"+strings.Join(pass, `', '`)+`'`), 0
}

func init() {
	processCmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVarP(&stopVerboseOutput, "verbose", "v", stopVerboseOutput, "verbose output")
}
