#!/bin/bash
echo "=== Testing ANSI Clear Screen Sequences ==="
echo "Line 1: This should disappear"
echo "Line 2: This should also disappear"
echo "Line 3: And this too"
echo "Line 4: All of these lines should be gone after clear"
echo ""
echo "Clearing screen in 3 seconds..."
sleep 3

# ESC[2J - Clear entire screen
# ESC[H - Move cursor to home position (top-left)
printf "\033[2J\033[H"

echo "After clear: This should be the ONLY visible line!"
echo "Success: Screen was cleared using ESC[2J + ESC[H"
