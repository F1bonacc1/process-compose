package api

// NameResponse represents a simple response containing a process name.
type NameResponse struct {
    Name string `json:"name"`
}

// ProjectNameResponse represents a response containing the project name.
type ProjectNameResponse struct {
    ProjectName string `json:"projectName"`
}

// StatusResponse represents a simple response containing a status string.
type StatusResponse struct {
    Status string `json:"status"`
}

// LogsResponse represents a response containing logs lines.
type LogsResponse struct {
    Logs []string `json:"logs"`
}

