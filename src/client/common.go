package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
)

type pcError struct {
	Error string `json:"error"`
}

// escapePathSegment escapes a URL path segment. "+" is escaped explicitly
// because gin decodes path params with url.QueryUnescape ("+" -> " ").
func escapePathSegment(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), "+", "%2B")
}

// parseErrorResponse extracts the server error from a non-OK response,
// falling back to the raw body for non-JSON responses (e.g. gin's plain
// text "404 page not found").
func parseErrorResponse(resp *http.Response, actionName string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Msgf("failed to read %s response: %v", actionName, err)
		return fmt.Errorf("%s failed: HTTP %d", actionName, resp.StatusCode)
	}
	var respErr pcError
	if json.Unmarshal(body, &respErr) == nil && respErr.Error != "" {
		return errors.New(respErr.Error)
	}
	return fmt.Errorf("%s failed: HTTP %d: %s", actionName, resp.StatusCode, strings.TrimSpace(string(body)))
}

func (p *PcClient) doActionWithBody(method, url, actionName string, payload any) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Error().Msgf("failed to marshal %s payload: %v", actionName, err)
		return err
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return parseErrorResponse(resp, actionName)
}

func (p *PcClient) doAction(method, url, actionName string) error {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}

	return parseErrorResponse(resp, actionName)
}
