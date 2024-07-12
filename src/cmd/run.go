package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// runCmd represents the up command
var runCmd = &cobra.Command{
	Use:   "run PROCESS [flags] -- [process_args]",
	Short: "Run PROCESS in the foreground, and its dependencies in the background",
	Long: `Run selected process with std(in|out|err) attached, while other processes run in the background.
Command line arguments, provided after --, are passed to the PROCESS.`,
	Args: cobra.MinimumNArgs(1),
	// Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		*pcFlags.IsTuiEnabled = false

		processName := args[0]

		if len(args) > 1 {
			argsLenAtDash := cmd.ArgsLenAtDash()
			if argsLenAtDash != 1 {
				message := "Extra positional arguments provided! To pass args to PROCESS, separate them from process-compose arguments with: --"
				fmt.Println(message)
				os.Exit(1)
			}
			args = args[argsLenAtDash:]
		} else {
			// Clease args as they will contain the processName
			args = []string{}
		}

		runner := getProjectRunner(
			[]string{processName},
			*pcFlags.NoDependencies,
			processName,
			args,
		)

		startHttpServerIfEnabled(false, runner)
		err := runProject(runner)
		handleErrorAndExit(err)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().BoolVarP(pcFlags.NoDependencies, "no-deps", "", *pcFlags.NoDependencies, "don't start dependent processes")
	runCmd.Flags().AddFlag(rootCmd.Flags().Lookup("config"))

}
