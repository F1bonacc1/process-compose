package client

import (
	"fmt"
	"github.com/f1bonacc1/process-compose/src/api"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"os"
	"os/signal"
	"time"
)

func ReadProcessLogs(address string, port int, name string, offset int, follow bool) error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	url := fmt.Sprintf("ws://%s:%d/process/logs/ws/%s/%d/%v", address, port, name, offset, follow)
	log.Info().Msgf("Connecting to %s", url)
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Error().Msgf("failed to dial to %s error: %v", url, err)
		return err
	}
	defer ws.Close()
	done := make(chan struct{})

	go readLogs(done, ws, follow)

	for {
		select {
		case <-done:
			return nil
		case <-interrupt:
			fmt.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				fmt.Println("write close:", err)
				return nil
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return nil
		}
	}
}

func readLogs(done chan struct{}, ws *websocket.Conn, follow bool) {
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
			log.Error().Msgf("failed to read message: %v", err)
			return
		}
		if len(message.ProcessName) > 0 {
			fmt.Printf("%s:\t%s\n", message.ProcessName, message.Message)
		}
	}
}
