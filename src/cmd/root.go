package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	port      int
	isTui     bool
	opts      *loader.LoaderOptions
	pcAddress string
	logPath   string

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "process-compose",
		Short: "Processes scheduler and orchestrator",
		Run:   run,
	}
)

const (
	defaultPortNum   = 8080
	portEnvVarName   = "PC_PORT_NUM"
	tuiEnvVarName    = "PC_DISABLE_TUI"
	configEnvVarName = "PC_CONFIG_FILES"
)

func run(cmd *cobra.Command, args []string) {

	if !cmd.Flags().Changed("tui") {
		isTui = getTuiDefault()
	}
	runner := getProjectRunner([]string{}, false)
	api.StartHttpServer(!isTui, port, runner)
	runProject(runner)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	file, err := setupLogger()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = file.Close()
	}()

	log.Info().Msgf("Process Compose %s", config.Version)
	err = rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	opts = &loader.LoaderOptions{
		FileNames: []string{},
	}

	rootCmd.Flags().BoolVarP(&isTui, "tui", "t", true, "disable tui (-t=false) (env: "+tuiEnvVarName+")")
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", getPortDefault(), "port number (env: "+portEnvVarName+")")
	rootCmd.Flags().StringArrayVarP(&opts.FileNames, "config", "f", getConfigDefault(), "path to config files to load (env: "+configEnvVarName+")")
	rootCmd.Flags().StringVarP(&logPath, "logFile", "", config.GetLogFilePath(), "Specify the log file path (env: "+config.LogPathEnvVarName+")")
}

func getTuiDefault() bool {
	_, found := os.LookupEnv(tuiEnvVarName)
	return !found
}

func getPortDefault() int {
	val, found := os.LookupEnv(portEnvVarName)
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
	val, found := os.LookupEnv(configEnvVarName)
	if found {
		return strings.Split(val, ",")
	}
	return []string{}
}

func logFatal(err error, format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Printf(": %v\n", err)
	log.Fatal().Err(err).Msgf(format, args...)
}

func setupLogger() (*os.File, error) {
	file, err := os.OpenFile(logPath, config.LogFileFlags, config.LogFileMode)
	if err != nil {
		return nil, err
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        file,
		TimeFormat: "06-01-02 15:04:05.000",
	})
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	return file, nil
}
