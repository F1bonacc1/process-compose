package mcp

import (
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog/log"
)

// handleResourceRequest handles an MCP resource request for a process
func (s *Server) handleResourceRequest(processName string, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	log.Info().
		Str("process", processName).
		Str("uri", request.Params.URI).
		Msg("MCP resource request")

	// Get process config
	_, ok := s.processes[processName]
	if !ok {
		return nil, fmt.Errorf("process not found: %s", processName)
	}

	// Get mutex for queuing
	s.mutexesMu.RLock()
	mu, ok := s.processMutexes[processName]
	s.mutexesMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("process mutex not found: %s", processName)
	}

	// Acquire lock to queue invocations
	mu.Lock()
	defer mu.Unlock()

	// Start the process
	log.Info().Str("process", processName).Msg("Starting MCP resource process")
	if err := s.runner.TruncateProcessLogs(processName); err != nil {
		log.Error().Err(err).Str("process", processName).Msg("Failed to truncate process logs")
	}
	if err := s.runner.StartProcess(processName); err != nil {
		return nil, fmt.Errorf("failed to start process: %v", err)
	}

	// Compute timeout for this process
	timeout := s.getTimeout(processName)

	// Wait for process to complete
	exitCode, output, err := s.waitForProcess(processName, timeout)
	if err != nil {
		return nil, fmt.Errorf("error waiting for process: %v", err)
	}

	// Check exit code
	if exitCode != 0 {
		return nil, fmt.Errorf("process exited with code %d\n\nOutput:\n%s", exitCode, output)
	}

	// Detect if output is JSON and set MIME type accordingly
	mimeType := "text/plain"
	if _, isJSON := parseJSONIfValid(output); isJSON {
		mimeType = "application/json"
		log.Debug().Str("process", processName).Msg("Detected JSON output, setting MIME type to application/json")
	}

	// Return successful result as text resource
	content := mcp.TextResourceContents{
		URI:      request.Params.URI,
		MIMEType: mimeType,
		Text:     output,
	}

	return []mcp.ResourceContents{content}, nil
}
