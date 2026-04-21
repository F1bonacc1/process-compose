package mcpctl

import (
	"context"
	"fmt"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog/log"
)

// Default limits for log-fetching tools.
const (
	defaultGetLogsLines   = 100
	maxGetLogsLines       = 1000
	defaultSearchTopK     = 20
	maxSearchTopK         = 100
	defaultSearchLogLimit = 500
	maxSearchLogLimit     = 5000
)

// registerBuiltinTools registers the Control MCP tools on the server.
func (s *Server) registerBuiltinTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("list_processes",
			mcp.WithDescription(`List all processes managed by process-compose with their current status, CPU/memory usage, uptime, and restart count. Use this to get an overview of what's running in the local dev environment.`),
		),
		s.toolListProcesses,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_process",
			mcp.WithDescription(`Get detailed information about a specific process including its command, dependencies, status, and resource usage.`),
			mcp.WithString("name", mcp.Required(), mcp.Description(`The process name (e.g. "web-server", "chort")`)),
		),
		s.toolGetProcess,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_logs",
			mcp.WithDescription(`Get the most recent log lines from a process (tail). Use this for time-sensitive queries like "what just happened", "latest errors", or "recent output" — it returns only the tail end of the log buffer.`),
			mcp.WithString("name", mcp.Required(), mcp.Description(`The process name (e.g. "web-server", "chort")`)),
			mcp.WithNumber("lines", mcp.Description(`Number of recent lines to fetch (default 100, max 1000)`)),
			mcp.WithNumber("offset", mcp.Description(`Offset from the end of the log buffer (default 0)`)),
		),
		s.toolGetLogs,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("search_logs",
			mcp.WithDescription(`Search the entire log buffer using BM25 text ranking. Results are ranked by relevance, NOT by time — they may be old or out of chronological order. For recent/latest logs, use get_logs instead.

Examples:
- Search for errors: query="error failed exception"
- Search for a specific module: query="GraphQL resolver timeout"
- Search across all processes: omit the name parameter`),
			mcp.WithString("query", mcp.Required(), mcp.Description(`Search query (space-separated terms)`)),
			mcp.WithString("name", mcp.Description(`Process name to search. If omitted, searches all processes.`)),
			mcp.WithNumber("top_k", mcp.Description(`Number of top results to return (default 20, max 100)`)),
			mcp.WithNumber("log_limit", mcp.Description(`Max log lines to fetch per process before searching (default 500, max 5000)`)),
		),
		s.toolSearchLogs,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("start_process",
			mcp.WithDescription(`Start a stopped process managed by process-compose.`),
			mcp.WithString("name", mcp.Required(), mcp.Description(`The process name to start`)),
		),
		s.toolStartProcess,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("stop_process",
			mcp.WithDescription(`Stop a running process managed by process-compose.`),
			mcp.WithString("name", mcp.Required(), mcp.Description(`The process name to stop`)),
		),
		s.toolStopProcess,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("restart_process",
			mcp.WithDescription(`Restart a process managed by process-compose.`),
			mcp.WithString("name", mcp.Required(), mcp.Description(`The process name to restart (e.g. "web-server")`)),
		),
		s.toolRestartProcess,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_dependency_graph",
			mcp.WithDescription(`Get the process dependency graph showing which processes depend on which others. Useful for understanding startup order and failure cascading.`),
		),
		s.toolGetDependencyGraph,
	)

	log.Info().Msg("Registered Control MCP tools")
}

// --- Tool handlers ---

type processSummary struct {
	Name      string  `json:"name"`
	Namespace string  `json:"namespace,omitempty"`
	Status    string  `json:"status"`
	IsReady   string  `json:"is_ready"`
	Pid       int     `json:"pid"`
	Restarts  int     `json:"restarts"`
	ExitCode  int     `json:"exit_code"`
	AgeSec    float64 `json:"age_seconds"`
}

func (s *Server) toolListProcesses(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	states, err := s.runner.GetProcessesState()
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get processes state: %v", err), nil
	}
	out := struct {
		Processes []processSummary `json:"processes"`
	}{
		Processes: make([]processSummary, 0, len(states.States)),
	}
	for _, st := range states.States {
		out.Processes = append(out.Processes, processSummary{
			Name:      st.Name,
			Namespace: st.Namespace,
			Status:    st.Status,
			IsReady:   st.Health,
			Pid:       st.Pid,
			Restarts:  st.Restarts,
			ExitCode:  st.ExitCode,
			AgeSec:    st.Age.Seconds(),
		})
	}
	return mcp.NewToolResultJSON(out)
}

type processDetail struct {
	Name      string  `json:"name"`
	Namespace string  `json:"namespace,omitempty"`
	Status    string  `json:"status"`
	IsReady   string  `json:"is_ready"`
	Pid       int     `json:"pid"`
	Restarts  int     `json:"restarts"`
	ExitCode  int     `json:"exit_code"`
	AgeSec    float64 `json:"age_seconds"`
	CPU       float64 `json:"cpu"`
	Mem       int64   `json:"mem"`
	IsRunning bool    `json:"is_running"`
}

func (s *Server) toolGetProcess(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultErrorf("missing 'name': %v", err), nil
	}
	st, err := s.runner.GetProcessState(name)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get process state for %q: %v", name, err), nil
	}
	return mcp.NewToolResultJSON(processDetail{
		Name:      st.Name,
		Namespace: st.Namespace,
		Status:    st.Status,
		IsReady:   st.Health,
		Pid:       st.Pid,
		Restarts:  st.Restarts,
		ExitCode:  st.ExitCode,
		AgeSec:    st.Age.Seconds(),
		CPU:       st.CPU,
		Mem:       st.Mem,
		IsRunning: st.IsRunning,
	})
}

func (s *Server) toolGetLogs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultErrorf("missing 'name': %v", err), nil
	}
	lines := req.GetInt("lines", defaultGetLogsLines)
	if lines <= 0 {
		lines = defaultGetLogsLines
	}
	if lines > maxGetLogsLines {
		lines = maxGetLogsLines
	}
	offset := req.GetInt("offset", 0)
	if offset < 0 {
		offset = 0
	}

	logs, err := s.runner.GetProcessLog(name, offset, lines)
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get logs for %q: %v", name, err), nil
	}
	return mcp.NewToolResultJSON(struct {
		Name  string   `json:"name"`
		Lines []string `json:"lines"`
	}{Name: name, Lines: logs})
}

// searchHit is one result from search_logs. ChunkIdx is the line's 0-based
// index within the log chunk we searched (tail of log_limit lines per
// process), not an absolute position in the process's log buffer — lines
// may roll out between calls.
type searchHit struct {
	Process  string  `json:"process"`
	ChunkIdx int     `json:"chunk_idx"`
	Score    float64 `json:"score"`
	Text     string  `json:"text"`
}

func (s *Server) toolSearchLogs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultErrorf("missing 'query': %v", err), nil
	}
	topK := req.GetInt("top_k", defaultSearchTopK)
	if topK <= 0 {
		topK = defaultSearchTopK
	}
	if topK > maxSearchTopK {
		topK = maxSearchTopK
	}
	logLimit := req.GetInt("log_limit", defaultSearchLogLimit)
	if logLimit <= 0 {
		logLimit = defaultSearchLogLimit
	}
	if logLimit > maxSearchLogLimit {
		logLimit = maxSearchLogLimit
	}

	procNames, err := s.targetProcessNames(req.GetString("name", ""))
	if err != nil {
		return mcp.NewToolResultErrorf("%v", err), nil
	}

	// Gather log lines per process, tracking which doc belongs to which process.
	var (
		docs      [][]string
		lineTexts []string
		lineProcs []string
		chunkIdxs []int
	)
	for _, pname := range procNames {
		lines, err := s.runner.GetProcessLog(pname, 0, logLimit)
		if err != nil {
			log.Warn().Err(err).Str("process", pname).Msg("search_logs: skipping process")
			continue
		}
		for i, line := range lines {
			docs = append(docs, Tokenize(line))
			lineTexts = append(lineTexts, line)
			lineProcs = append(lineProcs, pname)
			chunkIdxs = append(chunkIdxs, i)
		}
	}

	queryTokens := Tokenize(query)
	corpus := NewCorpus(docs, 0, 0)
	hits := corpus.TopN(queryTokens, topK)

	out := struct {
		Query string      `json:"query"`
		Hits  []searchHit `json:"hits"`
	}{
		Query: query,
		Hits:  make([]searchHit, 0, len(hits)),
	}
	for _, h := range hits {
		out.Hits = append(out.Hits, searchHit{
			Process:  lineProcs[h.DocID],
			ChunkIdx: chunkIdxs[h.DocID],
			Score:    h.Score,
			Text:     lineTexts[h.DocID],
		})
	}
	return mcp.NewToolResultJSON(out)
}

// targetProcessNames returns a single-element slice if name is set, otherwise
// all known process names. Returns an error if name is set but unknown.
func (s *Server) targetProcessNames(name string) ([]string, error) {
	if name != "" {
		if _, err := s.runner.GetProcessState(name); err != nil {
			return nil, fmt.Errorf("unknown process %q: %w", name, err)
		}
		return []string{name}, nil
	}
	names, err := s.runner.GetLexicographicProcessNames()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}
	return names, nil
}

func (s *Server) toolStartProcess(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultErrorf("missing 'name': %v", err), nil
	}
	if err := s.runner.StartProcess(name); err != nil {
		return mcp.NewToolResultErrorf("failed to start %q: %v", name, err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("started %s", name)), nil
}

func (s *Server) toolStopProcess(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultErrorf("missing 'name': %v", err), nil
	}
	if err := s.runner.StopProcess(name); err != nil {
		return mcp.NewToolResultErrorf("failed to stop %q: %v", name, err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("stopped %s", name)), nil
}

func (s *Server) toolRestartProcess(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultErrorf("missing 'name': %v", err), nil
	}
	if err := s.runner.RestartProcess(name); err != nil {
		return mcp.NewToolResultErrorf("failed to restart %q: %v", name, err), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("restarted %s", name)), nil
}

func (s *Server) toolGetDependencyGraph(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	graph := types.BuildDependencyGraph(s.processes)

	states, err := s.runner.GetProcessesState()
	if err != nil {
		return mcp.NewToolResultErrorf("failed to get processes state: %v", err), nil
	}

	// Overlay live state in a single pass. graph.Nodes aliases the pointers in
	// graph.AllNodes, so mutating AllNodes propagates to Nodes automatically.
	for i := range states.States {
		st := &states.States[i]
		if node, ok := graph.AllNodes[st.Name]; ok {
			node.Status = st.Status
			node.IsReady = st.Health
		}
	}

	return mcp.NewToolResultJSON(graph)
}
