package app

import (
	"github.com/f1bonacc1/process-compose/src/loader"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func getFixtures() []string {
	matches, err := filepath.Glob("../../fixtures/process-compose-*.yaml")
	if err != nil {
		panic("no fixtures found")
	}
	return matches
}

func TestSystem_TestFixtures(t *testing.T) {
	fixtures := getFixtures()
	for _, fixture := range fixtures {

		if strings.Contains(fixture, "process-compose-with-log.yaml") {
			//there is a dedicated test for that TestSystem_TestComposeWithLog
			continue
		}

		if strings.Contains(fixture, "process-compose-chain-exit.yaml") {
			//there is a dedicated test for that TestSystem_TestComposeWithLog
			continue
		}

		t.Run(fixture, func(t *testing.T) {
			project, err := loader.Load(&loader.LoaderOptions{
				FileNames: []string{fixture},
			})
			if err != nil {
				t.Errorf(err.Error())
				return
			}
			runner, err := NewProjectRunner(project, []string{}, false)
			if err != nil {
				t.Errorf(err.Error())
				return
			}
			runner.Run()
		})
	}
}

func TestSystem_TestComposeWithLog(t *testing.T) {
	fixture := filepath.Join("..", "..", "fixtures", "process-compose-with-log.yaml")
	t.Run(fixture, func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner, err := NewProjectRunner(project, []string{}, false)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner.Run()
		if _, err := os.Stat(runner.project.LogLocation); err != nil {
			t.Errorf("log file %s not found", runner.project.LogLocation)
		}
		if err := os.Remove(runner.project.LogLocation); err != nil {
			t.Errorf("failed to delete the log file %s, %s", runner.project.LogLocation, err.Error())
		}

		proc6log := runner.project.Processes["process6"].LogLocation
		if _, err := os.Stat(proc6log); err != nil {
			t.Errorf("log file %s not found", proc6log)
		}
		if err := os.Remove(proc6log); err != nil {
			t.Errorf("failed to delete the log file %s, %s", proc6log, err.Error())
		}
	})
}

func TestSystem_TestComposeChain(t *testing.T) {
	fixture := filepath.Join("..", "..", "fixtures", "process-compose-chain.yaml")
	t.Run(fixture, func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner, err := NewProjectRunner(project, []string{}, false)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		names, err := runner.GetDependenciesOrderNames()
		if err != nil {
			t.Errorf("GetDependenciesOrderNames() error = %v", err)
			return
		}
		want := []string{
			"process8",
			"process7",
			"process6",
			"process5",
			"process4",
			"process3",
			"process2",
			"process1",
		}
		if !reflect.DeepEqual(names, want) {
			t.Errorf("Project.GetDependenciesOrderNames() = %v, want %v", names, want)
		}
	})
}

func TestSystem_TestComposeChainExit(t *testing.T) {
	fixture := filepath.Join("..", "..", "fixtures", "process-compose-chain-exit.yaml")
	t.Run(fixture, func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner, err := NewProjectRunner(project, []string{}, false)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		exitCode := runner.Run()
		want := 42
		if want != exitCode {
			t.Errorf("Project.Run() = %v, want %v", exitCode, want)
		}
	})
}
