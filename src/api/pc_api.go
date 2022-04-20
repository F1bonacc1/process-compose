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
