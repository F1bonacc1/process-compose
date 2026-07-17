package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
)

// deriveProcessURL builds a browser-openable URL for a process, using (in order):
//  1. The process's http_get readiness probe (scheme://host:port/path). An empty
//     host defaults to 127.0.0.1 and an empty scheme defaults to http.
//  2. The first discovered TCP port -> http://127.0.0.1:<port>/.
//
// It returns ok=false when neither source yields a usable URL.
func deriveProcessURL(cfg *types.ProcessConfig, ports *types.ProcessPorts) (string, bool) {
	if cfg != nil && cfg.ReadinessProbe != nil && cfg.ReadinessProbe.HttpGet != nil {
		if url, ok := urlFromHttpGet(cfg.ReadinessProbe.HttpGet); ok {
			return url, true
		}
	}

	if ports != nil && len(ports.TcpPorts) > 0 {
		return fmt.Sprintf("http://127.0.0.1:%d/", ports.TcpPorts[0]), true
	}

	return "", false
}

func urlFromHttpGet(h *health.HttpProbe) (string, bool) {
	port := h.NumPort
	if port == 0 && h.Port != "" {
		port, _ = strconv.Atoi(h.Port)
	}
	if port < 1 || port > 65535 {
		return "", false
	}

	scheme := h.Scheme
	if strings.TrimSpace(scheme) == "" {
		scheme = "http"
	}
	host := h.Host
	if strings.TrimSpace(host) == "" {
		host = "127.0.0.1"
	}
	path := h.Path
	if strings.TrimSpace(path) == "" {
		path = "/"
	} else if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path), true
}
