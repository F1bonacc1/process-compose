package api

type LogMessage struct {
	Message     string `json:"message"`
	ProcessName string `json:"process_name"`
}
