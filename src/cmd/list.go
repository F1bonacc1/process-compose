package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/client"

	"github.com/spf13/cobra"
)

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List available processes",
	Aliases: []string{"ls"},
	Run: func(cmd *cobra.Command, args []string) {
		processNames, err := client.GetProcessesName(pcAddress, port)
		if err != nil {
			logFatal(err, "failed to list processes")
		}
		for _, proc := range processNames {
			fmt.Println(proc)
		}
	},
}

func init() {
	processCmd.AddCommand(listCmd)
}
