# process-compose dependency-restart wedge — reproduction

Deterministic reproduction of a bug where restarting a process that is parked waiting on
a `depends_on` condition wedges it permanently in `Terminating` with no recovery path.
Verified on process-compose v1.103.0 and v1.120.0.

See `REPORT.md` for the full error report and root-cause analysis.

## Contents

- `process-compose.yaml` — two processes: `a` (slow-starting, `ready_log_line`) and `b`
  (`depends_on: a` with `condition: process_log_ready`).
- `a.sh` — fake server, ready after `A_STARTUP_SECS` (default 20s).
- `b.sh` — trivial long-running process.
- `repro.sh` — runs the whole scenario and prints the wedged states.

## Run

```bash
PC=/path/to/process-compose ./repro.sh [port]   # port defaults to 28080
```

Requires `jq`. Takes ~45 seconds.

Every observation is checked against the expected (correct) behaviour and marked `PASS`
or `FAIL`; the script exits 0 when all checks pass and 1 when any fail.

On an affected build 4 checks fail: `b` wedges in `Terminating`/`pid=0`/`exit_code=0`
after step 2, survives start/stop/restart attempts, and after `a` becomes ready ends up
as an untracked running process that a subsequent `start` duplicates. Note that the three
replies printed in step 3 (`already running`, `Successfully stopped: 'b'`,
`restart b -> reported success`) are identical either way — only the state line below them
tells a wedged `b` from a healthy one.
