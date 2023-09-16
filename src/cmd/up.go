package cmd

import (
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/spf13/cobra"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [PROCESS...]",
	Short: "Run process compose project",
	Long: `Run all the process compose processes.
If one or more process names are passed as arguments,
will start them and their dependencies only`,
	Run: func(cmd *cobra.Command, args []string) {
		runner := getProjectRunner(args, *pcFlags.NoDependencies)
		api.StartHttpServer(!*pcFlags.Headless, *pcFlags.PortNum, runner)
		runProject(runner)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	upCmd.Flags().BoolVarP(pcFlags.Headless, "tui", "t", *pcFlags.Headless, "disable tui (-t=false) (env: "+config.TuiEnvVarName+")")
	upCmd.Flags().IntVarP(pcFlags.RefreshRate, "ref-rate", "r", *pcFlags.RefreshRate, "tui refresh rate in seconds")
	upCmd.Flags().BoolVarP(pcFlags.NoDependencies, "no-deps", "", *pcFlags.NoDependencies, "don't start dependent processes")
	upCmd.Flags().StringArrayVarP(&opts.FileNames, "config", "f", config.GetConfigDefault(), "path to config files to load (env: "+config.ConfigEnvVarName+")")

}
