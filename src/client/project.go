package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (p *PcClient) shutDownProject() error {
	url := fmt.Sprintf("http://%s/project/stop/", p.address)
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

func (p *PcClient) getProjectState(withMemory bool) (*types.ProjectState, error) {
	url := fmt.Sprintf("http://%s/project/state/?withMemory=%v", p.address, withMemory)
	resp, err := p.client.Get(url)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("failed to get project state - unexpected status code: %s", resp.Status)
	}
	var sResp types.ProjectState

	//Decode the data
	if err = json.NewDecoder(resp.Body).Decode(&sResp); err != nil {
		log.Err(err).Msgf("failed to decode process states")
		return nil, err
	}
	return &sResp, nil
}

func (p *PcClient) updateProject(project *types.Project) (map[string]string, error) {
	url := fmt.Sprintf("http://%s/project", p.address)
	jsonData, err := json.Marshal(project)
	if err != nil {
		log.Err(err).Msg("failed to marshal project")
		return nil, err
	}
	resp, err := p.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Err(err).Msg("failed to update project")
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus {
		status := map[string]string{}
		if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
			log.Err(err).Msg("failed to decode updated processes")
			return status, err
		}
		log.Info().Msgf("status: %v", status)

		return status, nil
	}
	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Err(err).Msg("failed to decode err update project")
		return nil, err
	}
	return nil, fmt.Errorf(respErr.Error)
}

func (p *PcClient) reloadProject() (map[string]string, error) {
	url := fmt.Sprintf("http://%s/project/configuration", p.address)
	resp, err := p.client.Post(url, "application/json", nil)
	if err != nil {
		log.Err(err).Msg("failed to update project")
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus {
		status := map[string]string{}
		if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
			log.Err(err).Msg("failed to decode updated processes")
			return status, err
		}
		log.Info().Msgf("status: %v", status)

		return status, nil
	}
	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Err(err).Msg("failed to decode err update project")
		return nil, err
	}
	return nil, fmt.Errorf(respErr.Error)
}
