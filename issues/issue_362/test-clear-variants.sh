#!/bin/bash
echo "=== Testing Different ANSI Clear Variants ==="

# Test 1: ESC[2J + ESC[H (most common)
echo "Test 1: Standard clear (ESC[2J + ESC[H)"
echo "Some content before clear..."
echo "More content..."
sleep 2
printf "\033[2J\033[H"
echo "✓ Cleared with ESC[2J + ESC[H"
sleep 2

# Test 2: ESCc (full reset)
echo "Adding more content..."
echo "This will be cleared with ESCc"
sleep 2
printf "\033c"
echo "✓ Cleared with ESCc (full reset)"
sleep 2

# Test 3: ESC[H + ESC[2J (reverse order)
echo "More test content..."
echo "Will clear with reverse sequence"
sleep 2
printf "\033[H\033[2J"
echo "✓ Cleared with ESC[H + ESC[2J (reverse order)"
