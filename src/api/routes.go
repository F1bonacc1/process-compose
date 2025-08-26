package api

import (
	"net/http"
	"net/url"

	_ "github.com/f1bonacc1/process-compose/src/docs"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// InitRoutes initialize routing information
func InitRoutes(useLogger bool, handler *PcApi) *gin.Engine {
	r := gin.New()
	if useLogger {
		r.Use(gin.Logger())
	}
	r.Use(gin.Recovery())

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
	r.PATCH("/processes/stop", handler.StopProcesses)
	r.POST("/process/start/:name", handler.StartProcess)
	r.POST("/process/restart/:name", handler.RestartProcess)
	r.POST("/project/stop", handler.ShutDownProject)
	r.POST("/project", handler.UpdateProject)
	r.POST("/project/configuration", handler.ReloadProject)
	r.GET("/project/name", handler.GetProjectName)
	r.GET("/project/state", handler.GetProjectState)
	r.PATCH("/process/scale/:name/:scale", handler.ScaleProcess)
	r.GET("/process/logs/ws", handler.HandleLogsStream)

	return r
}
