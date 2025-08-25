package cmd

import (
    "fmt"
    "github.com/rs/zerolog/log"
    "github.com/spf13/cobra"
    "os"
    "sort"
)

var namespaceDisableCmd = &cobra.Command{
    Use:   "disable [NAMESPACE]",
    Short: "Disable all processes in a namespace",
    Args:  cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        ns := args[0]
        statusMap, err := getClient().DisableNamespace(ns)
        if err != nil {
            log.Fatal().Err(err).Msgf("failed to disable namespace %s", ns)
        }

        names := make([]string, 0, len(statusMap))
        for name := range statusMap {
            names = append(names, name)
        }
        sort.Strings(names)

        if stopVerboseOutput {
            out, code := formatActionVerbose(statusMap, names, "disable")
            fmt.Print(out)
            os.Exit(code)
        } else {
            out, code := formatActionConcise(statusMap, names, "disable")
            fmt.Print(out)
            os.Exit(code)
        }
    },
}

func init() {
    namespaceCmd.AddCommand(namespaceDisableCmd)
    namespaceDisableCmd.Flags().BoolVarP(&stopVerboseOutput, "verbose", "v", stopVerboseOutput, "verbose output")
}

