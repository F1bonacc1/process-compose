package main

import (
	"flag"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"os"

	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/tui"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const EnvDebugMode = "PC_DEBUG_MODE"

var version = "undefined"

func setupLogger() {

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "06-01-02 15:04:05",
	})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func init() {
	setupLogger()
}

func quiet() func() {
	null, _ := os.Open(os.DevNull)
	sout := os.Stdout
	serr := os.Stderr
	os.Stdout = null
	os.Stderr = null
	zerolog.SetGlobalLevel(zerolog.Disabled)
	return func() {
		defer null.Close()
		os.Stdout = sout
		os.Stderr = serr
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

func runHeadless(project *app.Project) {
	cancelChan := make(chan os.Signal, 1)
	// catch SIGTERM or SIGINTERRUPT
	signal.Notify(cancelChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		project.Run()
	}()
	sig := <-cancelChan
	log.Info().Msgf("Caught %v - Shutting down the running processes...", sig)
	project.ShutDownProject()
}

func main() {
	fileName := ""
	port := 8080
	isTui := true
	flag.StringVar(&fileName, "f", app.DefaultFileNames[0], "path to file to load")
	flag.IntVar(&port, "p", port, "port number")
	flag.BoolVar(&isTui, "t", isTui, "disable tui (-t=false)")
	flag.Parse()
	if !isFlagPassed("f") {
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

	if os.Getenv(EnvDebugMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	routersInit := api.InitRoutes(!isTui)
	readTimeout := time.Duration(60) * time.Second
	writeTimeout := time.Duration(60) * time.Second
	endPoint := fmt.Sprintf(":%d", port)
	maxHeaderBytes := 1 << 20

	server := &http.Server{
		Addr:           endPoint,
		Handler:        routersInit,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		MaxHeaderBytes: maxHeaderBytes,
	}

	log.Info().Msgf("start http server listening %s", endPoint)

	go server.ListenAndServe()

	project := app.CreateProject(fileName)

	if isTui {
		defer quiet()()
		go project.Run()
		tui.SetupTui(version, project.LogLength)
	} else {
		runHeadless(project)
	}

	log.Info().Msg("Thank you for using proccess-compose")

}
