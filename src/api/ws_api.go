package api

import (
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

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
	if follow {
		go handleIncoming(ws, done)
	}
	for _, processName := range processNames {
		logChan := make(chan LogMessage, 256)
		chanCloseMtx := &sync.Mutex{}
		isChannelClosed := false
		connector := pclog.NewConnector(
			func(messages []string) {
				for _, message := range messages {
					msg := LogMessage{
						Message:     message,
						ProcessName: processName,
					}
					logChan <- msg
				}
				if !follow {
					chanCloseMtx.Lock()
					defer chanCloseMtx.Unlock()
					close(logChan)
					isChannelClosed = true
				}
			},
			func(message string) (n int, err error) {
				msg := LogMessage{
					Message:     message,
					ProcessName: processName,
				}
				chanCloseMtx.Lock()
				defer chanCloseMtx.Unlock()
				if isChannelClosed {
					return 0, nil
				}
				logChan <- msg
				return len(message), nil
			},
			endOffset)
		go api.handleLog(ws, processName, connector, logChan, done)

		err = api.project.GetLogsAndSubscribe(processName, connector)
		if err != nil {
			log.Err(err).Msg("Failed to subscribe to logger")
			return
		}
	}

}

func (api *PcApi) handleLog(ws *websocket.Conn, procName string, connector *pclog.Connector, logChan chan LogMessage, done chan struct{}) {
	defer func(project app.IProject, name string, observer pclog.LogObserver) {
		err := project.UnSubscribeLogger(name, observer)
		if err != nil {
			log.Err(err).Msg("Failed to unsubscribe from logger")
		}
	}(api.project, procName, connector)
	defer ws.Close()
	for {
		select {
		case msg, open := <-logChan:
			api.wsMtx.Lock()
			err := ws.WriteJSON(&msg)
			api.wsMtx.Unlock()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.Err(err).Msg("Failed to write to socket")
				return
			}
			if !open {
				return
			}
		case <-done:
			log.Warn().Msg("Socket closed remotely")
			close(logChan)
			return
		}

	}
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
