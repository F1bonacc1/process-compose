package recipe

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

const (
	DefaultRepoURL     = "https://api.github.com/repos/f1bonacc1/process-compose-recipes"
	DefaultTimeout     = 30 * time.Second
	RecipeMetadataFile = "recipe.yaml"
	ProcessComposeFile = "process-compose.yaml"
)

// Client handles communication with the recipe repository
type Client struct {
	httpClient *http.Client
	repoURL    string
	userAgent  string
}

// NewClient creates a new recipe client
func NewClient(repoURL string) *Client {
	if repoURL == "" {
		repoURL = DefaultRepoURL
	}

	return &Client{
		httpClient: &http.Client{Timeout: DefaultTimeout},
		repoURL:    repoURL,
		userAgent:  "process-compose-recipe-client/1.0",
	}
}

// GitHubContent represents a GitHub API content response
type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	DownloadURL string `json:"download_url"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
}

// GetRecipeIndex fetches and parses the recipe index
func (c *Client) GetRecipeIndex(ctx context.Context) (*RecipeIndex, error) {
	// Get all directories from the recipes repository
	url := fmt.Sprintf("%s/contents", c.repoURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repository contents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to decode repository contents: %w", err)
	}

	index := &RecipeIndex{
		Recipes:     make(map[string]Recipe),
		LastUpdated: time.Now(),
		Version:     "1.0",
	}

	// Fetch metadata for each directory
	for _, content := range contents {
		if content.Type == "dir" {
			recipe, err := c.getRecipeMetadata(ctx, content.Name)
			if err != nil {
				log.Err(err).Msgf("Failed to fetch metadata for %s", content.Name)
				continue
			}
			index.Recipes[content.Name] = *recipe
		}
	}

	return index, nil
}

// getRecipeMetadata fetches recipe metadata from a specific recipe directory
func (c *Client) getRecipeMetadata(ctx context.Context, recipeName string) (*Recipe, error) {
	url := fmt.Sprintf("%s/contents/%s/%s", c.repoURL, recipeName, RecipeMetadataFile)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("recipe metadata not found for %s", recipeName)
	}

	var content GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, fmt.Errorf("failed to decode metadata response: %w", err)
	}

	//decode content if needed
	if content.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(content.Content)
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 content: %w", err)
		}
		content.Content = string(decoded)
	}
	//unmarshal recipe metadata
	var recipe Recipe
	if err := yaml.Unmarshal([]byte(content.Content), &recipe); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recipe metadata: %w", err)
	}

	return &recipe, nil
}

// DownloadRecipe downloads a complete recipe (metadata + process-compose.yaml)
func (c *Client) DownloadRecipe(ctx context.Context, recipeName string) (map[string][]byte, error) {
	files := make(map[string][]byte)

	// Download recipe metadata
	metadataContent, err := c.downloadFile(ctx, recipeName, RecipeMetadataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to download recipe metadata: %w", err)
	}
	files[RecipeMetadataFile] = metadataContent

	// Download process-compose.yaml
	processComposeContent, err := c.downloadFile(ctx, recipeName, ProcessComposeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to download process-compose.yaml: %w", err)
	}
	files[ProcessComposeFile] = processComposeContent

	return files, nil
}

// downloadFile downloads a specific file from a recipe directory
func (c *Client) downloadFile(ctx context.Context, recipeName, fileName string) ([]byte, error) {
	url := fmt.Sprintf("%s/contents/%s/%s", c.repoURL, recipeName, fileName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("file not found: %s/%s", recipeName, fileName)
	}

	var content GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, fmt.Errorf("failed to decode file response: %w", err)
	}

	// If download_url is available, fetch the raw content
	if content.DownloadURL != "" {
		return c.downloadRawFile(ctx, content.DownloadURL)
	}

	return nil, fmt.Errorf("no download URL available for file: %s", fileName)
}

// downloadRawFile downloads raw file content from a direct URL
func (c *Client) downloadRawFile(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download raw file, status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
