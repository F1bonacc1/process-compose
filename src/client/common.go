package client

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
)

type pcError struct {
	Error string `json:"error"`
}

func (p *PcClient) doAction(method, url, actionName string) error {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Error().Msgf("failed to decode %s response: %v", actionName, err)
		return err
	}
	return errors.New(respErr.Error)
}
