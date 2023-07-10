package client

import (
	"encoding/json"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"net/http"
	"sort"
)

func GetProcessesName(address string, port int) ([]string, error) {
	states, err := GetProcessesState(address, port)
	if err != nil {
		return nil, err
	}
	procs := make([]string, len(states.States))
	for i, proc := range states.States {
		procs[i] = proc.Name
	}
	sort.Strings(procs)
	return procs, nil
}

func GetProcessesState(address string, port int) (*types.ProcessesState, error) {
	url := fmt.Sprintf("http://%s:%d/processes", address, port)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	//Create a variable of the same type as our model
	var sResp types.ProcessesState

	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		log.Err(err).Msgf("failed to decode process states")
		return nil, err
	}
	return &sResp, nil
}

func GetProcessState(address string, port int, name string) (*types.ProcessState, error) {
	url := fmt.Sprintf("http://%s:%d/process/%s", address, port, name)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	//Create a variable of the same type as our model
	var sResp types.ProcessState

	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		return nil, err
	}

	return &sResp, nil
}

func GetProcessInfo(address string, port int, name string) (*types.ProcessConfig, error) {
	url := fmt.Sprintf("http://%s:%d/process/info/%s", address, port, name)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var sResp types.ProcessConfig

	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		log.Err(err).Msgf("what I got: %s", err.Error())
		return nil, err
	}

	return &sResp, nil
}

func GetProcessPorts(address string, port int, name string) (*types.ProcessPorts, error) {
	url := fmt.Sprintf("http://%s:%d/process/ports/%s", address, port, name)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var sResp types.ProcessPorts

	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		log.Err(err).Msgf("what I got: %s", err.Error())
		return nil, err
	}

	return &sResp, nil
}
