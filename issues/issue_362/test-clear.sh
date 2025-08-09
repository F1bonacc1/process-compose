#!/bin/bash
echo "Before clear"
echo "Line 2"
echo "Line 3"
sleep 2
# Clear screen sequence
printf "\033[2J\033[H"
echo "After clear - this should be the only visible line"
