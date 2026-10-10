package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	neturl "net/url"
	"os"
	"strconv"
	"sync/atomic"
	"syscall"

	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

type LogClient struct {
	ws               *websocket.Conn
	Format           string
	isClosed         atomic.Bool
	readDone         atomic.Bool
	socketPath       string
	address          string
	PrintProcessName bool
}

func NewLogClient(address, socketPath string) *LogClient {
	return &LogClient{
		Format:           "%s",
		PrintProcessName: false,
		address:          address,
		socketPath:       socketPath,
	}
}

func (l *LogClient) ReadProcessLogs(name string, offset int, follow bool, fn func(api.LogMessage)) (done chan struct{}, err error) {

	q := neturl.Values{}
	q.Set("name", name)
	q.Set("offset", strconv.Itoa(offset))
	q.Set("follow", strconv.FormatBool(follow))
	url := fmt.Sprintf("ws://%s/process/logs/ws?%s", l.address, q.Encode())
	log.Info().Msgf("Connecting to %s", url)

	dialer := websocket.DefaultDialer
	if l.address == "unix" {
		dialer.NetDialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, l.address, l.socketPath)
		}
	}

	var header http.Header
	token := config.GetApiToken()
	if token != "" {
		header = make(http.Header)
		header.Set(config.TokenHeader, token)
	}

	var resp *http.Response
	l.ws, resp, err = dialer.Dial(url, header)

	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			log.Fatal().Msgf("authentication failed: invalid or missing %s", config.EnvVarApiToken)
		}
		log.Error().Msgf("failed to dial to %s error: %v", url, err)
		return done, fmt.Errorf("failed to connect to %s: %w", l.address, err)
	}
	//defer l.ws.Close()
	done = make(chan struct{})

	go l.readLogs(done, l.ws, follow, fn)

	return done, nil
}

// CloseChannel Cleanly close the connection by sending a close message and then
// waiting (with timeout) for the server to close the connection.
//
// Non-follow requests always reach this via <-done after readLogs has already
// observed the server-sent close, at which point the underlying conn is torn
// down and a courtesy close frame from us races into an EPIPE. Skip the write
// when the read side has finished, and treat "peer already closed" write
// errors as benign so users don't see a spurious "write close: ... broken
// pipe" on stderr after a successful log tail. Errors from the final
// ws.Close are also swallowed — they signal an already-torn-down conn, not
// a client fault worth surfacing.
func (l *LogClient) CloseChannel() error {
	if l.ws == nil {
		return nil
	}
	// Forget the connection before closing it, so a repeated close is a no-op
	// instead of a "websocket: close sent" error.
	ws := l.ws
	l.ws = nil
	l.isClosed.Store(true)
	if l.readDone.Load() {
		_ = ws.Close()
		return nil
	}
	err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		_ = ws.Close()
		if isPeerClosedErr(err) {
			return nil
		}
		fmt.Fprintln(os.Stderr, "write close:", err)
		return err
	}
	_ = ws.Close()
	return nil
}

// isPeerClosedErr reports whether err is the benign consequence of the peer
// having closed the connection while we were sending the courtesy close
// frame — EPIPE from the raw socket, EOF/net.ErrClosed from the closed
// conn, or a WebSocket close frame received in reply.
func isPeerClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.EPIPE) || errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
		return true
	}
	var closeErr *websocket.CloseError
	return errors.As(err, &closeErr)
}

func (l *LogClient) readLogs(done chan struct{}, ws *websocket.Conn, follow bool, fn func(api.LogMessage)) {
	// LIFO defer order: mark readDone before signalling done, so a
	// CloseChannel racing on <-done observes the flag and skips its
	// close-frame write.
	defer close(done)
	defer l.readDone.Store(true)
	for {
		var message api.LogMessage
		if err := ws.ReadJSON(&message); err != nil {
			if !follow && websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				return
			}
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			if l.isClosed.Load() {
				return
			}
			log.Error().Msgf("failed to read message: %v", err)
			return
		}
		if message.ProcessName != "" {
			fn(message)
		}
	}
}

func (p *PcClient) getProcessLog(name string, offsetFromEnd, limit int) ([]string, error) {
	url := fmt.Sprintf("http://%s/process/logs/%s/%d/%d", p.address, escapePathSegment(name), offsetFromEnd, limit)
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseErrorResponse(resp, fmt.Sprintf("get process %s logs", name))
	}
	var logsResp api.LogsResponse
	if err = json.NewDecoder(resp.Body).Decode(&logsResp); err != nil {
		return nil, err
	}
	return logsResp.Logs, nil
}

func (p *PcClient) truncateProcessLogs(name string) error {
	url := fmt.Sprintf("http://%s/process/logs/%s", p.address, escapePathSegment(name))
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return parseErrorResponse(resp, fmt.Sprintf("truncate process %s logs", name))
}
