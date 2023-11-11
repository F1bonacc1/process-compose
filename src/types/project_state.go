package types

import "time"

type ProjectState struct {
	FileNames         []string      `json:"fileNames"`
	UpTime            time.Duration `json:"upTime"`
	StartTime         time.Time     `json:"startTime"`
	ProcessNum        int           `json:"processNum"`
	RunningProcessNum int           `json:"runningProcessNum"`
	UserName          string        `json:"userName"`
	HostName          string        `json:"hostName"`
	Version           string        `json:"version"`
}
