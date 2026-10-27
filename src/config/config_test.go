package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLogFilePaths(t *testing.T) {
	defaultServer := filepath.Join(os.TempDir(), fmt.Sprintf("process-compose%s.log", userSuffix()))
	defaultClient := filepath.Join(os.TempDir(), fmt.Sprintf("process-compose%s-client.log", userSuffix()))

	tests := []struct {
		name       string
		logFile    string
		clientFile string
		wantServer string
		wantClient string
	}{
		{
			name:       "no environment variables",
			wantServer: defaultServer,
			wantClient: defaultClient,
		},
		{
			name:       "PC_LOG_FILE gives the client a sibling file",
			logFile:    "/var/log/pc.log",
			wantServer: "/var/log/pc.log",
			wantClient: "/var/log/pc-client.log",
		},
		{
			name:       "PC_LOG_FILE without an extension",
			logFile:    "/var/log/pc",
			wantServer: "/var/log/pc",
			wantClient: "/var/log/pc-client",
		},
		{
			name:       "PC_LOG_FILE as a dotfile",
			logFile:    "/var/log/.pc",
			wantServer: "/var/log/.pc",
			wantClient: "/var/log/.pc-client",
		},
		{
			name:       "PC_CLIENT_LOG_FILE wins over PC_LOG_FILE",
			logFile:    "/var/log/pc.log",
			clientFile: "/var/log/client.log",
			wantServer: "/var/log/pc.log",
			wantClient: "/var/log/client.log",
		},
		{
			name:       "PC_CLIENT_LOG_FILE alone leaves the server path alone",
			clientFile: "/var/log/client.log",
			wantServer: defaultServer,
			wantClient: "/var/log/client.log",
		},
		{
			name:       "empty values count as unset",
			logFile:    "",
			clientFile: "",
			wantServer: defaultServer,
			wantClient: defaultClient,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(LogPathEnvVarName, tt.logFile)
			t.Setenv(ClientLogPathEnvVarName, tt.clientFile)
			if got := GetLogFilePath(); got != tt.wantServer {
				t.Errorf("GetLogFilePath() = %q, want %q", got, tt.wantServer)
			}
			if got := GetClientLogFilePath(); got != tt.wantClient {
				t.Errorf("GetClientLogFilePath() = %q, want %q", got, tt.wantClient)
			}
			if GetLogFilePath() == GetClientLogFilePath() {
				t.Errorf("the client and the server share the log file %q", GetLogFilePath())
			}
		})
	}
}

func userSuffix() string {
	if userName := getUser(); len(userName) != 0 {
		return "-" + userName
	}
	return ""
}
