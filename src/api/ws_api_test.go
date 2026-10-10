package api

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/gorilla/websocket"
)

// backlogLines is well above the per-subscriber log channel capacity.
const backlogLines = 1000

func makeBacklog(prefix string, n int) []string {
	lines := make([]string, n)
	for i := range n {
		lines[i] = fmt.Sprintf("%s %d", prefix, i)
	}
	return lines
}

// logsMock serves the given backlogs; any other process name doesn't exist.
func logsMock(backlogs map[string][]string) *mockProject {
	return &mockProject{
		getLogsAndSubscribeFn: func(name string, observer pclog.LogObserver) error {
			backlog, ok := backlogs[name]
			if !ok {
				return errors.New("process " + name + " doesn't exist")
			}
			observer.SetLines(backlog)
			return nil
		},
	}
}

func dialLogsStream(t *testing.T, mock *mockProject, names string, follow bool) *websocket.Conn {
	t.Helper()
	srv := httptest.NewServer(setupRouter(mock))
	t.Cleanup(srv.Close)
	q := url.Values{}
	q.Set("name", names)
	q.Set("offset", fmt.Sprint(backlogLines))
	q.Set("follow", fmt.Sprint(follow))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/process/logs/ws?" + q.Encode()
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial %s: %v", wsURL, err)
	}
	t.Cleanup(func() { _ = ws.Close() })
	return ws
}

func readLogLines(t *testing.T, ws *websocket.Conn, n int) []string {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	lines := make([]string, 0, n)
	for len(lines) < n {
		var msg LogMessage
		if err := ws.ReadJSON(&msg); err != nil {
			t.Fatalf("got %d of %d log lines before read error: %v", len(lines), n, err)
		}
		lines = append(lines, msg.Message)
	}
	return lines
}

// readUntilClosed returns the lines received per process until the server
// closes the socket.
func readUntilClosed(t *testing.T, ws *websocket.Conn) map[string][]string {
	t.Helper()
	_ = ws.SetReadDeadline(time.Now().Add(5 * time.Second))
	lines := map[string][]string{}
	for {
		var msg LogMessage
		err := ws.ReadJSON(&msg)
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseAbnormalClosure) {
				t.Fatalf("expected the server to close the socket, got: %v", err)
			}
			return lines
		}
		lines[msg.ProcessName] = append(lines[msg.ProcessName], msg.Message)
	}
}

func assertLines(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHandleLogsStream_SendsFullBacklog(t *testing.T) {
	backlog := makeBacklog("line", backlogLines)
	ws := dialLogsStream(t, logsMock(map[string][]string{"proc": backlog}), "proc", false)

	got := readUntilClosed(t, ws)
	assertLines(t, got["proc"], backlog)
}

func TestHandleLogsStream_FollowSendsBacklogBeforeLiveLines(t *testing.T) {
	backlog := makeBacklog("line", backlogLines)
	mock := &mockProject{
		getLogsAndSubscribeFn: func(_ string, observer pclog.LogObserver) error {
			observer.SetLines(backlog)
			// A line written right after subscribing, while the backlog is
			// still being sent.
			_, _ = observer.WriteString("live 0")
			return nil
		},
	}
	ws := dialLogsStream(t, mock, "proc", true)

	assertLines(t, readLogLines(t, ws, backlogLines+1), append(backlog, "live 0"))
}

func TestHandleLogsStream_MultipleProcessesSendEveryBacklog(t *testing.T) {
	backlogs := map[string][]string{
		"a": makeBacklog("a", backlogLines),
		"b": makeBacklog("b", backlogLines),
	}
	ws := dialLogsStream(t, logsMock(backlogs), "a,b", false)

	got := readUntilClosed(t, ws)
	assertLines(t, got["a"], backlogs["a"])
	assertLines(t, got["b"], backlogs["b"])
}

func TestHandleLogsStream_SkipsUnknownProcess(t *testing.T) {
	for _, names := range []string{"a,nosuch", "nosuch,a"} {
		t.Run(names, func(t *testing.T) {
			backlogs := map[string][]string{"a": makeBacklog("a", backlogLines)}
			ws := dialLogsStream(t, logsMock(backlogs), names, false)

			got := readUntilClosed(t, ws)
			assertLines(t, got["a"], backlogs["a"])
		})
	}
}

func TestHandleLogsStream_OnlyUnknownProcessesCloseSocket(t *testing.T) {
	for _, follow := range []bool{false, true} {
		t.Run(fmt.Sprintf("follow=%v", follow), func(t *testing.T) {
			ws := dialLogsStream(t, logsMock(nil), "nosuch", follow)

			if got := readUntilClosed(t, ws); len(got) != 0 {
				t.Fatalf("expected no log lines, got %v", got)
			}
		})
	}
}
