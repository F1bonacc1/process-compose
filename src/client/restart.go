package client

import (
	"fmt"
	"net/http"
)

func (p *PcClient) restartProcess(name string) error {
	url := fmt.Sprintf("http://%s/process/restart/%s", p.address, name)
	return p.doAction(http.MethodPost, url, fmt.Sprintf("restart process %s", name))
}
