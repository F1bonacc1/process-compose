package client

import (
	"encoding/json"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"sort"
)

func (p *PcClient) GetProcessesName() ([]string, error) {
	states, err := p.GetRemoteProcessesState()
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

func (p *PcClient) GetRemoteProcessesState() (*types.ProcessesState, error) {
	url := fmt.Sprintf("http://%s/processes", p.address)
	resp, err := p.client.Get(url)
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

func (p *PcClient) getProcessState(name string) (*types.ProcessState, error) {
	url := fmt.Sprintf("http://%s/process/%s", p.address, name)
	resp, err := p.client.Get(url)
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

func (p *PcClient) getProcessInfo(name string) (*types.ProcessConfig, error) {
	url := fmt.Sprintf("http://%s/process/info/%s", p.address, name)
	resp, err := p.client.Get(url)
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

func (p *PcClient) getProcessPorts(name string) (*types.ProcessPorts, error) {
	url := fmt.Sprintf("http://%s/process/ports/%s", p.address, name)
	resp, err := p.client.Get(url)
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
