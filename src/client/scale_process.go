package client

import (
	"fmt"
	"net/http"
)

func (p *PcClient) scaleProcess(name string, scale int) error {
	url := fmt.Sprintf("http://%s/process/scale/%s/%d", p.address, name, scale)
	return p.doAction(http.MethodPatch, url, fmt.Sprintf("scale process %s", name))
}
