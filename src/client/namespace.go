package client

import (
    "encoding/json"
    "errors"
    "fmt"
    "net/http"

    "github.com/rs/zerolog/log"
)

func (p *PcClient) StopNamespace(name string) (map[string]string, error) {
    url := fmt.Sprintf("http://%s/namespace/stop/%s", p.address, name)
    req, err := http.NewRequest(http.MethodPatch, url, nil)
    if err != nil {
        return nil, err
    }
    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
        stopped := map[string]string{}
        if err = json.NewDecoder(resp.Body).Decode(&stopped); err != nil {
            log.Err(err).Msg("failed to decode stop namespace response")
            return stopped, err
        }
        if resp.StatusCode == http.StatusBadRequest {
            return stopped, errors.New("failed to stop some processes")
        }
        return stopped, nil
    }

    var respErr pcError
    if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
        log.Err(err).Msg("failed to decode stop namespace error")
        return nil, err
    }
    return nil, errors.New(respErr.Error)
}

func (p *PcClient) DisableNamespace(name string) (map[string]string, error) {
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

    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
        results := map[string]string{}
        if err = json.NewDecoder(resp.Body).Decode(&results); err != nil {
            log.Err(err).Msg("failed to decode disable namespace response")
            return results, err
        }
        if resp.StatusCode == http.StatusBadRequest {
            return results, errors.New("failed to disable some processes")
        }
        return results, nil
    }

    var respErr pcError
    if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
        log.Err(err).Msg("failed to decode disable namespace error")
        return nil, err
    }
    return nil, errors.New(respErr.Error)
}

func (p *PcClient) EnableNamespace(name string) (map[string]string, error) {
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

    if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadRequest {
        results := map[string]string{}
        if err = json.NewDecoder(resp.Body).Decode(&results); err != nil {
            log.Err(err).Msg("failed to decode enable namespace response")
            return results, err
        }
        if resp.StatusCode == http.StatusBadRequest {
            return results, errors.New("failed to enable some processes")
        }
        return results, nil
    }

    var respErr pcError
    if err = json.NewDecoder(resp.Body).Decode(&respErr); err != nil {
        log.Err(err).Msg("failed to decode enable namespace error")
        return nil, err
    }
    return nil, errors.New(respErr.Error)
}

