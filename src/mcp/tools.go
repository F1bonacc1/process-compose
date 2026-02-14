package mcp

import (
	"fmt"
	"strings"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog/log"
)

// handleToolInvocation handles an MCP tool invocation for a process
func (s *Server) handleToolInvocation(processName string, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Info().
		Str("process", processName).
		Interface("arguments", request.Params.Arguments).
		Msg("MCP tool invocation")

	// Get process config
	proc, ok := s.processes[processName]
	if !ok {
		return mcp.NewToolResultErrorf("process not found: %s", processName), nil
	}

	// Get mutex for queuing
	s.mutexesMu.RLock()
	mu, ok := s.processMutexes[processName]
	s.mutexesMu.RUnlock()
	if !ok {
		return mcp.NewToolResultErrorf("process mutex not found: %s", processName), nil
	}

	// Acquire lock to queue invocations
	mu.Lock()
	defer mu.Unlock()

	// Extract arguments from request using the GetArguments helper
	args := request.GetArguments()
	log.Debug().Str("process", processName).Interface("args", args).Msg("Extracted arguments from request")

	// Create substituted process config
	modifiedProc, err := SubstituteProcessConfig(proc, args)
	if err != nil {
		log.Error().Err(err).Str("process", processName).Msg("Failed to substitute arguments")
		return mcp.NewToolResultErrorf("failed to substitute arguments: %v", err), nil
	}

	// Check if substitution actually happened by looking for remaining @{...} patterns
	if strings.Contains(modifiedProc.Command, "@{") {
		log.Error().
			Str("process", processName).
			Str("command", modifiedProc.Command).
			Interface("args", args).
			Msg("Command still contains unsubstituted @{...} placeholders - check argument names match")
		return mcp.NewToolResultError("command contains unsubstituted placeholders - check that argument names in the request match those defined in the configuration"), nil
	}

	// Log the substituted command for debugging
	log.Debug().
		Str("process", processName).
		Str("original", proc.Command).
		Str("substituted", modifiedProc.Command).
		Interface("args", args).
		Msg("Command substitution result")

		// Update process info with modified config
	if err := s.runner.SetProcessInfo(modifiedProc); err != nil {
		log.Error().Err(err).Str("process", processName).Msg("Failed to set process info")
		return mcp.NewToolResultErrorf("failed to configure process: %v", err), nil
	}

	// Start the process
	log.Info().Str("process", processName).Msg("Starting MCP process")
	if err := s.runner.StartProcess(processName); err != nil {
		return mcp.NewToolResultErrorf("failed to start process: %v", err), nil
	}

	// Compute timeout for this process
	timeout := s.getTimeout(processName)

	// Wait for process to complete
	exitCode, output, err := s.waitForProcess(processName, timeout)
	if err != nil {
		return mcp.NewToolResultErrorf("error waiting for process: %v", err), nil
	}

	// Check exit code
	if exitCode != 0 {
		return mcp.NewToolResultErrorf("process exited with code %d\n\nOutput:\n%s", exitCode, output), nil
	}

	// Check if output is JSON and return appropriate result type
	if jsonData, isJSON := parseJSONIfValid(output); isJSON {
		return mcp.NewToolResultJSON(jsonData)
	}
	return mcp.NewToolResultText(output), nil
}

// waitForProcess waits for a process to complete and returns its output
func (s *Server) waitForProcess(processName string, timeout time.Duration) (int, string, error) {
	// Poll for process completion
	pollInterval := 100 * time.Millisecond

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ticker.C:
			// Check process state
			state, err := s.runner.GetProcessState(processName)
			if err != nil {
				// Process might not exist yet, keep waiting
				continue
			}

			// If process is completed or failed, get output
			if state.Status == types.ProcessStateCompleted ||
				state.Status == types.ProcessStateError ||
				state.Status == types.ProcessStateSkipped {
				// Get output from log buffer
				output := s.getProcessOutput(processName)
				return state.ExitCode, output, nil
			}

			// Process is still running, continue waiting

		case <-timeoutTimer.C:
			// Timeout - try to stop the process
			log.Warn().Str("process", processName).Msg("MCP process execution timeout")
			err := s.runner.StopProcess(processName)
			if err != nil {
				log.Error().Err(err).Str("process", processName).Msg("Failed to stop MCP process")
			}
			return -1, "", fmt.Errorf("process execution timeout")

		case <-s.ctx.Done():
			return -1, "", fmt.Errorf("server shutting down")
		}
	}
}

// getProcessOutput retrieves the output from a process's log buffer
func (s *Server) getProcessOutput(processName string) string {
	// Get log length
	logLength := s.runner.GetProcessLogLength(processName)
	if logLength == 0 {
		return ""
	}

	// Get all logs
	logs, err := s.runner.GetProcessLog(processName, 0, logLength)
	if err != nil {
		log.Error().Err(err).Str("process", processName).Msg("Failed to get process logs")
		return ""
	}

	return strings.Join(logs, "\n")
}

// getTimeout computes the timeout for a process
// Priority: 1) process-specific timeout, 2) global timeout, 3) default (5m)
func (s *Server) getTimeout(processName string) time.Duration {
	defaultTimeout := 5 * time.Minute

	// Check process-specific timeout
	proc, ok := s.processes[processName]
	if ok && proc.MCP != nil {
		if procTimeout, err := proc.MCP.GetTimeout(); err == nil && procTimeout > 0 {
			log.Debug().
				Str("process", processName).
				Dur("timeout", procTimeout).
				Msg("Using process-specific timeout")
			return procTimeout
		}
	}

	// Check global timeout
	if s.config != nil {
		if globalTimeout, err := s.config.GetTimeout(); err == nil && globalTimeout > 0 {
			log.Debug().
				Str("process", processName).
				Dur("timeout", globalTimeout).
				Msg("Using global MCP timeout")
			return globalTimeout
		}
	}

	// Return default
	log.Debug().
		Str("process", processName).
		Dur("timeout", defaultTimeout).
		Msg("Using default timeout")
	return defaultTimeout
}
