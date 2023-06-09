package client

import (
	"encoding/json"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
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
	var sResp types.ProcessStates

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

func GetProcessesState(address string, port int) ([]types.ProcessState, error) {
	url := fmt.Sprintf("http://%s:%d/processes", address, port)
	resp, err := http.Get(url)
	if err != nil {
		return []types.ProcessState{}, err
	}
	defer resp.Body.Close()
	//Create a variable of the same type as our model
	var sResp types.ProcessStates

	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		return []types.ProcessState{}, err
	}
	return sResp.States, nil
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
	//Create a variable of the same type as our model
	var sResp types.ProcessConfig

	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		log.Err(err).Msgf("what I got: %s", err.Error())
		return nil, err
	}

	return &sResp, nil
}
