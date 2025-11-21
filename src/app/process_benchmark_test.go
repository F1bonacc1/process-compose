package app

import (
	"os/exec"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/types"
	puproc "github.com/shirou/gopsutil/v4/process"
)

func setupBenchmark(b *testing.B) (*Process, *exec.Cmd, func()) {
	// Start a process with some children
	cmd := exec.Command("sh", "-c", "sleep 10 & sleep 10 & wait")
	if err := cmd.Start(); err != nil {
		b.Fatal(err)
	}

	cleanup := func() {
		_ = cmd.Process.Kill()
	}

	procConfig := &types.ProcessConfig{
		Name: "bench_proc",
	}
	procState := &types.ProcessState{
		Pid: cmd.Process.Pid,
	}

	p := NewProcess(
		withProcConf(procConfig),
		withProcState(procState),
		withRecursiveMetrics(true),
	)

	var err error
	p.metricsProc, err = puproc.NewProcess(int32(p.procState.Pid))
	if err != nil {
		cleanup()
		b.Fatal(err)
	}

	return p, cmd, cleanup
}

func BenchmarkGetProcResourcesRecursive_WithCache(b *testing.B) {
	p, _, cleanup := setupBenchmark(b)
	defer cleanup()

	tree := NewProcessTree(1 * time.Second)
	// Manually inject the tree
	p.processTree = tree

	// Populate tree
	if err := tree.Update(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.getProcResourcesRecursive(p.metricsProc)
	}
}

func BenchmarkGetProcResourcesRecursive_NoCache(b *testing.B) {
	p, _, cleanup := setupBenchmark(b)
	defer cleanup()

	// Ensure no tree is set
	p.processTree = nil

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.getProcResourcesRecursive(p.metricsProc)
	}
}
