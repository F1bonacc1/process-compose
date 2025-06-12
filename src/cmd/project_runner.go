package cmd

import (
	"fmt"
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
	opts.DisableDotenv(*pcFlags.DisableDotEnv)
	opts.WithTuiDisabled(!*pcFlags.IsTuiEnabled)

	project, err := loader.Load(opts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load project")
	}
	*pcFlags.IsTuiEnabled = !project.IsTuiDisabled

	prjOpts := app.ProjectOpts{}

	runner, err := app.NewProjectRunner(
		prjOpts.WithIsTuiOn(*pcFlags.IsTuiEnabled).
			WithMainProcess(mainProcess).
			WithMainProcessArgs(mainProcessArgs).
			WithProject(project).
			WithProcessesToRun(process).
			WithOrderedShutDown(*pcFlags.IsOrderedShutDown).
			WithNoDeps(noDeps).
			WithLogTruncate(*pcFlags.LogsTruncate).
			WithSlowRefRate(*pcFlags.SlowRefreshRate).
			WithRecursiveMetrics(*pcFlags.WithRecursiveMetrics),
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
	if *pcFlags.KeepProjectOn {
		runner.WaitForProjectShutdown()
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
	startTui(project, true)
	err := project.Run()
	if !*pcFlags.KeepProjectOn && !*pcFlags.KeepTuiOn {
		tui.Stop()
	} else {
		tui.Wait()
	}
	return err
}

func startTui(runner app.IProject, isAsync bool) {
	if !*pcFlags.IsReadOnlyMode {
		config.CreateProcCompHome()
	}

	// Starting the TUI takes some time, so if let's check if we're going to
	// just disconnect immediately and avoid the overhead if possible. This
	// also prevents us from flashing the TUI open for a split second.
	//
	// A no-op `process-compose --detached-with-tui --detach-on-success` takes
	// ~40ms with this optimization and ~90ms without it (for my project).
	//
	// A 2x performance improvement is worth a special case here.
	if *pcFlags.DetachOnSuccess {
		states, err := runner.GetProcessesState()
		if err != nil {
			log.Err(err).Msgf("Failed to get process states")
		} else if states.IsReady() {
			// This is kind of a lie, because we've skipped starting the TUI
			// entirely at this point, but
			// 1. The line above this says "Starting Process Compose in
			//    detached mode", so it makes sense to say we're "detaching" here.
			// 2. This makes the output consistent regardless of if the TUI is
			//    actually opened or not.
			fmt.Println(tui.DetachOnSuccessMessage)
			app.PrintStatesAsTable(states.States)
			return
		}
	}

	settings := config.NewSettings().Load()
	tuiOptions := []tui.Option{
		tui.WithRefreshRate(*pcFlags.RefreshRate),
		tui.WithReadOnlyMode(*pcFlags.IsReadOnlyMode),
		tui.WithFullScreen(*pcFlags.IsTuiFullScreen),
		tui.WithDisabledHidden(*pcFlags.HideDisabled),
		tui.WithDisabledExitConfirm(settings.DisableExitConfirmation),
		tui.WithDetachOnSuccess(*pcFlags.DetachOnSuccess),
		tui.LoadExtraShortCutsPaths(*pcFlags.ShortcutPaths),
	}

	tuiOptions = append(tuiOptions,
		ternary(pcFlags.PcThemeChanged, tui.WithTheme(*pcFlags.PcTheme), tui.WithTheme(settings.Theme)))

	tuiOptions = append(tuiOptions,
		ternary(pcFlags.SortColumnChanged,
			tui.WithStateSorter(getColumnId(*pcFlags.SortColumn), !*pcFlags.IsReverseSort),
			tui.WithStateSorter(getColumnId(settings.Sort.By), !settings.Sort.IsReversed)),
	)

	if isAsync {
		tui.RunTUIAsync(runner, tuiOptions...)
	} else {
		tui.RunTUI(runner, tuiOptions...)
	}
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
