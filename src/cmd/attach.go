/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/tui"
	"github.com/spf13/cobra"
)

var (
	logLength = 1000
)

// attachCmd represents the attach command
var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach the Process Compose TUI Remotely to a Running Process Compose Server",
	Run: func(cmd *cobra.Command, args []string) {
		pcClient := client.NewClient(pcAddress, port, logLength)
		tui.SetupTui(pcClient)
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.Flags().StringVarP(&pcAddress, "address", "a", "localhost", "address of a running process compose server")
	attachCmd.Flags().IntVarP(&logLength, "log-length", "l", logLength, "log length to display in TUI")
}
