package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (p *PcClient) startNamespace(name string) error {
	u := fmt.Sprintf("http://%s/namespace/start/%s", p.address, escapePathSegment(name))
	return p.doAction(http.MethodPost, u, fmt.Sprintf("start namespace %s", name))
}

func (p *PcClient) stopNamespace(name string) error {
	u := fmt.Sprintf("http://%s/namespace/stop/%s", p.address, escapePathSegment(name))
	return p.doAction(http.MethodPost, u, fmt.Sprintf("stop namespace %s", name))
}

func (p *PcClient) restartNamespace(name string) error {
	u := fmt.Sprintf("http://%s/namespace/restart/%s", p.address, escapePathSegment(name))
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
		return nil, parseErrorResponse(resp, "get namespaces")
	}

	var namespaces []string
	if err = json.NewDecoder(resp.Body).Decode(&namespaces); err != nil {
		return nil, err
	}
	return namespaces, nil
}
