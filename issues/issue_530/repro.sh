#!/usr/bin/env bash
# Deterministic reproduction of the "dependent process parks in Terminating"
# wedge in process-compose (verified on v1.103.0 and v1.120.0).
#
# Each observation is checked against the expected (correct) behaviour:
#   PASS - process-compose behaved correctly
#   FAIL - the issue reproduced
# Exits 0 when every check passed, 1 when any check failed.
#
# Usage: PC=/path/to/process-compose ./repro.sh [port]
set -u
PC="${PC:-process-compose}"
PORT="${1:-28080}"
cd "$(cd "$(dirname "$0")" && pwd)"
chmod +x a.sh b.sh

FAILURES=0
if [ -t 1 ]; then GREEN=$'\033[32m'; RED=$'\033[31m'; RESET=$'\033[0m'; else GREEN=""; RED=""; RESET=""; fi

bstate() {
  "$PC" process list -o json -p "$PORT" 2>/dev/null \
    | jq -c '.[]|select(.name=="b")|{status,is_running,pid,exit_code,process_end_time}'
}

# field <json> <jq filter> - reads one value out of a captured bstate
field() {
  local value
  value="$(printf '%s' "${1:-}" | jq -r "$2" 2>/dev/null)"
  printf '%s' "${value:-unknown}"
}

# check <description> <actual> <expected>
check() {
  if [ "$2" = "$3" ]; then
    printf '   %sPASS%s %s (%s)\n' "$GREEN" "$RESET" "$1" "$2"
  else
    printf '   %sFAIL%s %s: got %s, want %s\n' "$RED" "$RESET" "$1" "$2" "$3"
    FAILURES=$((FAILURES + 1))
  fi
}

echo "1. Starting project (a needs ~20s to become ready; b waits on it)..."
"$PC" up -f process-compose.yaml --tui=false -p "$PORT" >/dev/null 2>&1 &
UP_PID=$!
sleep 5
B_WAITING="$(bstate)"
echo "   b while waiting on a:   $B_WAITING"
check "b waits on a" "$(field "$B_WAITING" .status)" "Pending"

echo "2. process restart b  (while b is still waiting for a)..."
"$PC" process restart b -p "$PORT" >/dev/null 2>&1
sleep 2
B_RESTARTED="$(bstate)"
echo "   b after restart:        $B_RESTARTED"   # wedge: Terminating, pid=0, exit=0
check "b is not wedged" "$(field "$B_RESTARTED" .status)" "Pending"

echo "3. Trying to recover:"
# These three replies read the same whether or not b is wedged - only the
# state below tells the two apart.
echo "   start b   -> $("$PC" process start b -p "$PORT" 2>&1 | grep -oE 'already running|Successfully started')"
echo "   stop b    -> $("$PC" process stop b -p "$PORT" 2>&1 | grep -oE "Successfully stopped: 'b'|not running")"
"$PC" process restart b -p "$PORT" >/dev/null 2>&1 && echo "   restart b -> reported success"
sleep 2
B_RECOVERED="$(bstate)"
echo "   b after recovery:       $B_RECOVERED"
check "b recovers" "$(field "$B_RECOVERED" .status)" "Pending"

echo "4. Waiting for a to become ready (~15s more)..."
sleep 20
B_READY="$(bstate)"
PROCS_READY="$(ps -eo args | grep -c '[b]\.sh')"
echo "   b after a became ready: $B_READY"       # display used to stay frozen at Terminating
echo "   actual b.sh processes:  $PROCS_READY"   # ran untracked while the display was frozen
check "b is tracked" "$(field "$B_READY" .is_running)" "true"
check "one b.sh runs" "$PROCS_READY" "1"

echo "5. start b again:"
"$PC" process start b -p "$PORT" >/dev/null 2>&1
sleep 2
B_STARTED="$(bstate)"
PROCS_STARTED="$(ps -eo args | grep -c '[b]\.sh')"
echo "   b state:                $B_STARTED"
echo "   actual b.sh processes:  $PROCS_STARTED"  # used to silently duplicate b
check "b is not duplicated" "$PROCS_STARTED" "1"

"$PC" down -p "$PORT" >/dev/null 2>&1
kill "$UP_PID" 2>/dev/null
for p in $(ps -eo pid,args | grep -E '[a]\.sh|[b]\.sh' | awk '{print $1}'); do kill -9 "$p" 2>/dev/null; done

echo
if [ "$FAILURES" -eq 0 ]; then
  printf '%sAll checks passed%s - issue #530 did not reproduce.\n' "$GREEN" "$RESET"
  exit 0
fi
printf '%s%d check(s) failed%s - issue #530 reproduced.\n' "$RED" "$FAILURES" "$RESET"
exit 1
