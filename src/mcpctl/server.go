// Package mcpctl implements the Control MCP server: an MCP server that exposes
// tools for AI agents to introspect and control the running process-compose
// instance itself (list/get/start/stop/restart processes, read logs, search
// logs, read the dependency graph).
package mcpctl

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
)

// ProcessRunner is the subset of *app.ProjectRunner used by the Control MCP tools.
type ProcessRunner interface {
	GetLexicographicProcessNames() ([]string, error)
	GetProcessesState() (*types.ProcessesState, error)
	GetProcessState(name string) (*types.ProcessState, error)
	StartProcess(name string) error
	StopProcess(name string) error
	RestartProcess(name string) error
	GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error)
}

type Server struct {
	mcpServer *server.MCPServer
	runner    ProcessRunner
	config    *types.MCPCtlServerConfig
	processes types.Processes
	ctx       context.Context
	cancel    context.CancelFunc
	sseServer *server.SSEServer
}

// NewServer constructs a Control MCP server and registers its tools.
// Returns nil if cfg is nil or not enabled.
func NewServer(runner ProcessRunner, cfg *types.MCPCtlServerConfig, processes types.Processes) *Server {
	if cfg == nil || !cfg.IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Snapshot the process map so get_dependency_graph can iterate it without
	// racing any runtime mutation of the project's process set.
	procSnapshot := make(types.Processes, len(processes))
	for k, v := range processes {
		procSnapshot[k] = v
	}

	s := &Server{
		mcpServer: server.NewMCPServer("process-compose-mcpctl", "1.0.0"),
		runner:    runner,
		config:    cfg,
		processes: procSnapshot,
		ctx:       ctx,
		cancel:    cancel,
	}

	s.registerBuiltinTools()
	return s
}

// Start starts the SSE server.
func (s *Server) Start() error {
	if s == nil || s.config == nil || !s.config.IsEnabled() {
		return nil
	}
	if !s.config.IsSSE() {
		return fmt.Errorf("unsupported mcpctl transport: %s", s.config.Transport)
	}

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Info().
		Str("address", addr).
		Msg("Starting Control MCP server with SSE transport")

	s.sseServer = server.NewSSEServer(s.mcpServer)
	go func() {
		err := s.sseServer.Start(addr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("Control MCP SSE server error")
		}
	}()

	return nil
}

// Stop tears down the server. For SSE, we call sseServer.Shutdown so the
// listener goroutine does not leak.
func (s *Server) Stop() error {
	if s == nil {
		return nil
	}
	log.Info().Msg("Stopping Control MCP server")
	s.cancel()
	if s.sseServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.sseServer.Shutdown(ctx); err != nil {
			return err
		}
	}
	return nil
}
