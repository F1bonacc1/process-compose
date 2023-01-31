package cmd

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/tui"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

func runProject(process []string, noDeps bool) {
	project, err := loader.Load(opts)
	if err != nil {
		fmt.Println(err)
		log.Fatal().Msg(err.Error())
	}

	runner, err := app.NewProjectRunner(project, process, noDeps)
	if err != nil {
		fmt.Println(err)
		log.Fatal().Msg(err.Error())
	}
	exitCode := 0
	if isTui {
		exitCode = runTui(runner)
	} else {
		exitCode = runHeadless(runner)
	}

	log.Info().Msg("Thank you for using process-compose")
	os.Exit(exitCode)
}

func setSignal(signalHandler func()) {
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, syscall.SIGTERM, os.Interrupt, syscall.SIGHUP)
	go func() {
		sig := <-cancelChan
		log.Info().Msgf("Caught %v - Shutting down the running processes...", sig)
		signalHandler()
		os.Exit(1)
	}()
}

func runHeadless(project *app.ProjectRunner) int {
	setSignal(func() {
		project.ShutDownProject()
	})
	exitCode := project.Run()
	return exitCode
}

func runTui(project *app.ProjectRunner) int {
	setSignal(func() {
		tui.Stop()
	})
	defer quiet()()
	go tui.SetupTui()
	exitCode := project.Run()
	tui.Stop()
	return exitCode
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
