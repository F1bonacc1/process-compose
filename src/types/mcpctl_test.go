package types

import (
	"testing"
)

func TestMCPCtlServerConfigIsEnabled(t *testing.T) {
	cases := []struct {
		name string
		cfg  *MCPCtlServerConfig
		want bool
	}{
		{"nil", nil, false},
		{"empty block", &MCPCtlServerConfig{}, false},
		{"port only", &MCPCtlServerConfig{Port: 11001}, true},
		{"host only", &MCPCtlServerConfig{Host: "localhost"}, true},
		{"transport only", &MCPCtlServerConfig{Transport: "sse"}, true},
		{"fully populated", &MCPCtlServerConfig{Host: "localhost", Port: 11001, Transport: "sse"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.cfg.IsEnabled(); got != tc.want {
				t.Errorf("IsEnabled = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMCPCtlServerConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *MCPCtlServerConfig
		wantErr bool
	}{
		{"nil", nil, false},
		{"valid sse", &MCPCtlServerConfig{Host: "localhost", Port: 11001, Transport: "sse"}, false},
		{"sse default transport", &MCPCtlServerConfig{Host: "localhost", Port: 11001}, false},
		{"uppercase sse accepted", &MCPCtlServerConfig{Host: "localhost", Port: 11001, Transport: "SSE"}, false},
		{"stdio rejected", &MCPCtlServerConfig{Transport: "stdio", Host: "localhost", Port: 11001}, true},
		{"unknown transport rejected", &MCPCtlServerConfig{Transport: "websocket", Host: "localhost", Port: 11001}, true},
		{"missing host", &MCPCtlServerConfig{Port: 11001}, true},
		{"missing port", &MCPCtlServerConfig{Host: "localhost"}, true},
		{"zero port", &MCPCtlServerConfig{Host: "localhost", Port: 0, Transport: "sse"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
