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

// registerBuiltinTools registers the 8 Control MCP tools on the server.
func (s *Server) registerBuiltinTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("list_processes",
			mcp.WithDescription("List all processes managed by process-compose with a summary of their state."),
		),
		s.toolListProcesses,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_process",
			mcp.WithDescription("Get detailed status for a single process (status, PID, uptime, CPU, memory, restarts, exit code, namespace, readiness)."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the process.")),
		),
		s.toolGetProcess,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_logs",
			mcp.WithDescription("Return recent log lines for a process."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the process.")),
			mcp.WithNumber("lines", mcp.Description(fmt.Sprintf("Number of lines to return from the tail (default %d, cap %d).", defaultGetLogsLines, maxGetLogsLines))),
			mcp.WithNumber("offset", mcp.Description("Offset from the end of the log buffer (default 0).")),
		),
		s.toolGetLogs,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("search_logs",
			mcp.WithDescription("Search process logs for a query string using BM25 ranking."),
			mcp.WithString("query", mcp.Required(), mcp.Description("Search query.")),
			mcp.WithString("name", mcp.Description("Optional process name. If omitted, search across all processes.")),
			mcp.WithNumber("top_k", mcp.Description(fmt.Sprintf("Number of top results to return (default %d, cap %d).", defaultSearchTopK, maxSearchTopK))),
			mcp.WithNumber("log_limit", mcp.Description(fmt.Sprintf("Max number of recent log lines to index per process (default %d, cap %d).", defaultSearchLogLimit, maxSearchLogLimit))),
		),
		s.toolSearchLogs,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("start_process",
			mcp.WithDescription("Start a process by name."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the process.")),
		),
		s.toolStartProcess,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("stop_process",
			mcp.WithDescription("Stop a running process by name."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the process.")),
		),
		s.toolStopProcess,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("restart_process",
			mcp.WithDescription("Restart a process by name."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Name of the process.")),
		),
		s.toolRestartProcess,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_dependency_graph",
			mcp.WithDescription("Return the dependency graph of processes along with their current live status."),
		),
		s.toolGetDependencyGraph,
	)

	log.Info().Int("count", 8).Msg("Registered Control MCP tools")
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

type searchHit struct {
	Process string  `json:"process"`
	LineIdx int     `json:"line_idx"`
	Score   float64 `json:"score"`
	Text    string  `json:"text"`
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
		docs       [][]string
		lineTexts  []string
		lineProcs  []string
		lineIndex  []int
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
			lineIndex = append(lineIndex, i)
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
			Process: lineProcs[h.DocID],
			LineIdx: lineIndex[h.DocID],
			Score:   h.Score,
			Text:    lineTexts[h.DocID],
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

type depLink struct {
	Name           string             `json:"name"`
	Status         string             `json:"process_status"`
	IsReady        string             `json:"is_ready"`
	DependencyType string             `json:"dependency_type"`
	DependsOn      map[string]depLink `json:"depends_on,omitempty"`
}

type depNodeJSON struct {
	Name      string             `json:"name"`
	Status    string             `json:"process_status"`
	IsReady   string             `json:"is_ready"`
	DependsOn map[string]depLink `json:"depends_on,omitempty"`
}

func (s *Server) toolGetDependencyGraph(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	graph := types.BuildDependencyGraph(s.processes)

	// Overlay live state onto each node in AllNodes.
	for name, node := range graph.AllNodes {
		if st, err := s.runner.GetProcessState(name); err == nil && st != nil {
			node.Status = st.Status
			node.IsReady = st.Health
		}
	}

	nodes := make(map[string]depNodeJSON, len(graph.Nodes))
	for name, node := range graph.Nodes {
		nodes[name] = toDepNodeJSON(node)
	}

	return mcp.NewToolResultJSON(struct {
		Nodes map[string]depNodeJSON `json:"nodes"`
	}{Nodes: nodes})
}

func toDepNodeJSON(node *types.DependencyNode) depNodeJSON {
	out := depNodeJSON{
		Name:    node.Name,
		Status:  node.Status,
		IsReady: node.IsReady,
	}
	if len(node.DependsOn) > 0 {
		out.DependsOn = make(map[string]depLink, len(node.DependsOn))
		for depName, link := range node.DependsOn {
			out.DependsOn[depName] = toDepLink(link)
		}
	}
	return out
}

func toDepLink(link types.DependencyLink) depLink {
	out := depLink{
		DependencyType: link.Type,
	}
	if link.DependencyNode != nil {
		out.Name = link.Name
		out.Status = link.Status
		out.IsReady = link.IsReady
		if len(link.DependsOn) > 0 {
			out.DependsOn = make(map[string]depLink, len(link.DependsOn))
			for depName, sub := range link.DependsOn {
				out.DependsOn[depName] = toDepLink(sub)
			}
		}
	}
	return out
}
