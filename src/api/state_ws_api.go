package api

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/f1bonacc1/process-compose/src/app"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// stateWsObserver adapts a buffered channel into a types.StateObserver. It
// drops the connection on backpressure: if the channel is full, Notify
// signals the writer goroutine to disconnect rather than blocking the
// broadcaster.
//
// nameFilter, when non-nil, restricts delivery to events whose State.Name is
// in the set. A nil filter means "deliver everything" (subscribe to all).
type stateWsObserver struct {
	id         string
	events     chan types.ProcessStateEvent
	closeCh    chan struct{}
	closed     atomic.Bool
	nameFilter map[string]struct{}
}

func newStateWsObserver(buf int, nameFilter map[string]struct{}) *stateWsObserver {
	return &stateWsObserver{
		id:         pclog.GenerateUniqueID(16),
		events:     make(chan types.ProcessStateEvent, buf),
		closeCh:    make(chan struct{}),
		nameFilter: nameFilter,
	}
}

func (o *stateWsObserver) UniqueID() string { return o.id }

func (o *stateWsObserver) Notify(ev types.ProcessStateEvent) {
	if o.closed.Load() {
		return
	}
	if o.nameFilter != nil {
		if _, ok := o.nameFilter[ev.State.Name]; !ok {
			return
		}
	}
	select {
	case o.events <- ev:
	default:
		// Slow consumer: signal the writer goroutine to disconnect.
		o.signalClose()
	}
}

func (o *stateWsObserver) signalClose() {
	if o.closed.CompareAndSwap(false, true) {
		close(o.closeCh)
	}
}

// @Schemes
// @Id                    StatesStream
// @Summary               Stream process state changes over WebSocket
// @Description           Upgrades HTTP to WebSocket and streams JSON events. On connect, an event with snapshot=true is sent for every (filtered) process; subsequently, every Status or Health transition (and final exit info) produces one event.
// @Tags                  Process
// @Produce               json
// @Param                 name query string false "Comma-separated process names to subscribe to. If omitted, subscribes to all processes."
// @Success               101 "Switching Protocols"
// @Failure               400 {object} api.ErrorResponse
// @Router                /process/states/ws [get]
func (api *PcApi) HandleStatesStream(c *gin.Context) {
	filter := api.parseStateNameFilter(c.Query("name"))

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	observer := newStateWsObserver(256, filter)
	done := make(chan struct{})
	go handleIncoming(ws, done)

	api.project.RegisterStateObserver(observer)

	go api.handleStateStream(ws, observer, done)
}

// parseStateNameFilter splits the comma-separated query value, drops blanks,
// and warns the log for any name that isn't a known process. An empty input
// returns nil meaning "subscribe to all".
func (api *PcApi) parseStateNameFilter(raw string) map[string]struct{} {
	if raw == "" {
		return nil
	}
	known, err := api.project.GetLexicographicProcessNames()
	knownSet := make(map[string]struct{}, len(known))
	if err == nil {
		for _, n := range known {
			knownSet[n] = struct{}{}
		}
	}
	filter := make(map[string]struct{})
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if err == nil {
			if _, ok := knownSet[name]; !ok {
				log.Warn().Str("name", name).Msg("State stream subscriber requested unknown process; will deliver no events for this name")
			}
		}
		filter[name] = struct{}{}
	}
	if len(filter) == 0 {
		return nil
	}
	return filter
}

func (api *PcApi) handleStateStream(ws *websocket.Conn, observer *stateWsObserver, done chan struct{}) {
	defer func(project app.IProject, o *stateWsObserver) {
		project.UnregisterStateObserver(o)
		o.signalClose()
	}(api.project, observer)
	defer ws.Close()

	var writeMu sync.Mutex
	writeJSON := func(ev types.ProcessStateEvent) error {
		writeMu.Lock()
		api.wsMtx.Lock()
		err := ws.WriteJSON(&ev)
		api.wsMtx.Unlock()
		writeMu.Unlock()
		return err
	}

	for {
		select {
		case ev := <-observer.events:
			if err := writeJSON(ev); err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.Err(err).Msg("Failed to write state event to socket")
				return
			}
		case <-observer.closeCh:
			log.Debug().Msg("State stream observer closed (slow consumer)")
			return
		case <-done:
			log.Debug().Msg("State stream socket closed remotely")
			return
		}
	}
}
