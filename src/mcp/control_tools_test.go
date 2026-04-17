package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/mark3labs/mcp-go/mcp"
)

// fakeRunner records calls and returns canned data for test assertions.
type fakeRunner struct {
	startCalled    string
	startErr       error
	stopCalled     []string
	stopResult     map[string]string
	stopErr        error
	restartCalled  string
	restartErr     error
	scaleCalled    string
	scaleValue     int
	scaleErr       error
	getStateName   string
	getStateResult *types.ProcessState
	getStateErr    error
	listResult     *types.ProcessesState
	listErr        error
	portsName      string
	portsResult    *types.ProcessPorts
	portsErr       error
	projectMem     bool
	projectResult  *types.ProjectState
	projectErr     error
	logName        string
	logOffset      int
	logLimit       int
	logResult      []string
	logErr         error
	truncCalled    string
	truncErr       error
}

func (f *fakeRunner) StartProcess(name string) error {
	f.startCalled = name
	return f.startErr
}
func (f *fakeRunner) StopProcess(_ string) error { return nil }
func (f *fakeRunner) StopProcesses(names []string) (map[string]string, error) {
	f.stopCalled = names
	return f.stopResult, f.stopErr
}
func (f *fakeRunner) RestartProcess(name string) error {
	f.restartCalled = name
	return f.restartErr
}
func (f *fakeRunner) ScaleProcess(name string, scale int) error {
	f.scaleCalled = name
	f.scaleValue = scale
	return f.scaleErr
}
func (f *fakeRunner) GetProcessState(name string) (*types.ProcessState, error) {
	f.getStateName = name
	return f.getStateResult, f.getStateErr
}
func (f *fakeRunner) GetProcessesState() (*types.ProcessesState, error) {
	return f.listResult, f.listErr
}
func (f *fakeRunner) GetProcessPorts(name string) (*types.ProcessPorts, error) {
	f.portsName = name
	return f.portsResult, f.portsErr
}
func (f *fakeRunner) GetProjectState(checkMem bool) (*types.ProjectState, error) {
	f.projectMem = checkMem
	return f.projectResult, f.projectErr
}
func (f *fakeRunner) GetProcessLog(name string, offsetFromEnd, limit int) ([]string, error) {
	f.logName = name
	f.logOffset = offsetFromEnd
	f.logLimit = limit
	return f.logResult, f.logErr
}
func (f *fakeRunner) GetProcessLogLength(_ string) int       { return 0 }
func (f *fakeRunner) SetProcessInfo(_ *types.ProcessConfig) error { return nil }
func (f *fakeRunner) TruncateProcessLogs(name string) error {
	f.truncCalled = name
	return f.truncErr
}

func newTestServer(runner ProcessRunner) *Server {
	return NewServer(runner, &types.MCPServerConfig{ExposeControlTools: true})
}

func callRequest(args map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

// resultIsError returns true if the tool result was a user-facing error.
func resultIsError(r *mcp.CallToolResult) bool {
	return r != nil && r.IsError
}

// resultText returns the concatenated text payload of a tool result.
func resultText(r *mcp.CallToolResult) string {
	if r == nil {
		return ""
	}
	var sb strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

func TestProcessStart(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner := &fakeRunner{}
		s := newTestServer(runner)
		res, err := s.handleProcessStart(context.Background(), callRequest(map[string]any{"name": "web"}))
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if resultIsError(res) {
			t.Fatalf("unexpected tool error: %s", resultText(res))
		}
		if runner.startCalled != "web" {
			t.Errorf("StartProcess called with %q, want %q", runner.startCalled, "web")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		runner := &fakeRunner{}
		s := newTestServer(runner)
		res, err := s.handleProcessStart(context.Background(), callRequest(map[string]any{}))
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !resultIsError(res) {
			t.Fatal("expected tool error for missing name")
		}
		if runner.startCalled != "" {
			t.Errorf("StartProcess should not be called when name missing")
		}
	})

	t.Run("runner error", func(t *testing.T) {
		runner := &fakeRunner{startErr: errors.New("boom")}
		s := newTestServer(runner)
		res, _ := s.handleProcessStart(context.Background(), callRequest(map[string]any{"name": "web"}))
		if !resultIsError(res) {
			t.Fatal("expected tool error from runner failure")
		}
		if !strings.Contains(resultText(res), "boom") {
			t.Errorf("expected error text to include runner err, got %q", resultText(res))
		}
	})
}

func TestProcessStop(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner := &fakeRunner{stopResult: map[string]string{"a": "ok", "b": "ok"}}
		s := newTestServer(runner)
		res, err := s.handleProcessStop(context.Background(), callRequest(map[string]any{
			"names": []any{"a", "b"},
		}))
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if resultIsError(res) {
			t.Fatalf("unexpected tool error: %s", resultText(res))
		}
		if len(runner.stopCalled) != 2 || runner.stopCalled[0] != "a" || runner.stopCalled[1] != "b" {
			t.Errorf("StopProcesses called with %v", runner.stopCalled)
		}
		var got map[string]string
		if err := json.Unmarshal([]byte(resultText(res)), &got); err != nil {
			t.Fatalf("result is not valid JSON: %v", err)
		}
		if got["a"] != "ok" {
			t.Errorf("unexpected JSON payload: %v", got)
		}
	})

	t.Run("missing names", func(t *testing.T) {
		runner := &fakeRunner{}
		s := newTestServer(runner)
		res, _ := s.handleProcessStop(context.Background(), callRequest(map[string]any{}))
		if !resultIsError(res) {
			t.Fatal("expected tool error for missing names")
		}
	})

	t.Run("empty names", func(t *testing.T) {
		runner := &fakeRunner{}
		s := newTestServer(runner)
		res, _ := s.handleProcessStop(context.Background(), callRequest(map[string]any{
			"names": []any{},
		}))
		if !resultIsError(res) {
			t.Fatal("expected tool error for empty names")
		}
	})
}

func TestProcessRestart(t *testing.T) {
	runner := &fakeRunner{}
	s := newTestServer(runner)
	res, _ := s.handleProcessRestart(context.Background(), callRequest(map[string]any{"name": "web"}))
	if resultIsError(res) {
		t.Fatalf("unexpected tool error: %s", resultText(res))
	}
	if runner.restartCalled != "web" {
		t.Errorf("RestartProcess called with %q", runner.restartCalled)
	}
}

func TestProcessScale(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runner := &fakeRunner{}
		s := newTestServer(runner)
		res, _ := s.handleProcessScale(context.Background(), callRequest(map[string]any{
			"name":  "web",
			"scale": float64(3),
		}))
		if resultIsError(res) {
			t.Fatalf("unexpected tool error: %s", resultText(res))
		}
		if runner.scaleCalled != "web" || runner.scaleValue != 3 {
			t.Errorf("ScaleProcess called with name=%q, scale=%d", runner.scaleCalled, runner.scaleValue)
		}
	})

	t.Run("missing scale", func(t *testing.T) {
		runner := &fakeRunner{}
		s := newTestServer(runner)
		res, _ := s.handleProcessScale(context.Background(), callRequest(map[string]any{"name": "web"}))
		if !resultIsError(res) {
			t.Fatal("expected tool error for missing scale")
		}
	})
}

func TestProcessGet(t *testing.T) {
	state := &types.ProcessState{Name: "web", Status: "Running"}
	runner := &fakeRunner{getStateResult: state}
	s := newTestServer(runner)
	res, _ := s.handleProcessGet(context.Background(), callRequest(map[string]any{"name": "web"}))
	if resultIsError(res) {
		t.Fatalf("unexpected tool error: %s", resultText(res))
	}
	var got types.ProcessState
	if err := json.Unmarshal([]byte(resultText(res)), &got); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if got.Name != "web" || got.Status != "Running" {
		t.Errorf("unexpected state payload: %+v", got)
	}
}

func TestProcessList(t *testing.T) {
	runner := &fakeRunner{listResult: &types.ProcessesState{States: []types.ProcessState{
		{Name: "a", Status: "Running"},
		{Name: "b", Status: "Completed"},
	}}}
	s := newTestServer(runner)
	res, _ := s.handleProcessList(context.Background(), callRequest(nil))
	if resultIsError(res) {
		t.Fatalf("unexpected tool error: %s", resultText(res))
	}
	var got types.ProcessesState
	if err := json.Unmarshal([]byte(resultText(res)), &got); err != nil {
		t.Fatalf("result not valid JSON: %v", err)
	}
	if len(got.States) != 2 {
		t.Errorf("expected 2 states, got %d", len(got.States))
	}
}

func TestProcessPorts(t *testing.T) {
	runner := &fakeRunner{portsResult: &types.ProcessPorts{Name: "web", TcpPorts: []uint16{8080, 9090}}}
	s := newTestServer(runner)
	res, _ := s.handleProcessPorts(context.Background(), callRequest(map[string]any{"name": "web"}))
	if resultIsError(res) {
		t.Fatalf("unexpected tool error: %s", resultText(res))
	}
	if runner.portsName != "web" {
		t.Errorf("GetProcessPorts called with %q", runner.portsName)
	}
}

func TestProcessLogs(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		runner := &fakeRunner{logResult: []string{"line1", "line2"}}
		s := newTestServer(runner)
		res, _ := s.handleProcessLogs(context.Background(), callRequest(map[string]any{"name": "web"}))
		if resultIsError(res) {
			t.Fatalf("unexpected tool error: %s", resultText(res))
		}
		if runner.logName != "web" || runner.logLimit != 100 || runner.logOffset != 0 {
			t.Errorf("GetProcessLog called with name=%q offset=%d limit=%d", runner.logName, runner.logOffset, runner.logLimit)
		}
		var got processLogsResult
		if err := json.Unmarshal([]byte(resultText(res)), &got); err != nil {
			t.Fatalf("result not valid JSON: %v", err)
		}
		if got.Name != "web" || len(got.Lines) != 2 {
			t.Errorf("unexpected payload: %+v", got)
		}
	})

	t.Run("custom tail", func(t *testing.T) {
		runner := &fakeRunner{logResult: []string{"x"}}
		s := newTestServer(runner)
		_, _ = s.handleProcessLogs(context.Background(), callRequest(map[string]any{
			"name":            "web",
			"tail":            float64(50),
			"offset_from_end": float64(10),
		}))
		if runner.logLimit != 50 || runner.logOffset != 10 {
			t.Errorf("expected limit=50 offset=10, got limit=%d offset=%d", runner.logLimit, runner.logOffset)
		}
	})

	t.Run("invalid tail clamped", func(t *testing.T) {
		runner := &fakeRunner{logResult: []string{}}
		s := newTestServer(runner)
		_, _ = s.handleProcessLogs(context.Background(), callRequest(map[string]any{
			"name": "web",
			"tail": float64(-5),
		}))
		if runner.logLimit != 100 {
			t.Errorf("expected tail clamped to default 100, got %d", runner.logLimit)
		}
	})
}

func TestProcessLogsTruncate(t *testing.T) {
	runner := &fakeRunner{}
	s := newTestServer(runner)
	res, _ := s.handleProcessLogsTruncate(context.Background(), callRequest(map[string]any{"name": "web"}))
	if resultIsError(res) {
		t.Fatalf("unexpected tool error: %s", resultText(res))
	}
	if runner.truncCalled != "web" {
		t.Errorf("TruncateProcessLogs called with %q", runner.truncCalled)
	}
}

func TestProjectState(t *testing.T) {
	t.Run("default no memory", func(t *testing.T) {
		runner := &fakeRunner{projectResult: &types.ProjectState{ProjectName: "demo"}}
		s := newTestServer(runner)
		res, _ := s.handleProjectState(context.Background(), callRequest(nil))
		if resultIsError(res) {
			t.Fatalf("unexpected tool error: %s", resultText(res))
		}
		if runner.projectMem {
			t.Error("expected with_memory=false by default")
		}
	})

	t.Run("with memory", func(t *testing.T) {
		runner := &fakeRunner{projectResult: &types.ProjectState{}}
		s := newTestServer(runner)
		_, _ = s.handleProjectState(context.Background(), callRequest(map[string]any{"with_memory": true}))
		if !runner.projectMem {
			t.Error("expected with_memory=true")
		}
	})
}

func TestProjectIsReady(t *testing.T) {
	t.Run("all ready", func(t *testing.T) {
		runner := &fakeRunner{listResult: &types.ProcessesState{States: []types.ProcessState{
			{Name: "a", Status: types.ProcessStateRunning, IsRunning: true, Health: types.ProcessHealthReady},
			{Name: "b", Status: types.ProcessStateCompleted, ExitCode: 0, Health: types.ProcessHealthReady},
		}}}
		s := newTestServer(runner)
		res, _ := s.handleProjectIsReady(context.Background(), callRequest(nil))
		if resultIsError(res) {
			t.Fatalf("unexpected tool error: %s", resultText(res))
		}
		var got projectReadyResult
		if err := json.Unmarshal([]byte(resultText(res)), &got); err != nil {
			t.Fatalf("result not valid JSON: %v", err)
		}
		if !got.Ready {
			t.Errorf("expected ready=true, got payload %+v", got)
		}
		if got.Total != 2 {
			t.Errorf("expected total=2, got %d", got.Total)
		}
	})

	t.Run("some not ready", func(t *testing.T) {
		runner := &fakeRunner{listResult: &types.ProcessesState{States: []types.ProcessState{
			{Name: "a", Status: types.ProcessStateRunning, IsRunning: true, Health: types.ProcessHealthReady},
			{Name: "b", Status: types.ProcessStatePending},
		}}}
		s := newTestServer(runner)
		res, _ := s.handleProjectIsReady(context.Background(), callRequest(nil))
		var got projectReadyResult
		if err := json.Unmarshal([]byte(resultText(res)), &got); err != nil {
			t.Fatalf("result not valid JSON: %v", err)
		}
		if got.Ready {
			t.Errorf("expected ready=false, got payload %+v", got)
		}
		if len(got.NotReady) != 1 || got.NotReady[0].Name != "b" {
			t.Errorf("expected only b in not_ready, got %+v", got.NotReady)
		}
	})
}

func TestRegisterControlTools(t *testing.T) {
	runner := &fakeRunner{}
	s := newTestServer(runner)
	if err := s.RegisterControlTools(); err != nil {
		t.Fatalf("RegisterControlTools failed: %v", err)
	}
}
