package api

import (
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
	"sync"
)

var upgrader = websocket.Upgrader{}

func (api *PcApi) HandleLogsStream(c *gin.Context) {
	procName := c.Query("name")
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
	logChan := make(chan LogMessage, 256)
	chanCloseMtx := &sync.Mutex{}
	isChannelClosed := false
	connector := pclog.NewConnector(
		func(messages []string) {
			for _, message := range messages {
				msg := LogMessage{
					Message:     message,
					ProcessName: procName,
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
				ProcessName: procName,
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
	go api.handleLog(ws, procName, connector, logChan, done)
	if follow {
		go handleIncoming(ws, done)
	}
	api.project.GetLogsAndSubscribe(procName, connector)
}

func (api *PcApi) handleLog(ws *websocket.Conn, procName string, connector *pclog.Connector, logChan chan LogMessage, done chan struct{}) {
	defer api.project.UnSubscribeLogger(procName, connector)
	defer ws.Close()
	for {
		select {
		case msg, open := <-logChan:
			if err := ws.WriteJSON(&msg); err != nil {
				log.Err(err).Msg("Failed to write to socket")
				return
			}
			if !open {
				return
			}
		case <-done:
			log.Warn().Msg("Socket closed remotely")
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
