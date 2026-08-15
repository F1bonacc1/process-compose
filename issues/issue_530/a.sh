#!/usr/bin/env bash
# Slow-starting "server": becomes ready after A_STARTUP_SECS (default 20).
STARTUP="${A_STARTUP_SECS:-20}"
echo "a: starting, ready in ${STARTUP}s"
sleep "$STARTUP"
echo "Server running on port 8080"
while true; do sleep 5; done
