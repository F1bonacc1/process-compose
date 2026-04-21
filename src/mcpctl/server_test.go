package mcpctl

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/mark3labs/mcp-go/mcp"
)

// fakeRunner is a minimal ProcessRunner for handler tests.
type fakeRunner struct {
	names  []string
	states map[string]*types.ProcessState
	logs   map[string][]string
}

func (f *fakeRunner) GetLexicographicProcessNames() ([]string, error) { return f.names, nil }

func (f *fakeRunner) GetProcessesState() (*types.ProcessesState, error) {
	states := make([]types.ProcessState, 0, len(f.names))
	for _, n := range f.names {
		if st, ok := f.states[n]; ok {
			states = append(states, *st)
		}
	}
	return &types.ProcessesState{States: states}, nil
}

func (f *fakeRunner) GetProcessState(name string) (*types.ProcessState, error) {
	if st, ok := f.states[name]; ok {
		return st, nil
	}
	return nil, fmt.Errorf("process not found: %s", name)
}

func (f *fakeRunner) StartProcess(string) error   { return nil }
func (f *fakeRunner) StopProcess(string) error    { return nil }
func (f *fakeRunner) RestartProcess(string) error { return nil }

func (f *fakeRunner) GetProcessLog(name string, _, limit int) ([]string, error) {
	lines := f.logs[name]
	if limit < len(lines) {
		lines = lines[len(lines)-limit:]
	}
	return lines, nil
}

func TestNewManagerDisabledReturnsNil(t *testing.T) {
	if m := NewManager(&fakeRunner{}, nil, nil); m != nil {
		t.Errorf("nil config: expected nil manager, got %v", m)
	}
	if m := NewManager(&fakeRunner{}, &types.MCPCtlServerConfig{}, nil); m != nil {
		t.Errorf("empty config: expected nil manager, got %v", m)
	}
	// Nil manager must be safe to Start/Stop.
	var m *Manager
	if err := m.Start(); err != nil {
		t.Errorf("nil Manager.Start = %v, want nil", err)
	}
	if err := m.Stop(); err != nil {
		t.Errorf("nil Manager.Stop = %v, want nil", err)
	}
}

func TestNewManagerEnabledReturnsNonNil(t *testing.T) {
	cfg := &types.MCPCtlServerConfig{Host: "localhost", Port: 11001}
	m := NewManager(&fakeRunner{}, cfg, types.Processes{})
	if m == nil {
		t.Fatal("expected non-nil manager for enabled config")
	}
}

// TestGetDependencyGraphOverlaysLiveState verifies the AllNodes/Nodes aliasing
// claim in the handler: mutating AllNodes also mutates Nodes (because Nodes
// holds the same *DependencyNode pointers).
func TestGetDependencyGraphOverlaysLiveState(t *testing.T) {
	procs := types.Processes{
		"a": types.ProcessConfig{Name: "a"},
		"b": types.ProcessConfig{
			Name: "b",
			DependsOn: types.DependsOnConfig{
				"a": types.ProcessDependency{Condition: types.ProcessConditionStarted},
			},
		},
	}
	runner := &fakeRunner{
		names: []string{"a", "b"},
		states: map[string]*types.ProcessState{
			"a": {Name: "a", Status: "Running", Health: "Ready"},
			"b": {Name: "b", Status: "Completed", Health: "-"},
		},
	}
	srv := NewServer(runner, &types.MCPCtlServerConfig{Host: "localhost", Port: 11001}, procs)
	if srv == nil {
		t.Fatal("NewServer returned nil")
	}

	res, err := srv.toolGetDependencyGraph(context.Background(), mcp.CallToolRequest{})
	if err != nil {
		t.Fatalf("handler err: %v", err)
	}
	if res.IsError {
		t.Fatalf("handler returned error result: %+v", res.Content)
	}

	var payload struct {
		Nodes map[string]struct {
			Name      string `json:"name"`
			Status    string `json:"process_status"`
			IsReady   string `json:"is_ready"`
			DependsOn map[string]struct {
				Name    string `json:"name"`
				Status  string `json:"process_status"`
				IsReady string `json:"is_ready"`
			} `json:"depends_on"`
		} `json:"nodes"`
	}
	text := extractJSONText(t, res)
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("unmarshal: %v (raw=%s)", err, text)
	}

	// b is the only leaf; verify its overlay + dep link overlay.
	b, ok := payload.Nodes["b"]
	if !ok {
		t.Fatalf("missing node 'b' in %+v", payload.Nodes)
	}
	if b.Status != "Completed" || b.IsReady != "-" {
		t.Errorf("b overlay wrong: status=%s is_ready=%s", b.Status, b.IsReady)
	}
	a, ok := b.DependsOn["a"]
	if !ok {
		t.Fatalf("missing depends_on 'a' in %+v", b.DependsOn)
	}
	if a.Status != "Running" || a.IsReady != "Ready" {
		t.Errorf("a overlay wrong via DependsOn: status=%s is_ready=%s", a.Status, a.IsReady)
	}
}

func extractJSONText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("no TextContent in result")
	return ""
}
