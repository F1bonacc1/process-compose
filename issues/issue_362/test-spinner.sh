#!/bin/bash
chars="/-\|"
for (( i=0; i<50; i++ )); do
    printf "\r${chars:i%4:1} Processing..."
    sleep 0.1
done
printf "\rDone!            \n"
