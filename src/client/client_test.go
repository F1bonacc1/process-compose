package client

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// fakeProject implements the subset of app.IProject these tests exercise;
// any other method call panics via the embedded nil interface.
type fakeProject struct {
	app.IProject
	restarted []string
	stopped   []string
}

func (f *fakeProject) RestartProcess(name string) error {
	f.restarted = append(f.restarted, name)
	return nil
}

func (f *fakeProject) StopProcess(name string) error {
	f.stopped = append(f.stopped, name)
	return nil
}

func (f *fakeProject) GetProcessState(name string) (*types.ProcessState, error) {
	return &types.ProcessState{Name: name}, nil
}

func newTestClient(t *testing.T, project app.IProject) *PcClient {
	t.Helper()
	srv := httptest.NewServer(api.InitRoutes(false, api.NewPcApi(project)))
	t.Cleanup(srv.Close)
	host, portStr, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("failed to parse test server address %s: %v", srv.URL, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse test server port %s: %v", portStr, err)
	}
	return NewTcpClient(host, port, 100)
}

func TestRestartProcess_NameWithSlash(t *testing.T) {
	fake := &fakeProject{}
	c := newTestClient(t, fake)
	if err := c.RestartProcess("@scope/pkg"); err != nil {
		t.Fatalf("failed to restart process: %v", err)
	}
	if len(fake.restarted) != 1 || fake.restarted[0] != "@scope/pkg" {
		t.Fatalf("expected restart of %q, got %v", "@scope/pkg", fake.restarted)
	}
}

func TestStopProcess_NameWithPlusAndSlash(t *testing.T) {
	fake := &fakeProject{}
	c := newTestClient(t, fake)
	if err := c.StopProcess("a+b/c"); err != nil {
		t.Fatalf("failed to stop process: %v", err)
	}
	if len(fake.stopped) != 1 || fake.stopped[0] != "a+b/c" {
		t.Fatalf("expected stop of %q, got %v", "a+b/c", fake.stopped)
	}
}

func TestGetProcessState_NameWithSlash(t *testing.T) {
	fake := &fakeProject{}
	c := newTestClient(t, fake)
	state, err := c.GetProcessState("@scope/pkg")
	if err != nil {
		t.Fatalf("failed to get process state: %v", err)
	}
	if state.Name != "@scope/pkg" {
		t.Fatalf("expected process name %q, got %q", "@scope/pkg", state.Name)
	}
}

func TestEscapePathSegment(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"web", "web"},
		{"@scope/pkg", "@scope%2Fpkg"},
		{"a+b/c", "a%2Bb%2Fc"},
		{"a b", "a%20b"},
	}
	for _, tt := range tests {
		if got := escapePathSegment(tt.name); got != tt.expected {
			t.Errorf("escapePathSegment(%q) = %q, expected %q", tt.name, got, tt.expected)
		}
	}
}

func TestParseErrorResponse(t *testing.T) {
	tests := []struct {
		desc     string
		status   int
		body     string
		expected string
	}{
		{"json error body", http.StatusBadRequest, `{"error":"no such process"}`, "no such process"},
		{"plain text 404", http.StatusNotFound, "404 page not found", "test failed: HTTP 404: 404 page not found"},
	}
	for _, tt := range tests {
		resp := &http.Response{
			StatusCode: tt.status,
			Body:       io.NopCloser(strings.NewReader(tt.body)),
		}
		if got := parseErrorResponse(resp, "test"); got.Error() != tt.expected {
			t.Errorf("%s: got %q, expected %q", tt.desc, got.Error(), tt.expected)
		}
	}
}
