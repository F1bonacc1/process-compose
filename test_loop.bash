#!/usr/bin/env bash

LOOPS=3
for (( i=1; i<=LOOPS; i++ ))
do
  sleep 1
  
  if [[ -z "${PRINT_ERR}" ]]; then
    echo "test loop $i $1 $ABC"
  else
    echo "test loop $i this is error $1 $PC_PROC_NAME" >&2
  fi
done
exit $EXIT_CODE
