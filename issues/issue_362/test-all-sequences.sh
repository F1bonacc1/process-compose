#!/bin/bash
echo "=== Comprehensive ANSI Sequence Test ==="

test_sequence() {
    local name=$1
    local sequence=$2
    local description=$3
    
    echo "Testing: $name"
    echo "Description: $description"
    echo "Sequence: $sequence (hex representation)"
    echo "Some content that should be cleared..."
    echo "More content..."
    echo "Even more content..."
    sleep 2
    
    # Apply the sequence
    printf "$sequence"
    
    echo "✓ Applied $name"
    echo "If you can see this and NOT the content above, it worked!"
    sleep 3
    echo "---"
}

# Test each sequence individually
test_sequence "ESC[2J" "\033[2J" "Clear entire screen (but cursor stays)"
test_sequence "ESC[H" "\033[H" "Move cursor to home (1,1)"
test_sequence "ESC[2J+ESC[H" "\033[2J\033[H" "Clear screen and move to home (most common)"
test_sequence "ESCc" "\033c" "Full terminal reset"
test_sequence "ESC[H+ESC[2J" "\033[H\033[2J" "Move to home then clear"

echo "All tests completed!"
