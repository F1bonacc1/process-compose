package cmd

import (
	"github.com/f1bonacc1/process-compose/src/admitter"
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
		runner := getProjectRunner(args, *pcFlags.NoDependencies, "", []string{})
		startHttpServerIfEnabled(!*pcFlags.Headless, runner)
		runProject(runner)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	nsAdmitter := &admitter.NamespaceAdmitter{}
	opts.AddAdmitter(nsAdmitter)

	upCmd.Flags().BoolVarP(pcFlags.NoDependencies, "no-deps", "", *pcFlags.NoDependencies, "don't start dependent processes")
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("namespace"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("config"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("ref-rate"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("tui"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("hide-disabled"))
	upCmd.Flags().AddFlag(commonFlags.Lookup("reverse"))
	upCmd.Flags().AddFlag(commonFlags.Lookup("sort"))

}
