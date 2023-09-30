package client

import (
	"fmt"
	"net/http"
)

func (p *PcClient) shutDownProject() error {
	url := fmt.Sprintf("http://%s:%d/project/stop/", p.address, p.port)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	} else {
		return fmt.Errorf("failed to stop project - unexpected status code: %s", resp.Status)
	}
}
