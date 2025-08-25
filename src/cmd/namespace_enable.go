package cmd

import (
    "fmt"
    "github.com/rs/zerolog/log"
    "github.com/spf13/cobra"
    "os"
    "sort"
    "strings"
)

var namespaceEnableCmd = &cobra.Command{
    Use:   "enable [NAMESPACE]",
    Short: "Enable all processes in a namespace",
    Args:  cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        ns := args[0]
        statusMap, err := getClient().EnableNamespace(ns)
        if err != nil {
            log.Fatal().Err(err).Msgf("failed to enable namespace %s", ns)
        }

        names := make([]string, 0, len(statusMap))
        for name := range statusMap {
            names = append(names, name)
        }
        sort.Strings(names)

        if stopVerboseOutput {
            out, code := formatActionVerbose(statusMap, names, "enable")
            fmt.Print(out)
            os.Exit(code)
        } else {
            out, code := formatActionConcise(statusMap, names, "enable")
            fmt.Print(out)
            os.Exit(code)
        }
    },
}

func init() {
    namespaceCmd.AddCommand(namespaceEnableCmd)
    namespaceEnableCmd.Flags().BoolVarP(&stopVerboseOutput, "verbose", "v", stopVerboseOutput, "verbose output")
}

func formatActionVerbose(status map[string]string, processes []string, action string) (string, int) {
    prefixOk := map[string]string{"disable": "Disabled", "enable": "Enabled"}[action]
    out := ""
    code := 0
    for _, name := range processes {
        s, ok := status[name]
        if !ok {
            out += fmt.Sprintf("✘ Unknown status for process %s\n", name)
            code = 1
            continue
        }
        if s == "ok" {
            out += fmt.Sprintf("✓ %s %s\n", prefixOk, name)
        } else {
            out += fmt.Sprintf("✘ Failed to %s %s: %s\n", action, name, s)
            code = 1
        }
    }
    return out, code
}

func formatActionConcise(status map[string]string, processes []string, action string) (string, int) {
    ok := make([]string, 0)
    fail := make([]string, 0)
    for _, name := range processes {
        if status[name] == "ok" {
            ok = append(ok, name)
        } else {
            fail = append(fail, name)
        }
    }
    prefixOk := map[string]string{"disable": "Disabled", "enable": "Enabled"}[action]
    if len(ok) == 0 && len(fail) > 0 {
        return fmt.Sprintf("Failed to %s: '%s'\n", action, strings.Join(fail, "', '")), 1
    }
    if len(fail) > 0 {
        return fmt.Sprintf("%s '%s' but encountered failures for: '%s'\n", prefixOk, strings.Join(ok, "', '"), strings.Join(fail, "', '")), 1
    }
    return fmt.Sprintf("%s: '%s'\n", prefixOk, strings.Join(ok, "', '")), 0
}
