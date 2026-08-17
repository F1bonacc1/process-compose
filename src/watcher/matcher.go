package watcher

import (
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// DefaultIgnoreDirs are directory names pruned from every watch unless
// disable_default_excludes is set. Pruning during the walk - rather than
// filtering events afterwards - is what keeps a watch under the inotify watch
// limit and the kqueue descriptor limit, not merely quieter.
var DefaultIgnoreDirs = []string{
	".git", ".hg", ".svn", ".idea", ".vscode",
	"node_modules", "vendor", "target", "dist", "build", "bin",
	".venv", "venv", "__pycache__", ".mypy_cache", ".pytest_cache",
	".next", ".nuxt", ".terraform", ".gradle", ".direnv", "result",
}

// DefaultIgnoreFiles are base-name patterns dropped from every watch. Log files
// matter because process-compose writes them into the project directory, and
// editor scratch files because they accompany every save.
var DefaultIgnoreFiles = []string{
	"*.log", "*.log.gz", "*.swp", "*.swx", "*~", ".#*", "#*#", "4913",
	".DS_Store", "*.tmp",
}

// matcher decides which paths under a watched root are interesting.
//
// Patterns follow one rule, chosen to avoid the classic trap where
// `exclude: ["*_test.go"]` silently fails below the top level: a pattern
// containing '/' is matched against the root-relative path, and a pattern
// without one is matched against the base name. Matching always uses forward
// slashes so a config authored on Linux behaves identically on Windows.
type matcher struct {
	root string
	// include, when non-empty, restricts triggering to matching files.
	include []string
	exclude []string
	// ignoreDirs and ignoreFiles hold the built-in lists, empty when the
	// process set disable_default_excludes.
	ignoreDirs  []string
	ignoreFiles []string
	// alwaysExclude holds absolute-path patterns that are dropped regardless of
	// disable_default_excludes - process-compose's own log files and their
	// rotated siblings, which are never user content and would otherwise make
	// a project-root watch retrigger on its own output.
	alwaysExclude []string
	// onlyBase restricts matching to a single file name. Set when the
	// configured path was a regular file: we watch its parent directory,
	// because a watch on the file itself is destroyed the moment an editor
	// saves atomically.
	onlyBase string
}

type matcherOpts struct {
	include        []string
	exclude        []string
	useDefaults    bool
	alwaysExclude  []string
	onlyBase       string
	skipValidation bool
}

// newMatcher builds a matcher rooted at root. Patterns are validated here so a
// typo surfaces as an error rather than as a watch that silently never fires;
// the loader validates them too, so this is the defence for programmatic use.
func newMatcher(root string, opts matcherOpts) (*matcher, error) {
	m := &matcher{
		root:          filepath.Clean(root),
		include:       opts.include,
		exclude:       opts.exclude,
		alwaysExclude: normalizeSlashes(opts.alwaysExclude),
		onlyBase:      opts.onlyBase,
	}
	if opts.useDefaults {
		m.ignoreDirs = DefaultIgnoreDirs
		m.ignoreFiles = DefaultIgnoreFiles
	}
	if !opts.skipValidation {
		for _, pattern := range append(append([]string{}, m.include...), m.exclude...) {
			if !doublestar.ValidatePattern(pattern) {
				return nil, &InvalidPatternError{Root: root, Pattern: pattern}
			}
		}
	}
	return m, nil
}

// MatchFile reports whether a change to abs should trigger a restart.
func (m *matcher) MatchFile(abs string) bool {
	rel, ok := m.relative(abs)
	if !ok {
		return false
	}
	base := path.Base(rel)

	if m.onlyBase != "" && base != m.onlyBase {
		return false
	}
	if m.isAlwaysExcluded(abs) {
		return false
	}
	if matchAny(m.ignoreFiles, rel, base) {
		return false
	}
	// A file inside an ignored directory is dropped even when the walk did not
	// prune it - a directory created after startup is watched before it is
	// re-examined, so events can arrive from a path we would have pruned.
	if m.hasIgnoredDirComponent(rel) {
		return false
	}
	if matchAny(m.exclude, rel, base) {
		return false
	}
	if len(m.include) > 0 && !matchAny(m.include, rel, base) {
		return false
	}
	return true
}

// MatchDir reports whether a directory should be descended into and watched.
// A false result prunes the whole subtree.
func (m *matcher) MatchDir(abs string) bool {
	rel, ok := m.relative(abs)
	if !ok {
		return false
	}
	if rel == "." {
		return true
	}
	base := path.Base(rel)

	if slices.Contains(m.ignoreDirs, base) {
		return false
	}
	if m.hasIgnoredDirComponent(rel) {
		return false
	}
	// `bin/**` matches `bin` itself in doublestar, so an exclude written to
	// cover a subtree also prunes its root.
	return !matchAny(m.exclude, rel, base)
}

// relative returns the root-relative, slash-separated form of abs.
func (m *matcher) relative(abs string) (string, bool) {
	rel, err := filepath.Rel(m.root, filepath.Clean(abs))
	if err != nil {
		return "", false
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}

func (m *matcher) hasIgnoredDirComponent(rel string) bool {
	if len(m.ignoreDirs) == 0 {
		return false
	}
	for component := range strings.SplitSeq(path.Dir(rel), "/") {
		if component == "." || component == "" {
			continue
		}
		if slices.Contains(m.ignoreDirs, component) {
			return true
		}
	}
	return false
}

func (m *matcher) isAlwaysExcluded(abs string) bool {
	if len(m.alwaysExclude) == 0 {
		return false
	}
	target := filepath.ToSlash(filepath.Clean(abs))
	for _, pattern := range m.alwaysExclude {
		if ok, err := doublestar.Match(pattern, target); err == nil && ok {
			return true
		}
	}
	return false
}

// matchAny applies the slash rule: a pattern containing '/' is matched against
// the root-relative path, one without it against the base name.
func matchAny(patterns []string, rel, base string) bool {
	for _, pattern := range patterns {
		target := base
		if strings.Contains(pattern, "/") {
			target = rel
		}
		if ok, err := doublestar.Match(pattern, target); err == nil && ok {
			return true
		}
	}
	return false
}

// LogExclusionPatterns returns absolute-path patterns covering a log file and
// the rotated siblings lumberjack writes beside it (`name-<timestamp>.ext` and
// its compressed form). Without these, a watch on the project root retriggers
// on process-compose's own output.
func LogExclusionPatterns(logPath string) []string {
	if strings.TrimSpace(logPath) == "" {
		return nil
	}
	clean := filepath.ToSlash(filepath.Clean(logPath))
	dir, base := path.Split(clean)
	ext := path.Ext(base)
	prefix := strings.TrimSuffix(base, ext)
	if prefix == "" {
		return []string{clean}
	}
	return []string{
		clean,
		dir + prefix + "-*" + ext,
		dir + prefix + "-*" + ext + ".gz",
	}
}

func normalizeSlashes(patterns []string) []string {
	if len(patterns) == 0 {
		return nil
	}
	out := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		out = append(out, filepath.ToSlash(pattern))
	}
	return out
}
