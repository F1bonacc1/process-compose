package cmd

import (
	"github.com/spf13/cobra"
)

// attachCmd represents the attach command
var attachCmd = &cobra.Command{
	Use:   "attach",
	Short: "Attach the Process Compose TUI Remotely to a Running Process Compose Server",
	Run: func(cmd *cobra.Command, args []string) {
		startTui(getClient())
	},
}

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.Flags().VarP(refreshRateFlag{pcFlags.RefreshRate}, "ref-rate", "r", "TUI refresh rate in seconds or as a Go duration string (e.g. 1s)")
	attachCmd.Flags().StringVarP(pcFlags.Address, "address", "a", *pcFlags.Address, "address of the target process compose server")
	attachCmd.Flags().IntVarP(pcFlags.LogLength, "log-length", "l", *pcFlags.LogLength, "log length to display in TUI")
	attachCmd.Flags().AddFlag(commonFlags.Lookup(flagReverse))
	attachCmd.Flags().AddFlag(commonFlags.Lookup(flagSort))
	attachCmd.Flags().AddFlag(commonFlags.Lookup(flagTheme))

}
