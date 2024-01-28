package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/admitter"
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path"
	"time"
)

var (
	opts    *loader.LoaderOptions
	logFile *os.File

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "process-compose",
		Short: "Processes scheduler and orchestrator",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			logFile = setupLogger()
			log.Info().Msgf("Process Compose %s", config.Version)
		},
		RunE: run,
	}
)

func run(cmd *cobra.Command, args []string) error {
	defer func() {
		_ = logFile.Close()
	}()
	runner := getProjectRunner([]string{}, false, "", []string{})
	startHttpServerIfEnabled(!*pcFlags.Headless, runner)
	runProject(runner)
	return nil
}

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

	nsAdmitter := &admitter.NamespaceAdmitter{}
	opts.AddAdmitter(nsAdmitter)

	rootCmd.Flags().BoolVarP(pcFlags.Headless, "tui", "t", *pcFlags.Headless, "enable TUI (-t=false) (env: "+config.EnvVarNameTui+")")
	rootCmd.PersistentFlags().BoolVar(pcFlags.KeepTuiOn, "keep-tui", *pcFlags.KeepTuiOn, "keep TUI running even after all processes exit")
	rootCmd.PersistentFlags().BoolVar(pcFlags.NoServer, "no-server", *pcFlags.NoServer, "disable HTTP server (env: "+config.EnvVarNameNoServer+")")
	rootCmd.Flags().BoolVarP(pcFlags.HideDisabled, "hide-disabled", "d", *pcFlags.HideDisabled, "hide disabled processes")
	rootCmd.Flags().IntVarP(pcFlags.RefreshRate, "ref-rate", "r", *pcFlags.RefreshRate, "TUI refresh rate in seconds")
	rootCmd.PersistentFlags().IntVarP(pcFlags.PortNum, "port", "p", *pcFlags.PortNum, "port number (env: "+config.EnvVarNamePort+")")
	rootCmd.Flags().StringArrayVarP(&opts.FileNames, "config", "f", config.GetConfigDefault(), "path to config files to load (env: "+config.EnvVarNameConfig+")")
	rootCmd.Flags().StringArrayVarP(&nsAdmitter.EnabledNamespaces, "namespace", "n", nil, "run only specified namespaces (default all)")
	rootCmd.PersistentFlags().StringVarP(pcFlags.LogFile, "log-file", "L", *pcFlags.LogFile, "Specify the log file path (env: "+config.LogPathEnvVarName+")")
	rootCmd.Flags().AddFlag(commonFlags.Lookup("reverse"))
	rootCmd.Flags().AddFlag(commonFlags.Lookup("sort"))
}

func logFatal(err error, format string, args ...interface{}) {
	fmt.Printf(format, args...)
	fmt.Printf(": %v\n", err)
	log.Fatal().Err(err).Msgf(format, args...)
}

func setupLogger() *os.File {
	dirName := path.Dir(*pcFlags.LogFile)
	if err := os.MkdirAll(dirName, 0700); err != nil && !os.IsExist(err) {
		fmt.Printf("Failed to create log directory: %s - %v\n", dirName, err)
		os.Exit(1)
	}
	file, err := os.OpenFile(*pcFlags.LogFile, config.LogFileFlags, config.LogFileMode)
	if err != nil {
		logFatal(err, "Failed to open log file: %s", *pcFlags.LogFile)
	}
	writers := []io.Writer{
		&zerolog.FilteredLevelWriter{

			Level: zerolog.DebugLevel,
			Writer: zerolog.LevelWriterAdapter{Writer: zerolog.ConsoleWriter{
				Out:        file,
				TimeFormat: "06-01-02 15:04:05.000",
			}},
		},
		&zerolog.FilteredLevelWriter{

			Level: zerolog.FatalLevel,
			Writer: zerolog.LevelWriterAdapter{Writer: zerolog.ConsoleWriter{
				Out:        os.Stderr,
				TimeFormat: "06-01-02 15:04:05.000",
			}},
		},
	}
	writer := zerolog.MultiLevelWriter(writers...)

	log.Logger = zerolog.New(writer).With().Timestamp().Logger()
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	return file
}

func startHttpServerIfEnabled(useLogger bool, runner *app.ProjectRunner) {
	if !*pcFlags.NoServer {
		api.StartHttpServer(useLogger, *pcFlags.PortNum, runner)
	}
}
