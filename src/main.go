package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"os"

	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func setupLogger() {

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "06-01-02 15:04:05",
	})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
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

func main() {
	fileName := ""
	flag.StringVar(&fileName, "f", app.DefaultFileNames[0], "path to file to load")
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

	gin.SetMode("")

	routersInit := api.InitRouter()
	readTimeout := time.Duration(60) * time.Second
	writeTimeout := time.Duration(60) * time.Second
	endPoint := fmt.Sprintf(":%d", 8080)
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
	project.Run()
}
