#!/bin/bash
echo "=== Simulating TypeScript Compiler Watch Mode ==="

simulate_compilation() {
    local status=$1
    local time=$(date +"%H:%M:%S")
    
    # Clear screen like tsc --watch does
    printf "\033[2J\033[H"
    
    echo "[$time] Starting compilation in watch mode..."
    echo ""
    
    if [ "$status" = "success" ]; then
        echo "Found 0 errors. Watching for file changes."
    else
        echo "src/app.ts:15:7 - error TS2322: Type 'string' is not assignable to type 'number'."
        echo ""
        echo "Found 1 error. Watching for file changes."
    fi
}

# Simulate multiple compilations
simulate_compilation "success"
sleep 3
simulate_compilation "error" 
sleep 3
simulate_compilation "success"
sleep 3
simulate_compilation "error"
