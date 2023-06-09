package client

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"io"
	"sync/atomic"
)

type LogClient struct {
	ws       *websocket.Conn
	Format   string
	isClosed atomic.Bool
}

func NewLogClient() *LogClient {
	return &LogClient{
		Format: "%s",
	}
}

func (l *LogClient) ReadProcessLogs(address string, port int, name string, offset int, follow bool, out io.StringWriter) (err error) {

	url := fmt.Sprintf("ws://%s:%d/process/logs/ws?name=%s&offset=%d&follow=%v", address, port, name, offset, follow)
	log.Info().Msgf("Connecting to %s", url)
	l.ws, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Error().Msgf("failed to dial to %s error: %v", url, err)
		return err
	}
	//defer l.ws.Close()
	done := make(chan struct{})

	go l.readLogs(done, l.ws, follow, out)

	/*for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			fmt.Println("interrupt")

			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}*/
	return nil
}

// CloseChannel Cleanly close the connection by sending a close message and then
// waiting (with timeout) for the server to close the connection.
func (l *LogClient) CloseChannel() error {
	err := l.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		fmt.Println("write close:", err)
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
