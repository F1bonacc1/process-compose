package watcher

import (
	"errors"
	"fmt"
	"syscall"
)

// translateWatchError turns a raw Windows error into something a user can act
// on. ReadDirectoryChangesW does not have the inotify/kqueue resource limits,
// so the interesting cases here are path length and handle exhaustion.
func translateWatchError(err error, path string) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, syscall.EMFILE), errors.Is(err, syscall.ENFILE):
		return fmt.Errorf("ran out of handles while watching %q: narrow the watch with 'exclude:' "+
			"or lower 'watch.max_entries': %w", path, err)
	case errors.Is(err, syscall.ERROR_PATH_NOT_FOUND), errors.Is(err, syscall.ERROR_FILE_NOT_FOUND):
		return fmt.Errorf("watch path %q does not exist: %w", path, err)
	default:
		return fmt.Errorf("failed to watch %q: %w", path, err)
	}
}
