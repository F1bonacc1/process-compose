package app

import (
	"bufio"
	"github.com/f1bonacc1/process-compose/src/command"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/types"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
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
		err = runner.Run()
		want := "project non-zero exit code: 42"
		if want != err.Error() {
			t.Errorf("Project.Run() = %v, want %v", err, want)
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

func TestSystem_TestProcShutDownNoRestart(t *testing.T) {
	restarting := "Restarting"
	notRestarting := "NotRestarting"
	shell := command.DefaultShellConfig()
	project := &types.Project{
		Processes: map[string]types.ProcessConfig{
			restarting: {
				Name:        restarting,
				ReplicaName: restarting,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "sleep 2"},
				RestartPolicy: types.RestartPolicyConfig{
					Restart: types.RestartPolicyAlways,
				},
			},
			notRestarting: {
				Name:        notRestarting,
				ReplicaName: notRestarting,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "sleep 2"},
				RestartPolicy: types.RestartPolicyConfig{
					Restart: types.RestartPolicyNo,
				},
			},
		},
		ShellConfig: shell,
	}
	runner, err := NewProjectRunner(&ProjectOpts{
		project: project,
	})
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	go runner.Run()
	time.Sleep(100 * time.Millisecond)
	state, err := runner.GetProcessState(restarting)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if state.Status != types.ProcessStateRunning {
		t.Errorf("process %s is not running", restarting)
		return
	}
	err = runner.StopProcess(restarting)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	time.Sleep(100 * time.Millisecond)
	state, err = runner.GetProcessState(restarting)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if state.Status != types.ProcessStateCompleted {
		t.Errorf("process %s want %s got %s", restarting, types.ProcessStateCompleted, state.Status)
		return
	}
	state, err = runner.GetProcessState(notRestarting)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if state.Status != types.ProcessStateRunning {
		t.Errorf("process %s is not running", notRestarting)
		return
	}
	err = runner.StopProcess(notRestarting)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

	time.Sleep(100 * time.Millisecond)
	state, err = runner.GetProcessState(notRestarting)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if state.Status != types.ProcessStateCompleted {
		t.Errorf("process %s is running", notRestarting)
		return
	}
}
func TestSystem_TestReadyLine(t *testing.T) {
	proc1 := "proc1"
	proc2 := "proc2"
	shell := command.DefaultShellConfig()
	project := &types.Project{
		Processes: map[string]types.ProcessConfig{
			proc1: {
				Name:         proc1,
				ReplicaName:  proc1,
				Executable:   shell.ShellCommand,
				Args:         []string{shell.ShellArgument, "sleep 0.3 && echo ready"},
				ReadyLogLine: "ready",
			},
			proc2: {
				Name:        proc2,
				ReplicaName: proc2,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "sleep 2"},
				DependsOn: map[string]types.ProcessDependency{
					proc1: {
						Condition: types.ProcessConditionLogReady,
					},
				},
			},
		},
		ShellConfig: shell,
	}
	runner, err := NewProjectRunner(&ProjectOpts{
		project: project,
	})
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	go runner.Run()
	time.Sleep(100 * time.Millisecond)
	state := runner.getRunningProcess(proc2).getStatusName()

	if state != types.ProcessStatePending {
		t.Errorf("process %s is %s want %s", proc2, state, types.ProcessStatePending)
		return
	}
	time.Sleep(400 * time.Millisecond)
	state = runner.getRunningProcess(proc2).getStatusName()
	if state != types.ProcessStateRunning {
		t.Errorf("process %s is %s want %s", proc2, state, types.ProcessStateRunning)
		return
	}
}

func TestUpdateProject(t *testing.T) {
	proc1 := "process1"
	proc2 := "process2"
	proc3 := "process3"
	shell := command.DefaultShellConfig()
	p, err := NewProjectRunner(&ProjectOpts{
		project: &types.Project{
			ShellConfig: shell,
			Processes: map[string]types.ProcessConfig{
				"process1": {
					Name:        proc1,
					ReplicaName: proc1,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, "echo process1"},
					Environment: []string{
						"VAR1=value1",
						"VAR2=value2",
					},
				},
				"process2": {
					Name:        proc2,
					ReplicaName: proc2,
					Executable:  shell.ShellCommand,
					Args:        []string{shell.ShellArgument, "echo process2"},
					Environment: []string{
						"VAR3=value3",
						"VAR4=value4",
					},
				},
			},
		},
	})
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	go p.Run()
	time.Sleep(100 * time.Millisecond)

	// Test when no changes are made
	project := &types.Project{
		ShellConfig: shell,
		Processes: map[string]types.ProcessConfig{
			"process1": {
				Name:        proc1,
				ReplicaName: proc1,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "echo process1"},
				Environment: []string{
					"VAR1=value1",
					"VAR2=value2",
				},
			},
			"process2": {
				Name:        proc2,
				ReplicaName: proc2,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "echo process2"},
				Environment: []string{
					"VAR3=value3",
					"VAR4=value4",
				},
			},
		},
	}
	status, err := p.UpdateProject(project)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(status) != 0 {
		t.Errorf("Unexpected status: %v", status)
	}

	// Test when a process is updated
	project = &types.Project{
		ShellConfig: shell,
		Processes: map[string]types.ProcessConfig{
			"process1": {
				Name:        proc1,
				ReplicaName: proc1,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "echo updated"},
				Environment: []string{
					"VAR1=value1",
					"VAR2=value2",
				},
			},
			"process2": {
				Name:        proc2,
				ReplicaName: proc2,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "echo process2"},
				Environment: []string{
					"VAR3=value3",
					"VAR4=value4",
				},
			},
		},
	}
	status, err = p.UpdateProject(project)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	proc, ok := p.project.Processes[proc1]
	if !ok {
		t.Errorf("Process 'process1' not found in updated project")
	}
	if proc.Args[1] != "echo updated" {
		t.Errorf("Process 'process1' command is %s want 'echo updated'", proc.Args[1])
	}
	updatedStatus := status[proc1]
	if updatedStatus != types.ProcessUpdateUpdated {
		t.Errorf("Process 'process1' status is %s want %s", updatedStatus, types.ProcessUpdateUpdated)
	}

	// Test when a process is deleted
	project = &types.Project{
		Processes: map[string]types.ProcessConfig{
			"process2": {
				Name:        proc2,
				ReplicaName: proc2,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "echo process2"},
				Environment: []string{
					"VAR3=value3",
					"VAR4=value4",
				},
			},
		},
	}
	status, err = p.UpdateProject(project)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if _, ok = p.project.Processes[proc1]; ok {
		t.Errorf("Process 'process1' still exists in updated project")
	}
	updatedStatus = status[proc1]
	if updatedStatus != types.ProcessUpdateRemoved {
		t.Errorf("Process 'process1' status is %s want %s", updatedStatus, types.ProcessUpdateRemoved)
	}

	// Test when a new process is added
	project = &types.Project{
		Processes: map[string]types.ProcessConfig{
			"process3": {
				Name:        proc3,
				ReplicaName: proc3,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, "echo process3"},
				Environment: []string{
					"VAR5=value5",
					"VAR6=value6",
				},
			},
		},
	}
	status, err = p.UpdateProject(project)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if _, ok = p.project.Processes[proc3]; !ok {
		t.Errorf("Process 'process3' not found in updated project")
	}
	updatedStatus = status[proc3]
	if updatedStatus != types.ProcessUpdateAdded {
		t.Errorf("Process 'process1' status is %s want %s", updatedStatus, types.ProcessUpdateAdded)
	}
}

func assertProcessStatus(t *testing.T, proc *Process, procName string, wantStatus string) {
	t.Helper()
	status := proc.getStatusName()
	if status != wantStatus {
		t.Fatalf("process %s status want %s got %s", procName, wantStatus, status)
	}
}

func TestSystem_TestProcShutDownWithConfiguredTimeOut(t *testing.T) {
	ignoresSigTerm := "IgnoresSIGTERM"
	shell := command.DefaultShellConfig()
	timeout := 3

	project := &types.Project{
		Processes: map[string]types.ProcessConfig{
			ignoresSigTerm: {
				Name:        ignoresSigTerm,
				ReplicaName: ignoresSigTerm,
				Executable:  shell.ShellCommand,
				Args:        []string{shell.ShellArgument, ""},
				ShutDownParams: types.ShutDownParams{
					ShutDownTimeout: timeout,
					Signal:          int(syscall.SIGTERM),
				},
			},
		},
		ShellConfig: shell,
	}
	t.Run("with timeout sigterm fail", func(t *testing.T) {
		procConf := project.Processes[ignoresSigTerm]
		procConf.Args[1] = "trap '' SIGTERM && sleep 60"
		project.Processes[ignoresSigTerm] = procConf
		runner, err := NewProjectRunner(&ProjectOpts{project: project})
		if err != nil {
			t.Fatalf("%s", err)
		}
		go runner.Run()
		time.Sleep(100 * time.Millisecond)
		proc := runner.getRunningProcess(ignoresSigTerm)
		assertProcessStatus(t, proc, ignoresSigTerm, types.ProcessStateRunning)

		// If the test fails, cleanup after ourselves
		defer proc.command.Stop(int(syscall.SIGKILL), true)

		go func() {
			err = runner.StopProcess(ignoresSigTerm)
			if err != nil {
				t.Fatalf("%s", err)
			}
		}()

		for i := 0; i < timeout-1; i++ {
			time.Sleep(time.Second)
			assertProcessStatus(t, proc, ignoresSigTerm, types.ProcessStateTerminating)
		}

		time.Sleep(2 * time.Second)
		assertProcessStatus(t, proc, ignoresSigTerm, types.ProcessStateCompleted)
	})

	t.Run("with timeout sigterm success", func(t *testing.T) {
		procConf := project.Processes[ignoresSigTerm]
		procConf.Args[1] = "sleep 60"
		project.Processes[ignoresSigTerm] = procConf
		runner, err := NewProjectRunner(&ProjectOpts{project: project})
		if err != nil {
			t.Fatalf("%s", err)
		}
		go runner.Run()
		time.Sleep(100 * time.Millisecond)
		proc := runner.getRunningProcess(ignoresSigTerm)
		assertProcessStatus(t, proc, ignoresSigTerm, types.ProcessStateRunning)
		go func() {
			err = runner.StopProcess(ignoresSigTerm)
			if err != nil {
				t.Fatalf("%s", err)
			}
		}()
		time.Sleep(200 * time.Millisecond)
		assertProcessStatus(t, proc, ignoresSigTerm, types.ProcessStateCompleted)
	})

}
