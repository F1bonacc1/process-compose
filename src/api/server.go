package api

import (
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

const EnvDebugMode = "PC_DEBUG_MODE"

func StartHttpServerWithUnixSocket(useLogger bool, unixSocket string, project app.IProject) (*http.Server, error) {
	router := getRouter(useLogger, project)
	log.Info().Msgf("start UDS http server listening %s", unixSocket)

	// Check if the unix socket is already in use
	// If it exists but we can't connect, remove it
	_, err := net.Dial("unix", unixSocket)
	if err == nil {
		log.Fatal().Msgf("unix socket %s is already in use", unixSocket)
	}
	os.Remove(unixSocket)

	server := &http.Server{
		Handler: router.Handler(),
	}

	listener, err := net.Listen("unix", unixSocket)
	if err != nil {
		return server, err
	}

	go func() {
		defer listener.Close()
		defer os.Remove(unixSocket)

		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msgf("start UDS http server on %s failed", unixSocket)
		}
	}()

	return server, nil
}

func StartHttpServerWithTCP(useLogger bool, address string, port int, project app.IProject) (*http.Server, error) {
	router := getRouter(useLogger, project)
	endPoint := fmt.Sprintf("%s:%d", address, port)
	log.Info().Msgf("start http server listening %s", endPoint)

	server := &http.Server{
		Addr:    endPoint,
		Handler: router.Handler(),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msgf("start http server on %s failed", endPoint)
		}
	}()

	return server, nil
}

func getRouter(useLogger bool, project app.IProject) *gin.Engine {
	if os.Getenv(EnvDebugMode) == "" {
		gin.SetMode(gin.ReleaseMode)
		useLogger = false
	}
	return InitRoutes(useLogger, NewPcApi(project))
}
