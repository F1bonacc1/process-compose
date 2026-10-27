package updater

import (
	"encoding/json"
	"net/http"

	"github.com/f1bonacc1/process-compose/src/config"
)

const (
	//UrlPath for retrieving the latest release version
	UrlPath = "https://api.github.com/repos/f1bonacc1/process-compose/releases/latest"
)

type Release struct {
	Name string `json:"name"`
}

func GetLatestReleaseName() (string, error) {
	return getLatestReleaseName(UrlPath)
}

// userAgent identifies this client to the GitHub API, as GitHub's API
// guidelines require.
func userAgent() string {
	return "process-compose/" + config.Version
}

func getLatestReleaseName(url string) (string, error) {

	// 1. Create a custom client
	client := &http.Client{}

	// 2. Create the request
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	// 3. Identify the client
	req.Header.Set("User-Agent", userAgent())

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
