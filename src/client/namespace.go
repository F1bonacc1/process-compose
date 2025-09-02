package client

import (
    "fmt"
    "net/http"
)

// namespacePatch is a helper to perform PATCH actions on namespace endpoints
// and unify response decoding and error handling.
func (p *PcClient) namespacePatch(url string, partialErrMsg string) (map[string]string, error) {
    return p.doMapRequest(http.MethodPatch, url, nil, partialErrMsg)
}

func (p *PcClient) StopNamespace(name string) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace/stop/%s", p.address, name)
    return p.namespacePatch(url, "failed to stop some processes")
}

func (p *PcClient) DisableNamespace(name string) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace/disable/%s", p.address, name)
    return p.namespacePatch(url, "failed to disable some processes")
}

func (p *PcClient) EnableNamespace(name string) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace/enable/%s", p.address, name)
    return p.namespacePatch(url, "failed to enable some processes")
}

func (p *PcClient) RemoveNamespace(name string) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace?name=%s", p.address, name)
    return p.doMapRequest(http.MethodDelete, url, nil, "failed to remove some processes")
}
