package client

import (
	"fmt"
	"net/http"
)

func (p *PcClient) startProcess(name string) error {
	url := fmt.Sprintf("http://%s/process/start/%s", p.address, name)
	return p.doAction(http.MethodPost, url, fmt.Sprintf("start process %s", name))
}
