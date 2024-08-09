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
	verboseOutput = false
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
		missing := findMissingProcesses(args, stopped)
		if len(missing) != 0 && !verboseOutput {
			fmt.Printf("Successfully stopped some processes but encountered failures for: %s\n",
				"'"+strings.Join(missing, `', '`)+`'`)
			os.Exit(1)
		}
		for _, name := range stopped {
			fmt.Printf("%s Successfully stopped %s\n", color.GreenString("✓"), name)
		}
		if len(missing) > 0 {
			log.Error().Msgf("failed to stop some processes: %v", missing)
			for _, name := range missing {
				fmt.Printf("%s Failed to stop %s\n", color.RedString("✘"), name)
			}
			os.Exit(1)
		}
	},
}

func findMissingProcesses(requested, stopped []string) []string {
	bMap := make(map[string]bool)
	for _, str := range stopped {
		bMap[str] = true
	}

	missing := []string{}
	for _, str := range requested {
		if !bMap[str] {
			missing = append(missing, str)
		}
	}
	return missing
}

func init() {
	processCmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVarP(&verboseOutput, "verbose", "v", verboseOutput, "verbose output")
}
