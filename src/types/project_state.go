package types

import "time"

type ProjectState struct {
	FileNames         []string      `json:"fileNames"`
	UpTime            time.Duration `json:"upTime" swaggertype:"primitive,integer"`
	StartTime         time.Time     `json:"startTime"`
	ProcessNum        int           `json:"processNum"`
	RunningProcessNum int           `json:"runningProcessNum"`
	UserName          string        `json:"userName"`
	Version           string        `json:"version"`
	ProjectName       string        `json:"projectName"`
	MemoryState       *MemoryState  `json:"memoryState,omitempty"`
}

type MemoryState struct {
	Allocated      uint64 `json:"allocated"`
	TotalAllocated uint64 `json:"totalAllocated"`
	SystemMemory   uint64 `json:"systemMemory"`
	GcCycles       uint32 `json:"gcCycles"`
}
