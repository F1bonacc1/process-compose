// Package mcpctl implements the Control MCP server: an MCP server that exposes
// tools for AI agents to introspect and control the running process-compose
// instance itself (list/get/start/stop/restart processes, read logs, search
// logs, read the dependency graph).
package mcpctl

import (
	"context"
	"fmt"
	"io"
	stdLog "log"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog/log"
)

// Transport names used in MCPCtlServerConfig.
const (
	TransportSSE   = "sse"
	TransportStdio = "stdio"
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
	stdin     io.Reader
	stdout    io.Writer
}

// NewServer constructs a Control MCP server and registers its tools.
// Returns nil if cfg is nil or not enabled.
func NewServer(runner ProcessRunner, cfg *types.MCPCtlServerConfig, processes types.Processes) *Server {
	if cfg == nil || !cfg.IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		mcpServer: server.NewMCPServer("process-compose-mcpctl", "1.0.0"),
		runner:    runner,
		config:    cfg,
		processes: processes,
		ctx:       ctx,
		cancel:    cancel,
	}

	s.registerBuiltinTools()
	return s
}

// SetStdio sets stdin/stdout for the stdio transport.
func (s *Server) SetStdio(stdin io.Reader, stdout io.Writer) {
	s.stdin = stdin
	s.stdout = stdout
}

// Start starts the server with the configured transport.
func (s *Server) Start() error {
	if s == nil || s.config == nil || !s.config.IsEnabled() {
		return nil
	}

	log.Info().
		Str("transport", s.config.Transport).
		Msg("Starting Control MCP server")

	if s.config.IsStdio() {
		return s.startStdio()
	}
	if s.config.IsSSE() {
		return s.startSSE()
	}
	return fmt.Errorf("unknown mcpctl transport: %s (only %q and %q are supported)", s.config.Transport, TransportSSE, TransportStdio)
}

func (s *Server) startStdio() error {
	if s.stdin == nil || s.stdout == nil {
		return fmt.Errorf("mcpctl stdio transport requires stdin and stdout to be set")
	}

	log.Info().Msg("Starting Control MCP server with stdio transport")

	stdioServer := server.NewStdioServer(s.mcpServer)
	stdioServer.SetErrorLogger(stdLog.New(io.Discard, "", 0))

	go func() {
		if err := stdioServer.Listen(s.ctx, s.stdin, s.stdout); err != nil {
			log.Error().Err(err).Msg("Control MCP stdio server error")
		}
	}()

	return nil
}

func (s *Server) startSSE() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	log.Info().
		Str("address", addr).
		Msg("Starting Control MCP server with SSE transport")

	s.sseServer = server.NewSSEServer(s.mcpServer)
	go func() {
		if err := s.sseServer.Start(addr); err != nil {
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
