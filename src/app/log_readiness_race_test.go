package app

import (
	"context"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/f1bonacc1/process-compose/src/types"
)

func TestHandleOutputConcurrentLogReadiness(t *testing.T) {
	for _, stderr := range []string{"ordinary stderr output\n", "READY\n"} {
		t.Run(strings.TrimSpace(stderr), func(t *testing.T) {
			conf := &types.ProcessConfig{Name: "both", ReplicaName: "both", ReadyLogLine: "READY"}
			readyCtx, cancel := context.WithCancelCause(context.Background())
			t.Cleanup(func() { cancel(nil) })
			var published atomic.Int32
			p := &Process{
				procConf:         conf,
				procState:        types.NewProcessState(conf),
				readyLogCancelFn: cancel,
				publishState:     func(types.ProcessStateEvent) { published.Add(1) },
			}
			start := make(chan struct{})
			stdoutDone, stderrDone := make(chan struct{}), make(chan struct{})
			go func() {
				<-start
				p.handleOutput(io.NopCloser(strings.NewReader("READY\n")), "stdout", func(string) {}, stdoutDone)
			}()
			go func() {
				<-start
				p.handleOutput(io.NopCloser(strings.NewReader(stderr)), "stderr", func(string) {}, stderrDone)
			}()
			close(start)
			<-stdoutDone
			<-stderrDone
			if p.procState.Health != types.ProcessHealthReady {
				t.Errorf("Health = %q, want Ready", p.procState.Health)
			}
			if p.procState.ProcessReadyTime == nil {
				t.Error("ready timestamp was not recorded")
			}
			select {
			case <-readyCtx.Done():
			default:
				t.Error("ready log dependency was not released")
			}
			if got := published.Load(); got != 1 {
				t.Errorf("published %d readiness transitions, want 1", got)
			}
		})
	}
}
