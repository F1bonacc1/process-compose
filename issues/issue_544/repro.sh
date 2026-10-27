#!/usr/bin/env bash
# Deterministic reproduction of issue #544: a late `process_healthy` dependent
# is skipped after its dependency transiently loses readiness, even though the
# dependency never stopped running.
#
# Each observation is checked against the expected (correct) behaviour:
#   PASS - process-compose behaved correctly
#   FAIL - the issue reproduced
# Exits 0 when every check passed, 1 when any check failed and 2 when a
# precondition did not hold (result inconclusive).
#
# Usage: PC=/path/to/process-compose ./repro.sh [port]
set -u
PC="${PC:-process-compose}"
PORT="${1:-19098}"
SRC="$(cd "$(dirname "$0")" && pwd)"

command -v jq >/dev/null || { echo "jq is required"; exit 2; }
# The script runs from a scratch dir, so a relative PC path must be resolved first.
case "$PC" in */*) PC="$(cd "$(dirname "$PC")" && pwd)/$(basename "$PC")" ;; esac

# Keep the probe's `health` file and child.log out of the repo.
WORK="$(mktemp -d)"
cp "$SRC/process-compose.yaml" "$SRC/ready.sh" "$WORK/"
chmod +x "$WORK/ready.sh"
cd "$WORK" || exit 2
LOG="$WORK/pc.log"

cleanup() {
  "$PC" down -p "$PORT" >/dev/null 2>&1
  [ -n "${UP_PID:-}" ] && kill "$UP_PID" 2>/dev/null
  rm -rf "$WORK"
}
trap cleanup EXIT

FAILURES=0
if [ -t 1 ]; then GREEN=$'\033[32m'; RED=$'\033[31m'; YELLOW=$'\033[33m'; RESET=$'\033[0m'; else GREEN=""; RED=""; YELLOW=""; RESET=""; fi

# state <process> - one-line JSON summary of a process
state() {
  "$PC" process list -o json -p "$PORT" 2>/dev/null \
    | jq -c --arg n "$1" '.[]|select(.name==$n)|{status,is_ready,exit_code,restarts}'
}

# field <json> <jq filter> - reads one value out of a captured state
field() {
  local value
  value="$(printf '%s' "${1:-}" | jq -r "$2" 2>/dev/null)"
  printf '%s' "${value:-unknown}"
}

# wait_for <process> <jq filter> <expected> <timeout seconds>
wait_for() {
  local i
  for ((i = 0; i < $4 * 10; i++)); do
    [ "$(field "$(state "$1")" "$2")" = "$3" ] && return 0
    sleep 0.1
  done
  return 1
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

# require <description> <actual> <expected> - a setup step that must hold for
# the remaining checks to mean anything
require() {
  if [ "$2" != "$3" ]; then
    printf '   %sINCONCLUSIVE%s %s: got %s, want %s\n' "$YELLOW" "$RESET" "$1" "$2" "$3"
    exit 2
  fi
}

printf '%s\n' ready > health

echo "1. Starting parent (readiness probe passes)..."
"$PC" -f process-compose.yaml -p "$PORT" -L "$LOG" up parent -t=false >/dev/null 2>&1 &
UP_PID=$!
wait_for parent .is_ready Ready 10
P1="$(state parent)"
echo "   parent: $P1"
require "parent is Running" "$(field "$P1" .status)" "Running"
require "parent is Ready" "$(field "$P1" .is_ready)" "Ready"

echo "2. Making the still-running parent NotReady (nonfatal probe failure)..."
printf '%s\n' not-ready > health
wait_for parent .is_ready "Not Ready" 5
P2="$(state parent)"
echo "   parent: $P2"
require "parent is still Running" "$(field "$P2" .status)" "Running"
require "parent is Not Ready" "$(field "$P2" .is_ready)" "Not Ready"

echo "3. Starting child (process_healthy on parent) while parent is NotReady..."
echo "   cli: $("$PC" process start child -p "$PORT" 2>&1)"
sleep 2
C3="$(state child)"
P3="$(state parent)"
echo "   child:  $C3"
echo "   parent: $P3"
# A disabled process started by hand reports Disabled (not Pending) while it waits.
check "child waits for parent" "$(field "$C3" 'if .status == "Pending" or .status == "Disabled" then "waiting" else .status end')" "waiting"
check "parent exit code untouched" "$(field "$P3" .exit_code)" "0"

echo "4. Letting the same parent recover without a restart..."
printf '%s\n' ready > health
wait_for parent .is_ready Ready 5
wait_for child .status Running 5
C4="$(state child)"
P4="$(state parent)"
echo "   child:  $C4"
echo "   parent: $P4"
require "parent recovered without restart" "$(field "$P4" .restarts)" "0"
check "child runs once parent is Ready" "$(field "$C4" .status)" "Running"
check "child command actually started" "$(grep -c CHILD_STARTED child.log 2>/dev/null || echo 0)" "1"
check "parent exit code untouched" "$(field "$P4" .exit_code)" "0"

echo "5. process-compose log excerpt:"
grep -E "is waiting for parent|won't become ready|depended on parent|won't run" "$LOG" 2>/dev/null \
  | sed -E 's/^/   /' || echo "   (no dependency errors logged)"

echo
if [ "$FAILURES" -eq 0 ]; then
  printf '%sAll checks passed%s - issue #544 did not reproduce.\n' "$GREEN" "$RESET"
  exit 0
fi
printf '%s%d check(s) failed%s - issue #544 reproduced.\n' "$RED" "$FAILURES" "$RESET"
exit 1
