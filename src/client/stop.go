package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (p *PcClient) stopProcess(name string) error {
	url := fmt.Sprintf("http://%s/process/stop/%s", p.address, escapePathSegment(name))
	return p.doAction(http.MethodPatch, url, fmt.Sprintf("stop process %s", name))
}

func (p *PcClient) sendSignal(name string, sig int) error {
	url := fmt.Sprintf("http://%s/process/signal/%s/%d", p.address, escapePathSegment(name), sig)
	return p.doAction(http.MethodPatch, url, fmt.Sprintf("send signal %s", name))
}

func (p *PcClient) stopProcesses(names []string) (map[string]string, error) {
	url := fmt.Sprintf("http://%s/processes/stop", p.address)
	jsonPayload, err := json.Marshal(names)
	if err != nil {
		log.Err(err).Msgf("failed to marshal names: %v", names)
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus {
		stopped := map[string]string{}
		if err = json.NewDecoder(resp.Body).Decode(&stopped); err != nil {
			log.Err(err).Msgf("failed to decode stop processes %v", names)
			return stopped, err
		}
		log.Info().Msgf("stopped: %v", stopped)

		return stopped, nil
	}
	return nil, parseErrorResponse(resp, fmt.Sprintf("stop processes %v", names))
}
