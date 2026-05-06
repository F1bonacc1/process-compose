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

	s.addTool(
		mcp.NewTool(controlToolPrefix+"process_logs_search",
			mcp.WithDescription(fmt.Sprintf(`Search process log buffers using BM25 text ranking. Results are ranked by relevance, NOT by time, and may be old or out of chronological order. For the latest output of a single process use %sprocess_logs instead.

Each hit includes chunk_idx (0 = oldest line in the searched window, higher = more recent), so sort hits by chunk_idx descending if you need recency. Process log lines pass through verbatim — timestamps appear in 'text' only when the process itself emits them. 'truncated': true means the per-process line budget was reduced below log_limit to bound total work.`, controlToolPrefix)),
			mcp.WithString("query", mcp.Description("Search query (space-separated terms)"), mcp.Required()),
			mcp.WithString("name", mcp.Description("Process name to search. If omitted, searches every process.")),
			mcp.WithNumber("top_k", mcp.Description("Number of top results to return (default 20, max 100)")),
			mcp.WithNumber("log_limit", mcp.Description("Max log lines to fetch per process before searching (default 500, max 5000)")),
		),
		s.handleProcessLogsSearch,
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

	s.addTool(
		mcp.NewTool(controlToolPrefix+"project_dependency_graph",
			mcp.WithDescription("Return the project dependency graph: each node carries the process name, current status, readiness, and a depends_on map of upstream processes with their startup conditions (process_started, process_healthy, process_completed, process_log_ready). Useful for diagnosing why a process is stuck Pending."),
		),
		s.handleProjectDependencyGraph,
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

// Default and max bounds for pc_process_logs_search.
const (
	searchDefaultTopK     = 20
	searchMaxTopK         = 100
	searchDefaultLogLimit = 500
	searchMaxLogLimit     = 5000
	// searchMaxCorpusLines bounds the total lines tokenized and scored across
	// all processes in one search. When log_limit * N > this, each process's
	// budget is reduced to a fair share (most-recent lines kept) and the
	// result is flagged truncated.
	searchMaxCorpusLines = 50000
)

// logSearchHit is one entry in the pc_process_logs_search result. ChunkIdx is
// the line's 0-based index within the per-process tail we searched, not an
// absolute position in the underlying ring buffer — lines may roll out
// between calls.
type logSearchHit struct {
	Process  string  `json:"process"`
	ChunkIdx int     `json:"chunk_idx"`
	Score    float64 `json:"score"`
	Text     string  `json:"text"`
}

type logSearchResult struct {
	Query     string         `json:"query"`
	Truncated bool           `json:"truncated,omitempty"`
	Hits      []logSearchHit `json:"hits"`
}

func (s *Server) handleProcessLogsSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	topK := clampInt(req.GetInt("top_k", searchDefaultTopK), searchDefaultTopK, searchMaxTopK)
	logLimit := clampInt(req.GetInt("log_limit", searchDefaultLogLimit), searchDefaultLogLimit, searchMaxLogLimit)

	procNames, err := s.searchTargetNames(req.GetString("name", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	perProcCap := logLimit
	if n := len(procNames); n > 0 {
		if share := searchMaxCorpusLines / n; share < perProcCap {
			perProcCap = share
		}
	}
	if perProcCap < 1 {
		perProcCap = 1
	}
	truncated := perProcCap < logLimit

	var (
		docs      [][]string
		lineTexts []string
		lineProcs []string
		chunkIdxs []int
	)
	for _, pname := range procNames {
		lines, err := s.runner.GetProcessLog(pname, 0, perProcCap)
		if err != nil {
			log.Warn().Err(err).Str("process", pname).Msg("pc_process_logs_search: skipping process")
			continue
		}
		for i, line := range lines {
			docs = append(docs, Tokenize(line))
			lineTexts = append(lineTexts, line)
			lineProcs = append(lineProcs, pname)
			chunkIdxs = append(chunkIdxs, i)
		}
	}

	corpus := NewCorpus(docs, 0, 0)
	hits := corpus.TopN(Tokenize(query), topK)

	result := logSearchResult{Query: query, Truncated: truncated, Hits: make([]logSearchHit, 0, len(hits))}
	for _, h := range hits {
		result.Hits = append(result.Hits, logSearchHit{
			Process:  lineProcs[h.DocID],
			ChunkIdx: chunkIdxs[h.DocID],
			Score:    h.Score,
			Text:     lineTexts[h.DocID],
		})
	}
	return mcp.NewToolResultJSON(result)
}

// searchTargetNames returns just `name` when provided (and known), otherwise
// every known process name from the runner.
func (s *Server) searchTargetNames(name string) ([]string, error) {
	if name != "" {
		if _, err := s.runner.GetProcessState(name); err != nil {
			return nil, fmt.Errorf("unknown process %q: %w", name, err)
		}
		return []string{name}, nil
	}
	states, err := s.runner.GetProcessesState()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}
	names := make([]string, 0, len(states.States))
	for _, st := range states.States {
		names = append(names, st.Name)
	}
	return names, nil
}

func clampInt(v, fallback, max int) int {
	if v <= 0 {
		v = fallback
	}
	if v > max {
		v = max
	}
	return v
}

func (s *Server) handleProjectDependencyGraph(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	graph, err := s.runner.GetDependencyGraph()
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get dependency graph: %v", err), nil
	}
	return mcp.NewToolResultJSON(graph)
}
