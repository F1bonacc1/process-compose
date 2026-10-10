package client

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/gorilla/websocket"
)

// TestCloseChannel_NoStderrAfterPeerClose exercises the end-to-end shape of
// the fix from the caller's point of view: ReadProcessLogs → wait on done →
// CloseChannel, against a real WebSocket server that closes the connection
// the way handleLog does for a non-follow tail. A successful log fetch must
// not emit anything on stderr; historically the courtesy close-frame write
// leaked a "write close: ... broken pipe" line here.
func TestCloseChannel_NoStderrAfterPeerClose(t *testing.T) {
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		// Mirror handleLog's non-follow teardown: close the underlying
		// conn without a graceful WS close frame.
		_ = ws.Close()
	}))
	t.Cleanup(srv.Close)

	client := NewLogClient(strings.TrimPrefix(srv.URL, "http://"), "")
	restore := captureStderr(t)

	done, err := client.ReadProcessLogs("any", 0, false, func(api.LogMessage) {})
	if err != nil {
		restore()
		t.Fatalf("ReadProcessLogs: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		restore()
		t.Fatal("timed out waiting for peer close to surface as done")
	}

	if err := client.CloseChannel(); err != nil {
		restore()
		t.Fatalf("CloseChannel returned error: %v", err)
	}
	if got := restore(); got != "" {
		t.Fatalf("CloseChannel wrote %q to stderr; expected nothing", got)
	}
}

// TestCloseChannel_SkipsWriteWhenReadClosed guards the primary fix: when the
// read loop has already observed a peer-closed connection (which non-follow
// requests always do — handleLog on the server side hard-closes the conn
// once the requested tail has streamed), CloseChannel must not try to send
// its courtesy close frame. Writing to the torn-down conn is what produced
// the spurious "write close: ... broken pipe" line on stderr that made
// successful `process logs` invocations look like failures.
func TestCloseChannel_SkipsWriteWhenReadClosed(t *testing.T) {
	// Stand up a real WebSocket server just so we can get a genuine
	// *websocket.Conn on the client side; the server's behavior after
	// upgrade doesn't matter because we drive the close path manually.
	upgrader := websocket.Upgrader{}
	handshakeDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		close(handshakeDone)
		// Hold the server side open until the test finishes; the
		// client-side conn is what we're driving.
		<-r.Context().Done()
		_ = ws.Close()
	}))
	t.Cleanup(srv.Close)

	dialer := websocket.DefaultDialer
	ws, _, err := dialer.Dial("ws://"+strings.TrimPrefix(srv.URL, "http://")+"/", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	select {
	case <-handshakeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handshake")
	}

	client := NewLogClient("unused", "")
	client.ws = ws
	client.readDone.Store(true) // simulate readLogs having observed peer close

	// Close the underlying conn from the client side so a hypothetical
	// WriteMessage would fail loudly (net.ErrClosed). If CloseChannel
	// respects readDone it never attempts the write and stderr stays clean.
	_ = ws.NetConn().Close()

	restore := captureStderr(t)
	if err := client.CloseChannel(); err != nil {
		restore()
		t.Fatalf("CloseChannel returned error: %v", err)
	}
	if got := restore(); got != "" {
		t.Fatalf("CloseChannel wrote %q to stderr; expected nothing", got)
	}
}

// TestCloseChannel_SwallowsPeerClosedWrite guards the defense-in-depth
// half of the fix: on the rare path where readDone is not yet set but the
// close-frame write still fails because the peer beat us to close, the
// error must be classified benign and not surface on stderr.
func TestCloseChannel_SwallowsPeerClosedWrite(t *testing.T) {
	upgrader := websocket.Upgrader{}
	handshakeDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		close(handshakeDone)
		<-r.Context().Done()
		_ = ws.Close()
	}))
	t.Cleanup(srv.Close)

	ws, _, err := websocket.DefaultDialer.Dial("ws://"+strings.TrimPrefix(srv.URL, "http://")+"/", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	select {
	case <-handshakeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handshake")
	}

	client := NewLogClient("unused", "")
	client.ws = ws
	// readDone deliberately left false — this test exercises the second
	// guard: WriteMessage is attempted and returns a peer-closed error.
	_ = ws.NetConn().Close()

	restore := captureStderr(t)
	if err := client.CloseChannel(); err != nil {
		restore()
		t.Fatalf("CloseChannel returned error: %v", err)
	}
	if got := restore(); got != "" {
		t.Fatalf("CloseChannel wrote %q to stderr; expected nothing", got)
	}
}

func TestIsPeerClosedErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"EPIPE", syscall.EPIPE, true},
		{"EOF", io.EOF, true},
		{"net.ErrClosed", net.ErrClosed, true},
		{"wrapped EPIPE", &net.OpError{Op: "write", Err: syscall.EPIPE}, true},
		{"wrapped ErrClosed", &net.OpError{Op: "write", Err: net.ErrClosed}, true},
		{"websocket normal close", &websocket.CloseError{Code: websocket.CloseNormalClosure}, true},
		{"websocket abnormal close", &websocket.CloseError{Code: websocket.CloseAbnormalClosure}, true},
		{"unrelated", errors.New("boom"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isPeerClosedErr(c.err); got != c.want {
				t.Fatalf("isPeerClosedErr(%v) = %v, want %v", c.err, got, c.want)
			}
		})
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
