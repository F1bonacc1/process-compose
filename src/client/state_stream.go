package client

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/f1bonacc1/process-compose/src/config"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// SubscribeProcessStates opens a WebSocket connection to the server's state
// stream and returns a channel of ProcessStateEvent values. The channel is
// closed when ctx is cancelled, when the server closes the connection, or on
// a read error.
//
// If names is non-empty, the server filters the stream to only those
// processes (snapshot frames included). Unknown names are accepted silently
// by the server and produce no events; the server logs a warning.
//
// The first events delivered after connect are an initial snapshot
// (ev.Snapshot == true) for the matching processes, followed by live
// transitions.
func (p *PcClient) SubscribeProcessStates(ctx context.Context, names ...string) (<-chan types.ProcessStateEvent, error) {
	wsURL := fmt.Sprintf("ws://%s/process/states/ws", p.address)
	if len(names) > 0 {
		q := url.Values{}
		q.Set("name", strings.Join(names, ","))
		wsURL = wsURL + "?" + q.Encode()
	}

	dialer := *websocket.DefaultDialer
	if p.address == "unix" {
		sockPath := p.logger.socketPath
		dialer.NetDialContext = func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", sockPath)
		}
	}

	var header http.Header
	if token := config.GetApiToken(); token != "" {
		header = make(http.Header)
		header.Set(config.TokenHeader, token)
	}

	ws, resp, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			log.Fatal().Msgf("authentication failed: invalid or missing %s", config.EnvVarApiToken)
		}
		return nil, fmt.Errorf("failed to connect to %s: %w", wsURL, err)
	}

	out := make(chan types.ProcessStateEvent, 256)

	go func() {
		<-ctx.Done()
		_ = ws.Close()
	}()

	go func() {
		defer close(out)
		defer ws.Close()
		for {
			var ev types.ProcessStateEvent
			if err := ws.ReadJSON(&ev); err != nil {
				if ctx.Err() != nil {
					return
				}
				if websocket.IsCloseError(err,
					websocket.CloseNormalClosure,
					websocket.CloseAbnormalClosure,
					websocket.CloseGoingAway) {
					return
				}
				log.Debug().Err(err).Msg("state stream read error")
				return
			}
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}
