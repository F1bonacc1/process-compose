package app

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/pclog"
	"github.com/f1bonacc1/process-compose/src/types"
)

// A process stopped before its command launches, for example while it waits
// out a restart backoff, must not launch the command.
func TestStopBeforeStartKeepsCommandFromRunning(t *testing.T) {
	shell := command.DefaultShellConfig()
	marker := filepath.Join(t.TempDir(), "started")
	p := NewProcess(
		withProcConf(&types.ProcessConfig{
			Name:          "p",
			ReplicaName:   "p",
			Executable:    shell.ShellCommand,
			Args:          []string{shell.ShellArgument, fmt.Sprintf("echo x >> %s", marker)},
			RestartPolicy: types.RestartPolicyConfig{Restart: types.RestartPolicyAlways},
		}),
		withLogger(pclog.NewNilLogger()),
		withProcState(&types.ProcessState{}),
		withProcLog(pclog.NewLogBuffer(100)),
	)
	_ = p.shutDown()

	done := make(chan struct{})
	go func() {
		_ = p.run()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("run() kept restarting a stopped process")
	}
	if n := countLines(t, marker); n != 0 {
		t.Errorf("the command ran %d times after the stop, want 0", n)
	}
}
