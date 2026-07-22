package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/f1bonacc1/process-compose/src/loader"
	"github.com/spf13/cobra"
)

const completionValidConfig = `version: "0.5"
processes:
  postgres:
    command: "echo postgres"
    description: "the database"
  redis:
    command: "echo redis"
  minio:
    command: "echo minio"
`

const completionUnclosedBraceConfig = `version: "0.5"
processes:
  postgres:
    command: "echo ${UNCLOSED_BRACE"
`

func writeCompletionConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "process-compose.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func TestCompleteProcessNamesFromConfig(t *testing.T) {
	// The helper reads the package-global opts; restore it after the test.
	orig := opts
	t.Cleanup(func() { opts = orig })

	setConfig := func(t *testing.T, content string) {
		opts = &loader.LoaderOptions{FileNames: []string{writeCompletionConfig(t, content)}}
	}

	t.Run("returns names and descriptions in lexicographic order", func(t *testing.T) {
		setConfig(t, completionValidConfig)
		got, directive := completeProcessNamesFromConfig(true)(nil, nil, "")
		if want := []string{"minio", "postgres\tthe database", "redis"}; !slices.Equal(got, want) {
			t.Errorf("names = %v, want %v", got, want)
		}
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %d, want NoFileComp (%d)", directive, cobra.ShellCompDirectiveNoFileComp)
		}
	})

	t.Run("single stops offering names once an arg is present", func(t *testing.T) {
		setConfig(t, completionValidConfig)
		got, directive := completeProcessNamesFromConfig(true)(nil, []string{"postgres"}, "")
		if len(got) != 0 {
			t.Errorf("names = %v, want none after first positional arg", got)
		}
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %d, want NoFileComp", directive)
		}
	})

	t.Run("non-single completes at every position", func(t *testing.T) {
		setConfig(t, completionValidConfig)
		got, _ := completeProcessNamesFromConfig(false)(nil, []string{"postgres"}, "")
		if len(got) != 3 {
			t.Errorf("names = %v, want all 3 names for a variadic command", got)
		}
	})

	t.Run("broken config yields no candidates instead of exiting", func(t *testing.T) {
		// completionUnclosedBraceConfig triggers an envsubst "missing closing
		// brace" error, which is fatal (and would abort the whole test binary)
		// *unless* completeProcessNamesFromConfig properly sets the
		// IsInternalLoader option.
		setConfig(t, completionUnclosedBraceConfig)
		got, directive := completeProcessNamesFromConfig(true)(nil, nil, "")
		if len(got) != 0 {
			t.Errorf("names = %v, want none on a broken config", got)
		}
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("directive = %d, want NoFileComp", directive)
		}
	})
}

func TestCompletionEntry(t *testing.T) {
	tests := []struct {
		name, description, want string
	}{
		{"web", "", "web"},
		{"web", "a server", "web\ta server"},
		{"web", "first line\nsecond line", "web\tfirst line"},
		{"web", "has\ttab", "web\thas tab"},
	}
	for _, tt := range tests {
		if got := completionEntry(tt.name, tt.description); got != tt.want {
			t.Errorf("completionEntry(%q, %q) = %q, want %q", tt.name, tt.description, got, tt.want)
		}
	}
}
