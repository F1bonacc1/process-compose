package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

func (p *PcClient) restartProcess(name string) error {
	url := fmt.Sprintf("http://%s/process/restart/%s", p.address, name)
	resp, err := p.client.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Error().Msgf("failed to decode restart process %s response: %v", name, err)
		return err
	}
	return errors.New(respErr.Error)
}

func (p *PcClient) restartAllProcesses() error {
	url := fmt.Sprintf("http://%s/processes/restart", p.address)
	resp, err := p.client.Post(url, "application/json", nil)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Error().Msgf("failed to decode restart all processes response: %v", err)
		return err
	}
	return errors.New(respErr.Error)
}