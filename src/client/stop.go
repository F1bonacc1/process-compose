package client

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func StopProcesses(address string, port int, name string) error {
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
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	defer resp.Body.Close()
	var respErr pcError
	if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
		log.Error().Msgf("failed to decode stop process %s response: %v", name, err)
		return err
	}
	return fmt.Errorf(respErr.Error)
}
