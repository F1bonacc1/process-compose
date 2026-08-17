---
sidebar_position: 8
---

# File Watching

Process Compose can restart a process when its source files change, and - with `cascade` - restart everything that depends on it. This replaces wrapping each command in a per-language watcher (`air`, `nodemon`, `watchexec`, `cargo-watch`).

Wrapping has a cost that is easy to miss: the wrapper becomes the process Process Compose manages, so `shutdown.signal`, `success_exit_codes` and PTY handling apply to the wrapper rather than to your program, `liveness_probe` measures the wrapper's lifetime (it never dies, so a crash-looping app looks healthy), and the `Restarts` counter never moves. A built-in watch keeps all of those pointed at the real process - and, unlike any per-language watcher, it knows your dependency graph.

## Configuration

Add a `watch` section to a process:

```yaml
processes:
  api:
    command: "go build -o bin/api ./cmd/api && ./bin/api"
    watch:
      debounce: 300ms
      paths:
        - path: ./src
          exclude: ["**/*_test.go", "**/testdata/**"]
```

Saving anything under `./src` restarts `api`, except test files.

### Watch Options

| Option | Type | Default | Description |
| -------- | ------ | --------- | ------------- |
| `paths` | list | - | Watched roots and their filters. Required |
| `debounce` | string | `300ms` | Go duration to let a burst of changes settle before restarting |
| `cascade` | bool | false | Also restart this process's dependents, transitively |
| `max_entries` | int | 8192 | Cap on watched directories |
| `buffer_size` | int | 65536 | Event buffer size, per watched directory. Windows only |
| `disable_default_excludes` | bool | false | Turn off the built-in ignore list |

### Path Options

| Option | Type | Default | Description |
| -------- | ------ | --------- | ------------- |
| `path` | string | - | Directory to watch, resolved against the process `working_dir`. The whole subtree is watched |
| `include` | list | - | If set, only matching files trigger a restart |
| `exclude` | list | - | Matching files are ignored and matching directories are not descended into |

## Pattern Matching

Patterns use `doublestar` glob syntax, where `**` matches across directories. One rule decides what a pattern is matched against:

- a pattern **containing `/`** is matched against the path relative to the watched root - `generated/**`
- a pattern **without `/`** is matched against the file's base name, at any depth - `*_test.go`

So `exclude: ["*_test.go"]` correctly excludes `pkg/inner/foo_test.go`, and `exclude: ["generated/**"]` excludes only that subtree.

### Default Excludes

Unless `disable_default_excludes` is set, these are ignored: `.git`, `.hg`, `.svn`, `.idea`, `.vscode`, `node_modules`, `vendor`, `target`, `dist`, `build`, `bin`, `.venv`, `venv`, `__pycache__`, `.mypy_cache`, `.pytest_cache`, `.next`, `.nuxt`, `.terraform`, `.gradle`, `.direnv`, `result`, plus `*.log`, `*.log.gz`, editor swap files and `.DS_Store`.

Process Compose's own log files - the global log and any `log_location`, including rotated `.gz` siblings - are **always** excluded, even with `disable_default_excludes`. Without that, a watch on the project directory would retrigger on the project's own output.

## Cascading Restarts

`cascade: true` restarts the watched process and then everything that depends on it, in dependency order.

```yaml
processes:
  assets:
    command: "make assets"
    availability:
      restart: "no"                    # one-shot
    watch:
      cascade: true
      paths:
        - path: ./assets

  api:
    command: "go build -o bin/api ./cmd/api && ./bin/api"
    depends_on:
      assets:
        condition: process_completed_successfully
    watch:
      paths:
        - path: ./src
```

- Editing `./assets/app.scss` re-runs `make assets`, **then** restarts `api`.
- Editing `./src/main.go` restarts `api` alone.

> [!IMPORTANT]
> Restarts propagate **downstream to dependents, never upstream to dependencies**. A change to `api` will not re-run `assets`. If a change should rebuild an artifact and then restart its consumers, put the watch on the process that produces the artifact and set `cascade: true` there.

## Process State

A process that has exited **successfully** but still has an armed watch reports the **`Watching`** state rather than `Completed`, in the TUI, in `process-compose process list` and over the REST API. It signals that the process is waiting for a file to change - and explains why the project is still running.

A process that failed keeps `Failed` or `Error`. `Watching` never stands in for a failure: a broken build is the most important thing a watch loop has to report, and the watch stays armed either way, so the next save still triggers a rebuild.

The TUI also reports each watch-triggered restart in the status bar, naming the file that caused it. Bursts are folded into a single summary so that rapid restarts stay readable.

## Project Lifetime

While any watch is armed, the project stays up even after every process has exited - otherwise a project of one-shot builders would exit before a watch could ever fire.

Stopping a process pauses its watch, so a stopped process is never brought back by a file change. Starting it again resumes the watch.

This works on a process that has already exited: stopping something in the `Watching` state disarms its watch, which is how you leave that state. The process drops back to `Completed`, and if nothing else is running or armed the project then finishes.

To run without watching at all - in CI, or in a script - use `--no-watch` (or `PC_NO_WATCH`). Then `up` behaves exactly as it did before watching existed, `watch` blocks are ignored, and the project exits when its processes finish.

## Feedback Loops

The classic trap is a process that writes into a directory it also watches, so each restart triggers the next. The default excludes cover the usual build output directories, but if a loop does occur Process Compose detects it, logs an error naming the process and the file that triggered it, and suspends that watch until the project is reloaded rather than looping forever.

If you hit this, narrow `paths` or add an `exclude` for your build output.

## Unsupported Combinations

`watch` cannot be combined with:

| Setting | Reason |
| --------- | -------- |
| `replicas > 1` | Each replica would install its own watcher on the same tree, multiplying descriptors and restarts |
| `schedule` | A restart does not reschedule the cron job, and scheduler concurrency limits would not see watch-driven starts |
| `is_foreground` | Foreground processes run inside the TUI with the terminal suspended, and are not managed as ordinary processes |

Template variables (`{{.Vars.x}}`) are also not supported in watch paths and are rejected at load time.

## Platform Notes

File watching uses [fsnotify](https://github.com/fsnotify/fsnotify) - inotify on Linux, kqueue on macOS and BSD, `ReadDirectoryChangesW` on Windows.

- **macOS and BSD** consume one file descriptor per watched *file*. Large trees can exhaust the limit, so keep `paths` narrow and use `exclude`.
- **Linux** limits total watches via `fs.inotify.max_user_watches`. Exceeding it is reported with a message naming the sysctl to raise.
- **Windows** uses a fixed event buffer that can overflow under heavy churn; raise `buffer_size` if changes are missed. The buffer is allocated per watched directory, so when two processes watch the same directory the larger of their two settings applies to it.
- **Network and virtual filesystems** (NFS, SMB, FUSE, `/proc`, `/sys`) do not deliver events, so watching them silently does nothing. This includes WSL2 paths under `/mnt/c`.
- **Symlinked directories are not followed.**

Watching a single file is supported, but its parent directory is watched instead and filtered to that name - a watch on a file alone would be destroyed the first time an editor saved it by writing a temporary file and renaming it into place.
