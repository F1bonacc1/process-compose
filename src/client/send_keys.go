package client

import (
	"fmt"
	"net/http"
)

func (p *PcClient) sendProcessKeys(name, keys string) error {
	url := fmt.Sprintf("http://%s/process/send-keys/%s", p.address, escapePathSegment(name))
	payload := map[string]string{"keys": keys}
	return p.doActionWithBody(http.MethodPost, url, fmt.Sprintf("send keys to process %s", name), payload)
}
