package cmd

import (
	"github.com/f1bonacc1/process-compose/src/admitter"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/tui"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

func getProjectRunner(process []string, noDeps bool, mainProcess string, mainProcessArgs []string) *app.ProjectRunner {
	if *pcFlags.HideDisabled {
		opts.AddAdmitter(&admitter.DisabledProcAdmitter{})
	}
	if *pcFlags.DisableDotEnv {
		opts.DisableDotenv()
	}
	project, err := loader.Load(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load project")
	}

	prjOpts := app.ProjectOpts{}

	runner, err := app.NewProjectRunner(
		prjOpts.WithIsTuiOn(*pcFlags.IsTuiEnabled).
			WithMainProcess(mainProcess).
			WithMainProcessArgs(mainProcessArgs).
			WithProject(project).
			WithProcessesToRun(process).
			WithOrderedShutDown(*pcFlags.IsOrderedShutDown).
			WithNoDeps(noDeps),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize the project")
	}
	return runner
}

func runProject(runner *app.ProjectRunner) error {
	var err error
	if *pcFlags.IsTuiEnabled {
		err = runTui(runner)
	} else {
		err = runHeadless(runner)
	}
	os.Remove(*pcFlags.UnixSocketPath)
	log.Info().Msg("Thank you for using process-compose")
	return err
}

func setSignal(signalHandler func()) {
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, os.Interrupt, syscall.SIGHUP)
	go func() {
		sig := <-cancelChan
		log.Info().Msgf("Caught %v - Shutting down the running processes...", sig)
		signalHandler()
	}()
}

func runHeadless(project *app.ProjectRunner) error {
	setSignal(func() {
		_ = project.ShutDownProject()
	})
	return project.Run()
}

func runTui(project *app.ProjectRunner) error {
	go startTui(project)
	err := project.Run()
	if !*pcFlags.KeepTuiOn {
		tui.Stop()
	} else {
		tui.Wait()
	}
	return err
}

func startTui(runner app.IProject) {
	tuiOptions := []tui.Option{
		tui.WithRefreshRate(*pcFlags.RefreshRate),
	}
	if !*pcFlags.IsReadOnlyMode {
		config.CreateProcCompHome()
	}
	settings := config.NewSettings().Load()

	tuiOptions = append(tuiOptions,
		ternary(pcFlags.PcThemeChanged, tui.WithTheme(*pcFlags.PcTheme), tui.WithTheme(settings.Theme)))

	tuiOptions = append(tuiOptions,
		ternary(pcFlags.SortColumnChanged,
			tui.WithStateSorter(getColumnId(*pcFlags.SortColumn), !*pcFlags.IsReverseSort),
			tui.WithStateSorter(getColumnId(settings.Sort.By), !settings.Sort.IsReversed)),
		tui.WithReadOnlyMode(*pcFlags.IsReadOnlyMode))

	tui.SetupTui(runner, tuiOptions...)
}

func getColumnId(columnName string) tui.ColumnID {
	col, err := tui.StringToColumnID(columnName)
	if err != nil {
		log.Err(err).Msgf("Invalid column name %s provided. Using %s", *pcFlags.SortColumn, config.DefaultSortColumn)
		col = tui.ProcessStateName
	}
	return col
}

func ternary[T any](cond bool, a, b T) T {
	if cond {
		return a
	}
	return b
}

func quiet() func() {
	null, _ := os.Open(os.DevNull)
	sout := os.Stdout
	serr := os.Stderr
	os.Stdout = null
	os.Stderr = null
	return func() {
		defer null.Close()
		os.Stdout = sout
		os.Stderr = serr
	}
}
