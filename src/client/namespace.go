package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
)

func (p *PcClient) startNamespace(name string) error {
	u := fmt.Sprintf("http://%s/namespace/start/%s", p.address, url.PathEscape(name))
	return p.doAction(http.MethodPost, u, fmt.Sprintf("start namespace %s", name))
}

func (p *PcClient) stopNamespace(name string) error {
	u := fmt.Sprintf("http://%s/namespace/stop/%s", p.address, url.PathEscape(name))
	return p.doAction(http.MethodPost, u, fmt.Sprintf("stop namespace %s", name))
}

func (p *PcClient) restartNamespace(name string) error {
	u := fmt.Sprintf("http://%s/namespace/restart/%s", p.address, url.PathEscape(name))
	return p.doAction(http.MethodPost, u, fmt.Sprintf("restart namespace %s", name))
}

func (p *PcClient) getNamespaces() ([]string, error) {
	url := fmt.Sprintf("http://%s/namespaces", p.address)
	resp, err := p.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var respErr pcError
		if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
			log.Error().Msgf("failed to decode get namespaces response: %v", err)
			return nil, err
		}
		return nil, errors.New(respErr.Error)
	}

	var namespaces []string
	if err = json.NewDecoder(resp.Body).Decode(&namespaces); err != nil {
		return nil, err
	}
	return namespaces, nil
}
