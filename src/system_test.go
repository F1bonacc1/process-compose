package main

import (
	"path/filepath"
	"testing"
)

func getFixtures() []string {
	matches, err := filepath.Glob("../fixtures/process-compose-*.yaml")
	if err != nil {
		panic("no fixtures found")
	}
	return matches
}

func TestSystem_TestFixtures(t *testing.T) {
	fixtures := getFixtures()
	for _, fixture := range fixtures {

		t.Run(fixture, func(t *testing.T) {
			project := createProject(fixture)
			project.Run()
		})
	}
}
