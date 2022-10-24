package client

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func RestartProcesses(address string, port int, name string) error {
	url := fmt.Sprintf("http://%s:%d/process/restart/%s", address, port, name)
	resp, err := http.Post(url, "application/json", nil)
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
	return fmt.Errorf(respErr.Error)
}
