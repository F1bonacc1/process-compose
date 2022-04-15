#!/usr/bin/env bash

$LOOPS=3
foreach($i in 1..$LOOPS)
{
  Start-Sleep 1
  
  if (Test-Path 'env:PRINT_ERR') { 

    # Write-Warning "test loop $i this is error $Args[0] $Env:PC_PROC_NAME"
    $Host.UI.WriteErrorLine("test loop $i this is error " + $Args[0] + " " + $Env:PC_PROC_NAME)
  }
  else{
    Write-Host "test loop $i"$Args[0] [$Env:ABC]
  }
}

exit $Env:EXIT_CODE
