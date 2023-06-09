package api

import (
	"github.com/f1bonacc1/process-compose/src/types"
	"net/http"
	"strconv"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gin-gonic/gin"
)

type PcApi struct {
	project app.IProject
}

func NewPcApi(project app.IProject) *PcApi {
	return &PcApi{project}
}

// @Schemes
// @Description Retrieves the given process and its status
// @Tags Process
// @Summary Get process state
// @Produce  json
// @Param name path string true "Process Name"
// @Success 200 {object} object "Process State"
// @Router /process/{name} [get]
func (api *PcApi) GetProcess(c *gin.Context) {
	name := c.Param("name")

	state, err := api.project.GetProcessState(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}

// @Schemes
// @Description Retrieves the given process and its config
// @Tags Process
// @Summary Get process config
// @Produce  json
// @Param name path string true "Process Name"
// @Success 200 {object} object "Process Config"
// @Router /process/info/{name} [get]
func (api *PcApi) GetProcessInfo(c *gin.Context) {
	name := c.Param("name")

	config, err := api.project.GetProcessInfo(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

// @Schemes
// @Description Retrieves all the configured processes and their status
// @Tags Process
// @Summary Get all processes
// @Produce  json
// @Success 200 {object} object "Processes Status"
// @Router /processes [get]
func (api *PcApi) GetProcesses(c *gin.Context) {
	procs, err := api.project.GetLexicographicProcessNames()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	states := []*types.ProcessState{}
	for _, name := range procs {
		state, err := api.project.GetProcessState(name)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		states = append(states, state)
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
func (api *PcApi) GetProcessLogs(c *gin.Context) {
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

	logs, err := api.project.GetProcessLog(name, endOffset, limit)
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
func (api *PcApi) StopProcess(c *gin.Context) {
	name := c.Param("name")
	err := api.project.StopProcess(name)
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
func (api *PcApi) StartProcess(c *gin.Context) {
	name := c.Param("name")
	err := api.project.StartProcess(name)
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
func (api *PcApi) RestartProcess(c *gin.Context) {
	name := c.Param("name")
	err := api.project.RestartProcess(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}

// @Schemes
// @Description Check if server is responding
// @Tags Liveness
// @Summary Liveness Check
// @Produce  json
// @Success 200
// @Router /live [get]
func (api *PcApi) IsAlive(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// @Schemes
// @Description Get process compose hostname
// @Tags Hostname
// @Summary Get Hostname
// @Produce  json
// @Success 200
// @Router /hostname [get]
func (api *PcApi) GetHostName(c *gin.Context) {
	name, err := api.project.GetHostName()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}
