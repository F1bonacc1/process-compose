package client

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func ScaleProcess(address string, port int, name string, scale int) error {
	url := fmt.Sprintf("http://%s:%d/process/scale/%s/%d", address, port, name, scale)
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
		log.Error().Msgf("failed to decode scale process %s response: %v", name, err)
		return err
	}
	return fmt.Errorf(respErr.Error)
}
