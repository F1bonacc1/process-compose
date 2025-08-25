package cmd

import (
    "github.com/spf13/cobra"
)

// namespaceCmd represents the namespace command group
var namespaceCmd = &cobra.Command{
    Use:   "namespace",
    Short: "Execute operations on namespaces",
}

func init() {
    rootCmd.AddCommand(namespaceCmd)
}

