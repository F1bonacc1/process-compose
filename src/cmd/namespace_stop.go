package cmd

import (
    "fmt"
    "github.com/rs/zerolog/log"
    "github.com/spf13/cobra"
    "os"
)

// namespaceStopCmd stops all processes in a namespace
var namespaceStopCmd = &cobra.Command{
    Use:   "stop [NAMESPACE]",
    Short: "Stop all processes in a namespace",
    Args:  cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        ns := args[0]

        stopped, err := getClient().StopNamespace(ns)
        if err != nil {
            log.Fatal().Err(err).Msgf("failed to stop namespace %s", ns)
        }

        // Build ordered process list from response keys for stable output
        processes := make([]string, 0, len(stopped))
        for name := range stopped {
            processes = append(processes, name)
        }

        if stopVerboseOutput {
            status, exitCode := prepareVerboseOutput(stopped, processes)
            fmt.Print(status)
            os.Exit(exitCode)
        } else {
            status, exitCode := prepareConciseOutput(stopped, processes)
            fmt.Print(status)
            os.Exit(exitCode)
        }
    },
}

func init() {
    namespaceCmd.AddCommand(namespaceStopCmd)
    namespaceStopCmd.Flags().BoolVarP(&stopVerboseOutput, "verbose", "v", stopVerboseOutput, "verbose output")
}
