package cmd

import (
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/tui"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"syscall"
)

func runProject(isDefConfigPath bool, process []string) {
	if isDefConfigPath {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
		file, err := app.AutoDiscoverComposeFile(pwd)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
		fileName = file
	}

	project := app.NewProject(fileName)
	exitCode := 0
	if isTui {
		exitCode = runTui(project)
	} else {
		exitCode = runHeadless(project)
	}

	log.Info().Msg("Thank you for using process-compose")
	os.Exit(exitCode)
}

func runHeadless(project *app.Project) int {
	cancelChan := make(chan os.Signal, 1)
	// catch SIGTERM or SIGINTERRUPT
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-cancelChan
		log.Info().Msgf("Caught %v - Shutting down the running processes...", sig)
		project.ShutDownProject()
		os.Exit(1)
	}()
	exitCode := project.Run()
	return exitCode
}

func runTui(project *app.Project) int {
	defer quiet()()
	go tui.SetupTui(project.LogLength)
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
