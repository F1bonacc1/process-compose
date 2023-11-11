package client

import (
	"encoding/json"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (p *PcClient) shutDownProject() error {
	url := fmt.Sprintf("http://%s:%d/project/stop/", p.address, p.port)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("failed to stop project - unexpected status code: %s", resp.Status)
	}
}

func (p *PcClient) getProjectState() (*types.ProjectState, error) {
	url := fmt.Sprintf("http://%s:%d/project/state", p.address, p.port)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var sResp types.ProjectState

	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		log.Err(err).Msgf("failed to decode process states")
		return nil, err
	}
	return &sResp, nil
}
