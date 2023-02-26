package api

import (
	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"net/http"
	"strconv"
)

var upgrader = websocket.Upgrader{}

func HandleLogsStream(c *gin.Context) {
	procName := c.Param("name")
	follow := c.Param("follow") == "true"
	endOffset, err := strconv.Atoi(c.Param("endOffset"))
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
	connector := pclog.NewConnector(func(messages []string) {
		for _, message := range messages {
			msg := LogMessage{
				Message:     message,
				ProcessName: procName,
			}
			logChan <- msg
		}
		if !follow {
			close(logChan)
		}
	},
		func(message string) {
			msg := LogMessage{
				Message:     message,
				ProcessName: procName,
			}
			logChan <- msg
		},
		endOffset)
	go handleLog(ws, procName, connector, logChan, done)
	if follow {
		go handleIncoming(ws, done)
	}
	app.PROJ.GetLogsAndSubscribe(procName, connector)
}

func handleLog(ws *websocket.Conn, procName string, connector *pclog.Connector, logChan chan LogMessage, done chan struct{}) {
	defer app.PROJ.UnSubscribeLogger(procName, connector)
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
			log.Err(err).Msgf("Failed to read from socket %d", msgType)
			return
		}
	}
}
