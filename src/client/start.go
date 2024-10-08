package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (p *PcClient) startProcess(name string) error {
	url := fmt.Sprintf("http://%s/process/start/%s", p.address, name)
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
		log.Error().Msgf("failed to decode start process %s response: %v", name, err)
		return err
	}
	return errors.New(respErr.Error)
}
