package client

import (
	"encoding/json"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/app"
	"net/http"
)

func GetProcessesName(address string, port int) ([]string, error) {
	url := fmt.Sprintf("http://%s:%d/processes", address, port)
	resp, err := http.Get(url)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()
	//Create a variable of the same type as our model
	var sResp app.ProcessStates

	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		return []string{}, err
	}
	procs := make([]string, len(sResp.States))
	for i, proc := range sResp.States {
		procs[i] = proc.Name
	}
	return procs, nil
}
