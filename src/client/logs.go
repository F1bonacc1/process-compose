package client

import (
	"context"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"io"
	"net"
    "os"
	"sync/atomic"
)

type LogClient struct {
	ws         *websocket.Conn
	Format     string
	isClosed   atomic.Bool
	socketPath string
	address    string
}

func NewLogClient(address, socketPath string) *LogClient {
	return &LogClient{
		Format:     "%s",
		address:    address,
		socketPath: socketPath,
	}
}

func (l *LogClient) ReadProcessLogs(name string, offset int, follow bool, out io.StringWriter) (done chan struct{}, err error) {

	url := fmt.Sprintf("ws://%s/process/logs/ws?name=%s&offset=%d&follow=%v", l.address, name, offset, follow)
	log.Info().Msgf("Connecting to %s", url)

	dialer := websocket.DefaultDialer
	if l.address == "unix" {
		dialer.NetDialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, l.address, l.socketPath)
		}
	}
	l.ws, _, err = dialer.Dial(url, nil)

	if err != nil {
		log.Error().Msgf("failed to dial to %s error: %v", url, err)
		return done, err
	}
	//defer l.ws.Close()
	done = make(chan struct{})

	go l.readLogs(done, l.ws, follow, out)

	return done, nil
}

// CloseChannel Cleanly close the connection by sending a close message and then
// waiting (with timeout) for the server to close the connection.
func (l *LogClient) CloseChannel() error {
	err := l.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		fmt.Fprintln(os.Stderr, "write close:", err)
		return err
	}
	l.isClosed.Store(true)
	return l.ws.Close()
}

func (l *LogClient) readLogs(done chan struct{}, ws *websocket.Conn, follow bool, out io.StringWriter) {
	defer close(done)
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
		if len(message.ProcessName) > 0 {
			_, _ = out.WriteString(fmt.Sprintf(l.Format, message.Message))
		}
	}
}
