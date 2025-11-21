package app

import (
	"sync"
	"time"

	puproc "github.com/shirou/gopsutil/v4/process"
)

type ProcessTree struct {
	sync.RWMutex
	tree      map[int32][]int32
	procs     map[int32]*puproc.Process
	lastCache time.Time
	cacheTTL  time.Duration
}

func NewProcessTree(cacheTTL time.Duration) *ProcessTree {
	return &ProcessTree{
		tree:  make(map[int32][]int32),
		procs: make(map[int32]*puproc.Process),
		cacheTTL: cacheTTL,
	}
}

func (pt *ProcessTree) Update() error {
	pt.Lock()
	defer pt.Unlock()

	// Don't update too frequently
	if time.Since(pt.lastCache) < pt.cacheTTL {
		return nil
	}

	procs, err := puproc.Processes()
	if err != nil {
		return err
	}

	newTree := make(map[int32][]int32)
	newProcs := make(map[int32]*puproc.Process)

	for _, p := range procs {
		// Reuse existing process objects to maintain CPU history
		if existing, ok := pt.procs[p.Pid]; ok {
			newProcs[p.Pid] = existing
		} else {
			newProcs[p.Pid] = p
		}

		ppid, err := p.Ppid()
		if err != nil {
			continue
		}
		newTree[ppid] = append(newTree[ppid], p.Pid)
	}

	pt.tree = newTree
	pt.procs = newProcs
	pt.lastCache = time.Now()
	return nil
}

func (pt *ProcessTree) GetDescendants(pid int32) []*puproc.Process {
	pt.RLock()
	defer pt.RUnlock()

	var descendants []*puproc.Process
	var queue []int32
	queue = append(queue, pid)
	visited := make(map[int32]bool)

	for len(queue) > 0 {
		currentPid := queue[0]
		queue = queue[1:]

		if visited[currentPid] {
			continue
		}
		visited[currentPid] = true

		children := pt.tree[currentPid]
		for _, childPid := range children {
			if p, ok := pt.procs[childPid]; ok {
				descendants = append(descendants, p)
				queue = append(queue, childPid)
			}
		}
	}

	return descendants
}
