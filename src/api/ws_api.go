package api

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{}

// @Schemes
// @Id                    LogsStream
// @Summary               Stream process logs over WebSocket
// @Description           Upgrades HTTP to WebSocket and streams JSON log messages. Each message is api.LogMessage.
// @Tags                  Process
// @Produce               json
// @Param                 name   query   string true  "Comma-separated process names to stream"
// @Param                 offset query   int    true  "Offset from the end of the log"
// @Param                 follow query   bool   false "If true, continue streaming new lines"
// @Success               101 "Switching Protocols"
// @Failure               400 {object} api.ErrorResponse
// @Router                /process/logs/ws [get]
func (api *PcApi) HandleLogsStream(c *gin.Context) {
	processNamesStr := c.Query("name")
	processNames := strings.Split(processNamesStr, ",")
	follow := c.Query("follow") == "true"
	endOffset, err := strconv.Atoi(c.Query("offset"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	done := make(chan struct{})
	wsWriteMtx := &sync.Mutex{}
	if follow {
		go handleIncoming(ws, done)
	}
	// Streams of several processes share ws, so it is closed once all of them
	// have ended rather than by the first one to finish.
	var streams sync.WaitGroup
	for _, processName := range processNames {
		if processName == "" {
			continue
		}
		logChan := make(chan LogMessage, 256)
		// The backlog is handed to handleLog in one piece instead of being
		// queued line by line on logChan: SetLines runs under the log buffer's
		// lock, so it must not block, and a backlog larger than logChan would
		// otherwise be mostly dropped.
		backlogChan := make(chan []string, 1)
		chanCloseMtx := &sync.Mutex{}
		isChannelClosed := false
		closeLogChan := func() {
			chanCloseMtx.Lock()
			defer chanCloseMtx.Unlock()
			if !isChannelClosed {
				close(logChan)
				isChannelClosed = true
			}
		}
		dropped := &atomic.Uint64{}
		warnedOnce := &atomic.Bool{}
		enqueue := func(msg LogMessage) bool {
			chanCloseMtx.Lock()
			defer chanCloseMtx.Unlock()
			if isChannelClosed {
				return false
			}
			select {
			case logChan <- msg:
			default:
				dropped.Add(1)
				if warnedOnce.CompareAndSwap(false, true) {
					log.Warn().Str("process", processName).Msg("ws subscriber backpressured; dropping log lines")
				}
			}
			return true
		}
		connector := pclog.NewConnector(
			func(messages []string) {
				// SetLines is called once per subscription, so this never
				// drops; the select only guarantees it never blocks.
				select {
				case backlogChan <- messages:
				default:
				}
				if !follow {
					closeLogChan()
				}
			},
			func(message string) (n int, err error) {
				msg := LogMessage{
					Message:     message,
					ProcessName: processName,
				}
				if !enqueue(msg) {
					return 0, nil
				}
				return len(message), nil
			},
			endOffset)
		streams.Go(func() {
			api.handleLog(ws, processName, connector, backlogChan, logChan, done, wsWriteMtx, dropped, closeLogChan)
		})

		err = api.project.GetLogsAndSubscribe(processName, connector)
		if err != nil {
			log.Err(err).Msg("Failed to subscribe to logger")
			// End this stream instead of leaving handleLog waiting forever,
			// and keep streaming the other processes.
			close(backlogChan)
			continue
		}
	}
	go func() {
		streams.Wait()
		_ = ws.Close()
	}()
}

func (api *PcApi) handleLog(
	ws *websocket.Conn,
	procName string,
	connector *pclog.Connector,
	backlogChan <-chan []string,
	logChan chan LogMessage,
	done chan struct{},
	wsWriteMtx *sync.Mutex,
	dropped *atomic.Uint64,
	closeLogChan func(),
) {
	defer func() {
		if count := dropped.Load(); count > 0 {
			log.Warn().Str("process", procName).Uint64("dropped", count).
				Msg("ws subscriber disconnected after dropped lines")
		}
	}()
	defer func(project app.IProject, name string, observer pclog.LogObserver) {
		err := project.UnSubscribeLogger(name, observer)
		if err != nil {
			log.Err(err).Msg("Failed to unsubscribe from logger")
		}
	}(api.project, procName, connector)
	// Send the backlog before any live line queued on logChan meanwhile.
	select {
	case lines, ok := <-backlogChan:
		if !ok {
			return
		}
		for _, line := range lines {
			if !writeLogMessage(ws, wsWriteMtx, &LogMessage{Message: line, ProcessName: procName}) {
				return
			}
		}
	case <-done:
		log.Warn().Msg("Socket closed remotely")
		closeLogChan()
		return
	}
	for {
		select {
		case msg, open := <-logChan:
			if !open {
				return
			}
			if !writeLogMessage(ws, wsWriteMtx, &msg) {
				return
			}
		case <-done:
			log.Warn().Msg("Socket closed remotely")
			closeLogChan()
			return
		}

	}
}

// writeLogMessage writes msg to ws and reports whether the stream can go on.
func writeLogMessage(ws *websocket.Conn, wsWriteMtx *sync.Mutex, msg *LogMessage) bool {
	// Serialize writes per ws.Conn. Multiple process streams share the
	// same connection when the request contains comma-separated names.
	wsWriteMtx.Lock()
	_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err := ws.WriteJSON(msg)
	wsWriteMtx.Unlock()
	if err != nil {
		if !errors.Is(err, net.ErrClosed) {
			log.Err(err).Msg("Failed to write to socket")
		}
		// The connection is unusable for every stream sharing it: close it
		// so they all end instead of waiting for their next line.
		_ = ws.Close()
		return false
	}
	return true
}

func handleIncoming(ws *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		msgType, _, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				return
			}
			if msgType == -1 {
				return
			}
			log.Err(err).Msgf("Failed to read from socket %d", msgType)
			return
		}
	}
}
