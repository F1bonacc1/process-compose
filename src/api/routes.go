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
func InitRoutes(useLogger bool) *gin.Engine {
	r := gin.New()
	if useLogger {
		r.Use(gin.Logger())
	}
	r.Use(gin.Recovery())

	//url := ginSwagger.URL("http://localhost:8080/swagger/doc.json") // The url pointing to API definition
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/", func(c *gin.Context) {
		location := url.URL{Path: "/swagger/index.html"}
		c.Redirect(http.StatusFound, location.RequestURI())
	})

	r.GET("/processes", GetProcesses)
	r.GET("/process/logs/:name/:endOffset/:limit", GetProcessLogs)
	r.PATCH("/process/stop/:name", StopProcess)
	r.POST("/process/start/:name", StartProcess)
	r.POST("/process/restart/:name", RestartProcess)

	//websocket
	r.GET("/process/logs/ws", HandleLogsStream)

	return r
}
