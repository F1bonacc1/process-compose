package updater

import (
	"encoding/json"
	"net/http"
)

const (
	//UrlPath for retrieving the latest release version
	UrlPath = "https://api.github.com/repos/f1bonacc1/process-compose/releases/latest"
	// UrlPath = "https://shr.pn/process-compose-latest"
)

type Release struct {
	Name string `json:"name"`
}

func GetLatestReleaseName() (string, error) {

	// 1. Create a custom client
	client := &http.Client{}

	// 2. Create the request
	req, err := http.NewRequest(http.MethodGet, UrlPath, nil)
	if err != nil {
		return "", err
	}

	// This makes ShortPen see the call as coming from a real Chrome browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// 4. Execute the request
	resp, err := client.Do(req)
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
