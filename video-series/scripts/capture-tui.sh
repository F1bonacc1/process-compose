#!/usr/bin/env bash
# Capture real Process Compose TUI frames and CLI output as ANSI text.
#   TUI  -> driven inside a detached tmux pane, captured with capture-pane -e
#   CLI  -> run under `script` so the commands see a tty and emit color
# Output: video-series/tui/*.ansi
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEMO="$ROOT/demo"
OUT="$ROOT/tui"
PC="$ROOT/../bin/process-compose"
SESSION="pcshots"
PORT=8099
COLS=120
ROWS=34

mkdir -p "$OUT"
rm -f "$OUT"/*.ansi

TMUXCONF="$(mktemp)"
cat >"$TMUXCONF" <<'EOF'
set -g default-terminal "tmux-256color"
set -ga terminal-overrides ",*256col*:Tc,tmux-256color:Tc,xterm-256color:Tc"
set -g status off
set -g escape-time 0
set -g history-limit 5000
EOF

tm() { tmux -f "$TMUXCONF" "$@"; }

cleanup() {
  tm kill-session -t "$SESSION" 2>/dev/null
  pkill -f 'services/(api|worker|mailer)\.py' 2>/dev/null
  pkill -f 'http\.server 811[12]' 2>/dev/null
  rm -f "$TMUXCONF"
}
trap cleanup EXIT

# --- preflight: nothing may be holding the demo ports ------------------------
cleanup_ports() {
  pkill -f 'services/(api|worker|mailer)\.py' 2>/dev/null
  pkill -f 'http\.server 811[12]' 2>/dev/null
  sleep 1
  for p in 8110 8111 "$PORT"; do
    if ss -ltn 2>/dev/null | grep -q ":$p "; then
      echo "ERROR: port $p still in use, aborting" >&2
      ss -ltnp 2>/dev/null | grep ":$p " >&2
      exit 1
    fi
  done
}
tm kill-session -t "$SESSION" 2>/dev/null
cleanup_ports
echo "preflight ok, ports free"

# --- helpers -----------------------------------------------------------------
# Every key press is followed by a settle delay. Sending two keys back to back
# makes tcell merge the escape sequences and the second key gets eaten.
send() { tm send-keys -t "$SESSION" "$@"; sleep 0.45; }

grab() { # grab <name> [settle_seconds]
  local name="$1" settle="${2:-0.9}"
  sleep "$settle"
  tm capture-pane -p -e -t "$SESSION" >"$OUT/$name.ansi"
  printf '  tui  %-26s %6s bytes\n' "$name" "$(wc -c <"$OUT/$name.ansi")"
}

cli() { # cli <name> <command...>
  local name="$1"; shift
  script -qec "$*" /dev/null 2>/dev/null | sed 's/\r$//' >"$OUT/$name.ansi"
  printf '  cli  %-26s %6s bytes\n' "$name" "$(wc -c <"$OUT/$name.ansi")"
}

# --- boot --------------------------------------------------------------------
echo "starting process-compose in tmux (${COLS}x${ROWS})"
rm -f "$DEMO/data/linkstash.db"
tm new-session -d -s "$SESSION" -x "$COLS" -y "$ROWS" \
  -c "$DEMO" "TERM=tmux-256color $PC up -p $PORT -r 500ms"

grab 01-boot-pending 1.3
grab 02-boot-launching 2.2

# steady state: migrate done, api healthy, workers up, web serving
grab 03-running 16

# --- CLI captures against the live server ------------------------------------
echo "capturing CLI output"
cd "$DEMO"
cli 20-cli-list        "$PC process list -o wide -p $PORT"
cli 21-cli-graph       "$PC graph -p $PORT"
cli 22-cli-graph-mermaid "$PC graph --format mermaid -p $PORT"
cli 23-cli-state       "$PC project state -p $PORT"
cli 24-cli-logs        "$PC process logs worker-0 -n 12 -p $PORT"
cli 25-cli-critical    "$PC analyze critical-chain -p $PORT"
cli 26-cli-namespaces  "$PC namespace list -p $PORT"
cli 27-cli-ports       "$PC process ports api -p $PORT"
cli 28-cli-info        "$PC info"
cd - >/dev/null

# --- TUI navigation ----------------------------------------------------------
# Table is sorted by NAME: api docs mailer migrate smoke-test vacuum web worker-0 worker-1
send Down; send Down; send Down; send Down   # -> smoke-test
send Up; send Up; send Up                    # -> docs ... land deterministically below
send Home 2>/dev/null || true
grab 04-table-focus 1.0

# F3 process info on the first row (api)
send F3
grab 05-process-info 1.4
send Escape

# F1 help
send F1
grab 06-help 1.4
send Escape

# Ctrl+Q dependency graph
send C-q
grab 07-dependency-graph 1.6
send Escape

# ':' command palette
send ':'
grab 08-command-palette 1.4
send Escape

# Ctrl+T theme selector
send C-t
grab 09-theme-selector 1.4
send Escape

# '/' process filter
send '/'
send 'work'
grab 10-process-filter 1.2
send Escape

# 'n' namespace operations
send 'n'
grab 11-namespace-ops 1.4
send Escape

# Ctrl+G namespace filter.
send C-g
grab 12-ns-filter 1.4
send 'x'

# F2 scale form (move to worker-0 first)
send Down; send Down; send Down; send Down; send Down; send Down; send Down
send F2
grab 13-scale-form 1.4
send Escape

# Ctrl+X signal dialog
send C-x
grab 14-signal-dialog 1.4
send Escape

# F4 maximized log view
send F4
grab 15-log-fullscreen 1.6

# Ctrl+F find in logs
send C-f
send 'batch'
grab 16-log-search 1.4
send Escape
send F4

# stop a process to show Terminating / Completed
send F9
grab 17-after-stop 2.5

echo "shutting down"
send F10
sleep 1
send Enter
sleep 4

echo
echo "frames written to $OUT"
ls -1 "$OUT" | sed 's/^/  /'
