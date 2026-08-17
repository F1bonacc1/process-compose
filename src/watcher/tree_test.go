package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fsnotify/fsnotify"
)

// mkTree builds a directory tree under a temp dir. Each entry is a path
// relative to the root; a trailing "/" makes it a directory, otherwise a file
// (and its parents) is created.
func mkTree(t *testing.T, entries ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, entry := range entries {
		full := filepath.Join(root, filepath.FromSlash(entry))
		if entry[len(entry)-1] == '/' {
			if err := os.MkdirAll(full, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", full, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", full, err)
		}
	}
	return root
}

// relDirs renders scanned absolute directories as sorted root-relative paths.
func relDirs(t *testing.T, root string, dirs []string) []string {
	t.Helper()
	out := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			t.Fatalf("Rel(%q) error = %v", dir, err)
		}
		out = append(out, filepath.ToSlash(rel))
	}
	slices.Sort(out)
	return out
}

func TestScanTree_CollectsDirectories(t *testing.T) {
	root := mkTree(t, "main.go", "pkg/handler.go", "pkg/inner/deep.go", "cmd/api/main.go")
	m, err := newMatcher(root, matcherOpts{})
	if err != nil {
		t.Fatalf("newMatcher() error = %v", err)
	}

	dirs, err := scanTree("api", root, m, defaultMaxEntries)
	if err != nil {
		t.Fatalf("scanTree() error = %v", err)
	}

	want := []string{".", "cmd", "cmd/api", "pkg", "pkg/inner"}
	if got := relDirs(t, root, dirs); !slices.Equal(got, want) {
		t.Errorf("scanTree() dirs = %v, want %v", got, want)
	}
}

// TestScanTree_PrunesIgnoredDirs is the load-bearing case: pruning during the
// walk is what keeps a watch under the inotify and kqueue limits. A filter
// applied only to events would still register every watch.
func TestScanTree_PrunesIgnoredDirs(t *testing.T) {
	root := mkTree(t,
		"main.go",
		"node_modules/left-pad/index.js",
		"node_modules/a/b/c/d.js",
		".git/objects/ab/cdef",
		"pkg/handler.go",
	)
	m, err := newMatcher(root, matcherOpts{useDefaults: true})
	if err != nil {
		t.Fatalf("newMatcher() error = %v", err)
	}

	dirs, err := scanTree("api", root, m, defaultMaxEntries)
	if err != nil {
		t.Fatalf("scanTree() error = %v", err)
	}

	want := []string{".", "pkg"}
	if got := relDirs(t, root, dirs); !slices.Equal(got, want) {
		t.Errorf("scanTree() dirs = %v, want %v (ignored trees must be pruned, not merely filtered)", got, want)
	}
}

func TestScanTree_PrunesUserExcludes(t *testing.T) {
	root := mkTree(t, "src/main.go", "generated/api.pb.go", "generated/deep/more.go")
	m, err := newMatcher(root, matcherOpts{exclude: []string{"generated/**"}})
	if err != nil {
		t.Fatalf("newMatcher() error = %v", err)
	}

	dirs, err := scanTree("api", root, m, defaultMaxEntries)
	if err != nil {
		t.Fatalf("scanTree() error = %v", err)
	}

	want := []string{".", "src"}
	if got := relDirs(t, root, dirs); !slices.Equal(got, want) {
		t.Errorf("scanTree() dirs = %v, want %v", got, want)
	}
}

// TestScanTree_EnforcesCap pins the guard that keeps a runaway watch from
// exhausting the descriptor limit and breaking process launching itself.
func TestScanTree_EnforcesCap(t *testing.T) {
	root := mkTree(t,
		"big/a/x.go", "big/b/x.go", "big/c/x.go", "big/d/x.go",
		"small/x.go",
	)
	m, err := newMatcher(root, matcherOpts{})
	if err != nil {
		t.Fatalf("newMatcher() error = %v", err)
	}

	_, err = scanTree("api", root, m, 3)
	if err == nil {
		t.Fatal("scanTree() error = nil, want a TooManyEntriesError")
	}

	var tooMany *TooManyEntriesError
	if !errors.As(err, &tooMany) {
		t.Fatalf("scanTree() error = %T, want *TooManyEntriesError", err)
	}
	if tooMany.Process != "api" {
		t.Errorf("Process = %v, want api", tooMany.Process)
	}
	if tooMany.Max != 3 {
		t.Errorf("Max = %v, want 3", tooMany.Max)
	}
	// The message must name what blew the budget, not just report a number.
	if len(tooMany.LargestSubdirs) == 0 {
		t.Error("LargestSubdirs is empty; the error cannot tell the user what to exclude")
	}
	msg := tooMany.Error()
	for _, want := range []string{"api", "exceeding the limit of 3", "max_entries", "exclude"} {
		if !contains(msg, want) {
			t.Errorf("error message %q does not mention %q", msg, want)
		}
	}
}

func TestScanTree_CapNotTrippedWhenUnderLimit(t *testing.T) {
	root := mkTree(t, "a/x.go", "b/x.go")
	m, err := newMatcher(root, matcherOpts{})
	if err != nil {
		t.Fatalf("newMatcher() error = %v", err)
	}
	if _, err := scanTree("api", root, m, 3); err != nil {
		t.Errorf("scanTree() error = %v, want nil at exactly the cap", err)
	}
}

func TestResolveRoot(t *testing.T) {
	root := mkTree(t, "config.yaml", "src/main.go")

	t.Run("directory", func(t *testing.T) {
		dir, onlyBase, err := resolveRoot(filepath.Join(root, "src"))
		if err != nil {
			t.Fatalf("resolveRoot() error = %v", err)
		}
		if want := filepath.Join(root, "src"); dir != want {
			t.Errorf("dir = %v, want %v", dir, want)
		}
		if onlyBase != "" {
			t.Errorf("onlyBase = %v, want empty for a directory", onlyBase)
		}
	})

	// A configured file is watched through its parent: a watch on the file
	// itself dies the first time an editor saves atomically.
	t.Run("regular file watches its parent", func(t *testing.T) {
		dir, onlyBase, err := resolveRoot(filepath.Join(root, "config.yaml"))
		if err != nil {
			t.Fatalf("resolveRoot() error = %v", err)
		}
		if dir != root {
			t.Errorf("dir = %v, want %v", dir, root)
		}
		if onlyBase != "config.yaml" {
			t.Errorf("onlyBase = %v, want config.yaml", onlyBase)
		}
	})

	t.Run("missing path", func(t *testing.T) {
		if _, _, err := resolveRoot(filepath.Join(root, "nope")); err == nil {
			t.Error("resolveRoot() error = nil, want a stat error")
		}
	})
}

// TestInteresting pins the op filter. Chmod must be dropped: Spotlight,
// antivirus and backup software generate a constant stream of them.
func TestInteresting(t *testing.T) {
	tests := []struct {
		name string
		op   fsnotify.Op
		want bool
	}{
		{name: "create", op: fsnotify.Create, want: true},
		{name: "write", op: fsnotify.Write, want: true},
		{name: "remove", op: fsnotify.Remove, want: true},
		{name: "rename", op: fsnotify.Rename, want: true},
		{name: "chmod alone is ignored", op: fsnotify.Chmod, want: false},
		{name: "chmod combined with write is kept", op: fsnotify.Chmod | fsnotify.Write, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := interesting(tt.op); got != tt.want {
				t.Errorf("interesting(%v) = %v, want %v", tt.op, got, tt.want)
			}
		})
	}
}

func TestRelativeTo(t *testing.T) {
	tests := []struct {
		name   string
		root   string
		target string
		want   string
	}{
		{name: "inside", root: "/repo/src", target: "/repo/src/pkg/main.go", want: "pkg/main.go"},
		{name: "root itself", root: "/repo/src", target: "/repo/src", want: "/repo/src"},
		{name: "outside falls back to absolute", root: "/repo/src", target: "/other/main.go", want: "/other/main.go"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTo(filepath.FromSlash(tt.root), filepath.FromSlash(tt.target))
			if got != tt.want && got != filepath.FromSlash(tt.want) {
				t.Errorf("relativeTo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTranslateWatchError(t *testing.T) {
	if got := translateWatchError(nil, "/repo"); got != nil {
		t.Errorf("translateWatchError(nil) = %v, want nil", got)
	}
	err := translateWatchError(errors.New("boom"), "/repo/src")
	if err == nil {
		t.Fatal("translateWatchError() = nil, want an error")
	}
	if !contains(err.Error(), "/repo/src") {
		t.Errorf("error %q does not name the path", err.Error())
	}
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
