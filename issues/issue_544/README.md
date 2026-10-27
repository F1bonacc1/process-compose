# process-compose #544: late `process_healthy` dependent skipped after transient readiness loss

Deterministic reproduction of a bug where a `process_healthy` dependent, started while its
still-running dependency is temporarily NotReady, is skipped immediately instead of waiting.
The dependency is also stamped with `exit_code=1` even though it never exited.
Verified on v1.122.0 (commit 1e358d7).

## Contents

- `process-compose.yaml`: the config from the issue. `parent` has a readiness probe with a
  nonfatal `failure_threshold: 300`. `child` is disabled and `depends_on: parent` with
  `condition: process_healthy`.
- `ready.sh`: the probe. It passes only while the `health` file contains `ready`.
- `repro.sh`: runs the whole scenario in a temp dir and checks every observation.

## Run

```bash
PC=/path/to/process-compose ./repro.sh [port]   # port defaults to 19098
```

Requires `jq` and takes about 10 seconds. The script exits 0 when every check passes, 1 when
the issue reproduces, and 2 when a setup step did not hold (inconclusive).

On an affected build 5 checks fail:

- `child` goes to `Skipped` instead of waiting for `parent`.
- `parent` gets `exit_code=1` while still `Running`.
- After `parent` recovers, `child` stays `Skipped`, its command never runs, and `parent`
  keeps `exit_code=1`.

## Root cause

- `procReadyCtx` is a one-shot latch. It is created once in `NewProcess`
  (`src/app/process.go:129`), cancelled by the first successful probe
  (`onReadinessCheckEnd`, `process.go:1167`), and never re-armed.
- A later nonfatal probe failure sets health to NotReady (`process.go:1169`), but the context
  stays done.
- A late waiter in `waitUntilReady` (`process.go:419`) wakes immediately, reads NotReady,
  treats that as "terminated" and calls `setExitCode(1)` on the dependency
  (`process.go:429`).
- `waitIfNeeded` (`src/app/project_runner.go:390`) turns this into "won't run", so the
  dependent ends up `Skipped`.
