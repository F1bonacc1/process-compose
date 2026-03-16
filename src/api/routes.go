package api

import (
	"net/http"
	"net/url"

	"github.com/f1bonacc1/process-compose/src/config"
	_ "github.com/f1bonacc1/process-compose/src/docs"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// TokenAuthMiddleware enforces API access using an auth token if configured.
func TokenAuthMiddleware(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqToken := c.GetHeader(config.TokenHeader)
		if reqToken != token {
			log.Error().
				Str("client_ip", c.ClientIP()).
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Msg("failed login attempt: invalid or missing token")
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}

// InitRoutes initialize routing information
func InitRoutes(useLogger bool, handler *PcApi) *gin.Engine {
	r := gin.New()
	if useLogger {
		r.Use(gin.Logger())
	}
	r.Use(gin.Recovery())

	authToken := config.GetApiToken()
	if authToken != "" {
		if len(authToken) < 20 {
			log.Fatal().Msgf("%s must be at least 20 characters long", config.EnvVarApiToken)
		}
		r.Use(TokenAuthMiddleware(authToken))
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/", func(c *gin.Context) {
		location := url.URL{Path: "/swagger/index.html"}
		c.Redirect(http.StatusFound, location.RequestURI())
	})

	r.GET("/live", handler.IsAlive)
	r.GET("/processes", handler.GetProcesses)
	r.GET("/process/:name", handler.GetProcess)
	r.GET("/process/info/:name", handler.GetProcessInfo)
	r.POST("/process", handler.UpdateProcess)
	r.GET("/process/ports/:name", handler.GetProcessPorts)
	r.GET("/process/logs/:name/:endOffset/:limit", handler.GetProcessLogs)
	r.DELETE("/process/logs/:name", handler.TruncateProcessLogs)
	r.PATCH("/process/stop/:name", handler.StopProcess)
	r.PATCH("/process/signal/:name/:signal", handler.SendSignal)
	r.PATCH("/processes/stop", handler.StopProcesses)
	r.POST("/process/start/:name", handler.StartProcess)
	r.POST("/process/restart/:name", handler.RestartProcess)
	r.POST("/project/stop", handler.ShutDownProject)
	r.POST("/project", handler.UpdateProject)
	r.POST("/project/configuration", handler.ReloadProject)
	r.GET("/project/name", handler.GetProjectName)
	r.GET("/project/state", handler.GetProjectState)
	r.POST("/namespace/start/:name", handler.StartNamespace)
	r.POST("/namespace/stop/:name", handler.StopNamespace)
	r.POST("/namespace/restart/:name", handler.RestartNamespace)
	r.GET("/namespaces", handler.GetNamespaces)
	r.PATCH("/process/scale/:name/:scale", handler.ScaleProcess)
	r.GET("/process/logs/ws", handler.HandleLogsStream)
	r.GET("/graph", handler.GetDependencyGraph)

	return r
}
