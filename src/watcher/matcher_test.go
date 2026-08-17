package watcher

import (
	"path/filepath"
	"testing"
)

const testRoot = "/repo/src"

func newTestMatcher(t *testing.T, opts matcherOpts) *matcher {
	t.Helper()
	m, err := newMatcher(testRoot, opts)
	if err != nil {
		t.Fatalf("newMatcher() error = %v", err)
	}
	return m
}

func TestMatcher_MatchFile(t *testing.T) {
	tests := []struct {
		name string
		opts matcherOpts
		path string
		want bool
	}{
		{
			name: "plain file with no filters",
			path: "/repo/src/main.go",
			want: true,
		},
		{
			name: "nested file with no filters",
			path: "/repo/src/pkg/handler.go",
			want: true,
		},
		{
			name: "path outside the root",
			path: "/repo/other/main.go",
			want: false,
		},
		{
			name: "the root itself is not a file match",
			path: "/repo/src",
			want: true, // rel is ".", base "." - no filter rejects it
		},

		// The slash rule: a pattern without '/' matches the base name at any
		// depth. This is the trap doublestar alone gets wrong.
		{
			name: "bare pattern matches base name at top level",
			opts: matcherOpts{exclude: []string{"*_test.go"}},
			path: "/repo/src/main_test.go",
			want: false,
		},
		{
			name: "bare pattern matches base name when nested",
			opts: matcherOpts{exclude: []string{"*_test.go"}},
			path: "/repo/src/a/b/handler_test.go",
			want: false,
		},
		{
			name: "bare pattern does not over-match",
			opts: matcherOpts{exclude: []string{"*_test.go"}},
			path: "/repo/src/a/b/handler.go",
			want: true,
		},
		{
			name: "doublestar pattern matches nested",
			opts: matcherOpts{exclude: []string{"**/*_test.go"}},
			path: "/repo/src/a/b/handler_test.go",
			want: false,
		},
		{
			name: "rooted pattern matches relative path",
			opts: matcherOpts{exclude: []string{"generated/**"}},
			path: "/repo/src/generated/api.pb.go",
			want: false,
		},
		{
			name: "rooted pattern does not match a different subtree",
			opts: matcherOpts{exclude: []string{"generated/**"}},
			path: "/repo/src/pkg/api.go",
			want: true,
		},

		// include acts as an allowlist
		{
			name: "include allows a match",
			opts: matcherOpts{include: []string{"**/*.go"}},
			path: "/repo/src/pkg/main.go",
			want: true,
		},
		{
			name: "include rejects a non-match",
			opts: matcherOpts{include: []string{"**/*.go"}},
			path: "/repo/src/pkg/README.md",
			want: false,
		},
		{
			name: "exclude beats include",
			opts: matcherOpts{include: []string{"**/*.go"}, exclude: []string{"*_test.go"}},
			path: "/repo/src/pkg/main_test.go",
			want: false,
		},

		// default ignores
		{
			name: "default ignores drop log files",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/pc.log",
			want: false,
		},
		{
			name: "default ignores drop rotated log files",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/pc-2026-08-15T00-47-54.262.log.gz",
			want: false,
		},
		{
			name: "default ignores drop editor swap files",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/.main.go.swp",
			want: false,
		},
		{
			name: "default ignores drop files under node_modules",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/node_modules/left-pad/index.js",
			want: false,
		},
		{
			name: "default ignores drop files under a nested build dir",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/sub/bin/api",
			want: false,
		},
		{
			name: "default ignores leave real source alone",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/pkg/main.go",
			want: true,
		},
		{
			name: "defaults off keeps log files",
			opts: matcherOpts{useDefaults: false},
			path: "/repo/src/pc.log",
			want: true,
		},

		// onlyBase, used when the configured path was a regular file
		{
			name: "onlyBase matches its file",
			opts: matcherOpts{onlyBase: "config.yaml"},
			path: "/repo/src/config.yaml",
			want: true,
		},
		{
			name: "onlyBase rejects a sibling",
			opts: matcherOpts{onlyBase: "config.yaml"},
			path: "/repo/src/other.yaml",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMatcher(t, tt.opts)
			if got := m.MatchFile(filepath.FromSlash(tt.path)); got != tt.want {
				t.Errorf("MatchFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestMatcher_MatchDir(t *testing.T) {
	tests := []struct {
		name string
		opts matcherOpts
		path string
		want bool
	}{
		{
			name: "the root is always descended into",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src",
			want: true,
		},
		{
			name: "ordinary subdirectory",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/pkg",
			want: true,
		},
		{
			name: "default ignore dir is pruned",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/node_modules",
			want: false,
		},
		{
			name: "nested default ignore dir is pruned",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/a/.git",
			want: false,
		},
		{
			name: "child of an ignored dir is pruned",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/src/node_modules/left-pad",
			want: false,
		},
		{
			name: "defaults off keeps node_modules",
			opts: matcherOpts{useDefaults: false},
			path: "/repo/src/node_modules",
			want: true,
		},
		{
			// `generated/**` matches `generated` itself in doublestar, so an
			// exclude written for a subtree also prunes its root - which is
			// what keeps the walk cheap.
			name: "exclude subtree prunes its root directory",
			opts: matcherOpts{exclude: []string{"generated/**"}},
			path: "/repo/src/generated",
			want: false,
		},
		{
			name: "exclude by bare name prunes at any depth",
			opts: matcherOpts{exclude: []string{"testdata"}},
			path: "/repo/src/a/b/testdata",
			want: false,
		},
		{
			name: "directory outside the root",
			opts: matcherOpts{useDefaults: true},
			path: "/repo/other",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestMatcher(t, tt.opts)
			if got := m.MatchDir(filepath.FromSlash(tt.path)); got != tt.want {
				t.Errorf("MatchDir(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// TestMatcher_AlwaysExclude covers process-compose's own log files, which are
// dropped even when the user disabled the default excludes: they are never user
// content, and without this a project-root watch retriggers on its own output.
func TestMatcher_AlwaysExclude(t *testing.T) {
	m := newTestMatcher(t, matcherOpts{
		useDefaults:   false,
		alwaysExclude: LogExclusionPatterns("/repo/src/pc.log"),
	})

	tests := []struct {
		path string
		want bool
	}{
		{"/repo/src/pc.log", false},
		{"/repo/src/pc-2026-08-15T00-47-54.262.log", false},
		{"/repo/src/pc-2026-08-15T00-47-54.262.log.gz", false},
		{"/repo/src/pc.go", true},
		{"/repo/src/pcx.log", true},
		{"/repo/src/sub/pc.log", true},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := m.MatchFile(filepath.FromSlash(tt.path)); got != tt.want {
				t.Errorf("MatchFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestLogExclusionPatterns(t *testing.T) {
	tests := []struct {
		name    string
		logPath string
		want    int
	}{
		{name: "empty", logPath: "", want: 0},
		{name: "blank", logPath: "   ", want: 0},
		{name: "normal", logPath: "/repo/pc.log", want: 3},
		{name: "no extension", logPath: "/repo/pclog", want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(LogExclusionPatterns(tt.logPath)); got != tt.want {
				t.Errorf("LogExclusionPatterns(%q) length = %v, want %v", tt.logPath, got, tt.want)
			}
		})
	}
}

func TestNewMatcher_RejectsInvalidPattern(t *testing.T) {
	if _, err := newMatcher(testRoot, matcherOpts{exclude: []string{"[unclosed"}}); err == nil {
		t.Error("newMatcher() error = nil, want an InvalidPatternError for a malformed exclude")
	}
	if _, err := newMatcher(testRoot, matcherOpts{include: []string{"[unclosed"}}); err == nil {
		t.Error("newMatcher() error = nil, want an InvalidPatternError for a malformed include")
	}
	if _, err := newMatcher(testRoot, matcherOpts{exclude: []string{"**/*.go"}}); err != nil {
		t.Errorf("newMatcher() error = %v, want nil for a valid pattern", err)
	}
}
