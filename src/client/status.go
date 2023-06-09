package client

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func isAlive(address string, port int) error {
	url := fmt.Sprintf("http://%s:%d/live", address, port)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return nil
}

func getHostName(address string, port int) (string, error) {
	url := fmt.Sprintf("http://%s:%d/hostname", address, port)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s", resp.Status)
	}

	nameMap := map[string]string{}
	//Decode the data
	if err := json.NewDecoder(resp.Body).Decode(&nameMap); err != nil {
		log.Err(err)
		return "", err
	}
	return nameMap["name"], nil
}
