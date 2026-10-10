package client

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/gorilla/websocket"
)

// logsProject hands a fixed backlog to every log stream subscriber.
type logsProject struct {
	app.IProject
	lines []string
}

func (p *logsProject) GetLogsAndSubscribe(_ string, observer pclog.LogObserver) error {
	observer.SetLines(p.lines)
	return nil
}

func (p *logsProject) UnSubscribeLogger(string, pclog.LogObserver) error {
	return nil
}

// newUnixServer serves handler on a unix socket and returns the socket path.
// Over TCP a write to a connection the peer has already closed usually still
// succeeds, so the broken pipe these tests guard against only shows up on a
// unix socket.
func newUnixServer(t *testing.T, handler http.Handler) string {
	t.Helper()
	// t.TempDir embeds the test name, which can push the socket path past the
	// unix socket path length limit.
	dir, err := os.MkdirTemp("", "pc") //nolint:usetesting
	if err != nil {
		t.Fatalf("failed to create socket dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	sockPath := filepath.Join(dir, "pc.sock")
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("failed to listen on %s: %v", sockPath, err)
	}
	srv := httptest.NewUnstartedServer(handler)
	_ = srv.Listener.Close()
	srv.Listener = listener
	srv.Start()
	t.Cleanup(srv.Close)
	return sockPath
}

// A non-follow stream ends with the server closing the connection. Closing the
// client afterwards must not send a close message into the closed socket,
// which used to print "write close: ... broken pipe" on stderr.
func TestCloseChannel_AfterServerEndedStream(t *testing.T) {
	project := &logsProject{lines: []string{"line 1", "line 2"}}
	sockPath := newUnixServer(t, api.InitRoutes(false, api.NewPcApi(project)))
	logClient := NewLogClient("unix", sockPath)

	var got []string
	done, err := logClient.ReadProcessLogs("proc", len(project.lines), false, func(msg api.LogMessage) {
		got = append(got, msg.Message)
	})
	if err != nil {
		t.Fatalf("failed to read logs: %v", err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the server to end the stream")
	}
	if !slices.Equal(got, project.lines) {
		t.Fatalf("expected lines %v, got %v", project.lines, got)
	}

	restore := captureStderr(t)
	err = logClient.CloseChannel()
	stderr := restore()
	if err != nil {
		t.Fatalf("failed to close channel: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected nothing on stderr, got %q", stderr)
	}
}

// The remote TUI reuses one LogClient for every process it switches to.
// Closing a live stream must still send a close message after an earlier
// stream on the same client has ended.
func TestCloseChannel_ReusedClientSendsCloseMessage(t *testing.T) {
	upgrader := websocket.Upgrader{}
	closeErrs := make(chan error, 2)
	sockPath := newUnixServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			if _, _, err := ws.ReadMessage(); err != nil {
				closeErrs <- err
				return
			}
		}
	}))
	logClient := NewLogClient("unix", sockPath)

	for i := range 2 {
		done, err := logClient.ReadProcessLogs("proc", 0, true, func(api.LogMessage) {})
		if err != nil {
			t.Fatalf("stream %d: failed to read logs: %v", i, err)
		}
		if err = logClient.CloseChannel(); err != nil {
			t.Fatalf("stream %d: failed to close channel: %v", i, err)
		}
		select {
		case err = <-closeErrs:
		case <-time.After(5 * time.Second):
			t.Fatalf("stream %d: timed out waiting for the server to see the close", i)
		}
		if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			t.Fatalf("stream %d: expected a normal close message, got %v", i, err)
		}
		// Let the stream's reader exit before opening the next one, as it
		// has by the time a user switches to another process.
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("stream %d: timed out waiting for the reader to exit", i)
		}
	}
}

// captureStderr redirects os.Stderr into an in-memory pipe until the
// returned closure is called, at which point stderr is restored and the
// captured bytes are returned as a string. Calling the closure more than
// once is safe (subsequent calls return "").
func captureStderr(t *testing.T) func() string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = w

	var (
		once sync.Once
		out  string
	)
	return func() string {
		once.Do(func() {
			_ = w.Close()
			b, _ := io.ReadAll(r)
			_ = r.Close()
			os.Stderr = orig
			out = string(b)
		})
		return out
	}
}
