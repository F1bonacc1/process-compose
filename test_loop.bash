#!/usr/bin/env bash

LOOPS=30000
for (( i=1; i<=LOOPS; i++ ))
do
  sleep 0.01

  if [[ -z "${PRINT_ERR}" ]]; then
    echo "test loop $i loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop loop $1 $ABC"
  else
    echo "test loop $i this is error $1 $PC_PROC_NAME" >&2
  fi

  if [[ $i -eq 7 ]]; then
    echo "test loop $i this is error $1 $PC_PROC_NAME" >&2
  fi

done
exit $EXIT_CODE
