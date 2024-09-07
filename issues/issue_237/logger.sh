#!/usr/bin/env bash

set -euo pipefail

cat << EOF > process-compose.yml
version: "0.5"
log_level: debug

processes:
  sleep:
    command: "sleep 100"
  echo:
    command: |
      echo 1
      sleep .1
      echo 2
EOF

../../bin/process-compose up -u pc.sock -L pc.log --tui=false >/dev/null 2>&1 &

# make sure process-compose has started
for i in {1..5}; do
  if ../../bin/process-compose process list -u pc.sock 2>/dev/null | grep sleep >/dev/null; then
    break;
  fi
  sleep .1
done
if [ "$i" == 5 ]; then
  echo "didn't start"
  exit 1
fi

sleep .2
../../bin/process-compose -u pc.sock process logs echo --tail 5 > echo.log
if ! grep 2 echo.log; then
  echo "waiting longer"
  sleep 10
  ../../bin/process-compose -u pc.sock process logs echo --tail 5
  ../../bin/process-compose down -u pc.sock
  exit 1
fi
../../bin/process-compose down -u pc.sock
