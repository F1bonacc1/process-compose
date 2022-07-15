package updater

import (
	"encoding/json"
	"net/http"
)

const (
	//UrlPath for retrieving the latest release version
	UrlPath = "https://api.github.com/repos/f1bonacc1/process-compose/releases/latest"
)

type Release struct {
	Name string `json:"name"`
}

func GetLatestReleaseName() (string, error) {
	resp, err := http.Get(UrlPath)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var cResp Release
	if err := json.NewDecoder(resp.Body).Decode(&cResp); err != nil {
		return "", err
	}
	return cResp.Name, nil
}
