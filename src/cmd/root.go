package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/admitter"
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/client"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path"
	"runtime"
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
			if isUnixSocketMode(cmd) {
				*pcFlags.IsUnixSocket = true
			}
			pcFlags.PcThemeChanged = cmd.Flags().Changed(flagTheme)
			pcFlags.SortColumnChanged = cmd.Flags().Changed(flagSort)
		},
		RunE: run,
	}
)

func run(cmd *cobra.Command, args []string) error {
	defer func() {
		_ = logFile.Close()
	}()
	runner := getProjectRunner([]string{}, false, "", []string{})
	startHttpServerIfEnabled(!*pcFlags.IsTuiEnabled, runner)
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

	rootCmd.Flags().BoolVarP(pcFlags.IsTuiEnabled, "tui", "t", *pcFlags.IsTuiEnabled, "enable TUI (disable with -t=false) (env: "+config.EnvVarNameTui+")")
	rootCmd.PersistentFlags().BoolVar(pcFlags.KeepTuiOn, "keep-tui", *pcFlags.KeepTuiOn, "keep TUI running even after all processes exit")
	rootCmd.PersistentFlags().BoolVar(pcFlags.NoServer, "no-server", *pcFlags.NoServer, "disable HTTP server (env: "+config.EnvVarNameNoServer+")")
	rootCmd.PersistentFlags().BoolVar(pcFlags.IsOrderedShutDown, "ordered-shutdown", *pcFlags.IsOrderedShutDown, "shut down processes in reverse dependency order")
	rootCmd.Flags().BoolVarP(pcFlags.HideDisabled, "hide-disabled", "d", *pcFlags.HideDisabled, "hide disabled processes")
	rootCmd.Flags().IntVarP(pcFlags.RefreshRate, "ref-rate", "r", *pcFlags.RefreshRate, "TUI refresh rate in seconds")
	rootCmd.PersistentFlags().IntVarP(pcFlags.PortNum, "port", "p", *pcFlags.PortNum, "port number (env: "+config.EnvVarNamePort+")")
	rootCmd.Flags().StringArrayVarP(&opts.FileNames, "config", "f", config.GetConfigDefault(), "path to config files to load (env: "+config.EnvVarNameConfig+")")
	rootCmd.Flags().StringArrayVarP(&nsAdmitter.EnabledNamespaces, "namespace", "n", nil, "run only specified namespaces (default all)")
	rootCmd.PersistentFlags().StringVarP(pcFlags.LogFile, "log-file", "L", *pcFlags.LogFile, "Specify the log file path (env: "+config.LogPathEnvVarName+")")
	rootCmd.PersistentFlags().BoolVar(pcFlags.IsReadOnlyMode, "read-only", *pcFlags.IsReadOnlyMode, "enable read-only mode (env: "+config.EnvVarReadOnlyMode+")")
	rootCmd.Flags().AddFlag(commonFlags.Lookup(flagReverse))
	rootCmd.Flags().AddFlag(commonFlags.Lookup(flagSort))
	rootCmd.Flags().AddFlag(commonFlags.Lookup(flagTheme))

	if runtime.GOOS != "windows" {
		//rootCmd.Flags().BoolVarP(pcFlags.IsDetached, "detached", "D", *pcFlags.IsDetached, "run process-compose in detached mode")
		rootCmd.PersistentFlags().StringVarP(pcFlags.UnixSocketPath, "unix-socket", "u", config.GetUnixSocketPath(), "path to unix socket (env: "+config.EnvVarUnixSocketPath+")")
		rootCmd.PersistentFlags().BoolVarP(pcFlags.IsUnixSocket, "use-uds", "U", *pcFlags.IsUnixSocket, "use unix domain sockets instead of tcp")
	}
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

	// add caller only in debug mode
	if os.Getenv("PC_DEBUG") != "" {
		log.Logger = zerolog.New(writer).With().Timestamp().Caller().Logger()
	} else {
		log.Logger = zerolog.New(writer).With().Timestamp().Logger()
	}
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	return file
}

func startHttpServerIfEnabled(useLogger bool, runner *app.ProjectRunner) {
	if !*pcFlags.NoServer {
		if *pcFlags.IsUnixSocket {
			api.StartHttpServerWithUnixSocket(useLogger, *pcFlags.UnixSocketPath, runner)
			return
		}
		api.StartHttpServerWithTCP(useLogger, *pcFlags.PortNum, runner)
	}
}

func getClient() *client.PcClient {
	if *pcFlags.IsUnixSocket {
		return client.NewUdsClient(*pcFlags.UnixSocketPath, *pcFlags.LogLength)
	}
	return client.NewTcpClient(*pcFlags.Address, *pcFlags.PortNum, *pcFlags.LogLength)
}

func isUnixSocketMode(cmd *cobra.Command) bool {
	cobra.OnFinalize()
	return *pcFlags.IsUnixSocket ||
		os.Getenv(config.EnvVarUnixSocketPath) != "" ||
		cmd.Flags().Changed("unix-socket")
}
