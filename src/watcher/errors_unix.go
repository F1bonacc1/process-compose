//go:build !windows

package watcher

import (
	"errors"
	"fmt"
	"syscall"
)

// translateWatchError turns the raw errno fsnotify surfaces into something a
// user can act on.
//
// This matters most on Linux, where exhausting the inotify watch limit is
// reported as ENOSPC - literally "no space left on device" - which sends people
// hunting for a full disk instead of a sysctl.
func translateWatchError(err error, path string) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, syscall.ENOSPC):
		return fmt.Errorf("the inotify watch limit was reached while watching %q: "+
			"raise it with 'sudo sysctl fs.inotify.max_user_watches=524288' "+
			"(persist it in /etc/sysctl.d/), or narrow the watch with 'exclude:': %w", path, err)
	case errors.Is(err, syscall.EMFILE), errors.Is(err, syscall.ENFILE):
		return fmt.Errorf("ran out of file descriptors while watching %q: "+
			"raise 'ulimit -n' or narrow the watch. On macOS and BSD every watched file "+
			"consumes a descriptor, so large trees exhaust the limit quickly: %w", path, err)
	default:
		return fmt.Errorf("failed to watch %q: %w", path, err)
	}
}
