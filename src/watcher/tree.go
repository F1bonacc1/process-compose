package watcher

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// resolveRoot inspects a configured watch path and returns the directory to
// walk plus, when the path named a regular file, the base name to restrict
// matching to.
//
// A file is never watched directly: editors save atomically by writing a temp
// file and renaming it over the target, which destroys a watch registered on
// the original. Watching the parent and filtering on the name survives that.
func resolveRoot(configured string) (dir string, onlyBase string, err error) {
	clean := filepath.Clean(configured)
	info, err := os.Stat(clean)
	if err != nil {
		return "", "", err
	}
	if info.IsDir() {
		return clean, "", nil
	}
	return filepath.Dir(clean), filepath.Base(clean), nil
}

// scanTree walks root and returns every directory that should carry a watch,
// root included. Directories rejected by the matcher are pruned with fs.SkipDir
// so an excluded node_modules is never descended into - pruning, not event
// filtering, is what keeps a watch within the inotify watch limit and the
// kqueue descriptor limit.
//
// The walk stops as soon as the cap is exceeded, reporting which subtrees were
// largest so the error can name what actually blew the budget.
func scanTree(procName, root string, m *matcher, maxEntries int) ([]string, error) {
	dirs := make([]string, 0, 64)
	// Tally per top-level subdirectory, for the cap error message.
	counts := make(map[string]int)

	walkErr := filepath.WalkDir(root, func(current string, entry fs.DirEntry, err error) error {
		if err != nil {
			// A directory that vanished mid-walk is normal during a rebuild or
			// a branch switch; skip it rather than failing the whole scan.
			if os.IsNotExist(err) || os.IsPermission(err) {
				if entry != nil && entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		// Symlinked directories are not followed: WalkDir does not descend into
		// them, which also keeps a symlink cycle from becoming a watch cycle.
		if current != root && !m.MatchDir(current) {
			return fs.SkipDir
		}

		dirs = append(dirs, current)
		if top := topLevelComponent(root, current); top != "" {
			counts[top]++
		}

		if len(dirs) > maxEntries {
			return &TooManyEntriesError{
				Process:        procName,
				Root:           root,
				Scanned:        len(dirs),
				Max:            maxEntries,
				LargestSubdirs: topSubdirs(counts, 3),
			}
		}
		return nil
	})

	if walkErr != nil {
		var tooMany *TooManyEntriesError
		if errors.As(walkErr, &tooMany) {
			return nil, tooMany
		}
		return nil, translateWatchError(walkErr, root)
	}
	return dirs, nil
}

// topLevelComponent returns the first path component of current relative to
// root, which is the granularity the cap error reports at.
func topLevelComponent(root, current string) string {
	rel, err := filepath.Rel(root, current)
	if err != nil {
		return ""
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" {
		return ""
	}
	if top, _, found := strings.Cut(rel, "/"); found {
		return top
	}
	return rel
}

// interesting reports whether a filesystem op should be acted on.
//
// Chmod is dropped unconditionally: Spotlight, antivirus and backup software
// generate a constant stream of attribute changes that have nothing to do with
// the user editing a file.
func interesting(op fsnotify.Op) bool {
	return op.Has(fsnotify.Create) || op.Has(fsnotify.Write) ||
		op.Has(fsnotify.Remove) || op.Has(fsnotify.Rename)
}

// relativeTo renders a path for display, preferring a short root-relative form
// so restart messages stay readable.
func relativeTo(root, target string) string {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return target
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || strings.HasPrefix(rel, "../") {
		return target
	}
	return path.Clean(rel)
}
