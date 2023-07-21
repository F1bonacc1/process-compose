package api

import (
	_ "github.com/f1bonacc1/process-compose/src/docs"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"net/http"
	"net/url"
)

// @title Process Compose API
// @version 1.0
// @description process compose description

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Appache 2.0
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @query.collection.format multi

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
	r.GET("/hostname", handler.GetHostName)
	r.GET("/processes", handler.GetProcesses)
	r.GET("/process/:name", handler.GetProcess)
	r.GET("/process/info/:name", handler.GetProcessInfo)
	r.GET("/process/ports/:name", handler.GetProcessPorts)
	r.GET("/process/logs/:name/:endOffset/:limit", handler.GetProcessLogs)
	r.PATCH("/process/stop/:name", handler.StopProcess)
	r.PATCH("/processes/stop", handler.StopProcesses)
	r.POST("/process/start/:name", handler.StartProcess)
	r.POST("/process/restart/:name", handler.RestartProcess)
	r.PATCH("/process/scale/:name/:scale", handler.ScaleProcess)

	//websocket
	r.GET("/process/logs/ws", handler.HandleLogsStream)

	return r
}
