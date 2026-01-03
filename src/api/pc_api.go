package api

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/f1bonacc1/process-compose/src/types"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/gin-gonic/gin"
)

//	@title			Process Compose API
//	@version		1.0
//	@description	This is a sample Process Compose server.

//	@contact.name	Process Compose Discord Channel
//	@contact.url	https://discord.gg/S4xgmRSHdC

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@externalDocs.url			https://f1bonacc1.github.io/process-compose/
//	@host						localhost:8080
//	@BasePath					/
//	@query.collection.format	multi

type PcApi struct {
	project app.IProject
	wsMtx   sync.Mutex
}

func NewPcApi(project app.IProject) *PcApi {
	return &PcApi{project: project}
}

// @Schemes
// @Id				GetProcess
// @Description	Retrieves the given process and its status
// @Tags			Process
// @Summary		Get process state
// @Produce		json
// @Param			name	path		string	true	"Process Name"
// @Success		200		{object}	types.ProcessState
// @Failure		400		{object}	map[string]string
// @Router			/process/{name} [get]
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
// @Id				GetProcessInfo
// @Description	Retrieves the given process and its config
// @Tags			Process
// @Summary		Get process config
// @Produce		json
// @Param			name	path		string	true	"Process Name"
// @Success		200		{object}	types.ProcessConfig
// @Failure		400		{object}	map[string]string
// @Router			/process/info/{name} [get]
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
// @Id				GetProcesses
// @Description	Retrieves all the configured processes and their status
// @Tags			Process
// @Summary		Get all processes
// @Produce		json
// @Success		200	{object}	types.ProcessesState	"Processes Status"
// @Failure		400	{object}	map[string]string
// @Router			/processes [get]
func (api *PcApi) GetProcesses(c *gin.Context) {
	states, err := api.project.GetProcessesState()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, states)
}

// @Schemes
// @Id				GetProcessLogs
// @Description	Retrieves the process logs
// @Tags			Process
// @Summary		Get process logs
// @Produce		json
// @Param			name		path		string				true	"Process Name"
// @Param			endOffset	path		int					true	"Offset from the end of the log"
// @Param			limit		path		int					true	"Limit of lines to get (0 will get all the lines till the end)"
// @Success		200			{object}	api.LogsResponse	"Process Logs"
// @Failure		400			{object}	map[string]string
// @Router			/process/logs/{name}/{endOffset}/{limit} [get]
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
// @Id				TruncateProcessLogs
// @Description	Truncates the process logs
// @Tags			Process
// @Summary		Truncate process logs
// @Produce		json
// @Param		name		path		string				true	"Process Name"
// @Success		200			{object}	api.NameResponse	"Truncated Process Name"
// @Failure		400			{object}	map[string]string
// @Router			/process/logs/{name} [delete]
func (api *PcApi) TruncateProcessLogs(c *gin.Context) {
	name := c.Param("name")
	err := api.project.TruncateProcessLogs(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}

// @Schemes
// @Id				StopProcess
// @Description	Sends kill signal to the process
// @Tags			Process
// @Summary		Stop a process
// @Produce		json
// @Param			name	path		string				true	"Process Name"
// @Success		200		{object}	api.NameResponse	"Stopped Process Name"
// @Failure		400		{object}	map[string]string
// @Router			/process/stop/{name} [patch]
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
// @Id				StopProcesses
// @Description	Sends kill signal to the processes list
// @Tags			Process
// @Summary		Stop processes
// @Accept			json
//
// @Param			[]string	body	[]string	true	"Processes Names"
//
// @Produce		json
// @Success		200	{object}	map[string]string	"Stopped Processes Names"
// @Success		207	{object}	map[string]string	"Stopped Processes Names"
// @Failure		400	{object}	map[string]string
// @Router			/processes/stop [patch]
func (api *PcApi) StopProcesses(c *gin.Context) {
	var names []string
	if err := c.ShouldBindJSON(&names); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	stopped, err := api.project.StopProcesses(names)
	if err != nil {
		if len(stopped) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusMultiStatus, stopped)
		}
		return
	}

	c.JSON(http.StatusOK, stopped)
}

// @Schemes
// @Id				StartProcess
// @Description	Starts the process if the state is not 'running' or 'pending'
// @Tags			Process
// @Summary		Start a process
// @Produce		json
// @Param			name	path		string				true	"Process Name"
// @Success		200		{object}	api.NameResponse	"Started Process Name"
// @Failure		400		{object}	map[string]string
// @Router			/process/start/{name} [post]
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
// @Id				RestartProcess
// @Description	Restarts the process
// @Tags			Process
// @Summary		Restart a process
// @Produce		json
// @Param			name	path		string				true	"Process Name"
// @Success		200		{object}	api.NameResponse	"Restarted Process Name"
// @Failure		400		{object}	map[string]string
// @Router			/process/restart/{name} [post]
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
// @Id				ScaleProcess
// @Description	Scale a process
// @Tags			Process
// @Summary		Scale a process to a given replicas count
// @Produce		json
// @Param			name	path		string				true	"Process Name"
// @Param			scale	path		int					true	"New amount of process replicas"
// @Success		200		{object}	api.NameResponse	"Scaled Process Name"
// @Failure		400		{object}	map[string]string
// @Router			/process/scale/{name}/{scale} [patch]
func (api *PcApi) ScaleProcess(c *gin.Context) {
	name := c.Param("name")
	scale, err := strconv.Atoi(c.Param("scale"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = api.project.ScaleProcess(name, scale)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}

// @Schemes
// @Id				IsAlive
// @Description	Check if server is responding
// @Tags			Liveness
// @Summary		Liveness Check
// @Produce		json
// @Success		200	{object}	api.StatusResponse	"Alive Status"
// @Router			/live [get]
func (api *PcApi) IsAlive(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// @Schemes
// @Id				GetProjectName
// @Description	Get process compose project name
// @Tags			ProjectName
// @Summary		Get Project Name
// @Produce		json
// @Success		200	{object}	api.ProjectNameResponse	"Project Name"
// @Failure		400	{object}	map[string]string
// @Router			/project/name [get]
func (api *PcApi) GetProjectName(c *gin.Context) {
	name, err := api.project.GetProjectName()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"projectName": name})
}

// @Schemes
// @Id				GetProcessPorts
// @Description	Retrieves process open ports
// @Tags			Process
// @Summary		Get process ports
// @Produce		json
// @Param			name	path		string				true	"Process Name"
// @Success		200		{object}	types.ProcessPorts	"Process Ports"
// @Failure		400		{object}	map[string]string
// @Router			/process/ports/{name} [get]
func (api *PcApi) GetProcessPorts(c *gin.Context) {
	name := c.Param("name")

	ports, err := api.project.GetProcessPorts(name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, ports)
}

// @Schemes
// @Id				ShutDownProject
// @Description	Shuts down the server
// @Tags			Project
// @Summary		Stops all the processes and the server
// @Produce		json
// @Success		200	{object}	api.StatusResponse	"Stopped Server"
// @Router			/project/stop [post]
func (api *PcApi) ShutDownProject(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "stopped"})
	_ = api.project.ShutDownProject()
}

// @Schemes
// @Id				UpdateProject
// @Description	Update running project
// @Tags			Project
// @Summary		Updates running processes
// @Produce		json
// @Success		200	{object}	map[string]string	"Update Project Status"
// @Success		207	{object}	map[string]string	"Update Project Status"
// @Failure		400	{object}	map[string]string
// @Router			/project [post]
func (api *PcApi) UpdateProject(c *gin.Context) {
	var project types.Project
	if err := c.ShouldBindJSON(&project); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, err := api.project.UpdateProject(&project)
	if err != nil {
		if len(status) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusMultiStatus, status)
		}
		return
	}
	c.JSON(http.StatusOK, status)
}

// @Schemes
// @Id				UpdateProcess
// @Description	Update process
// @Tags			Process
// @Summary		Updates process configuration
// @Accept			json
// @Param			process	body	types.ProcessConfig	true	"Process configuration to update"
// @Success		200	{object}	types.ProcessConfig	"Updated Process Config"
// @Failure		400	{object}	map[string]string
// @Router			/process [post]
func (api *PcApi) UpdateProcess(c *gin.Context) {
	var proc types.ProcessConfig
	if err := c.ShouldBindJSON(&proc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := api.project.UpdateProcess(&proc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, proc)
}

// @Schemes
// @Id				GetProjectState
// @Description	Retrieves project state information
// @Tags			Project
// @Summary		Get project state
// @Produce		json
// @Success		200	{object}	types.ProjectState	"Project State"
// @Failure		500	{object}	map[string]string
// @Router			/project/state [get]
func (api *PcApi) GetProjectState(c *gin.Context) {
	withMemory := c.DefaultQuery("withMemory", "false")
	checkMem, _ := strconv.ParseBool(withMemory)
	state, err := api.project.GetProjectState(checkMem)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, state)
}

// @Schemes
// @Id				ReloadProject
// @Description	Reload project state from config
// @Tags			Project
// @Summary		Reload project
// @Produce		json
// @Success		200	{object}	map[string]string	"Update Project Status"
// @Success		207	{object}	map[string]string	"Update Project Status"
// @Failure		400	{object}	map[string]string
// @Router			/project/configuration [post]
func (api *PcApi) ReloadProject(c *gin.Context) {
	status, err := api.project.ReloadProject()
	if err != nil {
		if len(status) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusMultiStatus, status)
		}
		return
	}
	c.JSON(http.StatusOK, status)
}

// @Schemes
// @Id              GetDependencyGraph
// @Description     Returns the process dependency graph with current status
// @Tags            Graph
// @Summary         Get dependency graph
// @Produce         json
// @Success         200 {object} types.DependencyGraph
// @Failure         400 {object} map[string]string
// @Router          /graph [get]
func (api *PcApi) GetDependencyGraph(c *gin.Context) {
	graph, err := api.project.GetDependencyGraph()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, graph)
}
