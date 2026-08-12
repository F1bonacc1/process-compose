package app

import (
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/f1bonacc1/process-compose/src/admitter"
	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/types"
)

func loadNamespaceFixture(t *testing.T, namespaces ...string) *types.Project {
	t.Helper()
	return loadFixtureWithNamespaces(t, "process-compose-namespace-deps.yaml", namespaces...)
}

func loadFixtureWithNamespaces(t *testing.T, fixtureName string, namespaces ...string) *types.Project {
	t.Helper()
	fixture := filepath.Join("..", "..", "fixtures-code", fixtureName)
	opts := &loader.LoaderOptions{
		FileNames: []string{fixture},
	}
	opts.AddAdmitter(&admitter.NamespaceAdmitter{EnabledNamespaces: namespaces})
	project, err := loader.Load(opts)
	if err != nil {
		t.Fatal(err.Error())
	}
	return project
}

func TestNamespaceFilter_PrunesCrossNamespaceDeps(t *testing.T) {
	project := loadNamespaceFixture(t, "foo")

	if _, ok := project.Processes["bar"]; ok {
		t.Fatal("excluded process bar should be removed from the project")
	}
	foo, ok := project.Processes["foo"]
	if !ok {
		t.Fatal("admitted process foo should stay in the project")
	}
	if len(foo.DependsOn) != 0 {
		t.Fatalf("foo's dependency on the excluded bar should be pruned, got %v", foo.DependsOn)
	}

	runner, err := NewProjectRunner(&ProjectOpts{
		project:         project,
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := runner.Run(); err != nil {
		t.Fatalf("Run() should succeed with a pruned cross-namespace dependency, got: %v", err)
	}

	state, err := runner.GetProcessState("foo")
	if err != nil {
		t.Fatal(err.Error())
	}
	if state.Status != types.ProcessStateCompleted || state.ExitCode != 0 {
		t.Errorf("foo should complete successfully, got status=%s exit=%d", state.Status, state.ExitCode)
	}

	states, err := runner.GetProcessesState()
	if err != nil {
		t.Fatal(err.Error())
	}
	if len(states.States) != 2 {
		t.Errorf("expected 2 visible processes, got %d", len(states.States))
	}
	for _, s := range states.States {
		if s.Name == "bar" {
			t.Error("excluded process bar should not appear in the states list")
		}
	}
}

func TestNamespaceFilter_IntraNamespaceDepsAreKept(t *testing.T) {
	// no admitted namespaces means admit all - nothing should be pruned
	project := loadNamespaceFixture(t)

	foo := project.Processes["foo"]
	if _, ok := foo.DependsOn["bar"]; !ok {
		t.Fatal("without namespace filtering foo's dependency on bar should be kept")
	}
}

func TestNamespaceFilter_ExplicitSelectionOfExcludedProcessFails(t *testing.T) {
	project := loadNamespaceFixture(t, "foo")

	_, err := NewProjectRunner(&ProjectOpts{
		project:         project,
		processesToRun:  []string{"bar"},
		mainProcessArgs: []string{},
	})
	if err == nil {
		t.Fatal("selecting an excluded process should fail")
	}
	if !strings.Contains(err.Error(), "no such process: bar") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNamespaceFilter_StartExcludedProcessFails(t *testing.T) {
	project := loadNamespaceFixture(t, "foo")

	runner, err := NewProjectRunner(&ProjectOpts{
		project:         project,
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := runner.StartProcess("bar"); err == nil {
		t.Fatal("starting an excluded process should fail")
	}
}

func TestNamespaceFilter_MultiNamespace(t *testing.T) {
	// -n foo admits the multi-namespace process and prunes the others
	project := loadFixtureWithNamespaces(t, "process-compose-multi-namespace.yaml", "foo")
	if len(project.Processes) != 1 {
		t.Fatalf("expected only foo to be admitted, got %v", project.Processes)
	}
	if _, ok := project.Processes["foo"]; !ok {
		t.Fatal("foo should be admitted via its first namespace")
	}

	// -n foobar admits every process that lists it
	project = loadFixtureWithNamespaces(t, "process-compose-multi-namespace.yaml", "foobar")
	if len(project.Processes) != 2 {
		t.Fatalf("expected foo and bar to be admitted, got %v", project.Processes)
	}
	if _, ok := project.Processes["baz"]; ok {
		t.Fatal("baz should be pruned when selecting foobar")
	}

	// a namespace no process belongs to prunes everything
	project = loadFixtureWithNamespaces(t, "process-compose-multi-namespace.yaml", "unknown")
	if len(project.Processes) != 0 {
		t.Fatalf("expected no admitted processes, got %v", project.Processes)
	}
}

func TestNamespaceFilter_MultiNamespaceProjectOps(t *testing.T) {
	project := loadFixtureWithNamespaces(t, "process-compose-multi-namespace.yaml")

	runner, err := NewProjectRunner(&ProjectOpts{
		project:         project,
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	namespaces, err := runner.GetNamespaces()
	if err != nil {
		t.Fatal(err.Error())
	}
	want := []string{"bar", "baz", "foo", "foobar"}
	if !reflect.DeepEqual(namespaces, want) {
		t.Errorf("GetNamespaces() = %v, want %v", namespaces, want)
	}

	procs, err := runner.getNamespaceProcesses("foobar")
	if err != nil {
		t.Fatal(err.Error())
	}
	slices.Sort(procs)
	if !reflect.DeepEqual(procs, []string{"bar", "foo"}) {
		t.Errorf("getNamespaceProcesses(foobar) = %v, want [bar foo]", procs)
	}

	procs, err = runner.getNamespaceProcesses("baz")
	if err != nil {
		t.Fatal(err.Error())
	}
	if !reflect.DeepEqual(procs, []string{"baz"}) {
		t.Errorf("getNamespaceProcesses(baz) = %v, want [baz]", procs)
	}
}

func TestNamespaceFilter_SurvivesProjectUpdate(t *testing.T) {
	nsAdmitter := &admitter.NamespaceAdmitter{EnabledNamespaces: []string{"foo"}}
	project := loadNamespaceFixture(t, "foo")

	runner, err := NewProjectRunner(&ProjectOpts{
		project:         project,
		processesToRun:  []string{},
		mainProcessArgs: []string{},
		admitters:       []admitter.Admitter{nsAdmitter},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	// Simulate a config reload: the freshly loaded project has no admission
	// applied (ReloadProject loads without admitters).
	reloaded := loadNamespaceFixture(t)
	if _, err := runner.UpdateProject(reloaded); err != nil {
		t.Fatal(err.Error())
	}

	if _, ok := runner.project.Processes["bar"]; ok {
		t.Fatal("excluded process bar should not be resurrected by a project update")
	}
	foo := runner.project.Processes["foo"]
	if len(foo.DependsOn) != 0 {
		t.Fatalf("foo's pruned dependency should stay pruned after update, got %v", foo.DependsOn)
	}
}

// TestNamespaceOps_StartExcludedFromUpSelection covers issue #528: processes
// left out of an `up <process>...` selection are only disabled, not removed,
// so a namespace operation must still be able to start them later.
func TestNamespaceOps_StartExcludedFromUpSelection(t *testing.T) {
	project := loadFixtureWithNamespaces(t, "process-compose-namespace-profiles.yaml")

	runner, err := NewProjectRunner(&ProjectOpts{
		project:         project,
		processesToRun:  []string{"customer-api"},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatal(err.Error())
	}

	for _, name := range []string{"admin-api", "admin-ui", "shared-db"} {
		if !runner.project.Processes[name].Disabled {
			t.Fatalf("%s should be disabled by the `up customer-api` selection", name)
		}
	}

	// The admin namespace is resolvable even though all of its members are
	// disabled - this is what used to fail with "no processes assigned".
	adminProcs, err := runner.getNamespaceProcesses("admin")
	if err != nil {
		t.Fatalf("getNamespaceProcesses(admin) error: %v", err)
	}
	if !reflect.DeepEqual(adminProcs, []string{"admin-api", "admin-ui"}) {
		t.Errorf("getNamespaceProcesses(admin) = %v, want [admin-api admin-ui]", adminProcs)
	}
	infraProcs, err := runner.getNamespaceProcesses("infra")
	if err != nil {
		t.Fatalf("getNamespaceProcesses(infra) error: %v", err)
	}
	if !reflect.DeepEqual(infraProcs, []string{"shared-db"}) {
		t.Errorf("getNamespaceProcesses(infra) = %v, want [shared-db]", infraProcs)
	}

	runErr := make(chan error, 1)
	go func() {
		runErr <- runner.Run()
	}()
	defer func() {
		_ = runner.ShutDownProject()
		<-runErr
	}()

	waitForProcessLaunched(t, runner, "customer-api", 10*time.Second)

	if err := runner.StartNamespace("admin"); err != nil {
		t.Fatalf("StartNamespace(admin) error: %v", err)
	}
	waitForProcessState(t, runner, "admin-api", types.ProcessStateCompleted, 10*time.Second)
	waitForProcessState(t, runner, "admin-ui", types.ProcessStateCompleted, 10*time.Second)

	// shared-db is a dependency of admin-api, but it lives in another
	// namespace - a namespace operation must not start it as a side effect.
	dbState, err := runner.GetProcessState("shared-db")
	if err != nil {
		t.Fatal(err.Error())
	}
	if dbState.Status != types.ProcessStateDisabled {
		t.Errorf("shared-db: status = %s, want %s (dependency outside the started namespace)",
			dbState.Status, types.ProcessStateDisabled)
	}

	// Stopping the admin namespace leaves the customer surface alone.
	if err := runner.StopNamespace("admin"); err != nil {
		t.Fatalf("StopNamespace(admin) error: %v", err)
	}
	customerState, err := runner.GetProcessState("customer-api")
	if err != nil {
		t.Fatal(err.Error())
	}
	if !customerState.IsRunning {
		t.Errorf("customer-api: status = %s, want it to keep running", customerState.Status)
	}
}
