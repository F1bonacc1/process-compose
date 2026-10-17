package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/f1bonacc1/process-compose/src/types"
)

const updateAPIProjectYaml = `
processes:
  keeper:
    command: "sleep 600"
    vars:
      PORT: 8080
    availability:
      restart: always
  oneshot:
    command: "echo done"
    availability:
      restart: "no"
`

// loadAsAPIPayload loads a project the way `process-compose project update`
// does and round-trips it through JSON, as the REST client and the POST
// /project handler do. JSON decodes every number in the free-form vars as
// float64 while the in-process loader yields int.
func loadAsAPIPayload(t *testing.T, yaml string) *types.Project {
	t.Helper()
	file := filepath.Join(t.TempDir(), "process-compose.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	project, err := loader.Load(&loader.LoaderOptions{FileNames: []string{file}})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(project)
	if err != nil {
		t.Fatal(err)
	}
	var decoded types.Project
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	return &decoded
}

// TestUpdateProject_IdenticalAPIPayloadIsNoOp is the regression test for
// `project update` with an unchanged file restarting every process: the
// JSON-decoded vars (float64) never DeepEqual'd the loaded ones (int). The
// status map is the decision that drives restarts, so this checks it directly
// without starting the processes.
func TestUpdateProject_IdenticalAPIPayloadIsNoOp(t *testing.T) {
	file := filepath.Join(t.TempDir(), "process-compose.yaml")
	if err := os.WriteFile(file, []byte(updateAPIProjectYaml), 0o600); err != nil {
		t.Fatal(err)
	}
	project, err := loader.Load(&loader.LoaderOptions{FileNames: []string{file}})
	if err != nil {
		t.Fatal(err)
	}
	runner, err := NewProjectRunner(&ProjectOpts{
		project:         project,
		processesToRun:  []string{},
		mainProcessArgs: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}

	status, err := runner.UpdateProject(loadAsAPIPayload(t, updateAPIProjectYaml))
	if err != nil {
		t.Fatal(err)
	}
	if len(status) != 0 {
		t.Fatalf("an identical project must not update anything, got %v", status)
	}
}
