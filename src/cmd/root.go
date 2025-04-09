package cmd

import (
	"context"
	"errors"
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
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
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
		Run: func(cmd *cobra.Command, args []string) {
			runProjectCmd([]string{})
		},
	}
)

func runProjectCmd(args []string) {
	defer func() {
		_ = logFile.Close()
	}()
	runner := getProjectRunner(args, *pcFlags.NoDependencies, "", []string{})
	if *pcFlags.IsDetached || *pcFlags.IsDetachedWithTui {
		//placing it here ensures that if the compose.yaml is invalid, the program will exit immediately
		runInDetachedMode()
	}
	err := waitForProjectAndServer(!*pcFlags.IsTuiEnabled, runner)
	handleErrorAndExit(err)
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
	rootCmd.Flags().StringArrayVar(pcFlags.ShortcutPaths, "shortcuts", config.GetShortCutsPaths(nil), "paths to shortcut config files to load (env: "+config.EnvVarNameShortcuts+")")
	rootCmd.Flags().BoolVar(pcFlags.KeepTuiOn, "keep-tui", *pcFlags.KeepTuiOn, "keep TUI running even after all processes exit")
	rootCmd.Flags().BoolVar(pcFlags.KeepProjectOn, "keep-project", *pcFlags.KeepProjectOn, "keep the project running even after all processes exit")
	rootCmd.PersistentFlags().BoolVar(pcFlags.NoServer, "no-server", *pcFlags.NoServer, "disable HTTP server (env: "+config.EnvVarNameNoServer+")")
	rootCmd.PersistentFlags().BoolVar(pcFlags.IsOrderedShutDown, "ordered-shutdown", *pcFlags.IsOrderedShutDown, "shut down processes in reverse dependency order")
	rootCmd.Flags().BoolVarP(pcFlags.HideDisabled, "hide-disabled", "d", *pcFlags.HideDisabled, "hide disabled processes (env: "+config.EnvVarHideDisabled+")")
	rootCmd.Flags().StringArrayVar(pcFlags.EnabledProcesses, "enable", *pcFlags.EnabledProcesses, "names of processes to enable (comma-delimited). Processes that are both enabled and disabled will be disabled. (env: "+config.EnvVarEnabledProcesses+")")
	rootCmd.Flags().StringArrayVar(pcFlags.DisabledProcesses, "disable", *pcFlags.DisabledProcesses, "names of processes to disable (comma-delimited) (env: "+config.EnvVarDisabledProcesses+")")
	rootCmd.Flags().VarP(refreshRateFlag{pcFlags.RefreshRate}, "ref-rate", "r", "TUI refresh rate in seconds or as a Go duration string (e.g. 1s)")
	rootCmd.PersistentFlags().IntVarP(pcFlags.PortNum, "port", "p", *pcFlags.PortNum, "port number (env: "+config.EnvVarNamePort+")")
	rootCmd.Flags().StringArrayVarP(&opts.FileNames, "config", "f", config.GetConfigDefault(), "path to config files to load (env: "+config.EnvVarNameConfig+")")
	rootCmd.Flags().StringArrayVarP(&opts.EnvFileNames, "env", "e", []string{".env"}, "path to env files to load")
	rootCmd.Flags().StringArrayVarP(&nsAdmitter.EnabledNamespaces, "namespace", "n", nil, "run only specified namespaces (default all)")
	rootCmd.PersistentFlags().StringVarP(pcFlags.LogFile, "log-file", "L", *pcFlags.LogFile, "Specify the log file path (env: "+config.LogPathEnvVarName+")")
	rootCmd.PersistentFlags().BoolVar(pcFlags.IsReadOnlyMode, "read-only", *pcFlags.IsReadOnlyMode, "enable read-only mode (env: "+config.EnvVarReadOnlyMode+")")
	rootCmd.Flags().BoolVar(pcFlags.DisableDotEnv, "disable-dotenv", *pcFlags.DisableDotEnv, "disable .env file loading (env: "+config.EnvVarDisableDotEnv+"=1)")
	rootCmd.Flags().BoolVar(pcFlags.IsTuiFullScreen, "tui-fs", *pcFlags.IsTuiFullScreen, "enable TUI full screen (env: "+config.EnvVarTuiFullScreen+"=1)")
	rootCmd.Flags().AddFlag(commonFlags.Lookup(flagReverse))
	rootCmd.Flags().AddFlag(commonFlags.Lookup(flagSort))
	rootCmd.Flags().AddFlag(commonFlags.Lookup(flagTheme))

	_ = rootCmd.Flags().MarkDeprecated("keep-tui", "use --keep-project instead")

	if runtime.GOOS != "windows" {
		rootCmd.Flags().BoolVarP(pcFlags.IsDetached, "detached", "D", *pcFlags.IsDetached, "run process-compose in detached mode")
		rootCmd.Flags().BoolVar(pcFlags.IsDetachedWithTui, "detached-with-tui", *pcFlags.IsDetachedWithTui, "run process-compose in detached mode with TUI")
		rootCmd.Flags().BoolVar(pcFlags.DetachOnSuccess, "detach-on-success", *pcFlags.DetachOnSuccess, "detach the process-compose TUI after successful startup. Requires --detached-with-tui")
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

// Logs and exits with a non-zero code if there are any errors.
func handleErrorAndExit(err error) {
	if err != nil {
		log.Error().Err(err).Send()
		var exitErr *app.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}

func startHttpServerIfEnabled(useLogger bool, runner *app.ProjectRunner) (*http.Server, error) {
	if !*pcFlags.NoServer {
		if *pcFlags.IsUnixSocket {
			return api.StartHttpServerWithUnixSocket(useLogger, *pcFlags.UnixSocketPath, runner)
		}
		return api.StartHttpServerWithTCP(useLogger, *pcFlags.PortNum, runner)
	}

	return nil, nil
}

func waitForProjectAndServer(useLogger bool, runner *app.ProjectRunner) error {
	server, err := startHttpServerIfEnabled(useLogger, runner)
	if err != nil {
		return err
	}
	// Blocks until shutdown.
	if err = runProject(runner); err != nil {
		return err
	}
	if server != nil {
		shutdownTimeout := 5 * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
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

// refreshRateFlag is a custom flag type for the TUI's refresh rate.
// It accepts both an integer in seconds and a duration string.
type refreshRateFlag struct {
	dst *time.Duration
}

func (f refreshRateFlag) String() string {
	d := *f.dst
	if d%time.Second == 0 {
		return strconv.Itoa(int(d / time.Second))
	}
	return d.String()
}

func (f refreshRateFlag) Set(str string) error {
	i, err := strconv.Atoi(str)
	if err == nil {
		*f.dst = time.Duration(i) * time.Second
		return nil
	}
	d, err := time.ParseDuration(str)
	if err == nil {
		*f.dst = d
		return nil
	}
	return fmt.Errorf(
		"invalid refresh rate %q, must be a duration or an integer in seconds",
		str)
}

func (f refreshRateFlag) Type() string {
	return "duration"
}
