#!/usr/bin/env bash

# Define function to spawn a sub-process
spawn_subprocess() {
  sleep 3600 &
  echo "Spawned subprocess with PID $!"
}

# Continuously spawn sub-processes
while true; do
  spawn_subprocess
  sleep 1
done
