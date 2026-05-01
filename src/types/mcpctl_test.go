package types

import "testing"

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
		{"fully populated", &MCPCtlServerConfig{Host: "localhost", Port: 11001}, true},
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
		{"valid", &MCPCtlServerConfig{Host: "localhost", Port: 11001}, false},
		{"missing host", &MCPCtlServerConfig{Port: 11001}, true},
		{"missing port", &MCPCtlServerConfig{Host: "localhost"}, true},
		{"zero port", &MCPCtlServerConfig{Host: "localhost", Port: 0}, true},
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
