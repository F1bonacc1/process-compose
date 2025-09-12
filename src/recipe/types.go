package recipe

import (
	"time"
)

// Recipe represents a process-compose recipe
type Recipe struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Version     string            `json:"version" yaml:"version"`
	Author      string            `json:"author" yaml:"author"`
	Tags        []string          `json:"tags" yaml:"tags"`
	LastUpdated time.Time         `json:"last_updated" yaml:"last_updated"`
	MinVersion  string            `json:"min_version" yaml:"min_version"`                 // Minimum process-compose version required
	Variables   map[string]string `json:"variables,omitempty" yaml:"variables,omitempty"` // Default variable values
	Repository  string            `json:"repository" yaml:"repository"`
}

// RecipeIndex represents the index of all available recipes
type RecipeIndex struct {
	Recipes     map[string]Recipe `json:"recipes" yaml:"recipes"`
	LastUpdated time.Time         `json:"last_updated" yaml:"last_updated"`
	Version     string            `json:"version" yaml:"version"`
}

// LocalRecipe represents a recipe stored locally
type LocalRecipe struct {
	Recipe
	Path string `json:"path"`
}

// SearchFilter contains criteria for searching recipes
type SearchFilter struct {
	Name        string
	Tags        []string
	Author      string
	Description string
}
