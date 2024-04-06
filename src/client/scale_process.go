package client

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (p *PcClient) scaleProcess(name string, scale int) error {
	url := fmt.Sprintf("http://%s/process/scale/%s/%d", p.address, name, scale)
	req, err := http.NewRequest(http.MethodPatch, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Error().Msgf("failed to decode scale process %s response: %v", name, err)
		return err
	}
	return fmt.Errorf(respErr.Error)
}
