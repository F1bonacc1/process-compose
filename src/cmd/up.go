package cmd

import (
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/app"

	"github.com/spf13/cobra"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [PROCESS]",
	Short: "Run process compose project",
	Long: `Run all the process compose processes.
If one or more process names are passed as arguments, will start only them`,
	Run: func(cmd *cobra.Command, args []string) {
		isDefConfigPath := !cmd.Flags().Changed("config")
		api.StartHttpServer(!isTui, port)
		runProject(isDefConfigPath, args)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	upCmd.Flags().StringVarP(&fileName, "config", "f", app.DefaultFileNames[0], "path to config file to load")
	upCmd.Flags().BoolVarP(&isTui, "tui", "t", true, "disable tui (-t=false)")
}
