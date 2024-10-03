package client

import (
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"net/http"
)

func (p *PcClient) isAlive() error {
	url := fmt.Sprintf("http://%s/live", p.address)
	resp, err := p.client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", resp.Status)
	}
	return nil
}

func (p *PcClient) getHostName() (string, error) {
	url := fmt.Sprintf("http://%s/hostname", p.address)
	resp, err := p.client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %s", resp.Status)
	}

	nameMap := map[string]string{}
	//Decode the data
	if err = json.NewDecoder(resp.Body).Decode(&nameMap); err != nil {
		log.Err(err).Send()
		return "", err
	}
	return nameMap["name"], nil
}
