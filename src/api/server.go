package api

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"os"
)

const EnvDebugMode = "PC_DEBUG_MODE"

func StartHttpServerWithUnixSocket(useLogger bool, unixSocket string, project app.IProject) {
	router := getRouter(useLogger, project)
	log.Info().Msgf("start UDS http server listening %s", unixSocket)
	go func() {
		os.Remove(unixSocket)
		err := router.RunUnix(unixSocket)
		if err != nil {
			log.Fatal().Err(err).Msgf("start UDS http server on %s failed", unixSocket)
		}
	}()
}

func StartHttpServerWithTCP(useLogger bool, port int, project app.IProject) {
	router := getRouter(useLogger, project)
	endPoint := fmt.Sprintf(":%d", port)
	log.Info().Msgf("start http server listening %s", endPoint)
	go func() {
		err := router.Run(endPoint)
		if err != nil {
			log.Fatal().Err(err).Msgf("start http server on %s failed", endPoint)
		}
	}()

}

func getRouter(useLogger bool, project app.IProject) *gin.Engine {
	if os.Getenv(EnvDebugMode) == "" {
		gin.SetMode(gin.ReleaseMode)
		useLogger = false
	}
	return InitRoutes(useLogger, NewPcApi(project))
}
