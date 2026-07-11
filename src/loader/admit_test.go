package loader

import (
	"strings"
	"testing"

	"github.com/f1bonacc1/process-compose/src/admitter"
	"github.com/f1bonacc1/process-compose/src/types"
)

func newNamespaceProject() *types.Project {
	return &types.Project{
		Processes: types.Processes{
			"foo": {
				Name:        "foo",
				ReplicaName: "foo",
				Namespace:   "foo",
				DependsOn: types.DependsOnConfig{
					"bar": {Condition: types.ProcessConditionCompletedSuccessfully},
				},
			},
			"bar": {
				Name:        "bar",
				ReplicaName: "bar",
				Namespace:   "bar",
			},
		},
	}
}

func TestAdmitProcesses_PrunesDependencyOnRemovedProcess(t *testing.T) {
	p := newNamespaceProject()
	opts := &LoaderOptions{}
	opts.AddAdmitter(&admitter.NamespaceAdmitter{EnabledNamespaces: []string{"foo"}})

	if _, err := admitProcesses(opts, p); err != nil {
		t.Fatalf("admitProcesses() unexpected error: %v", err)
	}

	if _, ok := p.Processes["bar"]; ok {
		t.Fatalf("process 'bar' should have been pruned by the namespace admitter")
	}
	foo, ok := p.Processes["foo"]
	if !ok {
		t.Fatalf("process 'foo' should have been retained")
	}
	if _, ok := foo.DependsOn["bar"]; ok {
		t.Errorf("dangling dependency 'bar' should have been pruned from 'foo', got %v", foo.DependsOn)
	}
}

func TestAdmitProcesses_KeepsIntraNamespaceDependency(t *testing.T) {
	p := &types.Project{
		Processes: types.Processes{
			"foo": {
				Name:        "foo",
				ReplicaName: "foo",
				Namespace:   "web",
				DependsOn: types.DependsOnConfig{
					"bar": {Condition: types.ProcessConditionCompletedSuccessfully},
				},
			},
			"bar": {
				Name:        "bar",
				ReplicaName: "bar",
				Namespace:   "web",
			},
		},
	}
	opts := &LoaderOptions{}
	opts.AddAdmitter(&admitter.NamespaceAdmitter{EnabledNamespaces: []string{"web"}})

	if _, err := admitProcesses(opts, p); err != nil {
		t.Fatalf("admitProcesses() unexpected error: %v", err)
	}

	foo := p.Processes["foo"]
	if _, ok := foo.DependsOn["bar"]; !ok {
		t.Errorf("intra-namespace dependency 'bar' should be preserved on 'foo', got %v", foo.DependsOn)
	}
}

func TestAdmitProcesses_StrictNamespaceErrorsOnUnsatisfiableDependency(t *testing.T) {
	p := newNamespaceProject()
	opts := &LoaderOptions{StrictNamespace: true}
	opts.AddAdmitter(&admitter.NamespaceAdmitter{EnabledNamespaces: []string{"foo"}})

	_, err := admitProcesses(opts, p)
	if err == nil {
		t.Fatalf("admitProcesses() expected an error in strict-namespace mode, got nil")
	}
	if !strings.Contains(err.Error(), "'foo' -> 'bar'") {
		t.Errorf("error should describe the unsatisfiable dependency, got: %v", err)
	}
	// The dependency must be left intact so the failure is not silently masked.
	if _, ok := p.Processes["foo"].DependsOn["bar"]; !ok {
		t.Errorf("strict mode must not prune the dependency")
	}
}

func TestAdmitProcesses_StrictNamespacePassesWhenSatisfiable(t *testing.T) {
	p := &types.Project{
		Processes: types.Processes{
			"foo": {
				Name:        "foo",
				ReplicaName: "foo",
				Namespace:   "web",
				DependsOn: types.DependsOnConfig{
					"bar": {Condition: types.ProcessConditionCompletedSuccessfully},
				},
			},
			"bar": {
				Name:        "bar",
				ReplicaName: "bar",
				Namespace:   "web",
			},
		},
	}
	opts := &LoaderOptions{StrictNamespace: true}
	opts.AddAdmitter(&admitter.NamespaceAdmitter{EnabledNamespaces: []string{"web"}})

	if _, err := admitProcesses(opts, p); err != nil {
		t.Fatalf("admitProcesses() should not error when deps stay within scope: %v", err)
	}
}
