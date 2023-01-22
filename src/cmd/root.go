package cmd

import (
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/spf13/cobra"
	"os"
)

var (
	port  int
	isTui bool
	opts  *loader.LoaderOptions

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "process-compose",
		Short: "Processes scheduler and orchestrator",
		Run: func(cmd *cobra.Command, args []string) {
			api.StartHttpServer(!isTui, port)
			runProject([]string{}, false)
		},
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	opts = &loader.LoaderOptions{
		FileNames: []string{},
	}
	rootCmd.Flags().BoolVarP(&isTui, "tui", "t", true, "disable tui (-t=false)")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 8080, "port number")
	rootCmd.PersistentFlags().StringArrayVarP(&opts.FileNames, "config", "f", []string{}, "path to config files to load")
}
