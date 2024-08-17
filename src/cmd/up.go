package cmd

import (
	"github.com/f1bonacc1/process-compose/src/admitter"
	"github.com/spf13/cobra"
	"runtime"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [PROCESS...]",
	Short: "Run process compose project",
	Long: `Run all the process compose processes.
If one or more process names are passed as arguments,
will start them and their dependencies only`,
	Run: func(cmd *cobra.Command, args []string) {
		runProjectCmd(args)
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	nsAdmitter := &admitter.NamespaceAdmitter{}
	opts.AddAdmitter(nsAdmitter)

	upCmd.Flags().BoolVarP(pcFlags.NoDependencies, "no-deps", "", *pcFlags.NoDependencies, "don't start dependent processes")
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("namespace"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("config"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("env"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("ref-rate"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("tui"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("hide-disabled"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("disable-dotenv"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("keep-tui"))
	upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("keep-project"))
	upCmd.Flags().AddFlag(commonFlags.Lookup(flagReverse))
	upCmd.Flags().AddFlag(commonFlags.Lookup(flagSort))
	upCmd.Flags().AddFlag(commonFlags.Lookup(flagTheme))

	if runtime.GOOS != "windows" {
		upCmd.Flags().AddFlag(rootCmd.Flags().Lookup("detached"))
	}
	_ = upCmd.Flags().MarkDeprecated("keep-tui", "use --keep-project instead")

}
