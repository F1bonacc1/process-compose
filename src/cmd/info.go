package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/spf13/cobra"
)

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Print configuration info",
	Run: func(cmd *cobra.Command, args []string) {
		printInfo()
	},
}

func printInfo() {
	format := "%-15s %s\n"
	fmt.Println("Process Compose")
	fmt.Printf(format, "Logs:", config.GetLogFilePath())

	path := config.GetShortCutsPath()
	if len(path) > 0 {
		fmt.Printf(format, "Shortcuts:", config.GetShortCutsPath())
	}
	themes := config.GetThemesPath()
	if len(themes) > 0 {
		fmt.Printf(format, "Theme:", config.GetThemesPath())
	}

}

func init() {
	rootCmd.AddCommand(infoCmd)
}
