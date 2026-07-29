package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var namespaceCmd = &cobra.Command{
	Use:         "namespace",
	Short:       "Perform operations on a namespace (start, stop, restart, list)",
	Aliases:     []string{"ns"},
	Annotations: map[string]string{clientModeAnnotation: "true"},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(0)
		}
	},
}

var namespaceStartCmd = &cobra.Command{
	Use:   "start [namespace]",
	Short: "Start all processes in a namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		client := getClient()
		err := client.StartNamespace(namespace)
		if err != nil {
			fmt.Printf("Failed to start namespace '%s': %v\n", namespace, err)
			os.Exit(1)
		}
		fmt.Printf("Namespace '%s' started successfully\n", namespace)
	},
}

var namespaceStopCmd = &cobra.Command{
	Use:   "stop [namespace]",
	Short: "Stop all processes in a namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		client := getClient()
		err := client.StopNamespace(namespace)
		if err != nil {
			fmt.Printf("Failed to stop namespace '%s': %v\n", namespace, err)
			os.Exit(1)
		}
		fmt.Printf("Namespace '%s' stopped successfully\n", namespace)
	},
}

var namespaceRestartCmd = &cobra.Command{
	Use:   "restart [namespace]",
	Short: "Restart all processes in a namespace",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		namespace := args[0]
		client := getClient()
		err := client.RestartNamespace(namespace)
		if err != nil {
			fmt.Printf("Failed to restart namespace '%s': %v\n", namespace, err)
			os.Exit(1)
		}
		fmt.Printf("Namespace '%s' restarted successfully\n", namespace)
	},
}

var namespaceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all available namespaces",
	Aliases: []string{"ls"},
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := getClient()
		namespaces, err := client.GetNamespaces()
		if err != nil {
			fmt.Printf("Failed to list namespaces: %v\n", err)
			os.Exit(1)
		}
		for _, ns := range namespaces {
			fmt.Println(ns)
		}
	},
}

func init() {
	namespaceCmd.AddCommand(namespaceStartCmd)
	namespaceCmd.AddCommand(namespaceStopCmd)
	namespaceCmd.AddCommand(namespaceRestartCmd)
	namespaceCmd.AddCommand(namespaceListCmd)
	rootCmd.AddCommand(namespaceCmd)
}
