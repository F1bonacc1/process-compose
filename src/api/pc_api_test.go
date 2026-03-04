package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRouter(mock *mockProject) *gin.Engine {
	handler := NewPcApi(mock)
	r := gin.New()

	r.GET("/live", handler.IsAlive)
	r.GET("/processes", handler.GetProcesses)
	r.GET("/process/:name", handler.GetProcess)
	r.GET("/process/info/:name", handler.GetProcessInfo)
	r.POST("/process", handler.UpdateProcess)
	r.GET("/process/ports/:name", handler.GetProcessPorts)
	r.GET("/process/logs/:name/:endOffset/:limit", handler.GetProcessLogs)
	r.DELETE("/process/logs/:name", handler.TruncateProcessLogs)
	r.PATCH("/process/stop/:name", handler.StopProcess)
	r.PATCH("/processes/stop", handler.StopProcesses)
	r.POST("/process/start/:name", handler.StartProcess)
	r.POST("/process/restart/:name", handler.RestartProcess)
	r.POST("/project/stop", handler.ShutDownProject)
	r.POST("/project", handler.UpdateProject)
	r.POST("/project/configuration", handler.ReloadProject)
	r.GET("/project/name", handler.GetProjectName)
	r.GET("/project/state", handler.GetProjectState)
	r.POST("/namespace/start/:name", handler.StartNamespace)
	r.POST("/namespace/stop/:name", handler.StopNamespace)
	r.POST("/namespace/restart/:name", handler.RestartNamespace)
	r.GET("/namespaces", handler.GetNamespaces)
	r.PATCH("/process/scale/:name/:scale", handler.ScaleProcess)
	r.GET("/graph", handler.GetDependencyGraph)

	return r
}

func performRequest(r *gin.Engine, method, path string, body string) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, _ := http.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response JSON: %v\nbody: %s", err, w.Body.String())
	}
	return result
}

// --- IsAlive ---

func TestIsAlive(t *testing.T) {
	r := setupRouter(&mockProject{})
	w := performRequest(r, http.MethodGet, "/live", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	if result["status"] != "alive" {
		t.Fatalf("expected status=alive, got %v", result["status"])
	}
}

// --- GetProcess ---

func TestGetProcess_Success(t *testing.T) {
	mock := &mockProject{
		getProcessStateFn: func(name string) (*types.ProcessState, error) {
			return &types.ProcessState{Name: name, Status: "Running"}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/process/web", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	if result["name"] != "web" {
		t.Fatalf("expected name=web, got %v", result["name"])
	}
}

func TestGetProcess_NotFound(t *testing.T) {
	mock := &mockProject{
		getProcessStateFn: func(name string) (*types.ProcessState, error) {
			return nil, errors.New("process not found")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/process/missing", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetProcessInfo ---

func TestGetProcessInfo_Success(t *testing.T) {
	mock := &mockProject{
		getProcessInfoFn: func(name string) (*types.ProcessConfig, error) {
			return &types.ProcessConfig{Name: name, Command: "echo hi"}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/process/info/web", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetProcessInfo_NotFound(t *testing.T) {
	mock := &mockProject{
		getProcessInfoFn: func(name string) (*types.ProcessConfig, error) {
			return nil, errors.New("not found")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/process/info/missing", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetProcesses ---

func TestGetProcesses_Success(t *testing.T) {
	mock := &mockProject{
		getProcessesStateFn: func() (*types.ProcessesState, error) {
			return &types.ProcessesState{
				States: []types.ProcessState{
					{Name: "a", Status: "Running"},
					{Name: "b", Status: "Completed"},
				},
			}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/processes", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetProcesses_Empty(t *testing.T) {
	mock := &mockProject{
		getProcessesStateFn: func() (*types.ProcessesState, error) {
			return &types.ProcessesState{States: []types.ProcessState{}}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/processes", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetProcesses_Error(t *testing.T) {
	mock := &mockProject{
		getProcessesStateFn: func() (*types.ProcessesState, error) {
			return nil, errors.New("internal error")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/processes", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetProcessLogs ---

func TestGetProcessLogs_Success(t *testing.T) {
	mock := &mockProject{
		getProcessLogFn: func(name string, offset, limit int) ([]string, error) {
			return []string{"line1", "line2"}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/process/logs/web/0/10", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	logs, ok := result["logs"].([]interface{})
	if !ok || len(logs) != 2 {
		t.Fatalf("expected 2 log lines, got %v", result["logs"])
	}
}

func TestGetProcessLogs_InvalidOffset(t *testing.T) {
	r := setupRouter(&mockProject{})
	w := performRequest(r, http.MethodGet, "/process/logs/web/abc/10", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetProcessLogs_InvalidLimit(t *testing.T) {
	r := setupRouter(&mockProject{})
	w := performRequest(r, http.MethodGet, "/process/logs/web/0/abc", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetProcessLogs_NotFound(t *testing.T) {
	mock := &mockProject{
		getProcessLogFn: func(name string, offset, limit int) ([]string, error) {
			return nil, errors.New("process not found")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/process/logs/missing/0/10", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- TruncateProcessLogs ---

func TestTruncateProcessLogs_Success(t *testing.T) {
	mock := &mockProject{
		truncateProcessLogsFn: func(name string) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodDelete, "/process/logs/web", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	if result["name"] != "web" {
		t.Fatalf("expected name=web, got %v", result["name"])
	}
}

func TestTruncateProcessLogs_NotFound(t *testing.T) {
	mock := &mockProject{
		truncateProcessLogsFn: func(name string) error { return errors.New("not found") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodDelete, "/process/logs/missing", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- StopProcess ---

func TestStopProcess_Success(t *testing.T) {
	mock := &mockProject{
		stopProcessFn: func(name string) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPatch, "/process/stop/web", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	if result["name"] != "web" {
		t.Fatalf("expected name=web, got %v", result["name"])
	}
}

func TestStopProcess_Error(t *testing.T) {
	mock := &mockProject{
		stopProcessFn: func(name string) error { return errors.New("cannot stop") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPatch, "/process/stop/web", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- StopProcesses ---

func TestStopProcesses_Success(t *testing.T) {
	mock := &mockProject{
		stopProcessesFn: func(names []string) (map[string]string, error) {
			return map[string]string{"web": "stopped"}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPatch, "/processes/stop", `["web"]`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestStopProcesses_Partial(t *testing.T) {
	mock := &mockProject{
		stopProcessesFn: func(names []string) (map[string]string, error) {
			return map[string]string{"web": "stopped"}, errors.New("partial failure")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPatch, "/processes/stop", `["web","db"]`)
	if w.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d", w.Code)
	}
}

func TestStopProcesses_TotalFailure(t *testing.T) {
	mock := &mockProject{
		stopProcessesFn: func(names []string) (map[string]string, error) {
			return map[string]string{}, errors.New("total failure")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPatch, "/processes/stop", `["web"]`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestStopProcesses_BadJSON(t *testing.T) {
	r := setupRouter(&mockProject{})
	w := performRequest(r, http.MethodPatch, "/processes/stop", `not json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- StartProcess ---

func TestStartProcess_Success(t *testing.T) {
	mock := &mockProject{
		startProcessFn: func(name string) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/process/start/web", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	if result["name"] != "web" {
		t.Fatalf("expected name=web, got %v", result["name"])
	}
}

func TestStartProcess_Error(t *testing.T) {
	mock := &mockProject{
		startProcessFn: func(name string) error { return errors.New("already running") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/process/start/web", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- RestartProcess ---

func TestRestartProcess_Success(t *testing.T) {
	mock := &mockProject{
		restartProcessFn: func(name string) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/process/restart/web", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRestartProcess_Error(t *testing.T) {
	mock := &mockProject{
		restartProcessFn: func(name string) error { return errors.New("restart failed") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/process/restart/web", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- ScaleProcess ---

func TestScaleProcess_Success(t *testing.T) {
	mock := &mockProject{
		scaleProcessFn: func(name string, scale int) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPatch, "/process/scale/web/3", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestScaleProcess_BadScale(t *testing.T) {
	r := setupRouter(&mockProject{})
	w := performRequest(r, http.MethodPatch, "/process/scale/web/abc", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestScaleProcess_Error(t *testing.T) {
	mock := &mockProject{
		scaleProcessFn: func(name string, scale int) error { return errors.New("scale failed") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPatch, "/process/scale/web/3", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- StartNamespace ---

func TestStartNamespace_Success(t *testing.T) {
	mock := &mockProject{
		startNamespaceFn: func(ns string) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/namespace/start/default", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestStartNamespace_Error(t *testing.T) {
	mock := &mockProject{
		startNamespaceFn: func(ns string) error { return errors.New("ns error") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/namespace/start/bad", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- StopNamespace ---

func TestStopNamespace_Success(t *testing.T) {
	mock := &mockProject{
		stopNamespaceFn: func(ns string) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/namespace/stop/default", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestStopNamespace_Error(t *testing.T) {
	mock := &mockProject{
		stopNamespaceFn: func(ns string) error { return errors.New("ns error") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/namespace/stop/bad", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- RestartNamespace ---

func TestRestartNamespace_Success(t *testing.T) {
	mock := &mockProject{
		restartNamespaceFn: func(ns string) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/namespace/restart/default", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRestartNamespace_Error(t *testing.T) {
	mock := &mockProject{
		restartNamespaceFn: func(ns string) error { return errors.New("ns error") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/namespace/restart/bad", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetNamespaces ---

func TestGetNamespaces_Success(t *testing.T) {
	mock := &mockProject{
		getNamespacesFn: func() ([]string, error) {
			return []string{"default", "staging"}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/namespaces", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result []string
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 namespaces, got %d", len(result))
	}
}

func TestGetNamespaces_Error(t *testing.T) {
	mock := &mockProject{
		getNamespacesFn: func() ([]string, error) {
			return nil, errors.New("error")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/namespaces", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetProjectName ---

func TestGetProjectName_Success(t *testing.T) {
	mock := &mockProject{
		getProjectNameFn: func() (string, error) {
			return "my-project", nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/project/name", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	if result["projectName"] != "my-project" {
		t.Fatalf("expected projectName=my-project, got %v", result["projectName"])
	}
}

func TestGetProjectName_Error(t *testing.T) {
	mock := &mockProject{
		getProjectNameFn: func() (string, error) {
			return "", errors.New("no name")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/project/name", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetProcessPorts ---

func TestGetProcessPorts_Success(t *testing.T) {
	mock := &mockProject{
		getProcessPortsFn: func(name string) (*types.ProcessPorts, error) {
			return &types.ProcessPorts{Name: name, TcpPorts: []uint16{8080}}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/process/ports/web", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetProcessPorts_Error(t *testing.T) {
	mock := &mockProject{
		getProcessPortsFn: func(name string) (*types.ProcessPorts, error) {
			return nil, errors.New("not found")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/process/ports/missing", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- ShutDownProject ---

func TestShutDownProject(t *testing.T) {
	called := false
	mock := &mockProject{
		shutDownProjectFn: func() error { called = true; return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/project/stop", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	result := parseJSON(t, w)
	if result["status"] != "stopped" {
		t.Fatalf("expected status=stopped, got %v", result["status"])
	}
	if !called {
		t.Fatal("expected ShutDownProject to be called")
	}
}

// --- UpdateProject ---

func TestUpdateProject_Success(t *testing.T) {
	mock := &mockProject{
		updateProjectFn: func(p *types.Project) (map[string]string, error) {
			return map[string]string{"web": "updated"}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/project", `{"processes":{}}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUpdateProject_Partial(t *testing.T) {
	mock := &mockProject{
		updateProjectFn: func(p *types.Project) (map[string]string, error) {
			return map[string]string{"web": "updated"}, errors.New("partial")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/project", `{"processes":{}}`)
	if w.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d", w.Code)
	}
}

func TestUpdateProject_Failure(t *testing.T) {
	mock := &mockProject{
		updateProjectFn: func(p *types.Project) (map[string]string, error) {
			return map[string]string{}, errors.New("failed")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/project", `{"processes":{}}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateProject_BadJSON(t *testing.T) {
	r := setupRouter(&mockProject{})
	w := performRequest(r, http.MethodPost, "/project", `not json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- UpdateProcess ---

func TestUpdateProcess_Success(t *testing.T) {
	mock := &mockProject{
		updateProcessFn: func(p *types.ProcessConfig) error { return nil },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/process", `{"name":"web","command":"echo hi"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestUpdateProcess_Error(t *testing.T) {
	mock := &mockProject{
		updateProcessFn: func(p *types.ProcessConfig) error { return errors.New("update failed") },
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/process", `{"name":"web","command":"echo hi"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateProcess_BadJSON(t *testing.T) {
	r := setupRouter(&mockProject{})
	w := performRequest(r, http.MethodPost, "/process", `not json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetProjectState ---

func TestGetProjectState_Success(t *testing.T) {
	mock := &mockProject{
		getProjectStateFn: func(checkMem bool) (*types.ProjectState, error) {
			return &types.ProjectState{ProcessNum: 5, ProjectName: "test"}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/project/state", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetProjectState_WithMemory(t *testing.T) {
	var gotCheckMem bool
	mock := &mockProject{
		getProjectStateFn: func(checkMem bool) (*types.ProjectState, error) {
			gotCheckMem = checkMem
			return &types.ProjectState{}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/project/state?withMemory=true", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !gotCheckMem {
		t.Fatal("expected checkMem=true to be passed")
	}
}

func TestGetProjectState_Error(t *testing.T) {
	mock := &mockProject{
		getProjectStateFn: func(checkMem bool) (*types.ProjectState, error) {
			return nil, errors.New("state error")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/project/state", "")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// --- ReloadProject ---

func TestReloadProject_Success(t *testing.T) {
	mock := &mockProject{
		reloadProjectFn: func() (map[string]string, error) {
			return map[string]string{"web": "reloaded"}, nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/project/configuration", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestReloadProject_Partial(t *testing.T) {
	mock := &mockProject{
		reloadProjectFn: func() (map[string]string, error) {
			return map[string]string{"web": "ok"}, errors.New("partial")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/project/configuration", "")
	if w.Code != http.StatusMultiStatus {
		t.Fatalf("expected 207, got %d", w.Code)
	}
}

func TestReloadProject_Failure(t *testing.T) {
	mock := &mockProject{
		reloadProjectFn: func() (map[string]string, error) {
			return map[string]string{}, errors.New("failed")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodPost, "/project/configuration", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetDependencyGraph ---

func TestGetDependencyGraph_Success(t *testing.T) {
	mock := &mockProject{
		getDependencyGraphFn: func() (*types.DependencyGraph, error) {
			return types.NewDependencyGraph(), nil
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/graph", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetDependencyGraph_Error(t *testing.T) {
	mock := &mockProject{
		getDependencyGraphFn: func() (*types.DependencyGraph, error) {
			return nil, errors.New("graph error")
		},
	}
	r := setupRouter(mock)
	w := performRequest(r, http.MethodGet, "/graph", "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
