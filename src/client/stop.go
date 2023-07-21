package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func StopProcess(address string, port int, name string) error {
	url := fmt.Sprintf("http://%s:%d/process/stop/%s", address, port, name)
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPatch, url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Error().Msgf("failed to decode stop process %s response: %v", name, err)
		return err
	}
	return fmt.Errorf(respErr.Error)
}

func StopProcesses(address string, port int, names []string) ([]string, error) {
	url := fmt.Sprintf("http://%s:%d/processes/stop", address, port)
	client := &http.Client{}

	jsonPayload, err := json.Marshal(names)

	if err != nil {
		log.Err(err).Msgf("failed to marshal names: %v", names)
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		stopped := []string{}
		if err = json.NewDecoder(resp.Body).Decode(&stopped); err != nil {
			log.Err(err).Msgf("failed to decode stop processes %v", names)
			return stopped, err
		}
		log.Info().Msgf("stopped: %v", stopped)

		return stopped, nil
	}
	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Err(err).Msgf("failed to decode err stop process %v", names)
		return nil, err
	}
	return nil, fmt.Errorf(respErr.Error)
}
