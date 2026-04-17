package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog/log"
)

// controlToolPrefix is the namespace for built-in control-plane tools.
// Using a prefix avoids collisions with user-defined process tools.
const controlToolPrefix = "pc_"

// RegisterControlTools registers all built-in process-compose control tools
// (process and project management) on the MCP server.
func (s *Server) RegisterControlTools() error {
	s.registerProcessControlTools()
	s.registerProjectControlTools()
	log.Info().Msg("Registered MCP control tools")
	return nil
}

// ---- process.* tools ------------------------------------------------------

func (s *Server) registerProcessControlTools() {
	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_start",
			mcp.WithDescription("Start a process by name."),
			mcp.WithString("name", mcp.Description("Process name"), mcp.Required()),
		),
		s.handleProcessStart,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_stop",
			mcp.WithDescription("Stop one or more running processes by name."),
			mcp.WithArray("names",
				mcp.Description("Process names to stop"),
				mcp.Required(),
				mcp.WithStringItems(),
			),
		),
		s.handleProcessStop,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_restart",
			mcp.WithDescription("Restart a process by name."),
			mcp.WithString("name", mcp.Description("Process name"), mcp.Required()),
		),
		s.handleProcessRestart,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_scale",
			mcp.WithDescription("Scale a process to a given replica count."),
			mcp.WithString("name", mcp.Description("Process name"), mcp.Required()),
			mcp.WithNumber("scale", mcp.Description("Replica count"), mcp.Required()),
		),
		s.handleProcessScale,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_get",
			mcp.WithDescription("Get the state of a single process."),
			mcp.WithString("name", mcp.Description("Process name"), mcp.Required()),
		),
		s.handleProcessGet,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_list",
			mcp.WithDescription("List all processes and their current states."),
		),
		s.handleProcessList,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_ports",
			mcp.WithDescription("Get TCP/UDP ports a process is listening on."),
			mcp.WithString("name", mcp.Description("Process name"), mcp.Required()),
		),
		s.handleProcessPorts,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_logs",
			mcp.WithDescription("Fetch the most recent log lines of a process (one-shot, non-streaming)."),
			mcp.WithString("name", mcp.Description("Process name"), mcp.Required()),
			mcp.WithNumber("tail", mcp.Description("Number of lines to return (default 100)")),
			mcp.WithNumber("offset_from_end", mcp.Description("Offset from end of log buffer (default 0)")),
		),
		s.handleProcessLogs,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_logs_truncate",
			mcp.WithDescription("Truncate the log buffer for a process."),
			mcp.WithString("name", mcp.Description("Process name"), mcp.Required()),
		),
		s.handleProcessLogsTruncate,
	)
}

// ---- project.* tools ------------------------------------------------------

func (s *Server) registerProjectControlTools() {
	s.addTool(
		mcp.NewTool(controlToolPrefix+"project_state",
			mcp.WithDescription("Get the overall process-compose project state (uptime, process counts, optional memory)."),
			mcp.WithBoolean("with_memory", mcp.Description("Include memory usage statistics (default false)")),
		),
		s.handleProjectState,
	)

	s.addTool(
		mcp.NewTool(controlToolPrefix+"project_is_ready",
			mcp.WithDescription("Check whether all processes are ready. Returns ready=true only when every process is ready."),
		),
		s.handleProjectIsReady,
	)
}

// addTool wraps mcpServer.AddTool with a uniform handler signature.
func (s *Server) addTool(tool mcp.Tool, handler func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)) {
	s.mcpServer.AddTool(tool, handler)
}

// ---- handlers -------------------------------------------------------------

func (s *Server) handleProcessStart(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.runner.StartProcess(name); err != nil {
		return mcp.NewToolResultErrorf("failed to start process %s: %v", name, err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Process %s started", name)), nil
}

func (s *Server) handleProcessStop(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	names, err := req.RequireStringSlice("names")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(names) == 0 {
		return mcp.NewToolResultError("names must contain at least one process"), nil
	}
	stopped, err := s.runner.StopProcesses(names)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to stop processes: %v", err), nil
	}
	return mcp.NewToolResultJSON(stopped)
}

func (s *Server) handleProcessRestart(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.runner.RestartProcess(name); err != nil {
		return mcp.NewToolResultErrorf("failed to restart process %s: %v", name, err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Process %s restarted", name)), nil
}

func (s *Server) handleProcessScale(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	scale, err := req.RequireInt("scale")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.runner.ScaleProcess(name, scale); err != nil {
		return mcp.NewToolResultErrorf("failed to scale process %s to %d: %v", name, scale, err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Process %s scaled to %d", name, scale)), nil
}

func (s *Server) handleProcessGet(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	state, err := s.runner.GetProcessState(name)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get process state for %s: %v", name, err), nil
	}
	return mcp.NewToolResultJSON(state)
}

func (s *Server) handleProcessList(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	states, err := s.runner.GetProcessesState()
	if err != nil {
		return mcp.NewToolResultErrorf("failed to list processes: %v", err), nil
	}
	return mcp.NewToolResultJSON(states)
}

func (s *Server) handleProcessPorts(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	ports, err := s.runner.GetProcessPorts(name)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get ports for %s: %v", name, err), nil
	}
	return mcp.NewToolResultJSON(ports)
}

func (s *Server) handleProcessLogs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	tail := req.GetInt("tail", 100)
	if tail <= 0 {
		tail = 100
	}
	offset := req.GetInt("offset_from_end", 0)
	if offset < 0 {
		offset = 0
	}
	lines, err := s.runner.GetProcessLog(name, offset, tail)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get logs for %s: %v", name, err), nil
	}
	return mcp.NewToolResultJSON(processLogsResult{Name: name, Lines: lines})
}

// processLogsResult wraps log lines in an object so MCP structuredContent is valid.
type processLogsResult struct {
	Name  string   `json:"name"`
	Lines []string `json:"lines"`
}

func (s *Server) handleProcessLogsTruncate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := s.runner.TruncateProcessLogs(name); err != nil {
		return mcp.NewToolResultErrorf("failed to truncate logs for %s: %v", name, err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Process %s logs truncated", name)), nil
}

func (s *Server) handleProjectState(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	withMemory := req.GetBool("with_memory", false)
	state, err := s.runner.GetProjectState(withMemory)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get project state: %v", err), nil
	}
	return mcp.NewToolResultJSON(state)
}

// projectReadyResult is the structured payload returned by pc_project_is_ready.
type projectReadyResult struct {
	Ready    bool                   `json:"ready"`
	Total    int                    `json:"total"`
	NotReady []notReadyProcessEntry `json:"not_ready,omitempty"`
}

type notReadyProcessEntry struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (s *Server) handleProjectIsReady(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	states, err := s.runner.GetProcessesState()
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get processes state: %v", err), nil
	}
	notReady := make([]notReadyProcessEntry, 0)
	for i := range states.States {
		state := &states.States[i]
		ready, reason := state.IsReadyReason()
		if !ready {
			notReady = append(notReady, notReadyProcessEntry{
				Name:   state.Name,
				Status: state.Status,
				Reason: reason,
			})
		}
	}
	result := projectReadyResult{
		Ready:    len(notReady) == 0,
		Total:    len(states.States),
		NotReady: notReady,
	}
	return mcp.NewToolResultJSON(result)
}
