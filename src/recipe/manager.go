package recipe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/f1bonacc1/process-compose/src/config"
	"gopkg.in/yaml.v3"
)

// Manager handles local recipe operations
type Manager struct {
	client     *Client
	recipesDir string
	indexFile  string
	localIndex *RecipeIndex
}

// NewManager creates a new recipe manager
func NewManager(recipesDir string) *Manager {
	if recipesDir == "" {
		recipesDir = config.GetRecipesDir()
	}

	return &Manager{
		client:     NewClient(""),
		recipesDir: recipesDir,
		indexFile:  filepath.Join(recipesDir, "index.yaml"),
	}
}

// Initialize sets up the recipes directory and loads the local index
func (m *Manager) Initialize() error {
	if err := os.MkdirAll(m.recipesDir, 0755); err != nil {
		return fmt.Errorf("failed to create recipes directory: %w", err)
	}

	return m.loadLocalIndex()
}

// loadLocalIndex loads the local recipe index
func (m *Manager) loadLocalIndex() error {
	if _, err := os.Stat(m.indexFile); os.IsNotExist(err) {
		m.localIndex = &RecipeIndex{
			Recipes:     make(map[string]Recipe),
			LastUpdated: time.Now(),
			Version:     "1.0",
		}
		return m.saveLocalIndex()
	}

	data, err := os.ReadFile(m.indexFile)
	if err != nil {
		return fmt.Errorf("failed to read index file: %w", err)
	}

	var index RecipeIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("failed to parse index file: %w", err)
	}

	m.localIndex = &index
	return nil
}

// saveLocalIndex saves the local recipe index
func (m *Manager) saveLocalIndex() error {
	data, err := yaml.Marshal(m.localIndex)
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	return os.WriteFile(m.indexFile, data, 0644)
}

// PullRecipe downloads and installs a recipe locally
func (m *Manager) PullRecipe(ctx context.Context, recipeName string, force bool, outputPath string) error {
	recipePath := filepath.Join(m.recipesDir, recipeName)
	if outputPath != "" {
		recipePath = filepath.Join(outputPath, recipeName)
	}

	// Check if recipe already exists
	if _, err := os.Stat(recipePath); err == nil && !force {
		return fmt.Errorf("recipe '%s' already exists. Use --force to overwrite", recipeName)
	}

	// Download recipe files
	files, err := m.client.DownloadRecipe(ctx, recipeName)
	if err != nil {
		return fmt.Errorf("failed to download recipe: %w", err)
	}

	// Create recipe directory
	if err := os.MkdirAll(recipePath, 0755); err != nil {
		return fmt.Errorf("failed to create recipe directory: %w", err)
	}

	// Write files to local directory
	for fileName, content := range files {
		filePath := filepath.Join(recipePath, fileName)
		if err := os.WriteFile(filePath, content, config.RecipeFileMode); err != nil {
			return fmt.Errorf("failed to write file %s: %w", fileName, err)
		}
	}

	// Parse recipe metadata and update local index
	var recipe Recipe
	if metadataContent, exists := files[RecipeMetadataFile]; exists {
		if err := yaml.Unmarshal(metadataContent, &recipe); err == nil {
			recipe.Name = recipeName
			m.localIndex.Recipes[recipeName] = recipe
			m.localIndex.LastUpdated = time.Now()
			if err := m.saveLocalIndex(); err != nil {
				return err
			}
		}
	}

	fmt.Printf("Recipe '%s' pulled successfully to %s\n", recipeName, recipePath)
	return nil
}

// SearchRecipes searches for recipes based on filter criteria
func (m *Manager) SearchRecipes(ctx context.Context, filter SearchFilter) ([]Recipe, error) {
	// Fetch latest recipe index
	remoteIndex, err := m.client.GetRecipeIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recipe index: %w", err)
	}

	var results []Recipe

	for _, recipe := range remoteIndex.Recipes {
		if m.matchesFilter(recipe, filter) {
			results = append(results, recipe)
		}
	}

	return results, nil
}

// matchesFilter checks if a recipe matches the search filter
func (m *Manager) matchesFilter(recipe Recipe, filter SearchFilter) bool {
	if filter.Name != "" && !strings.Contains(strings.ToLower(recipe.Name), strings.ToLower(filter.Name)) {
		return false
	}

	if filter.Author != "" && !strings.Contains(strings.ToLower(recipe.Author), strings.ToLower(filter.Author)) {
		return false
	}

	if filter.Description != "" && !strings.Contains(strings.ToLower(recipe.Description), strings.ToLower(filter.Description)) {
		return false
	}

	if len(filter.Tags) > 0 {
		tagMatch := false
		for _, filterTag := range filter.Tags {
			for _, recipeTag := range recipe.Tags {
				if strings.EqualFold(filterTag, recipeTag) {
					tagMatch = true
					break
				}
			}
			if tagMatch {
				break
			}
		}
		if !tagMatch {
			return false
		}
	}

	return true
}

// ListLocalRecipes returns all locally installed recipes
func (m *Manager) ListLocalRecipes() ([]LocalRecipe, error) {
	var recipes []LocalRecipe

	for name, recipe := range m.localIndex.Recipes {
		recipePath := filepath.Join(m.recipesDir, name, "process-compose.yaml")
		if _, err := os.Stat(recipePath); err == nil {
			recipes = append(recipes, LocalRecipe{
				Recipe: recipe,
				Path:   recipePath,
			})
		}
	}

	return recipes, nil
}

// GetRecipe returns a specific local recipe
func (m *Manager) GetRecipe(recipeName string) (*LocalRecipe, error) {
	recipe, exists := m.localIndex.Recipes[recipeName]
	if !exists {
		return nil, fmt.Errorf("recipe '%s' not found locally", recipeName)
	}

	recipePath := filepath.Join(m.recipesDir, recipeName)
	if _, err := os.Stat(recipePath); err != nil {
		return nil, fmt.Errorf("recipe directory not found: %s", recipePath)
	}

	return &LocalRecipe{
		Recipe: recipe,
		Path:   recipePath,
	}, nil
}

// GetRecipeContent returns the content of a local recipe's process-compose.yaml file
func (m *Manager) GetRecipeContent(recipeName string) ([]byte, error) {
	recipe, err := m.GetRecipe(recipeName)
	if err != nil {
		return nil, err
	}

	processComposeFile := filepath.Join(recipe.Path, "process-compose.yaml")
	content, err := os.ReadFile(processComposeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read process-compose.yaml: %w", err)
	}

	return content, nil
}

// RemoveRecipe removes a local recipe
func (m *Manager) RemoveRecipe(recipeName string) error {
	recipePath := filepath.Join(m.recipesDir, recipeName)

	if err := os.RemoveAll(recipePath); err != nil {
		return fmt.Errorf("failed to remove recipe directory: %w", err)
	}

	delete(m.localIndex.Recipes, recipeName)
	m.localIndex.LastUpdated = time.Now()

	return m.saveLocalIndex()
}
