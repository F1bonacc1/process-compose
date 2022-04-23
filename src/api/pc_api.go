package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/f1bonacc1/process-compose/src/app"
)

// @Summary Get all processes
// @Produce  json
// @Success 200 {object} object "Processes Status"
// @Router /processes [get]
func GetProcesses(c *gin.Context) {
	procs, err := app.PROJ.GetLexicographicProcessNames()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	states := []*app.ProcessState{}
	for _, name := range procs {
		states = append(states, app.PROJ.GetProcessState(name))
	}

	c.JSON(http.StatusOK, gin.H{"data": states})
}

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
