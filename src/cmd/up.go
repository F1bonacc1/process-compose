package cmd

import (
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/spf13/cobra"
)

var (
	noDeps = false
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [PROCESS...]",
	Short: "Run process compose project",
	Long: `Run all the process compose processes.
If one or more process names are passed as arguments,
will start them and their dependencies only`,
	Run: func(cmd *cobra.Command, args []string) {
		if !cmd.Flags().Changed("tui") {
			isTui = getTuiDefault()
		}
		runner := getProjectRunner([]string{}, false)
		api.StartHttpServer(!isTui, port, runner)
		runProject(runner)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	upCmd.Flags().BoolVarP(&isTui, "tui", "t", true, "disable tui (-t=false) (env: PC_DISABLE_TUI)")
	upCmd.Flags().BoolVarP(&noDeps, "no-deps", "", false, "don't start dependent processes")
	upCmd.Flags().StringArrayVarP(&opts.FileNames, "config", "f", getConfigDefault(), "path to config files to load (env: PC_CONFIG_FILES)")

}
