package main

import (
	"flag"
	"fmt"
	"net/http"
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

	routersInit := api.InitRoutes()
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
		tui.SetupTui()
	} else {
		project.Run()
	}

}
