package mcp

import (
	"context"
	"fmt"
	"io"
	stdLog "log"
	"sync"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
)

// ProcessRunner defines the interface needed from ProjectRunner
type ProcessRunner interface {
	StartProcess(name string) error
	StopProcess(name string) error
	StopProcesses(names []string) (map[string]string, error)
	RestartProcess(name string) error
	ScaleProcess(name string, scale int) error
	GetProcessState(name string) (*types.ProcessState, error)
	GetProcessesState() (*types.ProcessesState, error)
	GetProcessPorts(name string) (*types.ProcessPorts, error)
	GetProjectState(checkMem bool) (*types.ProjectState, error)
	GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error)
	GetProcessLogLength(name string) int
	SetProcessInfo(config *types.ProcessConfig) error
	TruncateProcessLogs(name string) error
}

// Server wraps the MCP server and integrates with process-compose
type Server struct {
	mcpServer      *server.MCPServer
	runner         ProcessRunner
	config         *types.MCPServerConfig
	processes      map[string]*types.ProcessConfig
	processMutexes map[string]*sync.Mutex // For queuing concurrent invocations
	mutexesMu      sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	stdin          io.Reader
	stdout         io.Writer
}

// NewServer creates a new MCP server instance
func NewServer(runner ProcessRunner, config *types.MCPServerConfig) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	// Create MCP server
	s := server.NewMCPServer("process-compose", "1.0.0")

	return &Server{
		mcpServer:      s,
		runner:         runner,
		config:         config,
		processes:      make(map[string]*types.ProcessConfig),
		processMutexes: make(map[string]*sync.Mutex),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// SetStdio sets the stdin and stdout for the MCP server when using stdio transport
func (s *Server) SetStdio(stdin io.Reader, stdout io.Writer) {
	s.stdin = stdin
	s.stdout = stdout
}

// RegisterProcess registers a process as an MCP tool or resource
func (s *Server) RegisterProcess(proc *types.ProcessConfig) error {
	if proc.MCP == nil {
		return nil
	}

	// Store process config
	s.processes[proc.Name] = proc

	// Create mutex for queuing
	s.mutexesMu.Lock()
	s.processMutexes[proc.Name] = &sync.Mutex{}
	s.mutexesMu.Unlock()

	if proc.MCP.IsTool() {
		return s.registerTool(proc)
	} else if proc.MCP.IsResource() {
		return s.registerResource(proc)
	}

	return fmt.Errorf("unknown MCP type for process %s", proc.Name)
}

// registerTool registers a process as an MCP tool
func (s *Server) registerTool(proc *types.ProcessConfig) error {
	// Build tool options
	options := []mcp.ToolOption{
		mcp.WithDescription(proc.Description),
	}

	// Add arguments as tool parameters
	for _, arg := range proc.MCP.Arguments {
		var opts []mcp.PropertyOption
		if arg.Description != "" {
			opts = append(opts, mcp.Description(arg.Description))
		}
		if arg.Required {
			opts = append(opts, mcp.Required())
		}

		var opt mcp.ToolOption
		switch arg.Type {
		case types.MCPArgTypeString:
			opt = mcp.WithString(arg.Name, opts...)
		case types.MCPArgTypeInteger:
			opt = mcp.WithNumber(arg.Name, opts...)
		case types.MCPArgTypeNumber:
			opt = mcp.WithNumber(arg.Name, opts...)
		case types.MCPArgTypeBoolean:
			opt = mcp.WithBoolean(arg.Name, opts...)
		}
		options = append(options, opt)
	}

	// Create and register the tool
	tool := mcp.NewTool(proc.Name, options...)

	s.mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return s.handleToolInvocation(proc.Name, request)
	})

	log.Info().Str("process", proc.Name).Msg("Registered MCP tool")
	return nil
}

// registerResource registers a process as an MCP resource
func (s *Server) registerResource(proc *types.ProcessConfig) error {
	uri := fmt.Sprintf("process://%s", proc.Name)

	s.mcpServer.AddResource(mcp.NewResource(
		uri,
		proc.Name,
		mcp.WithResourceDescription(proc.Description),
	), func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		return s.handleResourceRequest(proc.Name, request)
	})

	log.Info().Str("process", proc.Name).Str("uri", uri).Msg("Registered MCP resource")
	return nil
}

// Start starts the MCP server with the configured transport
func (s *Server) Start() error {
	if s.config == nil || !s.config.IsEnabled() {
		log.Info().Msg("MCP server is not enabled")
		return nil
	}

	log.Info().
		Str("transport", s.config.Transport).
		Msg("Starting MCP server")

	if s.config.IsStdio() {
		return s.startStdio()
	}

	if s.config.IsSSE() {
		return s.startSSE()
	}

	return fmt.Errorf("unknown MCP transport: %s (only 'sse' and 'stdio' are supported)", s.config.Transport)
}

// startStdio starts the MCP server with stdio transport
func (s *Server) startStdio() error {
	if s.stdin == nil || s.stdout == nil {
		return fmt.Errorf("stdio transport requires stdin and stdout to be set")
	}

	log.Info().Msg("Starting MCP server with stdio transport")

	// Create Stdio server
	stdioServer := server.NewStdioServer(s.mcpServer)

	// Redirect StdioServer's internal error logger to discard (we use zerolog)
	// This prevents the library from writing to stderr which might be used for
	// something else or closed. We rely on our own logging.
	stdioServer.SetErrorLogger(stdLog.New(io.Discard, "", 0))

	// Stdio transport blocks Listen, so run in goroutine
	go func() {
		if err := stdioServer.Listen(s.ctx, s.stdin, s.stdout); err != nil {
			log.Error().Err(err).Msg("MCP stdio server error")
		}
	}()

	return nil
}

// startSSE starts the MCP server with SSE transport
func (s *Server) startSSE() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Info().
		Str("address", addr).
		Msg("Starting MCP server with SSE transport")

	// Create SSE server
	sseServer := server.NewSSEServer(s.mcpServer)

	// SSE transport blocks, so run in goroutine
	go func() {
		if err := sseServer.Start(addr); err != nil {
			log.Error().Err(err).Msg("MCP SSE server error")
		}
	}()

	return nil
}

// Stop gracefully stops the MCP server
func (s *Server) Stop() error {
	log.Info().Msg("Stopping MCP server")
	s.cancel()
	return nil
}

// GetRegisteredProcesses returns the list of registered MCP processes
func (s *Server) GetRegisteredProcesses() []string {
	processes := make([]string, 0, len(s.processes))
	for name := range s.processes {
		processes = append(processes, name)
	}
	return processes
}

// IsMCPProcess returns true if the given process name is MCP-registered
func (s *Server) IsMCPProcess(name string) bool {
	_, ok := s.processes[name]
	return ok
}
