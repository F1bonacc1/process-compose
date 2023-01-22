package api

import (
	"github.com/f1bonacc1/process-compose/src/types"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/f1bonacc1/process-compose/src/app"
)

// @Schemes
// @Description Retrieves all the configured processes and their status
// @Tags Process
// @Summary Get all processes
// @Produce  json
// @Success 200 {object} object "Processes Status"
// @Router /processes [get]
func GetProcesses(c *gin.Context) {
	procs := app.PROJ.GetProject().GetLexicographicProcessNames()

	states := []*types.ProcessState{}
	for _, name := range procs {
		states = append(states, app.PROJ.GetProcessState(name))
	}

	c.JSON(http.StatusOK, gin.H{"data": states})
}

// @Schemes
// @Description Retrieves the process logs
// @Tags Process
// @Summary Get process logs
// @Produce  json
// @Param name path string true "Process Name"
// @Param endOffset path int true "Offset from the end of the log"
// @Param limit path int true "Limit of lines to get (0 will get all the lines till the end)"
// @Success 200 {object} object "Process Logs"
// @Router /process/logs/{name}/{endOffset}/{limit} [get]
func GetProcessLogs(c *gin.Context) {
	name := c.Param("name")
	endOffset, err := strconv.Atoi(c.Param("endOffset"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	limit, err := strconv.Atoi(c.Param("limit"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logs, err := app.PROJ.GetProcessLog(name, endOffset, limit)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// @Schemes
// @Description Sends kill signal to the process
// @Tags Process
// @Summary Stop a process
// @Produce  json
// @Param name path string true "Process Name"
// @Success 200 {string} string "Stopped Process Name"
// @Router /process/stop/{name} [patch]
func StopProcess(c *gin.Context) {
	name := c.Param("name")
	err := app.PROJ.StopProcess(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}

// @Schemes
// @Description Starts the process if the state is not 'running' or 'pending'
// @Tags Process
// @Summary Start a process
// @Produce  json
// @Param name path string true "Process Name"
// @Success 200 {string} string "Started Process Name"
// @Router /process/start/{name} [post]
func StartProcess(c *gin.Context) {
	name := c.Param("name")
	err := app.PROJ.StartProcess(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}

// @Schemes
// @Description Restarts the process
// @Tags Process
// @Summary Restart a process
// @Produce  json
// @Param name path string true "Process Name"
// @Success 200 {string} string "Restarted Process Name"
// @Router /process/restart/{name} [post]
func RestartProcess(c *gin.Context) {
	name := c.Param("name")
	err := app.PROJ.RestartProcess(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}
