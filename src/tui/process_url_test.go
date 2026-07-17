package tui

import (
	"testing"

	"github.com/f1bonacc1/process-compose/src/health"
	"github.com/f1bonacc1/process-compose/src/types"
)

func httpGetConfig(h *health.HttpProbe) *types.ProcessConfig {
	return &types.ProcessConfig{
		ReadinessProbe: &health.Probe{HttpGet: h},
	}
}

func TestDeriveProcessURL(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *types.ProcessConfig
		ports   *types.ProcessPorts
		wantURL string
		wantOk  bool
	}{
		{
			name:    "http_get with path",
			cfg:     httpGetConfig(&health.HttpProbe{Host: "localhost", NumPort: 8080, Path: "/health"}),
			wantURL: "http://localhost:8080/health",
			wantOk:  true,
		},
		{
			name:    "http_get without path defaults to /",
			cfg:     httpGetConfig(&health.HttpProbe{Host: "localhost", NumPort: 8080}),
			wantURL: "http://localhost:8080/",
			wantOk:  true,
		},
		{
			name:    "custom scheme host port",
			cfg:     httpGetConfig(&health.HttpProbe{Scheme: "https", Host: "example.com", NumPort: 9443, Path: "/status"}),
			wantURL: "https://example.com:9443/status",
			wantOk:  true,
		},
		{
			name:    "empty host defaults to 127.0.0.1",
			cfg:     httpGetConfig(&health.HttpProbe{NumPort: 3000}),
			wantURL: "http://127.0.0.1:3000/",
			wantOk:  true,
		},
		{
			name:    "port from string field",
			cfg:     httpGetConfig(&health.HttpProbe{Host: "localhost", Port: "8081"}),
			wantURL: "http://localhost:8081/",
			wantOk:  true,
		},
		{
			name:    "fallback to first tcp port",
			cfg:     &types.ProcessConfig{},
			ports:   &types.ProcessPorts{TcpPorts: []uint16{5000, 5001}},
			wantURL: "http://127.0.0.1:5000/",
			wantOk:  true,
		},
		{
			name:   "neither http_get nor ports",
			cfg:    &types.ProcessConfig{},
			wantOk: false,
		},
		{
			name:    "http_get without usable port falls back to tcp port",
			cfg:     httpGetConfig(&health.HttpProbe{Host: "localhost"}),
			ports:   &types.ProcessPorts{TcpPorts: []uint16{6000}},
			wantURL: "http://127.0.0.1:6000/",
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURL, gotOk := deriveProcessURL(tt.cfg, tt.ports)
			if gotOk != tt.wantOk {
				t.Fatalf("ok = %v, want %v", gotOk, tt.wantOk)
			}
			if gotURL != tt.wantURL {
				t.Errorf("url = %q, want %q", gotURL, tt.wantURL)
			}
		})
	}
}
