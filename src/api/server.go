package api

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"net/http"
	"os"
	"time"
)

const EnvDebugMode = "PC_DEBUG_MODE"

func StartHttpServer(useLogger bool, port int, project app.IProject) {

	if os.Getenv(EnvDebugMode) == "" {
		gin.SetMode(gin.ReleaseMode)
		useLogger = false
	}

	handler := NewPcApi(project)
	routersInit := InitRoutes(useLogger, handler)
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

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msgf("start http server on %s failed", endPoint)
		}
	}()
}
