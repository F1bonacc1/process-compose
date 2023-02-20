package cmd

import (
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
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
			if !cmd.Flags().Changed("tui") {
				isTui = getTuiDefault()
			}
			api.StartHttpServer(!isTui, port)
			runProject([]string{}, false)
		},
	}
)

const (
	defaultPortNum = 8080
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
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", getPortDefault(), "port number")
	rootCmd.PersistentFlags().StringArrayVarP(&opts.FileNames, "config", "f", getConfigDefault(), "path to config files to load")
}

func getTuiDefault() bool {
	_, found := os.LookupEnv("PC_DISABLE_TUI")
	return !found
}

func getPortDefault() int {
	val, found := os.LookupEnv("PC_PORT_NUM")
	if found {
		port, err := strconv.Atoi(val)
		if err != nil {
			log.Fatal().Msgf("Invalid port number: %s", val)
			return defaultPortNum
		}
		return port
	}
	return defaultPortNum
}

func getConfigDefault() []string {
	val, found := os.LookupEnv("PC_CONFIG_FILES")
	if found {
		return strings.Split(val, ",")
	}
	return []string{}
}
