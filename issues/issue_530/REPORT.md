# process-compose: restarting a dependency-parked process wedges it permanently in `Terminating`, with no recovery path

**Affected versions:** verified on v1.103.0 and v1.120.0 (built from source at those tags).

## Summary

If process `b` has `depends_on: a` with `condition: process_log_ready`, and you run
`process restart b` (or `process stop b` followed by more lifecycle commands) **while `b`
is parked waiting for `a` to become ready**, `b` wedges:

- `process list` shows `b` as `Terminating` with `pid=0`, `is_running=false`, `exit_code=0`
  (and `process_end_time` set, on v1.120.0).
- `process start b` fails with `process b is already running`.
- `process stop b` and `process restart b` report success but change nothing.
- The state stays frozen for as long as the dependency is not ready. For a dependency that
  takes a long time to start (or is crash-looping and never emits its ready line), this is
  effectively permanent, and the only recovery is restarting the whole process-compose
  instance.
- **Worse:** when the dependency *does* finally become ready, the bookkeeping is corrupted.
  One of the stacked replacement instances launches, but its entry is deleted from the
  running-process map by a stale generation's cleanup, so the real process runs
  **untracked** (display stays `Terminating` forever), and a subsequent `process start b`
  silently launches a **duplicate** instance.

## Deterministic reproduction

Files in this archive: `process-compose.yaml`, `a.sh`, `b.sh`, `repro.sh`.

```yaml
version: "0.5"
processes:
  a:
    command: ./a.sh          # takes ~20s to emit its ready line
    ready_log_line: "Server running on port"
  b:
    command: ./b.sh
    depends_on:
      a:
        condition: process_log_ready
```

Steps (automated by `repro.sh`; no timing luck required):

1. `process-compose up -f process-compose.yaml --tui=false -p <port>` — `a` starts warming
   up; `b` is `Pending`, parked waiting for `a`'s ready log line.
2. While `a` is still warming up: `process-compose process restart b`.
3. `b` is now wedged in `Terminating` / `pid=0` / `exit_code=0`.
4. `start` / `stop` / `restart` on `b` cannot recover it (see transcript below).
5. After `a` becomes ready: `b`'s display stays `Terminating`, while an actual `b.sh`
   instance runs untracked; `process start b` then creates a second instance.

### Observed transcript (v1.120.0, also identical on v1.103.0 except `process_end_time` stays null)

```
1. Starting project (a needs ~20s to become ready; b waits on it)...
   b while waiting on a:   {"status":"Pending","is_running":false,"pid":0,"exit_code":0,"process_end_time":null}
2. process restart b  (while b is still waiting for a)...
   b is now wedged:        {"status":"Terminating","is_running":false,"pid":0,"exit_code":0,"process_end_time":"2026-08-14T00:00:50-07:00"}
3. Trying to recover:
   start b   -> already running
   stop b    -> Successfully stopped: 'b'   (no actual effect)
   restart b -> reported success            (no actual effect)
   b still wedged:         {"status":"Terminating","is_running":false,"pid":0,"exit_code":0,...}
4. Waiting for a to become ready...
   b after a became ready: {"status":"Terminating","is_running":false,"pid":0,"exit_code":0,...}
   actual b.sh processes:  1  (running, but untracked by process-compose)
5. start b now silently duplicates b:
   b state:                {"status":"Running","is_running":true,"pid":42807,...}
   actual b.sh processes:  2
```

The `pid=0` / `exit_code=0` signature is the tell: the instance that got replaced never ran.

## Root-cause analysis (from source, verified with instrumented builds)

Four interacting problems:

1. **Stopping/restarting a dependency-parked process never aborts its dependency wait.**
   `runProcess` parks each new instance in a goroutine inside `waitIfNeeded` →
   `waitUntilLogReady()`, blocking on the dependency object's `procLogReadyCtx`
   (`src/app/project_runner.go`, `src/app/process.go`). `stopProcess` on a pre-`run()`
   instance takes the `!isRunning()` path: it marks a `Pending` instance terminal via
   `onProcessEnd(ProcessStateTerminating)` — which produces exactly the
   `Terminating`/`pid=0`/`exit_code=0` snapshot — but the goroutine stays parked and the
   instance stays in `runningProcesses`. Nothing plumbs the waiter's own shutdown into
   `waitIfNeeded`, and there is no re-resolution or timeout.

2. **The replacement instance inherits the previous instance's display state.**
   `doRestart` → `runProcess` does `procState, _ := p.GetProcessState(name)` →
   `withProcState(procState)`, so the new `b` object is born displaying the stopped
   instance's `Terminating` snapshot. All real state transitions happen inside `run()`,
   which the parked instance never reaches, so the stale `Terminating` is what the UI/API
   shows indefinitely. Notably `Process.run()` has a recovery for exactly this
   ("Resetting stale Terminating state before start"), but it can only fire after the
   dependency wait — too late.

3. **A parked pre-`run()` instance is unmanageable, and inheritance destroys the only
   escape hatch.** It is in the running-process map (so `StartProcess` refuses with
   "already running") but has no pid and no live lifecycle. `stopProcess`'s early-return
   path only handles `Pending`; the inherited state is `Terminating`, so stop becomes a
   pure no-op, and every further `restart` just stacks another parked goroutine on the
   same dependency context.

4. **`removeRunningProcess` deletes by name, not object identity**
   (`src/app/project_runner.go`). When the dependency finally becomes ready, all parked
   generations wake at once. A stale, already-stopped generation returns immediately from
   `run()` and its cleanup deletes the running-map entry **which now points to the newest
   generation**. That newest generation launches its process anyway → a live, untracked
   process; the published state map stays frozen at `Terminating`; a later `start`
   duplicates the process.

## Related (separate) bug found while validating: silent dependency bypass on restart

On **v1.103.0**, the sequence "restart `a`, then restart `b` a few seconds later while `a`
is still starting" does not wedge — instead `b` relaunches **immediately**, ~8s before `a`
is ready, silently bypassing the dependency gate. Cause: `getDoneOrRunningProcess`
preferred the **done** (stale) object in v1.103.0, and a done dependency object that ever
emitted its ready line has its one-shot `cancelReadyLogFunc` already consumed with a nil
cause, so `waitUntilLogReady` returns `true` instantly.

v1.120.0 flipped `getDoneOrRunningProcess` to prefer the running object, which fixes the
common case (verified: `b` correctly waits ~8s for the new `a`), but the stale-done
fallback is still reachable when the lookup lands in the ~1s restart backoff window while
the dependency's map entry is absent.

## Suggested directions

1. Make dependency waits abortable/re-resolvable: plumb the waiter's own shutdown/run
   context into `waitIfNeeded` so a subsequent stop/restart of the waiter aborts the wait,
   and/or re-resolve the dependency object (or signal waiters) when a dependency instance
   is replaced.
2. Set the new instance's state to `Pending` (or a new `Waiting` state) in `runProcess`
   instead of inheriting the prior instance's snapshot.
3. Allow `stop` to fully cancel a pre-`run()` instance: not just mark it terminal, but
   unpark its goroutine so it exits and removes itself from the running-process map.
4. Make `removeRunningProcess` delete the map entry only if it still points to the same
   `Process` object (compare identity, not just name).
5. For the bypass bug: never treat a done dependency object's consumed log-ready context
   as satisfied for a *new* wait; require the currently-running instance's readiness.

## Validation methodology

- Built v1.103.0 and v1.120.0 from source at the release tags.
- Confirmed the wedge repro is deterministic on both versions (multiple runs).
- Swept the "restart a, then b" timing (delays 0–4s, multiple rounds per delay): no wedge
  from that shape; v1.103.0 shows the premature-start bypass instead, v1.120.0 waits
  correctly (with a cosmetic stale-status display while parked).
- Instrumented builds logged object identities in `waitIfNeeded`, ready-line handling, and
  `removeRunningProcess`, confirming the generation-stacking, wrong-entry deletion, and
  orphaned relaunch described above, e.g.:

```
DBG runProcess b new obj=0x...a288 inherited status=Pending
DBG b resolving dep a: obj=0x...(a gen1) status=Pending          <- parked
DBG runProcess b new obj=0x...a788 inherited status=Terminating  <- restart while parked
DBG runProcess b new obj=0x...56008 inherited status=Terminating <- restart again
DBG a ready-line match (cancel fires=true)                       <- a becomes ready
DBG removeRunningProcess b obj=0x...a288 mapObj=0x...56008       <- stale gen deletes live gen's entry
WRN Resetting stale Terminating state before start process=b
INF Started ./b.sh                                               <- runs untracked
```
