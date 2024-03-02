package app

import (
	"bufio"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/types"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
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
		t.Run(fixture, func(t *testing.T) {
			project, err := loader.Load(&loader.LoaderOptions{
				FileNames: []string{fixture},
			})
			if err != nil {
				t.Errorf(err.Error())
				return
			}
			runner, err := NewProjectRunner(&ProjectOpts{
				project:         project,
				processesToRun:  []string{},
				noDeps:          false,
				mainProcess:     "",
				mainProcessArgs: []string{},
				isTuiOn:         false,
			})
			if err != nil {
				t.Errorf(err.Error())
				return
			}
			runner.Run()
		})
	}
}

func TestSystem_TestComposeWithLog(t *testing.T) {
	fixture := filepath.Join("..", "..", "fixtures-code", "process-compose-with-log.yaml")
	t.Run(fixture, func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner, err := NewProjectRunner(&ProjectOpts{
			project:         project,
			processesToRun:  []string{},
			mainProcessArgs: []string{},
		})
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
	fixture := filepath.Join("..", "..", "fixtures-code", "process-compose-chain.yaml")
	t.Run(fixture, func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner, err := NewProjectRunner(&ProjectOpts{
			project:         project,
			processesToRun:  []string{},
			mainProcessArgs: []string{},
		})
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
	fixture := filepath.Join("..", "..", "fixtures-code", "process-compose-chain-exit.yaml")
	t.Run(fixture, func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner, err := NewProjectRunner(&ProjectOpts{
			project:         project,
			processesToRun:  []string{},
			mainProcessArgs: []string{},
		})
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

func TestSystem_TestComposeCircular(t *testing.T) {
	fixture1 := filepath.Join("..", "..", "fixtures-code", "process-compose-circular.yaml")
	fixture2 := filepath.Join("..", "..", "fixtures-code", "process-compose-non-circular.yaml")
	t.Run(fixture1, func(t *testing.T) {
		_, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture1},
		})
		if err == nil {
			t.Errorf("should fail on cirlcular dependency")
			return
		}

		_, err = loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture2},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
	})
}

func TestSystem_TestComposeScale(t *testing.T) {
	fixture := filepath.Join("..", "..", "fixtures-code", "process-compose-scale.yaml")
	t.Run(fixture, func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner, err := NewProjectRunner(&ProjectOpts{
			project:         project,
			processesToRun:  []string{},
			mainProcessArgs: []string{},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		go runner.Run()
		time.Sleep(200 * time.Millisecond)
		states, err := runner.GetProcessesState()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		want := 4
		if len(states.States) != want {
			t.Errorf("len(states.States) = %d, want %d", len(states.States), want)
		}

		//scale to 10
		err = runner.ScaleProcess("process1-0", 10)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		states, err = runner.GetProcessesState()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		want = 12
		if len(states.States) != want {
			t.Errorf("len(states.States) = %d, want %d", len(states.States), want)
		}

		//check scale to 0 - should fail
		err = runner.ScaleProcess("process1-00", 0)
		if err == nil {
			t.Errorf("should fail on scale 0")
			return
		}

		//scale to 1 and new name with -00
		err = runner.ScaleProcess("process1-00", 1)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		states, err = runner.GetProcessesState()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		want = 3
		if len(states.States) != want {
			t.Errorf("len(states.States) = %d, want %d", len(states.States), want)
		}

		//scale to 5 process2
		err = runner.ScaleProcess("process2", 5)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		states, err = runner.GetProcessesState()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		want = 7
		if len(states.States) != want {
			t.Errorf("len(states.States) = %d, want %d", len(states.States), want)
		}

		//check no change
		err = runner.ScaleProcess("process2-0", 5)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		states, err = runner.GetProcessesState()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		want = 7
		if len(states.States) != want {
			t.Errorf("len(states.States) = %d, want %d", len(states.States), want)
		}

		//wrong process name
		err = runner.ScaleProcess("process2-00", 5)
		if err == nil {
			t.Errorf("should fail on wrong process name")
			return
		}
	})
}

func TestSystem_TestTransitiveDependency(t *testing.T) {
	fixture1 := filepath.Join("..", "..", "fixtures-code", "process-compose-transitive-dep.yaml")
	t.Run(fixture1, func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture1},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runner, err := NewProjectRunner(&ProjectOpts{
			project:         project,
			processesToRun:  []string{},
			mainProcessArgs: []string{},
		})
		runner.Run()

		states, err := runner.GetProcessesState()
		for _, state := range states.States {
			if state.ExitCode != 1 {
				t.Errorf("process %s exit code is not 1", state.Name)
			}
		}
	})
}

func TestSystem_TestProcListToRun(t *testing.T) {
	fixture1 := filepath.Join("..", "..", "fixtures-code", "process-compose-transitive-dep.yaml")
	t.Run("Single Proc", func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture1},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		numProc := len(project.Processes)
		runner, err := NewProjectRunner(&ProjectOpts{
			project:         project,
			processesToRun:  []string{"procA"},
			mainProcessArgs: []string{},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		if len(runner.project.Processes) != numProc {
			t.Errorf("should have %d processes", numProc)
		}
		for name, proc := range runner.project.Processes {
			if name == "procA" {
				if proc.Disabled {
					t.Errorf("process %s is disabled", name)
				}
			} else {
				if !proc.Disabled {
					t.Errorf("process %s is not disabled", name)
				}
			}
		}

	})
	t.Run("Single Proc with deps", func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture1},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		numProc := len(project.Processes)
		runner, err := NewProjectRunner(&ProjectOpts{
			project:         project,
			processesToRun:  []string{"procC"},
			mainProcessArgs: []string{},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		if len(runner.project.Processes) != numProc {
			t.Errorf("should have %d processes", numProc)
		}
		for name, proc := range runner.project.Processes {
			if proc.Disabled {
				t.Errorf("process %s is disabled", name)
			}
		}
	})
	t.Run("Single Proc no deps", func(t *testing.T) {
		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture1},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		numProc := len(project.Processes)
		runner, err := NewProjectRunner(&ProjectOpts{
			project:         project,
			processesToRun:  []string{"procC"},
			mainProcessArgs: []string{},
			noDeps:          true,
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		if len(runner.project.Processes) != numProc {
			t.Errorf("should have %d processes", numProc)
		}
		for name, proc := range runner.project.Processes {
			if name == "procC" {
				if proc.Disabled {
					t.Errorf("process %s is disabled", name)
				}
			} else {
				if !proc.Disabled {
					t.Errorf("process %s is not disabled", name)
				}
			}
		}
	})
}

func TestSystem_TestProcListShutsDownInOrder(t *testing.T) {
	fixture1 := filepath.Join("..", "..", "fixtures-code", "process-compose-shutdown-inorder.yaml")
	t.Run("Single Proc with deps", func(t *testing.T) {

		project, err := loader.Load(&loader.LoaderOptions{
			FileNames: []string{fixture1},
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		numProc := len(project.Processes)
		runner, err := NewProjectRunner(&ProjectOpts{
			project:           project,
			processesToRun:    []string{},
			mainProcessArgs:   []string{},
			isOrderedShutDown: true,
		})
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		if len(runner.project.Processes) != numProc {
			t.Errorf("should have %d processes", numProc)
		}
		for name, proc := range runner.project.Processes {
			if proc.Disabled {
				t.Errorf("process %s is disabled", name)
			}
		}
		file, err := os.CreateTemp("/tmp", "pc_log.*.log")
		defer os.Remove(file.Name())
		project.LogLocation = file.Name()
		project.LoggerConfig = &types.LoggerConfig{
			FieldsOrder:     []string{"message"},
			DisableJSON:     true,
			TimestampFormat: "",
			NoMetadata:      true,
			FlushEachLine:   true,
			NoColor:         true,
		}
		go runner.Run()
		time.Sleep(10 * time.Millisecond)
		states, err := runner.GetProcessesState()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		want := 3
		if len(states.States) != want {
			t.Errorf("len(states.States) = %d, want %d", len(states.States), want)
		}

		time.Sleep(10 * time.Millisecond)
		err = runner.ShutDownProject()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		states, err = runner.GetProcessesState()
		if err != nil {
			t.Errorf(err.Error())
			return
		}
		runningProcesses := 0
		for _, processState := range states.States {
			if processState.IsRunning {
				runningProcesses++
			}
		}
		want = 0
		if runningProcesses != want {
			t.Errorf("runningProcesses = %d, want %d", runningProcesses, want)
		}
		//read file and validate the shutdown order
		scanner := bufio.NewScanner(file)
		order := make([]string, 0)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "exit") {
				order = append(order, line)
			}
		}
		wantOrder := []string{"C: exit", "B: exit", "A: exit"}
		if !slices.Equal(order, wantOrder) {
			t.Errorf("content = %v, want %v", order, wantOrder)
			return
		}
	})
}
