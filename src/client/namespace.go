package client

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "github.com/rs/zerolog/log"
    "github.com/f1bonacc1/process-compose/src/types"
    "net/http"
)

func (p *PcClient) disableNamespace(name string) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace/disable/%s", p.address, name)
    req, err := http.NewRequest(http.MethodPatch, url, nil)
    if err != nil {
        return nil, err
    }
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus {
        status := map[string]string{}
        if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
            log.Err(err).Msgf("failed to decode disable namespace %s response", name)
            return status, err
        }
        return status, nil
    }
    var respErr pcError
    if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
        log.Err(err).Msgf("failed to decode err disable namespace %s", name)
        return nil, err
    }
    return nil, errors.New(respErr.Error)
}

func (p *PcClient) enableNamespace(name string) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace/enable/%s", p.address, name)
    req, err := http.NewRequest(http.MethodPatch, url, nil)
    if err != nil {
        return nil, err
    }
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus {
        status := map[string]string{}
        if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
            log.Err(err).Msgf("failed to decode enable namespace %s response", name)
            return status, err
        }
        return status, nil
    }
    var respErr pcError
    if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
        log.Err(err).Msgf("failed to decode err enable namespace %s", name)
        return nil, err
    }
    return nil, errors.New(respErr.Error)
}

func (p *PcClient) addNamespace(processes *types.Processes) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace", p.address)
    payload, err := json.Marshal(processes)
    if err != nil {
        return nil, err
    }
    req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus {
        status := map[string]string{}
        if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
            return status, err
        }
        return status, nil
    }
    var respErr pcError
    if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
        return nil, err
    }
    return nil, errors.New(respErr.Error)
}

func (p *PcClient) removeNamespace(name string) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace?name=%s", p.address, name)
    req, err := http.NewRequest(http.MethodDelete, url, nil)
    if err != nil {
        return nil, err
    }
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusMultiStatus {
        status := map[string]string{}
        if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
            return status, err
        }
        return status, nil
    }
    var respErr pcError
    if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
        return nil, err
    }
    return nil, errors.New(respErr.Error)
}
